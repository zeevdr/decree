package config

import (
	"crypto/sha256"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
	"github.com/zeevdr/central-config-service/internal/storage/dbstore"
)

// parseUUID parses a string UUID into pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return id, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}

// uuidToString converts pgtype.UUID to string.
func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", id.Bytes[0:4], id.Bytes[4:6], id.Bytes[6:8], id.Bytes[8:10], id.Bytes[10:16])
}

// computeChecksum computes a checksum for a config value.
func computeChecksum(value string) string {
	h := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", h[:8])
}

// configVersionToProto converts a DB config version to proto.
func configVersionToProto(v dbstore.ConfigVersion) *pb.ConfigVersion {
	result := &pb.ConfigVersion{
		Id:        uuidToString(v.ID),
		TenantId:  uuidToString(v.TenantID),
		Version:   v.Version,
		CreatedBy: v.CreatedBy,
		CreatedAt: timestamppb.New(v.CreatedAt.Time),
	}
	if v.Description != nil {
		result.Description = *v.Description
	}
	return result
}

// ptrString returns a pointer to s, or nil if empty.
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
