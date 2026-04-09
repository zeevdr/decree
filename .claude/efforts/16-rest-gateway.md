# REST/HTTP Gateway

**Status:** Planning
**Started:** 2026-04-09

---

## Goal

Expose the entire gRPC API as REST/JSON via grpc-gateway. Foundation for the admin GUI, non-Go SDKs, and curl-based usage.

## Approach

Embed grpc-gateway in the server binary (same pattern as MinIO Console, Grafana). Not a separate process.

## Work Items

### Proto Changes
- [ ] Add `buf.build/googleapis/googleapis` dependency to `buf.yaml`
- [ ] Add `import "google/api/annotations.proto"` to all 4 service protos
- [ ] Add `google.api.http` annotations to all 29 RPCs

### Build Tooling
- [ ] Add `protoc-gen-grpc-gateway` to `buf.gen.yaml` and `build/Dockerfile.tools`
- [ ] Add `protoc-gen-openapiv2` to `buf.gen.yaml` and `build/Dockerfile.tools`
- [ ] Run `make generate`, verify `.pb.gw.go` files and OpenAPI spec

### Server Integration
- [ ] New `internal/server/gateway.go` — HTTP mux with gateway handlers
- [ ] `HTTP_PORT` env var (default: 8080)
- [ ] Forward auth headers (x-subject, x-role, x-tenant-id, Authorization) from HTTP → gRPC metadata
- [ ] Wire into `cmd/server/main.go` — start HTTP server alongside gRPC
- [ ] Update `docker-compose.yml` — add port 8080

### URL Scheme
| Service | Method | URL |
|---------|--------|-----|
| GetServerVersion | GET | `/v1/version` |
| CreateSchema | POST | `/v1/schemas` |
| GetSchema | GET | `/v1/schemas/{id}` |
| ListSchemas | GET | `/v1/schemas` |
| UpdateSchema | PATCH | `/v1/schemas/{id}` |
| DeleteSchema | DELETE | `/v1/schemas/{id}` |
| PublishSchema | POST | `/v1/schemas/{id}/publish` |
| ExportSchema | GET | `/v1/schemas/{id}/export` |
| ImportSchema | POST | `/v1/schemas/import` |
| CreateTenant | POST | `/v1/tenants` |
| GetTenant | GET | `/v1/tenants/{id}` |
| ListTenants | GET | `/v1/tenants` |
| UpdateTenant | PATCH | `/v1/tenants/{id}` |
| DeleteTenant | DELETE | `/v1/tenants/{id}` |
| LockField | POST | `/v1/tenants/{tenant_id}/locks` |
| UnlockField | DELETE | `/v1/tenants/{tenant_id}/locks/{field_path}` |
| ListFieldLocks | GET | `/v1/tenants/{tenant_id}/locks` |
| GetConfig | GET | `/v1/tenants/{tenant_id}/config` |
| GetField | GET | `/v1/tenants/{tenant_id}/config/fields` |
| GetFields | POST | `/v1/tenants/{tenant_id}/config:batchGet` |
| SetField | PUT | `/v1/tenants/{tenant_id}/config/fields` |
| SetFields | POST | `/v1/tenants/{tenant_id}/config:batchSet` |
| ListVersions | GET | `/v1/tenants/{tenant_id}/versions` |
| GetVersion | GET | `/v1/tenants/{tenant_id}/versions/{version}` |
| RollbackToVersion | POST | `/v1/tenants/{tenant_id}/versions/{version}:rollback` |
| Subscribe | GET | `/v1/tenants/{tenant_id}/config:subscribe` |
| ExportConfig | GET | `/v1/tenants/{tenant_id}/config/export` |
| ImportConfig | POST | `/v1/tenants/{tenant_id}/config/import` |
| QueryWriteLog | GET | `/v1/audit/logs` |
| GetFieldUsage | GET | `/v1/tenants/{tenant_id}/usage/{field_path}` |
| GetTenantUsage | GET | `/v1/tenants/{tenant_id}/usage` |
| GetUnusedFields | GET | `/v1/tenants/{tenant_id}/unused-fields` |

Note: Field paths with dots use query params, not URL path segments.

### Outputs
- [ ] OpenAPI spec at `docs/api/openapi.json`
- [ ] Verify with curl: `curl http://localhost:8080/v1/version`

## Key Decisions
- Embedded in server binary, not a sidecar
- `HTTP_PORT` separate from `GRPC_PORT`
- Subscribe streams as newline-delimited JSON
