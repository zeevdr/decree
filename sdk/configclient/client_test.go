package configclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
)

func sp(s string) *string { return &s }

// --- Mock ---

type mockRPC struct {
	mock.Mock
}

func (m *mockRPC) GetConfig(ctx context.Context, in *pb.GetConfigRequest, opts ...grpc.CallOption) (*pb.GetConfigResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.GetConfigResponse), args.Error(1)
}

func (m *mockRPC) GetField(ctx context.Context, in *pb.GetFieldRequest, opts ...grpc.CallOption) (*pb.GetFieldResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.GetFieldResponse), args.Error(1)
}

func (m *mockRPC) GetFields(ctx context.Context, in *pb.GetFieldsRequest, opts ...grpc.CallOption) (*pb.GetFieldsResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.GetFieldsResponse), args.Error(1)
}

func (m *mockRPC) SetField(ctx context.Context, in *pb.SetFieldRequest, opts ...grpc.CallOption) (*pb.SetFieldResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.SetFieldResponse), args.Error(1)
}

func (m *mockRPC) SetFields(ctx context.Context, in *pb.SetFieldsRequest, opts ...grpc.CallOption) (*pb.SetFieldsResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.SetFieldsResponse), args.Error(1)
}

func (m *mockRPC) ListVersions(ctx context.Context, in *pb.ListVersionsRequest, opts ...grpc.CallOption) (*pb.ListVersionsResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.ListVersionsResponse), args.Error(1)
}

func (m *mockRPC) GetVersion(ctx context.Context, in *pb.GetVersionRequest, opts ...grpc.CallOption) (*pb.GetVersionResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.GetVersionResponse), args.Error(1)
}

func (m *mockRPC) RollbackToVersion(ctx context.Context, in *pb.RollbackToVersionRequest, opts ...grpc.CallOption) (*pb.RollbackToVersionResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.RollbackToVersionResponse), args.Error(1)
}

func (m *mockRPC) Subscribe(ctx context.Context, in *pb.SubscribeRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.SubscribeResponse], error) {
	args := m.Called(ctx, in)
	return nil, args.Error(1)
}

func (m *mockRPC) ExportConfig(ctx context.Context, in *pb.ExportConfigRequest, opts ...grpc.CallOption) (*pb.ExportConfigResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.ExportConfigResponse), args.Error(1)
}

func (m *mockRPC) ImportConfig(ctx context.Context, in *pb.ImportConfigRequest, opts ...grpc.CallOption) (*pb.ImportConfigResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.ImportConfigResponse), args.Error(1)
}

// --- Get ---

func TestGet_Success(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc, WithSubject("test"))
	ctx := context.Background()

	rpc.On("GetField", mock.Anything, mock.MatchedBy(func(r *pb.GetFieldRequest) bool {
		return r.TenantId == "t1" && r.FieldPath == "payments.fee"
	})).Return(&pb.GetFieldResponse{
		Value: &pb.ConfigValue{FieldPath: "payments.fee", Value: sp("0.5%"), Checksum: "abc"},
	}, nil)

	val, err := client.Get(ctx, "t1", "payments.fee")
	require.NoError(t, err)
	assert.Equal(t, "0.5%", val)
}

func TestGet_NotFound(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	rpc.On("GetField", mock.Anything, mock.Anything).
		Return((*pb.GetFieldResponse)(nil), status.Error(codes.NotFound, "not found"))

	_, err := client.Get(ctx, "t1", "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- GetAll ---

func TestGetAll_Success(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	rpc.On("GetConfig", mock.Anything, mock.Anything).Return(&pb.GetConfigResponse{
		Config: &pb.Config{
			TenantId: "t1",
			Version:  3,
			Values: []*pb.ConfigValue{
				{FieldPath: "a", Value: sp("1")},
				{FieldPath: "b", Value: sp("2")},
			},
		},
	}, nil)

	vals, err := client.GetAll(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, vals)
}

// --- Set ---

func TestSet_Success(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	rpc.On("SetField", mock.Anything, mock.MatchedBy(func(r *pb.SetFieldRequest) bool {
		return r.TenantId == "t1" && r.FieldPath == "a" && r.Value != nil && *r.Value == "new"
	})).Return(&pb.SetFieldResponse{}, nil)

	err := client.Set(ctx, "t1", "a", "new")
	require.NoError(t, err)
}

func TestSet_Locked(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	rpc.On("SetField", mock.Anything, mock.Anything).
		Return((*pb.SetFieldResponse)(nil), status.Error(codes.PermissionDenied, "locked"))

	err := client.Set(ctx, "t1", "a", "new")
	assert.ErrorIs(t, err, ErrLocked)
}

// --- SetMany ---

func TestSetMany_Success(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	rpc.On("SetFields", mock.Anything, mock.Anything).Return(&pb.SetFieldsResponse{}, nil)

	err := client.SetMany(ctx, "t1", map[string]string{"a": "1", "b": "2"}, "bulk update")
	require.NoError(t, err)
	rpc.AssertCalled(t, "SetFields", mock.Anything, mock.Anything)
}

// --- Snapshot ---

func TestSnapshot_PinnedVersion(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	// Snapshot resolves latest version
	rpc.On("GetConfig", mock.Anything, mock.MatchedBy(func(r *pb.GetConfigRequest) bool {
		return r.Version == nil
	})).Return(&pb.GetConfigResponse{
		Config: &pb.Config{TenantId: "t1", Version: 5},
	}, nil)

	snap, err := client.Snapshot(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int32(5), snap.Version())

	// Subsequent read uses pinned version
	v := int32(5)
	rpc.On("GetField", mock.Anything, mock.MatchedBy(func(r *pb.GetFieldRequest) bool {
		return r.Version != nil && *r.Version == v
	})).Return(&pb.GetFieldResponse{
		Value: &pb.ConfigValue{FieldPath: "a", Value: sp("val")},
	}, nil)

	val, err := snap.Get(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, "val", val)
}

func TestAtVersion(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	snap := client.AtVersion("t1", 3)
	assert.Equal(t, int32(3), snap.Version())

	v := int32(3)
	rpc.On("GetConfig", mock.Anything, mock.MatchedBy(func(r *pb.GetConfigRequest) bool {
		return r.Version != nil && *r.Version == v
	})).Return(&pb.GetConfigResponse{
		Config: &pb.Config{TenantId: "t1", Version: 3, Values: []*pb.ConfigValue{
			{FieldPath: "x", Value: sp("y")},
		}},
	}, nil)

	vals, err := snap.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"x": "y"}, vals)
}

// --- GetForUpdate + LockedValue.Set ---

func TestGetForUpdate_ThenSet(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	rpc.On("GetField", mock.Anything, mock.Anything).Return(&pb.GetFieldResponse{
		Value: &pb.ConfigValue{FieldPath: "a", Value: sp("old"), Checksum: "chk123"},
	}, nil)

	lv, err := client.GetForUpdate(ctx, "t1", "a")
	require.NoError(t, err)
	assert.Equal(t, "old", lv.Value)
	assert.Equal(t, "chk123", lv.Checksum)

	// Write with checksum
	rpc.On("SetField", mock.Anything, mock.MatchedBy(func(r *pb.SetFieldRequest) bool {
		return r.ExpectedChecksum != nil && *r.ExpectedChecksum == "chk123" && r.Value != nil && *r.Value == "new"
	})).Return(&pb.SetFieldResponse{}, nil)

	err = lv.Set(ctx, client, "new")
	require.NoError(t, err)
}

func TestGetForUpdate_ChecksumMismatch(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	rpc.On("GetField", mock.Anything, mock.Anything).Return(&pb.GetFieldResponse{
		Value: &pb.ConfigValue{FieldPath: "a", Value: sp("old"), Checksum: "stale"},
	}, nil)

	lv, err := client.GetForUpdate(ctx, "t1", "a")
	require.NoError(t, err)

	rpc.On("SetField", mock.Anything, mock.Anything).
		Return((*pb.SetFieldResponse)(nil), status.Error(codes.Aborted, "checksum mismatch"))

	err = lv.Set(ctx, client, "new")
	assert.ErrorIs(t, err, ErrChecksumMismatch)
}

// --- Update ---

func TestUpdate_Success(t *testing.T) {
	rpc := &mockRPC{}
	client := New(rpc)
	ctx := context.Background()

	rpc.On("GetField", mock.Anything, mock.Anything).Return(&pb.GetFieldResponse{
		Value: &pb.ConfigValue{FieldPath: "counter", Value: sp("5"), Checksum: "chk"},
	}, nil)

	rpc.On("SetField", mock.Anything, mock.MatchedBy(func(r *pb.SetFieldRequest) bool {
		return r.Value != nil && *r.Value == "6" && r.ExpectedChecksum != nil && *r.ExpectedChecksum == "chk"
	})).Return(&pb.SetFieldResponse{}, nil)

	err := client.Update(ctx, "t1", "counter", func(current string) (string, error) {
		// Simple increment
		return "6", nil
	})
	require.NoError(t, err)
}
