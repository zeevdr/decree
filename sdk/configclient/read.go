package configclient

import (
	"context"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
)

// Get returns the current value of a single configuration field.
// Returns [ErrNotFound] if the field has no value set.
func (c *Client) Get(ctx context.Context, tenantID, fieldPath string) (string, error) {
	resp, err := c.rpc.GetField(c.withAuth(ctx), &pb.GetFieldRequest{
		TenantId:  tenantID,
		FieldPath: fieldPath,
	})
	if err != nil {
		return "", mapError(err)
	}
	return derefString(resp.Value.Value), nil
}

// GetNullable returns the current value of a single configuration field as a *string.
// Returns nil if the field value is null, or [ErrNotFound] if the field has no value set.
func (c *Client) GetNullable(ctx context.Context, tenantID, fieldPath string) (*string, error) {
	resp, err := c.rpc.GetField(c.withAuth(ctx), &pb.GetFieldRequest{
		TenantId:  tenantID,
		FieldPath: fieldPath,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return resp.Value.Value, nil
}

// GetAll returns all configuration values for a tenant as a map of field path to value.
// Returns an empty map if no values are set.
func (c *Client) GetAll(ctx context.Context, tenantID string) (map[string]string, error) {
	resp, err := c.rpc.GetConfig(c.withAuth(ctx), &pb.GetConfigRequest{
		TenantId: tenantID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return configToMap(resp.Config), nil
}

// GetFields returns the values for the specified field paths.
// Fields that have no value set are omitted from the result.
func (c *Client) GetFields(ctx context.Context, tenantID string, fieldPaths []string) (map[string]string, error) {
	resp, err := c.rpc.GetFields(c.withAuth(ctx), &pb.GetFieldsRequest{
		TenantId:   tenantID,
		FieldPaths: fieldPaths,
	})
	if err != nil {
		return nil, mapError(err)
	}
	result := make(map[string]string, len(resp.Values))
	for _, v := range resp.Values {
		result[v.FieldPath] = derefString(v.Value)
	}
	return result, nil
}

func configToMap(cfg *pb.Config) map[string]string {
	if cfg == nil {
		return nil
	}
	m := make(map[string]string, len(cfg.Values))
	for _, v := range cfg.Values {
		m[v.FieldPath] = derefString(v.Value)
	}
	return m
}

// derefString safely dereferences a *string, returning "" for nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
