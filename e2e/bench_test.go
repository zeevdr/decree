//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/zeevdr/central-config-service/sdk/adminclient"
	"github.com/zeevdr/central-config-service/sdk/configclient"
)

// benchEnv sets up a schema + tenant for benchmarks and returns cleanup func.
func benchEnv(b *testing.B, name string) (*configclient.Client, string, func()) {
	b.Helper()
	conn := dialBench(b)
	admin := newAdminClient(conn)
	cfg := newConfigClient(conn)
	ctx := context.Background()

	s, err := admin.CreateSchema(ctx, name, []adminclient.Field{
		{Path: "bench.string", Type: "FIELD_TYPE_STRING"},
		{Path: "bench.int", Type: "FIELD_TYPE_INT"},
		{Path: "bench.bool", Type: "FIELD_TYPE_BOOL"},
		{Path: "bench.number", Type: "FIELD_TYPE_NUMBER"},
	}, "")
	require.NoError(b, err)
	_, err = admin.PublishSchema(ctx, s.ID, 1)
	require.NoError(b, err)

	tenant, err := admin.CreateTenant(ctx, name+"-tenant", s.ID, 1)
	require.NoError(b, err)

	// Seed initial values.
	require.NoError(b, cfg.Set(ctx, tenant.ID, "bench.string", "hello"))
	require.NoError(b, cfg.SetInt(ctx, tenant.ID, "bench.int", 42))
	require.NoError(b, cfg.SetBool(ctx, tenant.ID, "bench.bool", true))
	require.NoError(b, cfg.SetFloat(ctx, tenant.ID, "bench.number", 3.14))

	cleanup := func() {
		_ = admin.DeleteTenant(ctx, tenant.ID)
		_ = admin.DeleteSchema(ctx, s.ID)
	}

	return cfg, tenant.ID, cleanup
}

func dialBench(b *testing.B) *grpc.ClientConn {
	b.Helper()
	ctx := context.Background()
	conn, err := grpc.NewClient(serviceAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(b, err)
	_ = ctx
	b.Cleanup(func() { conn.Close() })
	return conn
}

// --- Sequential latency benchmarks ---

func BenchmarkGetField(b *testing.B) {
	cfg, tenantID, cleanup := benchEnv(b, "bench-get")
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = cfg.Get(ctx, tenantID, "bench.string")
	}
}

func BenchmarkGetAll(b *testing.B) {
	cfg, tenantID, cleanup := benchEnv(b, "bench-getall")
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = cfg.GetAll(ctx, tenantID)
	}
}

func BenchmarkSetField(b *testing.B) {
	cfg, tenantID, cleanup := benchEnv(b, "bench-set")
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		_ = cfg.Set(ctx, tenantID, "bench.string", fmt.Sprintf("val-%d", i))
	}
}

func BenchmarkSetInt(b *testing.B) {
	cfg, tenantID, cleanup := benchEnv(b, "bench-setint")
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		_ = cfg.SetInt(ctx, tenantID, "bench.int", int64(i))
	}
}

// --- Parallel throughput benchmarks ---

func BenchmarkGetField_Parallel(b *testing.B) {
	cfg, tenantID, cleanup := benchEnv(b, "bench-get-par")
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cfg.Get(ctx, tenantID, "bench.string")
		}
	})
}

func BenchmarkGetAll_Parallel(b *testing.B) {
	cfg, tenantID, cleanup := benchEnv(b, "bench-getall-par")
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cfg.GetAll(ctx, tenantID)
		}
	})
}

func BenchmarkSetField_Parallel(b *testing.B) {
	cfg, tenantID, cleanup := benchEnv(b, "bench-set-par")
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = cfg.Set(ctx, tenantID, "bench.string", fmt.Sprintf("val-%d", i))
			i++
		}
	})
}

// --- Mixed workload ---

func BenchmarkMixed_90Read_10Write(b *testing.B) {
	cfg, tenantID, cleanup := benchEnv(b, "bench-mixed")
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if i%10 == 0 {
			_ = cfg.Set(ctx, tenantID, "bench.string", fmt.Sprintf("val-%d", i))
		} else {
			_, _ = cfg.Get(ctx, tenantID, "bench.string")
		}
	}
}
