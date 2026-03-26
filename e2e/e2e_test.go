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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
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

func ptr[T any](v T) *T { return &v }

// --- Schema Lifecycle ---

func TestSchemaLifecycle(t *testing.T) {
	conn := dial(t)
	schema := pb.NewSchemaServiceClient(conn)
	ctx := context.Background()

	// Create schema with fields.
	createResp, err := schema.CreateSchema(ctx, &pb.CreateSchemaRequest{
		Name:        "payments-e2e",
		Description: ptr("E2E test schema"),
		Fields: []*pb.SchemaField{
			{Path: "payments.fee", Type: pb.FieldType_FIELD_TYPE_STRING, Description: ptr("Transaction fee percentage")},
			{Path: "payments.currency", Type: pb.FieldType_FIELD_TYPE_STRING, Description: ptr("Default currency")},
			{Path: "payments.timeout", Type: pb.FieldType_FIELD_TYPE_DURATION, Description: ptr("Payment timeout")},
		},
	})
	require.NoError(t, err)
	s := createResp.Schema
	assert.NotEmpty(t, s.Id)
	assert.Equal(t, "payments-e2e", s.Name)
	assert.Equal(t, int32(1), s.Version)
	assert.False(t, s.Published)
	assert.Len(t, s.Fields, 3)
	schemaID := s.Id

	// Get schema (latest).
	getResp, err := schema.GetSchema(ctx, &pb.GetSchemaRequest{Id: schemaID})
	require.NoError(t, err)
	assert.Equal(t, int32(1), getResp.Schema.Version)

	// List schemas.
	listResp, err := schema.ListSchemas(ctx, &pb.ListSchemasRequest{PageSize: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Schemas), 1)

	// Update schema — add a field, creates v2.
	updateResp, err := schema.UpdateSchema(ctx, &pb.UpdateSchemaRequest{
		Id:                 schemaID,
		VersionDescription: ptr("Add max_retries field"),
		Fields: []*pb.SchemaField{
			{Path: "payments.max_retries", Type: pb.FieldType_FIELD_TYPE_INT, Description: ptr("Max retry count")},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), updateResp.Schema.Version)
	assert.Len(t, updateResp.Schema.Fields, 4)

	// Get specific version.
	getV1, err := schema.GetSchema(ctx, &pb.GetSchemaRequest{Id: schemaID, Version: ptr(int32(1))})
	require.NoError(t, err)
	assert.Len(t, getV1.Schema.Fields, 3)

	// Publish v1.
	pubResp, err := schema.PublishSchema(ctx, &pb.PublishSchemaRequest{Id: schemaID, Version: 1})
	require.NoError(t, err)
	assert.True(t, pubResp.Schema.Published)

	// Publish already-published → should be idempotent or error.
	_, err = schema.PublishSchema(ctx, &pb.PublishSchemaRequest{Id: schemaID, Version: 1})
	// Accept either success (idempotent) or error.
	if err != nil {
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	}

	t.Run("cleanup", func(t *testing.T) {
		// Delete schema at the end (will cascade tenants etc.).
		_, err := schema.DeleteSchema(ctx, &pb.DeleteSchemaRequest{Id: schemaID})
		require.NoError(t, err)

		// Confirm it's gone.
		_, err = schema.GetSchema(ctx, &pb.GetSchemaRequest{Id: schemaID})
		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}

// --- Full Flow: Schema → Tenant → Config → Audit ---

func TestFullFlow(t *testing.T) {
	conn := dial(t)
	schemaSvc := pb.NewSchemaServiceClient(conn)
	configSvc := pb.NewConfigServiceClient(conn)
	auditSvc := pb.NewAuditServiceClient(conn)
	ctx := context.Background()

	// 1. Create and publish a schema.
	createResp, err := schemaSvc.CreateSchema(ctx, &pb.CreateSchemaRequest{
		Name: "settlement-e2e",
		Fields: []*pb.SchemaField{
			{Path: "settlement.window", Type: pb.FieldType_FIELD_TYPE_DURATION},
			{Path: "settlement.currency", Type: pb.FieldType_FIELD_TYPE_STRING},
			{Path: "settlement.fee", Type: pb.FieldType_FIELD_TYPE_STRING},
		},
	})
	require.NoError(t, err)
	schemaID := createResp.Schema.Id

	_, err = schemaSvc.PublishSchema(ctx, &pb.PublishSchemaRequest{Id: schemaID, Version: 1})
	require.NoError(t, err)

	// 2. Create a tenant.
	tenantResp, err := schemaSvc.CreateTenant(ctx, &pb.CreateTenantRequest{
		Name:          "acme-e2e",
		SchemaId:      schemaID,
		SchemaVersion: 1,
	})
	require.NoError(t, err)
	tenant := tenantResp.Tenant
	assert.NotEmpty(t, tenant.Id)
	assert.Equal(t, "acme-e2e", tenant.Name)
	tenantID := tenant.Id

	// Get tenant.
	getTenantResp, err := schemaSvc.GetTenant(ctx, &pb.GetTenantRequest{Id: tenantID})
	require.NoError(t, err)
	assert.Equal(t, schemaID, getTenantResp.Tenant.SchemaId)

	// List tenants.
	listTenantsResp, err := schemaSvc.ListTenants(ctx, &pb.ListTenantsRequest{SchemaId: &schemaID, PageSize: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listTenantsResp.Tenants), 1)

	// 3. Set config fields.
	setResp, err := configSvc.SetField(ctx, &pb.SetFieldRequest{
		TenantId:  tenantID,
		FieldPath: "settlement.window",
		Value:     "24h",
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), setResp.ConfigVersion.Version)

	// Set multiple fields.
	setFieldsResp, err := configSvc.SetFields(ctx, &pb.SetFieldsRequest{
		TenantId:    tenantID,
		Description: ptr("Bulk config update"),
		Updates: []*pb.FieldUpdate{
			{FieldPath: "settlement.currency", Value: "USD"},
			{FieldPath: "settlement.fee", Value: "0.5%"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), setFieldsResp.ConfigVersion.Version)

	// 4. Read config.
	getConfigResp, err := configSvc.GetConfig(ctx, &pb.GetConfigRequest{TenantId: tenantID})
	require.NoError(t, err)
	cfg := getConfigResp.Config
	assert.Equal(t, tenantID, cfg.TenantId)
	assert.GreaterOrEqual(t, len(cfg.Values), 3)

	valueMap := make(map[string]string)
	for _, v := range cfg.Values {
		valueMap[v.FieldPath] = v.Value
	}
	assert.Equal(t, "24h", valueMap["settlement.window"])
	assert.Equal(t, "USD", valueMap["settlement.currency"])
	assert.Equal(t, "0.5%", valueMap["settlement.fee"])

	// Get single field.
	getFieldResp, err := configSvc.GetField(ctx, &pb.GetFieldRequest{
		TenantId:  tenantID,
		FieldPath: "settlement.currency",
	})
	require.NoError(t, err)
	assert.Equal(t, "USD", getFieldResp.Value.Value)

	// Get multiple fields.
	getFieldsResp, err := configSvc.GetFields(ctx, &pb.GetFieldsRequest{
		TenantId:   tenantID,
		FieldPaths: []string{"settlement.window", "settlement.fee"},
	})
	require.NoError(t, err)
	assert.Len(t, getFieldsResp.Values, 2)

	// 5. Config versioning.
	listVersionsResp, err := configSvc.ListVersions(ctx, &pb.ListVersionsRequest{
		TenantId: tenantID,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listVersionsResp.Versions), 2)

	// Get specific version.
	getVersionResp, err := configSvc.GetVersion(ctx, &pb.GetVersionRequest{
		TenantId: tenantID,
		Version:  1,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), getVersionResp.ConfigVersion.Version)

	// 6. Update a field and verify checksum-based optimistic concurrency.
	getFieldResp2, err := configSvc.GetField(ctx, &pb.GetFieldRequest{
		TenantId:  tenantID,
		FieldPath: "settlement.fee",
	})
	require.NoError(t, err)
	checksum := getFieldResp2.Value.Checksum

	// Set with correct checksum → should succeed.
	_, err = configSvc.SetField(ctx, &pb.SetFieldRequest{
		TenantId:         tenantID,
		FieldPath:        "settlement.fee",
		Value:            "0.3%",
		ExpectedChecksum: &checksum,
	})
	require.NoError(t, err)

	// Set with stale checksum → should fail.
	_, err = configSvc.SetField(ctx, &pb.SetFieldRequest{
		TenantId:         tenantID,
		FieldPath:        "settlement.fee",
		Value:            "0.1%",
		ExpectedChecksum: &checksum, // stale
	})
	require.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))

	// 7. Rollback.
	rollbackResp, err := configSvc.RollbackToVersion(ctx, &pb.RollbackToVersionRequest{
		TenantId:    tenantID,
		Version:     1,
		Description: ptr("Rollback to v1"),
	})
	require.NoError(t, err)
	assert.Greater(t, rollbackResp.ConfigVersion.Version, int32(3))

	// Verify rollback: only settlement.window should exist from v1.
	afterRollback, err := configSvc.GetConfig(ctx, &pb.GetConfigRequest{TenantId: tenantID})
	require.NoError(t, err)
	rolledBackMap := make(map[string]string)
	for _, v := range afterRollback.Config.Values {
		rolledBackMap[v.FieldPath] = v.Value
	}
	assert.Equal(t, "24h", rolledBackMap["settlement.window"])

	// 8. Field locking (lock/unlock/list — enforcement requires auth, so we only test CRUD).
	_, err = schemaSvc.LockField(ctx, &pb.LockFieldRequest{
		TenantId:  tenantID,
		FieldPath: "settlement.currency",
	})
	require.NoError(t, err)

	// List locks.
	locksResp, err := schemaSvc.ListFieldLocks(ctx, &pb.ListFieldLocksRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.Len(t, locksResp.Locks, 1)
	assert.Equal(t, "settlement.currency", locksResp.Locks[0].FieldPath)

	// Unlock.
	_, err = schemaSvc.UnlockField(ctx, &pb.UnlockFieldRequest{
		TenantId:  tenantID,
		FieldPath: "settlement.currency",
	})
	require.NoError(t, err)

	locksResp, err = schemaSvc.ListFieldLocks(ctx, &pb.ListFieldLocksRequest{TenantId: tenantID})
	require.NoError(t, err)
	assert.Empty(t, locksResp.Locks)

	// 9. Audit log.
	auditResp, err := auditSvc.QueryWriteLog(ctx, &pb.QueryWriteLogRequest{
		TenantId: &tenantID,
		PageSize: 50,
	})
	require.NoError(t, err)
	assert.Greater(t, len(auditResp.Entries), 0, "expected audit entries for config changes")

	// 10. Cleanup.
	_, err = schemaSvc.DeleteTenant(ctx, &pb.DeleteTenantRequest{Id: tenantID})
	require.NoError(t, err)
	_, err = schemaSvc.DeleteSchema(ctx, &pb.DeleteSchemaRequest{Id: schemaID})
	require.NoError(t, err)
}

// --- Streaming Subscription ---

func TestConfigSubscription(t *testing.T) {
	conn := dial(t)
	schemaSvc := pb.NewSchemaServiceClient(conn)
	configSvc := pb.NewConfigServiceClient(conn)
	ctx := context.Background()

	// Setup: schema + tenant.
	createResp, err := schemaSvc.CreateSchema(ctx, &pb.CreateSchemaRequest{
		Name: "stream-e2e",
		Fields: []*pb.SchemaField{
			{Path: "notify.enabled", Type: pb.FieldType_FIELD_TYPE_STRING},
			{Path: "notify.channel", Type: pb.FieldType_FIELD_TYPE_STRING},
		},
	})
	require.NoError(t, err)
	schemaID := createResp.Schema.Id
	_, err = schemaSvc.PublishSchema(ctx, &pb.PublishSchemaRequest{Id: schemaID, Version: 1})
	require.NoError(t, err)

	tenantResp, err := schemaSvc.CreateTenant(ctx, &pb.CreateTenantRequest{
		Name:          "stream-tenant-e2e",
		SchemaId:      schemaID,
		SchemaVersion: 1,
	})
	require.NoError(t, err)
	tenantID := tenantResp.Tenant.Id

	// Subscribe with a short timeout.
	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	stream, err := configSvc.Subscribe(subCtx, &pb.SubscribeRequest{
		TenantId:   tenantID,
		FieldPaths: []string{"notify.enabled"},
	})
	require.NoError(t, err)

	// Give the subscription a moment to establish.
	time.Sleep(200 * time.Millisecond)

	// Write a config value — should trigger subscription.
	_, err = configSvc.SetField(ctx, &pb.SetFieldRequest{
		TenantId:  tenantID,
		FieldPath: "notify.enabled",
		Value:     "true",
	})
	require.NoError(t, err)

	// Read from stream.
	change, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, "notify.enabled", change.Change.FieldPath)
	assert.Equal(t, "true", change.Change.NewValue)

	cancel()

	// Cleanup.
	_, err = schemaSvc.DeleteTenant(context.Background(), &pb.DeleteTenantRequest{Id: tenantID})
	require.NoError(t, err)
	_, err = schemaSvc.DeleteSchema(context.Background(), &pb.DeleteSchemaRequest{Id: schemaID})
	require.NoError(t, err)
}

// --- Error Cases ---

func TestErrorCases(t *testing.T) {
	conn := dial(t)
	schemaSvc := pb.NewSchemaServiceClient(conn)
	configSvc := pb.NewConfigServiceClient(conn)
	ctx := context.Background()

	t.Run("get nonexistent schema", func(t *testing.T) {
		_, err := schemaSvc.GetSchema(ctx, &pb.GetSchemaRequest{Id: "00000000-0000-0000-0000-000000000000"})
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("create tenant with unpublished schema", func(t *testing.T) {
		resp, err := schemaSvc.CreateSchema(ctx, &pb.CreateSchemaRequest{
			Name: "unpublished-e2e",
			Fields: []*pb.SchemaField{
				{Path: "x", Type: pb.FieldType_FIELD_TYPE_STRING},
			},
		})
		require.NoError(t, err)
		sid := resp.Schema.Id

		_, err = schemaSvc.CreateTenant(ctx, &pb.CreateTenantRequest{
			Name:          "bad-tenant-e2e",
			SchemaId:      sid,
			SchemaVersion: 1,
		})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))

		// Cleanup.
		_, _ = schemaSvc.DeleteSchema(ctx, &pb.DeleteSchemaRequest{Id: sid})
	})

	t.Run("duplicate schema name", func(t *testing.T) {
		resp, err := schemaSvc.CreateSchema(ctx, &pb.CreateSchemaRequest{
			Name:   "dup-e2e",
			Fields: []*pb.SchemaField{{Path: "x", Type: pb.FieldType_FIELD_TYPE_STRING}},
		})
		require.NoError(t, err)
		sid := resp.Schema.Id

		_, err = schemaSvc.CreateSchema(ctx, &pb.CreateSchemaRequest{
			Name:   "dup-e2e",
			Fields: []*pb.SchemaField{{Path: "x", Type: pb.FieldType_FIELD_TYPE_STRING}},
		})
		require.Error(t, err, "duplicate schema name should fail")

		_, _ = schemaSvc.DeleteSchema(ctx, &pb.DeleteSchemaRequest{Id: sid})
	})

	t.Run("get config for nonexistent tenant", func(t *testing.T) {
		resp, err := configSvc.GetConfig(ctx, &pb.GetConfigRequest{TenantId: "00000000-0000-0000-0000-000000000000"})
		// Service may return empty config or error for nonexistent tenant.
		if err == nil {
			assert.Empty(t, resp.Config.Values)
		}
	})
}

// --- Schema Export/Import ---

func TestSchemaExportImport(t *testing.T) {
	conn := dial(t)
	schemaSvc := pb.NewSchemaServiceClient(conn)
	ctx := context.Background()

	// 1. Create a schema with fields.
	createResp, err := schemaSvc.CreateSchema(ctx, &pb.CreateSchemaRequest{
		Name:        "export-e2e",
		Description: ptr("Schema for export testing"),
		Fields: []*pb.SchemaField{
			{Path: "trade.fee", Type: pb.FieldType_FIELD_TYPE_STRING, Description: ptr("Fee percentage")},
			{Path: "trade.currency", Type: pb.FieldType_FIELD_TYPE_STRING, Constraints: &pb.FieldConstraints{
				EnumValues: []string{"USD", "EUR"},
			}},
			{Path: "trade.timeout", Type: pb.FieldType_FIELD_TYPE_DURATION},
		},
	})
	require.NoError(t, err)
	schemaID := createResp.Schema.Id

	// 2. Export.
	exportResp, err := schemaSvc.ExportSchema(ctx, &pb.ExportSchemaRequest{Id: schemaID})
	require.NoError(t, err)
	assert.NotEmpty(t, exportResp.YamlContent)

	yamlContent := exportResp.YamlContent
	t.Logf("Exported YAML:\n%s", string(yamlContent))

	// Verify YAML contains expected content.
	yamlStr := string(yamlContent)
	assert.Contains(t, yamlStr, "syntax:")
	assert.Contains(t, yamlStr, "name: export-e2e")
	assert.Contains(t, yamlStr, "trade.fee")
	assert.Contains(t, yamlStr, "trade.currency")

	// 3. Import identical YAML → should get AlreadyExists.
	_, err = schemaSvc.ImportSchema(ctx, &pb.ImportSchemaRequest{YamlContent: yamlContent})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))

	// 4. Modify YAML — add a field, re-import.
	modified := bytes.Replace(yamlContent,
		[]byte("    trade.timeout:"),
		[]byte("    trade.max_retries:\n        type: integer\n    trade.timeout:"),
		1,
	)
	importResp, err := schemaSvc.ImportSchema(ctx, &pb.ImportSchemaRequest{YamlContent: modified})
	require.NoError(t, err)
	assert.Equal(t, int32(2), importResp.Schema.Version)
	assert.Len(t, importResp.Schema.Fields, 4)
	assert.False(t, importResp.Schema.Published, "imported schema should be a draft")

	// 5. Verify via GetSchema that v2 exists.
	getResp, err := schemaSvc.GetSchema(ctx, &pb.GetSchemaRequest{Id: schemaID})
	require.NoError(t, err)
	assert.Equal(t, int32(2), getResp.Schema.Version)

	// 6. Import new schema by name that doesn't exist yet.
	newYAML := []byte(`syntax: "v1"
name: brand-new-e2e
description: Created via import
fields:
  config.enabled:
    type: string
    default: "true"
`)
	newImportResp, err := schemaSvc.ImportSchema(ctx, &pb.ImportSchemaRequest{YamlContent: newYAML})
	require.NoError(t, err)
	assert.Equal(t, "brand-new-e2e", newImportResp.Schema.Name)
	assert.Equal(t, int32(1), newImportResp.Schema.Version)

	// Cleanup.
	_, _ = schemaSvc.DeleteSchema(ctx, &pb.DeleteSchemaRequest{Id: schemaID})
	_, _ = schemaSvc.DeleteSchema(ctx, &pb.DeleteSchemaRequest{Id: newImportResp.Schema.Id})
}

// --- Config Export/Import ---

func TestConfigExportImport(t *testing.T) {
	conn := dial(t)
	schemaSvc := pb.NewSchemaServiceClient(conn)
	configSvc := pb.NewConfigServiceClient(conn)
	ctx := context.Background()

	// 1. Create and publish a schema with typed fields.
	createResp, err := schemaSvc.CreateSchema(ctx, &pb.CreateSchemaRequest{
		Name: "config-export-e2e",
		Fields: []*pb.SchemaField{
			{Path: "app.enabled", Type: pb.FieldType_FIELD_TYPE_BOOL},
			{Path: "app.max_retries", Type: pb.FieldType_FIELD_TYPE_INT},
			{Path: "app.fee_rate", Type: pb.FieldType_FIELD_TYPE_NUMBER},
			{Path: "app.name", Type: pb.FieldType_FIELD_TYPE_STRING},
			{Path: "app.timeout", Type: pb.FieldType_FIELD_TYPE_DURATION},
		},
	})
	require.NoError(t, err)
	schemaID := createResp.Schema.Id

	_, err = schemaSvc.PublishSchema(ctx, &pb.PublishSchemaRequest{Id: schemaID, Version: 1})
	require.NoError(t, err)

	// 2. Create a tenant.
	tenantResp, err := schemaSvc.CreateTenant(ctx, &pb.CreateTenantRequest{
		Name:          "config-export-tenant-e2e",
		SchemaId:      schemaID,
		SchemaVersion: 1,
	})
	require.NoError(t, err)
	tenantID := tenantResp.Tenant.Id

	// 3. Set config values.
	_, err = configSvc.SetFields(ctx, &pb.SetFieldsRequest{
		TenantId:    tenantID,
		Description: ptr("Initial config"),
		Updates: []*pb.FieldUpdate{
			{FieldPath: "app.enabled", Value: "true"},
			{FieldPath: "app.max_retries", Value: "3"},
			{FieldPath: "app.fee_rate", Value: "0.025"},
			{FieldPath: "app.name", Value: "MyApp"},
			{FieldPath: "app.timeout", Value: "30s"},
		},
	})
	require.NoError(t, err)

	// 4. Export config.
	exportResp, err := configSvc.ExportConfig(ctx, &pb.ExportConfigRequest{TenantId: tenantID})
	require.NoError(t, err)
	require.NotEmpty(t, exportResp.YamlContent)

	yamlStr := string(exportResp.YamlContent)
	t.Logf("Exported config YAML:\n%s", yamlStr)

	// Verify typed values in YAML.
	assert.Contains(t, yamlStr, "syntax:")
	assert.Contains(t, yamlStr, "app.enabled")
	assert.Contains(t, yamlStr, "app.max_retries")
	assert.Contains(t, yamlStr, "value: true")  // bool, not "true"
	assert.Contains(t, yamlStr, "value: 3")     // int, not "3"
	assert.Contains(t, yamlStr, "value: 0.025") // number, not "0.025"

	// 5. Modify YAML and import — change some values.
	modified := bytes.Replace(exportResp.YamlContent,
		[]byte("value: 3"),
		[]byte("value: 5"),
		1,
	)
	modified = bytes.Replace(modified,
		[]byte("value: true"),
		[]byte("value: false"),
		1,
	)

	importResp, err := configSvc.ImportConfig(ctx, &pb.ImportConfigRequest{
		TenantId:    tenantID,
		YamlContent: modified,
		Description: ptr("Updated via import"),
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), importResp.ConfigVersion.Version)

	// 6. Verify the imported values.
	getResp, err := configSvc.GetConfig(ctx, &pb.GetConfigRequest{TenantId: tenantID})
	require.NoError(t, err)

	valueMap := make(map[string]string)
	for _, v := range getResp.Config.Values {
		valueMap[v.FieldPath] = v.Value
	}
	assert.Equal(t, "false", valueMap["app.enabled"])
	assert.Equal(t, "5", valueMap["app.max_retries"])
	assert.Equal(t, "0.025", valueMap["app.fee_rate"])
	assert.Equal(t, "MyApp", valueMap["app.name"])
	assert.Equal(t, "30s", valueMap["app.timeout"])

	// Cleanup.
	_, _ = schemaSvc.DeleteTenant(ctx, &pb.DeleteTenantRequest{Id: tenantID})
	_, _ = schemaSvc.DeleteSchema(ctx, &pb.DeleteSchemaRequest{Id: schemaID})
}
