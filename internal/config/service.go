package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
	"github.com/zeevdr/central-config-service/internal/auth"
	"github.com/zeevdr/central-config-service/internal/cache"
	"github.com/zeevdr/central-config-service/internal/pubsub"
	"github.com/zeevdr/central-config-service/internal/storage/dbstore"
)

const defaultCacheTTL = 5 * time.Minute

// Service implements the ConfigService gRPC server.
type Service struct {
	pb.UnimplementedConfigServiceServer
	store      Store
	cache      cache.ConfigCache
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber
	logger     *slog.Logger
}

// NewService creates a new ConfigService.
func NewService(store Store, cache cache.ConfigCache, pub pubsub.Publisher, sub pubsub.Subscriber, logger *slog.Logger) *Service {
	return &Service{
		store:      store,
		cache:      cache,
		publisher:  pub,
		subscriber: sub,
		logger:     logger,
	}
}

// --- Read operations ---

func (s *Service) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}

	// Resolve version.
	version, err := s.resolveVersion(ctx, tenantID, req.Version)
	if err != nil {
		return nil, err
	}

	// If descriptions not requested, try cache.
	if !req.IncludeDescriptions {
		if cached, err := s.cache.Get(ctx, req.TenantId, version); err == nil && cached != nil {
			values := make([]*pb.ConfigValue, 0, len(cached))
			for path, val := range cached {
				values = append(values, &pb.ConfigValue{
					FieldPath: path,
					Value:     val,
					Checksum:  computeChecksum(val),
				})
			}
			return &pb.GetConfigResponse{
				Config: &pb.Config{TenantId: req.TenantId, Version: version, Values: values},
			}, nil
		}
	}

	// Fetch from DB.
	rows, err := s.store.GetFullConfigAtVersion(ctx, dbstore.GetFullConfigAtVersionParams{
		TenantID: tenantID,
		Version:  version,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get config")
	}

	values := make([]*pb.ConfigValue, 0, len(rows))
	cacheMap := make(map[string]string, len(rows))
	for _, row := range rows {
		cv := &pb.ConfigValue{
			FieldPath: row.FieldPath,
			Value:     row.Value,
			Checksum:  computeChecksum(row.Value),
		}
		if req.IncludeDescriptions && row.Description != nil {
			cv.Description = row.Description
		}
		values = append(values, cv)
		cacheMap[row.FieldPath] = row.Value
	}

	// Populate cache (values only, no descriptions).
	if !req.IncludeDescriptions {
		if err := s.cache.Set(ctx, req.TenantId, version, cacheMap, defaultCacheTTL); err != nil {
			s.logger.WarnContext(ctx, "failed to populate cache", "error", err)
		}
	}

	return &pb.GetConfigResponse{
		Config: &pb.Config{TenantId: req.TenantId, Version: version, Values: values},
	}, nil
}

func (s *Service) GetField(ctx context.Context, req *pb.GetFieldRequest) (*pb.GetFieldResponse, error) {
	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}

	version, err := s.resolveVersion(ctx, tenantID, req.Version)
	if err != nil {
		return nil, err
	}

	row, err := s.store.GetConfigValueAtVersion(ctx, dbstore.GetConfigValueAtVersionParams{
		TenantID:  tenantID,
		FieldPath: req.FieldPath,
		Version:   version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "field not found")
		}
		return nil, status.Error(codes.Internal, "failed to get field")
	}

	cv := &pb.ConfigValue{
		FieldPath: row.FieldPath,
		Value:     row.Value,
		Checksum:  computeChecksum(row.Value),
	}
	if req.IncludeDescription && row.Description != nil {
		cv.Description = row.Description
	}

	return &pb.GetFieldResponse{Value: cv}, nil
}

func (s *Service) GetFields(ctx context.Context, req *pb.GetFieldsRequest) (*pb.GetFieldsResponse, error) {
	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}

	version, err := s.resolveVersion(ctx, tenantID, req.Version)
	if err != nil {
		return nil, err
	}

	values := make([]*pb.ConfigValue, 0, len(req.FieldPaths))
	for _, path := range req.FieldPaths {
		row, err := s.store.GetConfigValueAtVersion(ctx, dbstore.GetConfigValueAtVersionParams{
			TenantID:  tenantID,
			FieldPath: path,
			Version:   version,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue // Skip missing fields.
			}
			return nil, status.Error(codes.Internal, "failed to get field")
		}
		cv := &pb.ConfigValue{
			FieldPath: row.FieldPath,
			Value:     row.Value,
			Checksum:  computeChecksum(row.Value),
		}
		if req.IncludeDescriptions && row.Description != nil {
			cv.Description = row.Description
		}
		values = append(values, cv)
	}

	return &pb.GetFieldsResponse{Values: values}, nil
}

// --- Write operations ---

func (s *Service) SetField(ctx context.Context, req *pb.SetFieldRequest) (*pb.SetFieldResponse, error) {
	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}

	actor := s.getActor(ctx)

	// Optimistic concurrency check.
	if req.ExpectedChecksum != nil {
		if err := s.checkChecksum(ctx, tenantID, req.FieldPath, *req.ExpectedChecksum); err != nil {
			return nil, err
		}
	}

	// Check field locks.
	if err := s.checkFieldLock(ctx, tenantID, req.FieldPath); err != nil {
		return nil, err
	}

	// Get latest version to derive from.
	latestVersion, err := s.getOrCreateVersion(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Get old value for audit.
	oldValue := s.getCurrentValue(ctx, tenantID, req.FieldPath, latestVersion)

	// Create new config version.
	newVersion, err := s.store.CreateConfigVersion(ctx, dbstore.CreateConfigVersionParams{
		TenantID:    tenantID,
		Version:     latestVersion + 1,
		Description: ptrString(req.GetDescription()),
		CreatedBy:   actor,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "create config version", "error", err)
		return nil, status.Error(codes.Internal, "failed to create config version")
	}

	// Set the value.
	if err := s.store.SetConfigValue(ctx, dbstore.SetConfigValueParams{
		ConfigVersionID: newVersion.ID,
		FieldPath:       req.FieldPath,
		Value:           req.Value,
		Description:     ptrString(req.GetValueDescription()),
	}); err != nil {
		s.logger.ErrorContext(ctx, "set config value", "error", err)
		return nil, status.Error(codes.Internal, "failed to set config value")
	}

	// Invalidate cache.
	if err := s.cache.Invalidate(ctx, req.TenantId); err != nil {
		s.logger.WarnContext(ctx, "failed to invalidate cache", "error", err)
	}

	// Publish change event.
	s.publishChange(ctx, req.TenantId, newVersion.Version, req.FieldPath, oldValue, req.Value, actor)

	// Audit log.
	s.auditWrite(ctx, tenantID, actor, "set_field", req.FieldPath, oldValue, req.Value, newVersion.Version)

	return &pb.SetFieldResponse{ConfigVersion: configVersionToProto(newVersion)}, nil
}

func (s *Service) SetFields(ctx context.Context, req *pb.SetFieldsRequest) (*pb.SetFieldsResponse, error) {
	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}

	actor := s.getActor(ctx)

	// Check all checksums and locks first.
	for _, update := range req.Updates {
		if update.ExpectedChecksum != nil {
			if err := s.checkChecksum(ctx, tenantID, update.FieldPath, *update.ExpectedChecksum); err != nil {
				return nil, err
			}
		}
		if err := s.checkFieldLock(ctx, tenantID, update.FieldPath); err != nil {
			return nil, err
		}
	}

	latestVersion, err := s.getOrCreateVersion(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Create new config version.
	newVersion, err := s.store.CreateConfigVersion(ctx, dbstore.CreateConfigVersionParams{
		TenantID:    tenantID,
		Version:     latestVersion + 1,
		Description: ptrString(req.GetDescription()),
		CreatedBy:   actor,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "create config version", "error", err)
		return nil, status.Error(codes.Internal, "failed to create config version")
	}

	// Set all values and audit each.
	for _, update := range req.Updates {
		oldValue := s.getCurrentValue(ctx, tenantID, update.FieldPath, latestVersion)

		if err := s.store.SetConfigValue(ctx, dbstore.SetConfigValueParams{
			ConfigVersionID: newVersion.ID,
			FieldPath:       update.FieldPath,
			Value:           update.Value,
			Description:     ptrString(update.GetValueDescription()),
		}); err != nil {
			s.logger.ErrorContext(ctx, "set config value", "field", update.FieldPath, "error", err)
			return nil, status.Errorf(codes.Internal, "failed to set field %s", update.FieldPath)
		}

		s.publishChange(ctx, req.TenantId, newVersion.Version, update.FieldPath, oldValue, update.Value, actor)
		s.auditWrite(ctx, tenantID, actor, "set_field", update.FieldPath, oldValue, update.Value, newVersion.Version)
	}

	// Invalidate cache.
	if err := s.cache.Invalidate(ctx, req.TenantId); err != nil {
		s.logger.WarnContext(ctx, "failed to invalidate cache", "error", err)
	}

	return &pb.SetFieldsResponse{ConfigVersion: configVersionToProto(newVersion)}, nil
}

// --- Version operations ---

func (s *Service) ListVersions(ctx context.Context, req *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}

	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	versions, err := s.store.ListConfigVersions(ctx, dbstore.ListConfigVersionsParams{
		TenantID: tenantID,
		Limit:    pageSize,
		Offset:   0,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list versions")
	}

	pbVersions := make([]*pb.ConfigVersion, 0, len(versions))
	for _, v := range versions {
		pbVersions = append(pbVersions, configVersionToProto(v))
	}

	return &pb.ListVersionsResponse{Versions: pbVersions}, nil
}

func (s *Service) GetVersion(ctx context.Context, req *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}

	version, err := s.store.GetConfigVersion(ctx, dbstore.GetConfigVersionParams{
		TenantID: tenantID,
		Version:  req.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "version not found")
		}
		return nil, status.Error(codes.Internal, "failed to get version")
	}

	return &pb.GetVersionResponse{ConfigVersion: configVersionToProto(version)}, nil
}

func (s *Service) RollbackToVersion(ctx context.Context, req *pb.RollbackToVersionRequest) (*pb.RollbackToVersionResponse, error) {
	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}

	actor := s.getActor(ctx)

	// Get the target version's full config.
	targetRows, err := s.store.GetFullConfigAtVersion(ctx, dbstore.GetFullConfigAtVersionParams{
		TenantID: tenantID,
		Version:  req.Version,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get target version config")
	}
	if len(targetRows) == 0 {
		return nil, status.Error(codes.NotFound, "target version not found or empty")
	}

	// Get latest version number.
	latest, err := s.store.GetLatestConfigVersion(ctx, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get latest version")
	}

	// Create new version as rollback.
	desc := fmt.Sprintf("Rollback to version %d", req.Version)
	if req.Description != nil {
		desc = *req.Description
	}
	newVersion, err := s.store.CreateConfigVersion(ctx, dbstore.CreateConfigVersionParams{
		TenantID:    tenantID,
		Version:     latest.Version + 1,
		Description: &desc,
		CreatedBy:   actor,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create rollback version")
	}

	// Copy all values from target version.
	for _, row := range targetRows {
		if err := s.store.SetConfigValue(ctx, dbstore.SetConfigValueParams{
			ConfigVersionID: newVersion.ID,
			FieldPath:       row.FieldPath,
			Value:           row.Value,
			Description:     row.Description,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to copy field %s", row.FieldPath)
		}
	}

	// Invalidate cache.
	if err := s.cache.Invalidate(ctx, req.TenantId); err != nil {
		s.logger.WarnContext(ctx, "failed to invalidate cache", "error", err)
	}

	// Audit.
	s.auditWrite(ctx, tenantID, actor, "rollback", "", "", fmt.Sprintf("v%d", req.Version), newVersion.Version)

	return &pb.RollbackToVersionResponse{ConfigVersion: configVersionToProto(newVersion)}, nil
}

// --- Subscriptions ---

func (s *Service) Subscribe(req *pb.SubscribeRequest, stream grpc.ServerStreamingServer[pb.SubscribeResponse]) error {
	ctx := stream.Context()

	events, cancel, err := s.subscriber.Subscribe(ctx, req.TenantId)
	if err != nil {
		return status.Error(codes.Internal, "failed to subscribe")
	}
	defer cancel()

	filterPaths := make(map[string]struct{}, len(req.FieldPaths))
	for _, p := range req.FieldPaths {
		filterPaths[p] = struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}

			// Filter by field paths if specified.
			if len(filterPaths) > 0 {
				if _, ok := filterPaths[event.FieldPath]; !ok {
					continue
				}
			}

			if err := stream.Send(&pb.SubscribeResponse{
				Change: &pb.ConfigChange{
					TenantId:  event.TenantID,
					Version:   event.Version,
					FieldPath: event.FieldPath,
					OldValue:  event.OldValue,
					NewValue:  event.NewValue,
					ChangedBy: event.ChangedBy,
					ChangedAt: timestamppb.New(event.ChangedAt),
				},
			}); err != nil {
				return err
			}
		}
	}
}

// --- Import/export ---

func (s *Service) ExportConfig(ctx context.Context, req *pb.ExportConfigRequest) (*pb.ExportConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "export not yet implemented")
}

func (s *Service) ImportConfig(ctx context.Context, req *pb.ImportConfigRequest) (*pb.ImportConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "import not yet implemented")
}

// --- Helpers ---

func (s *Service) resolveVersion(ctx context.Context, tenantID pgtype.UUID, requested *int32) (int32, error) {
	if requested != nil {
		return *requested, nil
	}
	latest, err := s.store.GetLatestConfigVersion(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil // No versions yet.
		}
		return 0, status.Error(codes.Internal, "failed to get latest version")
	}
	return latest.Version, nil
}

func (s *Service) getOrCreateVersion(ctx context.Context, tenantID pgtype.UUID) (int32, error) {
	latest, err := s.store.GetLatestConfigVersion(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, status.Error(codes.Internal, "failed to get latest version")
	}
	return latest.Version, nil
}

func (s *Service) getActor(ctx context.Context) string {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return "unknown"
	}
	return claims.Subject
}

func (s *Service) getCurrentValue(ctx context.Context, tenantID pgtype.UUID, fieldPath string, version int32) string {
	if version == 0 {
		return ""
	}
	row, err := s.store.GetConfigValueAtVersion(ctx, dbstore.GetConfigValueAtVersionParams{
		TenantID:  tenantID,
		FieldPath: fieldPath,
		Version:   version,
	})
	if err != nil {
		return ""
	}
	return row.Value
}

func (s *Service) checkChecksum(ctx context.Context, tenantID pgtype.UUID, fieldPath, expected string) error {
	latest, err := s.store.GetLatestConfigVersion(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // No existing value — checksum check passes.
		}
		return status.Error(codes.Internal, "failed to get latest version")
	}
	row, err := s.store.GetConfigValueAtVersion(ctx, dbstore.GetConfigValueAtVersionParams{
		TenantID:  tenantID,
		FieldPath: fieldPath,
		Version:   latest.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // Field doesn't exist yet.
		}
		return status.Error(codes.Internal, "failed to get current value for checksum")
	}
	actual := computeChecksum(row.Value)
	if actual != expected {
		return status.Errorf(codes.Aborted, "checksum mismatch for %s: expected %s, got %s", fieldPath, expected, actual)
	}
	return nil
}

func (s *Service) checkFieldLock(ctx context.Context, tenantID pgtype.UUID, fieldPath string) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims.Role == auth.RoleSuperAdmin {
		return nil // SuperAdmin bypasses locks.
	}

	locks, err := s.store.GetFieldLocks(ctx, tenantID)
	if err != nil {
		return status.Error(codes.Internal, "failed to check field locks")
	}
	for _, lock := range locks {
		if lock.FieldPath == fieldPath {
			return status.Errorf(codes.PermissionDenied, "field %s is locked", fieldPath)
		}
	}
	return nil
}

func (s *Service) publishChange(ctx context.Context, tenantID string, version int32, fieldPath, oldValue, newValue, actor string) {
	event := pubsub.ConfigChangeEvent{
		TenantID:  tenantID,
		Version:   version,
		FieldPath: fieldPath,
		OldValue:  oldValue,
		NewValue:  newValue,
		ChangedBy: actor,
		ChangedAt: time.Now(),
	}
	if err := s.publisher.Publish(ctx, event); err != nil {
		s.logger.WarnContext(ctx, "failed to publish change event", "error", err)
	}
}

func (s *Service) auditWrite(ctx context.Context, tenantID pgtype.UUID, actor, action, fieldPath, oldValue, newValue string, version int32) {
	if err := s.store.InsertAuditWriteLog(ctx, dbstore.InsertAuditWriteLogParams{
		TenantID:      tenantID,
		Actor:         actor,
		Action:        action,
		FieldPath:     ptrString(fieldPath),
		OldValue:      ptrString(oldValue),
		NewValue:      ptrString(newValue),
		ConfigVersion: &version,
	}); err != nil {
		s.logger.WarnContext(ctx, "failed to write audit log", "error", err)
	}
}
