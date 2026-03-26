//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zeevdr/central-config-service/sdk/adminclient"
	"github.com/zeevdr/central-config-service/sdk/configclient"
)

// --- Constraint Validation ---

func TestConstraintValidation(t *testing.T) {
	conn := dial(t)
	admin := newAdminClient(conn)
	cfg := newConfigClient(conn)
	ctx := context.Background()

	// Create schema with constrained fields.
	s, err := admin.CreateSchema(ctx, "validation-e2e", []adminclient.Field{
		{
			Path: "app.retries", Type: "FIELD_TYPE_INT",
			Constraints: &adminclient.FieldConstraints{Min: ptr(0.0), Max: ptr(10.0)},
		},
		{
			Path: "app.rate", Type: "FIELD_TYPE_NUMBER",
			Constraints: &adminclient.FieldConstraints{Min: ptr(0.0), Max: ptr(1.0)},
		},
		{
			Path: "app.name", Type: "FIELD_TYPE_STRING",
			Constraints: &adminclient.FieldConstraints{Min: ptr(2.0), Max: ptr(50.0)},
		},
		{
			Path: "app.env", Type: "FIELD_TYPE_STRING",
			Constraints: &adminclient.FieldConstraints{Enum: []string{"dev", "staging", "prod"}},
		},
		{
			Path: "app.webhook", Type: "FIELD_TYPE_URL",
		},
		{
			Path: "app.enabled", Type: "FIELD_TYPE_BOOL",
		},
	}, "")
	require.NoError(t, err)
	_, err = admin.PublishSchema(ctx, s.ID, 1)
	require.NoError(t, err)

	tenant, err := admin.CreateTenant(ctx, "validation-tenant-e2e", s.ID, 1)
	require.NoError(t, err)

	// --- Valid values should pass ---

	t.Run("valid values accepted", func(t *testing.T) {
		require.NoError(t, cfg.SetInt(ctx, tenant.ID, "app.retries", 5))
		require.NoError(t, cfg.SetFloat(ctx, tenant.ID, "app.rate", 0.5))
		require.NoError(t, cfg.Set(ctx, tenant.ID, "app.name", "MyApp"))
		require.NoError(t, cfg.Set(ctx, tenant.ID, "app.env", "prod"))
		require.NoError(t, cfg.SetBool(ctx, tenant.ID, "app.enabled", true))
	})

	// --- Constraint violations should fail with informative errors ---

	t.Run("integer above max", func(t *testing.T) {
		err := cfg.SetInt(ctx, tenant.ID, "app.retries", 11)
		assert.ErrorIs(t, err, configclient.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "maximum")
	})

	t.Run("integer below min", func(t *testing.T) {
		err := cfg.SetInt(ctx, tenant.ID, "app.retries", -1)
		assert.ErrorIs(t, err, configclient.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "minimum")
	})

	t.Run("number out of range", func(t *testing.T) {
		err := cfg.SetFloat(ctx, tenant.ID, "app.rate", 1.5)
		assert.ErrorIs(t, err, configclient.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "maximum")
	})

	t.Run("string too short", func(t *testing.T) {
		err := cfg.Set(ctx, tenant.ID, "app.name", "x")
		assert.ErrorIs(t, err, configclient.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "minimum")
	})

	t.Run("enum violation", func(t *testing.T) {
		err := cfg.Set(ctx, tenant.ID, "app.env", "local")
		assert.ErrorIs(t, err, configclient.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "not in allowed")
	})

	t.Run("invalid url", func(t *testing.T) {
		err := cfg.Set(ctx, tenant.ID, "app.webhook", "not-a-url")
		assert.ErrorIs(t, err, configclient.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "URL")
	})

	// --- Strict mode: unknown fields rejected ---

	t.Run("unknown field rejected", func(t *testing.T) {
		err := cfg.Set(ctx, tenant.ID, "app.nonexistent", "value")
		assert.ErrorIs(t, err, configclient.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "not defined")
	})

	// --- Type mismatch ---

	t.Run("wrong type rejected", func(t *testing.T) {
		err := cfg.Set(ctx, tenant.ID, "app.retries", "not-a-number")
		assert.ErrorIs(t, err, configclient.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "expected integer")
	})

	// --- Import validation ---

	t.Run("import valid YAML accepted", func(t *testing.T) {
		validYAML := []byte(`syntax: "v1"
values:
  app.retries:
    value: 5
  app.name:
    value: "ValidApp"
  app.env:
    value: "dev"
`)
		_, err := admin.ImportConfig(ctx, tenant.ID, validYAML, "valid import")
		require.NoError(t, err)
	})

	t.Run("import rejects constraint violation", func(t *testing.T) {
		badYAML := []byte(`syntax: "v1"
values:
  app.retries:
    value: 99
`)
		_, err := admin.ImportConfig(ctx, tenant.ID, badYAML, "bad import")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maximum")
	})

	t.Run("import rejects unknown field", func(t *testing.T) {
		badYAML := []byte(`syntax: "v1"
values:
  app.nonexistent:
    value: "hello"
`)
		_, err := admin.ImportConfig(ctx, tenant.ID, badYAML, "unknown field import")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not defined")
	})

	// Cleanup.
	_ = admin.DeleteTenant(ctx, tenant.ID)
	_ = admin.DeleteSchema(ctx, s.ID)
}
