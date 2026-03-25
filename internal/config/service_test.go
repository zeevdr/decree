package config

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
	"github.com/zeevdr/central-config-service/internal/auth"
	"github.com/zeevdr/central-config-service/internal/storage/dbstore"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func testUUID(b byte) pgtype.UUID {
	var id pgtype.UUID
	id.Bytes[0] = b
	id.Valid = true
	return id
}

func newTestService() (*Service, *mockStore, *mockCache, *mockPublisher, *mockSubscriber) {
	store := &mockStore{}
	cache := &mockCache{}
	pub := &mockPublisher{}
	sub := &mockSubscriber{}
	svc := NewService(store, cache, pub, sub, testLogger)
	return svc, store, cache, pub, sub
}

// --- GetConfig ---

func TestGetConfig_CacheHit(t *testing.T) {
	svc, store, cache, _, _ := newTestService()
	ctx := context.Background()

	tenantID := testUUID(1)
	tenantIDStr := uuidToString(tenantID)

	store.On("GetLatestConfigVersion", ctx, tenantID).
		Return(dbstore.ConfigVersion{Version: 5}, nil)
	cache.On("Get", ctx, tenantIDStr, int32(5)).
		Return(map[string]string{"payments.fee": "0.5"}, nil)

	resp, err := svc.GetConfig(ctx, &pb.GetConfigRequest{TenantId: tenantIDStr})

	require.NoError(t, err)
	assert.Len(t, resp.Config.Values, 1)
	assert.Equal(t, "payments.fee", resp.Config.Values[0].FieldPath)
	assert.Equal(t, "0.5", resp.Config.Values[0].Value)
	// Should not hit DB.
	store.AssertNotCalled(t, "GetFullConfigAtVersion")
	cache.AssertExpectations(t)
}

func TestGetConfig_CacheMiss(t *testing.T) {
	svc, store, cache, _, _ := newTestService()
	ctx := context.Background()

	tenantID := testUUID(1)
	tenantIDStr := uuidToString(tenantID)

	store.On("GetLatestConfigVersion", ctx, tenantID).
		Return(dbstore.ConfigVersion{Version: 3}, nil)
	cache.On("Get", ctx, tenantIDStr, int32(3)).
		Return(nil, nil)
	store.On("GetFullConfigAtVersion", ctx, dbstore.GetFullConfigAtVersionParams{TenantID: tenantID, Version: 3}).
		Return([]dbstore.GetFullConfigAtVersionRow{
			{FieldPath: "a.b", Value: "123"},
		}, nil)
	cache.On("Set", ctx, tenantIDStr, int32(3), mock.AnythingOfType("map[string]string"), mock.Anything).
		Return(nil)

	resp, err := svc.GetConfig(ctx, &pb.GetConfigRequest{TenantId: tenantIDStr})

	require.NoError(t, err)
	assert.Len(t, resp.Config.Values, 1)
	cache.AssertCalled(t, "Set", ctx, tenantIDStr, int32(3), mock.AnythingOfType("map[string]string"), mock.Anything)
}

func TestGetConfig_IncludeDescriptions_BypassesCache(t *testing.T) {
	svc, store, cache, _, _ := newTestService()
	ctx := context.Background()

	tenantID := testUUID(1)
	tenantIDStr := uuidToString(tenantID)
	desc := "fee per transaction"

	store.On("GetLatestConfigVersion", ctx, tenantID).
		Return(dbstore.ConfigVersion{Version: 1}, nil)
	store.On("GetFullConfigAtVersion", ctx, dbstore.GetFullConfigAtVersionParams{TenantID: tenantID, Version: 1}).
		Return([]dbstore.GetFullConfigAtVersionRow{
			{FieldPath: "fee", Value: "0.5", Description: &desc},
		}, nil)

	resp, err := svc.GetConfig(ctx, &pb.GetConfigRequest{
		TenantId:            tenantIDStr,
		IncludeDescriptions: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "fee per transaction", *resp.Config.Values[0].Description)
	// Cache should NOT be read or written.
	cache.AssertNotCalled(t, "Get")
	cache.AssertNotCalled(t, "Set")
}

// --- SetField ---

func TestSetField_Success(t *testing.T) {
	svc, store, cache, pub, _ := newTestService()
	ctx := context.Background()

	tenantID := testUUID(1)
	tenantIDStr := uuidToString(tenantID)
	versionID := testUUID(2)

	store.On("GetFieldLocks", ctx, tenantID).
		Return([]dbstore.TenantFieldLock{}, nil)
	store.On("GetLatestConfigVersion", ctx, tenantID).
		Return(dbstore.ConfigVersion{}, pgx.ErrNoRows)
	store.On("CreateConfigVersion", ctx, mock.AnythingOfType("dbstore.CreateConfigVersionParams")).
		Return(dbstore.ConfigVersion{ID: versionID, TenantID: tenantID, Version: 1, CreatedBy: "unknown"}, nil)
	store.On("SetConfigValue", ctx, mock.AnythingOfType("dbstore.SetConfigValueParams")).
		Return(nil)
	cache.On("Invalidate", ctx, tenantIDStr).Return(nil)
	pub.On("Publish", ctx, mock.AnythingOfType("pubsub.ConfigChangeEvent")).Return(nil)
	store.On("InsertAuditWriteLog", ctx, mock.AnythingOfType("dbstore.InsertAuditWriteLogParams")).Return(nil)

	resp, err := svc.SetField(ctx, &pb.SetFieldRequest{
		TenantId:  tenantIDStr,
		FieldPath: "payments.fee",
		Value:     "0.5",
	})

	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.ConfigVersion.Version)
	cache.AssertCalled(t, "Invalidate", ctx, tenantIDStr)
	pub.AssertCalled(t, "Publish", ctx, mock.AnythingOfType("pubsub.ConfigChangeEvent"))
}

func TestSetField_ChecksumMismatch(t *testing.T) {
	svc, store, _, _, _ := newTestService()
	ctx := context.Background()

	tenantID := testUUID(1)
	tenantIDStr := uuidToString(tenantID)
	wrongChecksum := "wrong"

	store.On("GetLatestConfigVersion", ctx, tenantID).
		Return(dbstore.ConfigVersion{Version: 1}, nil)
	store.On("GetConfigValueAtVersion", ctx, mock.AnythingOfType("dbstore.GetConfigValueAtVersionParams")).
		Return(dbstore.GetConfigValueAtVersionRow{Value: "old-value"}, nil)

	_, err := svc.SetField(ctx, &pb.SetFieldRequest{
		TenantId:         tenantIDStr,
		FieldPath:        "payments.fee",
		Value:            "0.5",
		ExpectedChecksum: &wrongChecksum,
	})

	require.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))
}

func TestSetField_LockedField(t *testing.T) {
	svc, store, _, _, _ := newTestService()
	// Use admin context — lock checks only apply to non-superadmin.
	ctx := auth.ContextWithClaims(context.Background(), &auth.Claims{
		Role:     auth.RoleAdmin,
		TenantID: "test-tenant",
	})

	tenantID := testUUID(1)
	tenantIDStr := uuidToString(tenantID)

	store.On("GetFieldLocks", ctx, tenantID).
		Return([]dbstore.TenantFieldLock{
			{TenantID: tenantID, FieldPath: "payments.fee"},
		}, nil)

	_, err := svc.SetField(ctx, &pb.SetFieldRequest{
		TenantId:  tenantIDStr,
		FieldPath: "payments.fee",
		Value:     "0.5",
	})

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

// --- GetField ---

func TestGetField_NotFound(t *testing.T) {
	svc, store, _, _, _ := newTestService()
	ctx := context.Background()

	tenantID := testUUID(1)
	tenantIDStr := uuidToString(tenantID)

	store.On("GetLatestConfigVersion", ctx, tenantID).
		Return(dbstore.ConfigVersion{Version: 1}, nil)
	store.On("GetConfigValueAtVersion", ctx, mock.AnythingOfType("dbstore.GetConfigValueAtVersionParams")).
		Return(dbstore.GetConfigValueAtVersionRow{}, pgx.ErrNoRows)

	_, err := svc.GetField(ctx, &pb.GetFieldRequest{
		TenantId:  tenantIDStr,
		FieldPath: "nonexistent",
	})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// --- RollbackToVersion ---

func TestRollbackToVersion_Success(t *testing.T) {
	svc, store, cache, _, _ := newTestService()
	ctx := context.Background()

	tenantID := testUUID(1)
	tenantIDStr := uuidToString(tenantID)
	newVersionID := testUUID(3)

	store.On("GetFullConfigAtVersion", ctx, dbstore.GetFullConfigAtVersionParams{TenantID: tenantID, Version: 2}).
		Return([]dbstore.GetFullConfigAtVersionRow{
			{FieldPath: "a", Value: "1"},
			{FieldPath: "b", Value: "2"},
		}, nil)
	store.On("GetLatestConfigVersion", ctx, tenantID).
		Return(dbstore.ConfigVersion{Version: 5}, nil)
	store.On("CreateConfigVersion", ctx, mock.AnythingOfType("dbstore.CreateConfigVersionParams")).
		Return(dbstore.ConfigVersion{ID: newVersionID, TenantID: tenantID, Version: 6, CreatedBy: "unknown"}, nil)
	store.On("SetConfigValue", ctx, mock.AnythingOfType("dbstore.SetConfigValueParams")).
		Return(nil)
	cache.On("Invalidate", ctx, tenantIDStr).Return(nil)
	store.On("InsertAuditWriteLog", ctx, mock.AnythingOfType("dbstore.InsertAuditWriteLogParams")).Return(nil)

	resp, err := svc.RollbackToVersion(ctx, &pb.RollbackToVersionRequest{
		TenantId: tenantIDStr,
		Version:  2,
	})

	require.NoError(t, err)
	assert.Equal(t, int32(6), resp.ConfigVersion.Version)
	// Should copy 2 values.
	store.AssertNumberOfCalls(t, "SetConfigValue", 2)
}
