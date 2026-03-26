//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
	"github.com/zeevdr/central-config-service/sdk/adminclient"
	"github.com/zeevdr/central-config-service/sdk/configclient"
)

func serviceAddr() string {
	if addr := os.Getenv("SERVICE_ADDR"); addr != "" {
		return addr
	}
	return "localhost:9090"
}

func dial(t *testing.T) *grpc.ClientConn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, serviceAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	require.NoError(t, err, "failed to connect to service at %s", serviceAddr())
	t.Cleanup(func() { conn.Close() })
	return conn
}

func newAdminClient(conn *grpc.ClientConn) *adminclient.Client {
	return adminclient.New(
		pb.NewSchemaServiceClient(conn),
		pb.NewConfigServiceClient(conn),
		pb.NewAuditServiceClient(conn),
		adminclient.WithSubject("e2e-test"),
	)
}

func newConfigClient(conn *grpc.ClientConn) *configclient.Client {
	return configclient.New(
		pb.NewConfigServiceClient(conn),
		configclient.WithSubject("e2e-test"),
	)
}

// --- Schema Lifecycle (adminclient) ---

func TestSchemaLifecycle(t *testing.T) {
	conn := dial(t)
	admin := newAdminClient(conn)
	ctx := context.Background()

	// Create schema with fields.
	s, err := admin.CreateSchema(ctx, "payments-e2e", []adminclient.Field{
		{Path: "payments.fee", Type: "FIELD_TYPE_STRING", Description: "Transaction fee percentage"},
		{Path: "payments.currency", Type: "FIELD_TYPE_STRING", Description: "Default currency"},
		{Path: "payments.timeout", Type: "FIELD_TYPE_DURATION", Description: "Payment timeout"},
	}, "E2E test schema")
	require.NoError(t, err)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, "payments-e2e", s.Name)
	assert.Equal(t, int32(1), s.Version)
	assert.False(t, s.Published)
	assert.Len(t, s.Fields, 3)

	// Get schema (latest).
	got, err := admin.GetSchema(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, int32(1), got.Version)

	// List schemas.
	schemas, err := admin.ListSchemas(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(schemas), 1)

	// Update schema — add a field, creates v2.
	updated, err := admin.UpdateSchema(ctx, s.ID, []adminclient.Field{
		{Path: "payments.max_retries", Type: "FIELD_TYPE_INT", Description: "Max retry count"},
	}, nil, "Add max_retries field")
	require.NoError(t, err)
	assert.Equal(t, int32(2), updated.Version)
	assert.Len(t, updated.Fields, 4)

	// Get specific version.
	v1, err := admin.GetSchemaVersion(ctx, s.ID, 1)
	require.NoError(t, err)
	assert.Len(t, v1.Fields, 3)

	// Publish v1.
	published, err := admin.PublishSchema(ctx, s.ID, 1)
	require.NoError(t, err)
	assert.True(t, published.Published)

	// Publish already-published → should error.
	_, err = admin.PublishSchema(ctx, s.ID, 1)
	if err != nil {
		assert.ErrorIs(t, err, adminclient.ErrFailedPrecondition)
	}

	// Cleanup.
	require.NoError(t, admin.DeleteSchema(ctx, s.ID))

	_, err = admin.GetSchema(ctx, s.ID)
	assert.ErrorIs(t, err, adminclient.ErrNotFound)
}

// --- Full Flow: Schema → Tenant → Config → Audit (adminclient + configclient) ---

func TestFullFlow(t *testing.T) {
	conn := dial(t)
	admin := newAdminClient(conn)
	cfg := newConfigClient(conn)
	ctx := context.Background()

	// 1. Create and publish a schema.
	s, err := admin.CreateSchema(ctx, "settlement-e2e", []adminclient.Field{
		{Path: "settlement.window", Type: "FIELD_TYPE_DURATION"},
		{Path: "settlement.currency", Type: "FIELD_TYPE_STRING"},
		{Path: "settlement.fee", Type: "FIELD_TYPE_STRING"},
	}, "")
	require.NoError(t, err)
	_, err = admin.PublishSchema(ctx, s.ID, 1)
	require.NoError(t, err)

	// 2. Create a tenant.
	tenant, err := admin.CreateTenant(ctx, "acme-e2e", s.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "acme-e2e", tenant.Name)

	// Get tenant.
	got, err := admin.GetTenant(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.SchemaID)

	// List tenants.
	tenants, err := admin.ListTenants(ctx, s.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tenants), 1)

	// 3. Set config fields via configclient.
	require.NoError(t, cfg.Set(ctx, tenant.ID, "settlement.window", "24h"))

	require.NoError(t, cfg.SetMany(ctx, tenant.ID, map[string]string{
		"settlement.currency": "USD",
		"settlement.fee":      "0.5%",
	}, "Bulk config update"))

	// 4. Read config via configclient.
	allVals, err := cfg.GetAll(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "24h", allVals["settlement.window"])
	assert.Equal(t, "USD", allVals["settlement.currency"])
	assert.Equal(t, "0.5%", allVals["settlement.fee"])

	// Get single field.
	val, err := cfg.Get(ctx, tenant.ID, "settlement.currency")
	require.NoError(t, err)
	assert.Equal(t, "USD", val)

	// Get multiple fields.
	fields, err := cfg.GetFields(ctx, tenant.ID, []string{"settlement.window", "settlement.fee"})
	require.NoError(t, err)
	assert.Len(t, fields, 2)

	// 5. Snapshot reads (pinned version).
	snap, err := cfg.Snapshot(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Greater(t, snap.Version(), int32(0))

	snapVal, err := snap.Get(ctx, "settlement.fee")
	require.NoError(t, err)
	assert.Equal(t, "0.5%", snapVal)

	// 6. Config versioning via adminclient.
	versions, err := admin.ListConfigVersions(ctx, tenant.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(versions), 2)

	v, err := admin.GetConfigVersion(ctx, tenant.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, int32(1), v.Version)

	// 7. Optimistic concurrency via configclient.
	lv, err := cfg.GetForUpdate(ctx, tenant.ID, "settlement.fee")
	require.NoError(t, err)
	assert.Equal(t, "0.5%", lv.Value)

	// Set with correct checksum → succeeds.
	require.NoError(t, lv.Set(ctx, cfg, "0.3%"))

	// Set with stale checksum → fails.
	err = lv.Set(ctx, cfg, "0.1%")
	assert.ErrorIs(t, err, configclient.ErrChecksumMismatch)

	// 8. Update convenience CAS.
	require.NoError(t, cfg.Update(ctx, tenant.ID, "settlement.fee", func(current string) (string, error) {
		return "0.2%", nil
	}))
	updated, err := cfg.Get(ctx, tenant.ID, "settlement.fee")
	require.NoError(t, err)
	assert.Equal(t, "0.2%", updated)

	// 9. Rollback via adminclient.
	rv, err := admin.RollbackConfig(ctx, tenant.ID, 1, "Rollback to v1")
	require.NoError(t, err)
	assert.Greater(t, rv.Version, int32(3))

	afterRollback, err := cfg.GetAll(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "24h", afterRollback["settlement.window"])

	// 10. Field locking via adminclient.
	require.NoError(t, admin.LockField(ctx, tenant.ID, "settlement.currency"))

	locks, err := admin.ListFieldLocks(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Len(t, locks, 1)
	assert.Equal(t, "settlement.currency", locks[0].FieldPath)

	require.NoError(t, admin.UnlockField(ctx, tenant.ID, "settlement.currency"))

	locks, err = admin.ListFieldLocks(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Empty(t, locks)

	// 11. Audit log via adminclient.
	entries, err := admin.QueryWriteLog(ctx, adminclient.WithAuditTenant(tenant.ID))
	require.NoError(t, err)
	assert.Greater(t, len(entries), 0, "expected audit entries for config changes")

	// 12. Cleanup.
	require.NoError(t, admin.DeleteTenant(ctx, tenant.ID))
	require.NoError(t, admin.DeleteSchema(ctx, s.ID))
}

// --- Streaming Subscription (raw proto — Subscribe is not in configclient SDK) ---

func TestConfigSubscription(t *testing.T) {
	conn := dial(t)
	admin := newAdminClient(conn)
	configSvc := pb.NewConfigServiceClient(conn)
	cfg := newConfigClient(conn)
	ctx := context.Background()

	// Setup: schema + tenant via adminclient.
	s, err := admin.CreateSchema(ctx, "stream-e2e", []adminclient.Field{
		{Path: "notify.enabled", Type: "FIELD_TYPE_STRING"},
		{Path: "notify.channel", Type: "FIELD_TYPE_STRING"},
	}, "")
	require.NoError(t, err)
	_, err = admin.PublishSchema(ctx, s.ID, 1)
	require.NoError(t, err)

	tenant, err := admin.CreateTenant(ctx, "stream-tenant-e2e", s.ID, 1)
	require.NoError(t, err)

	// Subscribe with raw proto client (not available in SDK).
	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	stream, err := configSvc.Subscribe(subCtx, &pb.SubscribeRequest{
		TenantId:   tenant.ID,
		FieldPaths: []string{"notify.enabled"},
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Write via configclient SDK.
	require.NoError(t, cfg.Set(ctx, tenant.ID, "notify.enabled", "true"))

	// Read from raw stream.
	change, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, "notify.enabled", change.Change.FieldPath)
	assert.Equal(t, "true", change.Change.NewValue)

	cancel()

	// Cleanup.
	require.NoError(t, admin.DeleteTenant(context.Background(), tenant.ID))
	require.NoError(t, admin.DeleteSchema(context.Background(), s.ID))
}

// --- Error Cases (SDK sentinel errors) ---

func TestErrorCases(t *testing.T) {
	conn := dial(t)
	admin := newAdminClient(conn)
	cfg := newConfigClient(conn)
	ctx := context.Background()

	t.Run("get nonexistent schema", func(t *testing.T) {
		_, err := admin.GetSchema(ctx, "00000000-0000-0000-0000-000000000000")
		assert.ErrorIs(t, err, adminclient.ErrNotFound)
	})

	t.Run("create tenant with unpublished schema", func(t *testing.T) {
		s, err := admin.CreateSchema(ctx, "unpublished-e2e", []adminclient.Field{
			{Path: "x", Type: "FIELD_TYPE_STRING"},
		}, "")
		require.NoError(t, err)

		_, err = admin.CreateTenant(ctx, "bad-tenant-e2e", s.ID, 1)
		assert.ErrorIs(t, err, adminclient.ErrFailedPrecondition)

		_ = admin.DeleteSchema(ctx, s.ID)
	})

	t.Run("get nonexistent config field", func(t *testing.T) {
		// Create schema + tenant to have a valid tenant ID.
		s, err := admin.CreateSchema(ctx, "err-cfg-e2e", []adminclient.Field{
			{Path: "x", Type: "FIELD_TYPE_STRING"},
		}, "")
		require.NoError(t, err)
		_, err = admin.PublishSchema(ctx, s.ID, 1)
		require.NoError(t, err)
		tenant, err := admin.CreateTenant(ctx, "err-cfg-tenant-e2e", s.ID, 1)
		require.NoError(t, err)

		_, err = cfg.Get(ctx, tenant.ID, "nonexistent.field")
		assert.ErrorIs(t, err, configclient.ErrNotFound)

		_ = admin.DeleteTenant(ctx, tenant.ID)
		_ = admin.DeleteSchema(ctx, s.ID)
	})
}

// --- Schema Export/Import (adminclient) ---

func TestSchemaExportImport(t *testing.T) {
	conn := dial(t)
	admin := newAdminClient(conn)
	ctx := context.Background()

	// 1. Create a schema with fields.
	s, err := admin.CreateSchema(ctx, "export-e2e", []adminclient.Field{
		{Path: "trade.fee", Type: "FIELD_TYPE_STRING", Description: "Fee percentage"},
		{Path: "trade.currency", Type: "FIELD_TYPE_STRING"},
		{Path: "trade.timeout", Type: "FIELD_TYPE_DURATION"},
	}, "Schema for export testing")
	require.NoError(t, err)

	// 2. Export.
	yamlContent, err := admin.ExportSchema(ctx, s.ID, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, yamlContent)

	yamlStr := string(yamlContent)
	t.Logf("Exported YAML:\n%s", yamlStr)
	assert.Contains(t, yamlStr, "syntax:")
	assert.Contains(t, yamlStr, "name: export-e2e")
	assert.Contains(t, yamlStr, "trade.fee")

	// 3. Import identical YAML → should get AlreadyExists.
	_, err = admin.ImportSchema(ctx, yamlContent)
	assert.ErrorIs(t, err, adminclient.ErrAlreadyExists)

	// 4. Modify YAML — add a field, re-import.
	modified := bytes.Replace(yamlContent,
		[]byte("    trade.timeout:"),
		[]byte("    trade.max_retries:\n        type: integer\n    trade.timeout:"),
		1,
	)
	imported, err := admin.ImportSchema(ctx, modified)
	require.NoError(t, err)
	assert.Equal(t, int32(2), imported.Version)
	assert.Len(t, imported.Fields, 4)
	assert.False(t, imported.Published)

	// 5. Verify via GetSchema.
	got, err := admin.GetSchema(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, int32(2), got.Version)

	// 6. Import new schema by name.
	newYAML := []byte(`syntax: "v1"
name: brand-new-e2e
description: Created via import
fields:
  config.enabled:
    type: string
    default: "true"
`)
	newSchema, err := admin.ImportSchema(ctx, newYAML)
	require.NoError(t, err)
	assert.Equal(t, "brand-new-e2e", newSchema.Name)
	assert.Equal(t, int32(1), newSchema.Version)

	// Cleanup.
	_ = admin.DeleteSchema(ctx, s.ID)
	_ = admin.DeleteSchema(ctx, newSchema.ID)
}

// --- Config Export/Import (adminclient + configclient) ---

func TestConfigExportImport(t *testing.T) {
	conn := dial(t)
	admin := newAdminClient(conn)
	cfg := newConfigClient(conn)
	ctx := context.Background()

	// 1. Create and publish a schema with typed fields.
	s, err := admin.CreateSchema(ctx, "config-export-e2e", []adminclient.Field{
		{Path: "app.enabled", Type: "FIELD_TYPE_BOOL"},
		{Path: "app.max_retries", Type: "FIELD_TYPE_INT"},
		{Path: "app.fee_rate", Type: "FIELD_TYPE_NUMBER"},
		{Path: "app.name", Type: "FIELD_TYPE_STRING"},
		{Path: "app.timeout", Type: "FIELD_TYPE_DURATION"},
	}, "")
	require.NoError(t, err)
	_, err = admin.PublishSchema(ctx, s.ID, 1)
	require.NoError(t, err)

	// 2. Create a tenant.
	tenant, err := admin.CreateTenant(ctx, "config-export-tenant-e2e", s.ID, 1)
	require.NoError(t, err)

	// 3. Set config values via configclient.
	require.NoError(t, cfg.SetMany(ctx, tenant.ID, map[string]string{
		"app.enabled":     "true",
		"app.max_retries": "3",
		"app.fee_rate":    "0.025",
		"app.name":        "MyApp",
		"app.timeout":     "30s",
	}, "Initial config"))

	// 4. Export config via adminclient.
	yamlContent, err := admin.ExportConfig(ctx, tenant.ID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, yamlContent)

	yamlStr := string(yamlContent)
	t.Logf("Exported config YAML:\n%s", yamlStr)
	assert.Contains(t, yamlStr, "value: true")
	assert.Contains(t, yamlStr, "value: 3")
	assert.Contains(t, yamlStr, "value: 0.025")

	// 5. Modify YAML and import.
	modified := bytes.Replace(yamlContent, []byte("value: 3"), []byte("value: 5"), 1)
	modified = bytes.Replace(modified, []byte("value: true"), []byte("value: false"), 1)

	v, err := admin.ImportConfig(ctx, tenant.ID, modified, "Updated via import")
	require.NoError(t, err)
	assert.Equal(t, int32(2), v.Version)

	// 6. Verify via configclient.
	vals, err := cfg.GetAll(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "false", vals["app.enabled"])
	assert.Equal(t, "5", vals["app.max_retries"])
	assert.Equal(t, "0.025", vals["app.fee_rate"])
	assert.Equal(t, "MyApp", vals["app.name"])
	assert.Equal(t, "30s", vals["app.timeout"])

	// Cleanup.
	_ = admin.DeleteTenant(ctx, tenant.ID)
	_ = admin.DeleteSchema(ctx, s.ID)
}

// --- Typed Values + Null Handling ---

func TestTypedValuesAndNull(t *testing.T) {
	conn := dial(t)
	admin := newAdminClient(conn)
	cfg := newConfigClient(conn)
	ctx := context.Background()

	// 1. Create schema with all types.
	s, err := admin.CreateSchema(ctx, "typed-e2e", []adminclient.Field{
		{Path: "app.retries", Type: "FIELD_TYPE_INT"},
		{Path: "app.rate", Type: "FIELD_TYPE_NUMBER"},
		{Path: "app.name", Type: "FIELD_TYPE_STRING"},
		{Path: "app.enabled", Type: "FIELD_TYPE_BOOL"},
		{Path: "app.timeout", Type: "FIELD_TYPE_DURATION"},
	}, "")
	require.NoError(t, err)
	_, err = admin.PublishSchema(ctx, s.ID, 1)
	require.NoError(t, err)

	tenant, err := admin.CreateTenant(ctx, "typed-tenant-e2e", s.ID, 1)
	require.NoError(t, err)

	// 2. Set values with typed setters.
	require.NoError(t, cfg.SetInt(ctx, tenant.ID, "app.retries", 5))
	require.NoError(t, cfg.SetFloat(ctx, tenant.ID, "app.rate", 0.025))
	require.NoError(t, cfg.Set(ctx, tenant.ID, "app.name", "MyApp"))
	require.NoError(t, cfg.SetBool(ctx, tenant.ID, "app.enabled", true))
	require.NoError(t, cfg.SetDuration(ctx, tenant.ID, "app.timeout", 30*time.Second))

	// 3. Read with typed getters.
	retries, err := cfg.GetInt(ctx, tenant.ID, "app.retries")
	require.NoError(t, err)
	assert.Equal(t, int64(5), retries)

	rate, err := cfg.GetFloat(ctx, tenant.ID, "app.rate")
	require.NoError(t, err)
	assert.Equal(t, 0.025, rate)

	name, err := cfg.GetString(ctx, tenant.ID, "app.name")
	require.NoError(t, err)
	assert.Equal(t, "MyApp", name)

	enabled, err := cfg.GetBool(ctx, tenant.ID, "app.enabled")
	require.NoError(t, err)
	assert.True(t, enabled)

	timeout, err := cfg.GetDuration(ctx, tenant.ID, "app.timeout")
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, timeout)

	// 4. Get as string (always works).
	retriesStr, err := cfg.Get(ctx, tenant.ID, "app.retries")
	require.NoError(t, err)
	assert.Equal(t, "5", retriesStr)

	// 5. Null handling — set to null and verify.
	require.NoError(t, cfg.SetNull(ctx, tenant.ID, "app.retries"))

	// GetInt on null returns zero value.
	retriesAfterNull, err := cfg.GetInt(ctx, tenant.ID, "app.retries")
	require.NoError(t, err)
	assert.Equal(t, int64(0), retriesAfterNull)

	// GetIntNullable distinguishes null from zero.
	retriesNullable, err := cfg.GetIntNullable(ctx, tenant.ID, "app.retries")
	require.NoError(t, err)
	assert.Nil(t, retriesNullable)

	// 6. Null vs empty string distinction.
	require.NoError(t, cfg.Set(ctx, tenant.ID, "app.name", "")) // empty string, not null
	nameNullable, err := cfg.GetStringNullable(ctx, tenant.ID, "app.name")
	require.NoError(t, err)
	require.NotNil(t, nameNullable, "empty string should not be null")
	assert.Equal(t, "", *nameNullable)

	require.NoError(t, cfg.SetNull(ctx, tenant.ID, "app.name")) // now actually null
	nameNullable, err = cfg.GetStringNullable(ctx, tenant.ID, "app.name")
	require.NoError(t, err)
	assert.Nil(t, nameNullable, "should be null after SetNull")

	// Cleanup.
	_ = admin.DeleteTenant(ctx, tenant.ID)
	_ = admin.DeleteSchema(ctx, s.ID)
}
