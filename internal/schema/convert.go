package schema

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

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

// schemaToProto converts a DB schema + version + fields to a proto Schema.
func schemaToProto(s dbstore.Schema, v dbstore.SchemaVersion, fields []dbstore.SchemaField) *pb.Schema {
	pbFields := make([]*pb.SchemaField, 0, len(fields))
	for _, f := range fields {
		pbFields = append(pbFields, fieldToProto(f))
	}

	result := &pb.Schema{
		Id:                 uuidToString(s.ID),
		Name:               s.Name,
		Version:            v.Version,
		Checksum:           v.Checksum,
		Published:          v.Published,
		Fields:             pbFields,
		CreatedAt:          timestamppb.New(v.CreatedAt.Time),
	}
	if s.Description != nil {
		result.Description = *s.Description
	}
	if v.ParentVersion != nil {
		result.ParentVersion = v.ParentVersion
	}
	if v.Description != nil {
		result.VersionDescription = *v.Description
	}
	return result
}

// fieldToProto converts a DB schema field to a proto SchemaField.
func fieldToProto(f dbstore.SchemaField) *pb.SchemaField {
	result := &pb.SchemaField{
		Path:       f.Path,
		Type:       fieldTypeToProto(f.FieldType),
		Nullable:   f.Nullable,
		Deprecated: f.Deprecated,
	}

	if f.Constraints != nil {
		var c pb.FieldConstraints
		if err := json.Unmarshal(f.Constraints, &c); err == nil {
			result.Constraints = &c
		}
	}
	if f.RedirectTo != nil {
		result.RedirectTo = f.RedirectTo
	}
	if f.DefaultValue != nil {
		result.DefaultValue = f.DefaultValue
	}
	if f.Description != nil {
		result.Description = f.Description
	}
	return result
}

// fieldTypeToProto converts a string field type to proto enum.
func fieldTypeToProto(t string) pb.FieldType {
	switch t {
	case "int":
		return pb.FieldType_FIELD_TYPE_INT
	case "string":
		return pb.FieldType_FIELD_TYPE_STRING
	case "time":
		return pb.FieldType_FIELD_TYPE_TIME
	case "duration":
		return pb.FieldType_FIELD_TYPE_DURATION
	case "url":
		return pb.FieldType_FIELD_TYPE_URL
	case "json":
		return pb.FieldType_FIELD_TYPE_JSON
	default:
		return pb.FieldType_FIELD_TYPE_UNSPECIFIED
	}
}

// protoFieldType converts a proto enum to string for DB storage.
func protoFieldType(t pb.FieldType) string {
	switch t {
	case pb.FieldType_FIELD_TYPE_INT:
		return "int"
	case pb.FieldType_FIELD_TYPE_STRING:
		return "string"
	case pb.FieldType_FIELD_TYPE_TIME:
		return "time"
	case pb.FieldType_FIELD_TYPE_DURATION:
		return "duration"
	case pb.FieldType_FIELD_TYPE_URL:
		return "url"
	case pb.FieldType_FIELD_TYPE_JSON:
		return "json"
	default:
		return "unspecified"
	}
}

// tenantToProto converts a DB tenant to a proto Tenant.
func tenantToProto(t dbstore.Tenant) *pb.Tenant {
	return &pb.Tenant{
		Id:            uuidToString(t.ID),
		Name:          t.Name,
		SchemaId:      uuidToString(t.SchemaID),
		SchemaVersion: t.SchemaVersion,
		CreatedAt:     timestamppb.New(t.CreatedAt.Time),
		UpdatedAt:     timestamppb.New(t.UpdatedAt.Time),
	}
}

// fieldLockToProto converts a DB field lock to a proto FieldLock.
func fieldLockToProto(fl dbstore.TenantFieldLock) *pb.FieldLock {
	result := &pb.FieldLock{
		TenantId:  uuidToString(fl.TenantID),
		FieldPath: fl.FieldPath,
	}
	if fl.LockedValues != nil {
		var values []string
		if err := json.Unmarshal(fl.LockedValues, &values); err == nil {
			result.LockedValues = values
		}
	}
	return result
}

// computeChecksum computes a deterministic checksum for a set of schema fields.
func computeChecksum(fields []*pb.SchemaField) string {
	// Sort by path for determinism.
	sorted := make([]*pb.SchemaField, len(fields))
	copy(sorted, fields)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	h := sha256.New()
	for _, f := range sorted {
		fmt.Fprintf(h, "%s:%s:%v:%v", f.Path, f.Type.String(), f.Nullable, f.Deprecated)
		if f.Constraints != nil {
			data, _ := json.Marshal(f.Constraints)
			h.Write(data)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// slugRe matches valid slug names: lowercase alphanumeric with hyphens, not starting/ending with hyphen.
var slugRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// isValidSlug checks if name is a valid slug (lowercase alphanumeric, hyphens allowed, 1-63 chars).
func isValidSlug(name string) bool {
	return len(name) >= 1 && len(name) <= 63 && slugRe.MatchString(name)
}

// ptrString returns a pointer to s, or nil if empty.
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
