# Central Config Service — Implementation

**Status:** In Progress
**Started:** 2025-03-25

---

## Progress

### Phase 1: Server Wiring (completed)

Core infrastructure — everything needed before implementing business logic.

- [x] **Database layer** (`internal/storage/postgres.go`) — pgxpool with separate read/write connection pools
- [x] **Cache interface + Redis impl** (`internal/cache/`) — `ConfigCache` interface, Redis hash-per-tenant-version
- [x] **Pub/Sub interface + Redis impl** (`internal/pubsub/`) — `Publisher` and `Subscriber` interfaces, Redis Pub/Sub with JSON events
- [x] **JWT auth interceptor** (`internal/auth/jwt.go`) — JWKS-based JWT validation, role extraction (superadmin/admin/user), unary + stream interceptors, health check bypass
- [x] **gRPC server** (`internal/server/server.go`) — listener, health checks, reflection, graceful shutdown, selective service registration via `--enable-services`
- [x] **Main wiring** (`cmd/server/main.go`) — connects all components: DB → Redis → cache → pubsub → auth → server → signal handling
- [x] **Context-aware logging** — all log calls use `slog.*Context(ctx, ...)` for future OTel trace correlation

### Phase 2: SchemaService (completed)

- [x] Schema CRUD (create with initial v1, get by version/latest, list, update creates new version, delete)
- [x] Schema versioning (publish, immutability, parent version lineage)
- [x] Schema field management (add, modify, remove via update)
- [x] Tenant CRUD (create with published schema validation, get, list, update name/version, delete)
- [x] Field locking (lock, unlock, list with JSON-encoded locked values)
- [x] Schema import/export (YAML) — OAS-style constraints, syntax v1, full-replace import with checksum dedup
- [x] Store interface with PostgreSQL implementation (read/write pool routing)
- [x] Proto <-> DB conversion helpers, deterministic checksum computation

### Phase 3: ConfigService (completed)

- [x] Config reads (GetConfig, GetField, GetFields) with Redis caching + `include_descriptions` bypass
- [x] Config writes (SetField, SetFields) with optimistic concurrency (checksum), field lock enforcement, audit logging, cache invalidation, pub/sub change events
- [x] Config versioning (ListVersions, GetVersion, RollbackToVersion — creates new version with target's values)
- [x] gRPC streaming subscriptions (Subscribe with field path filtering via Redis Pub/Sub)
- [ ] Config import/export (YAML) — stubbed as unimplemented
- [ ] Usage stats recording (async, batched) — deferred to AuditService
- [x] Store interface with PostgreSQL implementation (read/write pool routing)

### Phase 4: AuditService (completed)

- [x] Write log queries (QueryWriteLog with tenant/actor/field/time filters)
- [x] Usage statistics (GetFieldUsage — aggregated across periods, GetTenantUsage, GetUnusedFields)
- [x] Store interface with PostgreSQL implementation (read/write pool routing)
- [x] UpsertUsageStats available for future batched read tracking

### Unit Tests (in progress)

- [x] Schema service tests — CreateSchema, GetSchema (latest/specific), UpdateSchema (versioning), PublishSchema, CreateTenant (requires published schema)
- [x] Config service tests — GetConfig (cache hit/miss, include_descriptions bypass), SetField (success, checksum mismatch, locked field), GetField (not found), RollbackToVersion
- [x] Convert helper tests — UUID roundtrip, checksum determinism/order-independence, field type roundtrip, ptrString
- [x] Mock stores using testify/mock for schema, config, cache, pubsub
- [x] `auth.ContextWithClaims()` helper for injecting claims in tests
- [x] Auth interceptor tests — 20 tests: valid/expired/wrong-key tokens, health bypass, missing/bad auth headers, issuer validation, role validation, tenant_id enforcement, stream interceptor, ClaimsFromContext roundtrip

### Phase 5: Polish

- [ ] Helm chart
- [x] E2E tests — docker-compose stack (PG + Redis + migrate + service), 4 test suites: schema lifecycle, full flow (schema→tenant→config→lock→audit), streaming subscription, error cases
- [ ] OpenTelemetry integration (tracing + custom metrics)
- [ ] CI (GitHub Actions)

---

## Key Decisions During Implementation

1. **Go 1.24** — pinned to 1.24 for broader compatibility (otel v1.35, pgx v5.8, grpc v1.72)
2. **Tools image stays Go 1.25** — buf v1.66.1 requires it, but tools image is separate from the project's Go version
3. **JWT auth is optional** — if `JWT_JWKS_URL` is not set, auth is disabled (logged as warning). Useful for local dev.
4. **Context-aware logging** — `slog.*Context()` everywhere for future OTel trace ID correlation
5. **keyfunc v3** — uses context cancellation (not `End()`) for cleanup
6. **Makefile sentinel caching** — tools image only rebuilds when Dockerfile.tools changes; generate and lint-proto run in single containers
7. **Atomic config writes + audit** — `Store.RunInTx` callback wraps CreateConfigVersion + SetConfigValue + InsertAuditWriteLog in a single DB transaction. Side effects (cache invalidation, pub/sub) happen after commit. Audit failures now roll back the entire write (previously fire-and-forget).
