package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/zeevdr/decree/api/centralconfig/v1"
	"github.com/zeevdr/decree/internal/storage/domain"
)

// computeChecksum computes a checksum for a config value using xxHash.
func computeChecksum(value string) string {
	h := xxhash.Sum64String(value)
	return strconv.FormatUint(h, 16)
}

// checksumPtr computes a checksum for a *string value. Returns nil for nil input.
func checksumPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cs := computeChecksum(*value)
	return &cs
}

// configVersionToProto converts a domain config version to proto.
func configVersionToProto(v domain.ConfigVersion) *pb.ConfigVersion {
	result := &pb.ConfigVersion{
		Id:        v.ID,
		TenantId:  v.TenantID,
		Version:   v.Version,
		CreatedBy: v.CreatedBy,
		CreatedAt: timestamppb.New(v.CreatedAt),
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

// strPtr returns a pointer to s (always non-nil, even for empty string).
func strPtr(s string) *string {
	return &s
}

// derefString safely dereferences a *string, returning "" for nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// --- TypedValue <-> string conversion ---

// typedValueToString serializes a TypedValue to its string representation for DB storage.
// Returns nil for a nil (null) TypedValue.
func typedValueToString(tv *pb.TypedValue) *string {
	if tv == nil {
		return nil
	}
	var s string
	switch v := tv.Kind.(type) {
	case *pb.TypedValue_IntegerValue:
		s = strconv.FormatInt(v.IntegerValue, 10)
	case *pb.TypedValue_NumberValue:
		s = strconv.FormatFloat(v.NumberValue, 'f', -1, 64)
	case *pb.TypedValue_StringValue:
		s = v.StringValue
	case *pb.TypedValue_BoolValue:
		s = strconv.FormatBool(v.BoolValue)
	case *pb.TypedValue_TimeValue:
		if v.TimeValue != nil {
			s = v.TimeValue.AsTime().Format(time.RFC3339Nano)
		}
	case *pb.TypedValue_DurationValue:
		if v.DurationValue != nil {
			s = v.DurationValue.AsDuration().String()
		}
	case *pb.TypedValue_UrlValue:
		s = v.UrlValue
	case *pb.TypedValue_JsonValue:
		s = v.JsonValue
	default:
		return nil
	}
	return &s
}

// stringToTypedValue converts a DB string value to a TypedValue using the field type.
// Returns nil for a nil (null) string.
func stringToTypedValue(s *string, ft domain.FieldType) *pb.TypedValue {
	if s == nil {
		return nil
	}
	switch ft {
	case domain.FieldTypeInteger:
		v, _ := strconv.ParseInt(*s, 10, 64)
		return &pb.TypedValue{Kind: &pb.TypedValue_IntegerValue{IntegerValue: v}}
	case domain.FieldTypeNumber:
		v, _ := strconv.ParseFloat(*s, 64)
		return &pb.TypedValue{Kind: &pb.TypedValue_NumberValue{NumberValue: v}}
	case domain.FieldTypeString:
		return &pb.TypedValue{Kind: &pb.TypedValue_StringValue{StringValue: *s}}
	case domain.FieldTypeBool:
		v, _ := strconv.ParseBool(*s)
		return &pb.TypedValue{Kind: &pb.TypedValue_BoolValue{BoolValue: v}}
	case domain.FieldTypeTime:
		t, _ := time.Parse(time.RFC3339Nano, *s)
		return &pb.TypedValue{Kind: &pb.TypedValue_TimeValue{TimeValue: timestamppb.New(t)}}
	case domain.FieldTypeDuration:
		d, _ := time.ParseDuration(*s)
		return &pb.TypedValue{Kind: &pb.TypedValue_DurationValue{DurationValue: durationpb.New(d)}}
	case domain.FieldTypeURL:
		return &pb.TypedValue{Kind: &pb.TypedValue_UrlValue{UrlValue: *s}}
	case domain.FieldTypeJSON:
		return &pb.TypedValue{Kind: &pb.TypedValue_JsonValue{JsonValue: *s}}
	default:
		return &pb.TypedValue{Kind: &pb.TypedValue_StringValue{StringValue: *s}}
	}
}

// typedValueChecksum computes a checksum for a TypedValue by serializing to string first.
func typedValueChecksum(tv *pb.TypedValue) string { //nolint:unused // used in Phase 2 validation
	s := typedValueToString(tv)
	if s == nil {
		return computeChecksum("")
	}
	return computeChecksum(*s)
}

// typedValueToDisplayString returns a human-readable string for a TypedValue (for audit/events).
func typedValueToDisplayString(tv *pb.TypedValue) string {
	s := typedValueToString(tv)
	if s == nil {
		return ""
	}
	return *s
}

// validateTypedValueType checks that a TypedValue matches the expected field type.
func validateTypedValueType(tv *pb.TypedValue, expected domain.FieldType) error { //nolint:unused // used in Phase 2 validation
	if tv == nil {
		return nil // null is valid for any type (nullable check is separate)
	}
	switch expected {
	case domain.FieldTypeInteger:
		if _, ok := tv.Kind.(*pb.TypedValue_IntegerValue); !ok {
			return fmt.Errorf("expected integer value")
		}
	case domain.FieldTypeNumber:
		if _, ok := tv.Kind.(*pb.TypedValue_NumberValue); !ok {
			return fmt.Errorf("expected number value")
		}
	case domain.FieldTypeString:
		if _, ok := tv.Kind.(*pb.TypedValue_StringValue); !ok {
			return fmt.Errorf("expected string value")
		}
	case domain.FieldTypeBool:
		if _, ok := tv.Kind.(*pb.TypedValue_BoolValue); !ok {
			return fmt.Errorf("expected bool value")
		}
	case domain.FieldTypeTime:
		if _, ok := tv.Kind.(*pb.TypedValue_TimeValue); !ok {
			return fmt.Errorf("expected time value")
		}
	case domain.FieldTypeDuration:
		if _, ok := tv.Kind.(*pb.TypedValue_DurationValue); !ok {
			return fmt.Errorf("expected duration value")
		}
	case domain.FieldTypeURL:
		if v, ok := tv.Kind.(*pb.TypedValue_UrlValue); ok {
			u, err := url.Parse(v.UrlValue)
			if err != nil || !u.IsAbs() {
				return fmt.Errorf("invalid absolute URL")
			}
		} else {
			return fmt.Errorf("expected url value")
		}
	case domain.FieldTypeJSON:
		if v, ok := tv.Kind.(*pb.TypedValue_JsonValue); ok {
			if !json.Valid([]byte(v.JsonValue)) {
				return fmt.Errorf("invalid JSON")
			}
		} else {
			return fmt.Errorf("expected json value")
		}
	}
	return nil
}
