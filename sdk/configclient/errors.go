// Package configclient provides an ergonomic Go client for reading and writing
// configuration values via the Central Config Service gRPC API.
//
// This is an application-runtime SDK — for admin operations (schema management,
// import/export, rollback) see the adminclient package.
package configclient

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrNotFound is returned when a requested field or version does not exist.
	ErrNotFound = errors.New("not found")

	// ErrLocked is returned when attempting to write a field that is locked.
	ErrLocked = errors.New("field is locked")

	// ErrChecksumMismatch is returned when an optimistic concurrency check fails
	// because the value was modified between read and write.
	ErrChecksumMismatch = errors.New("checksum mismatch: value was modified")

	// ErrAlreadyExists is returned when attempting to create a resource that already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrTypeMismatch is returned when a typed getter is called on a field
	// whose value type doesn't match (e.g. GetInt on a string field).
	ErrTypeMismatch = errors.New("value type mismatch")
)

// mapError converts gRPC status errors to sentinel errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.NotFound:
		return ErrNotFound
	case codes.PermissionDenied:
		return ErrLocked
	case codes.Aborted:
		return ErrChecksumMismatch
	case codes.AlreadyExists:
		return ErrAlreadyExists
	default:
		return err
	}
}
