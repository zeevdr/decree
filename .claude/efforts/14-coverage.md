# Improve Test Coverage

**Status:** In Progress
**Started:** 2026-03-30

---

## Goal

Raise test coverage across all modules. The coverage ratchet prevents regression; this effort pushes numbers up.

## Results

| Module | Start | Current | Target | Status |
|--------|-------|---------|--------|--------|
| sdk/tools | 95.2% | 95.2% | maintain | done |
| sdk/configwatcher | 61.6% | 90.9% | 80% | exceeded |
| sdk/adminclient | 36.3% | 89.3% | 55% | exceeded |
| sdk/configclient | 58.0% | 81.6% | 75% | exceeded |
| cmd/decree | 57.8% | 57.8% | 65% | not started |
| internal | 41.1% | 43.5% | 65% | in progress |

## Completed

### Round 1 (PR #2)
- [x] internal/telemetry — ConfigFromEnv, envBool, AnyMetrics
- [x] sdk/configclient — With* builders, withAuth metadata/bearer paths
- [x] sdk/configwatcher — New, With* options, typed field registration, parsers, typedValueToString, Value.close, channel overflow
- [x] internal/audit — QueryWriteLog, GetFieldUsage, GetTenantUsage, GetUnusedFields, UUID helpers, proto conversion
- [x] sdk/adminclient — all CRUD, locks, audit, pagination, error mapping, ServiceNotConfigured
- [x] internal/server — New, Serve, GracefulStop, IsServiceEnabled

### Round 2 (current)
- [x] internal/telemetry — CacheMetrics, ConfigMetrics, SchemaMetrics nil-safe + enabled paths, StartDBPoolMetrics disabled
- [x] sdk/configclient — SetTime, SetDuration, SetTyped, GetTime, GetDuration, GetFields, GetBoolNullable, derefString, typedValueToString
- [x] internal/config/store_pg — all *FromDB mapping functions
- [x] internal/schema/store_pg — all *FromDB mapping functions
- [x] internal/audit/store_pg — all *FromDB mapping functions
- [x] Removed dead code: typedValueChecksum, validateTypedValueType (validation package handles this)

## Remaining

### Still needed to reach targets
- [ ] cmd/decree (57.8→65%) — CLI handlers with mocked gRPC
- [ ] internal (43.5→65%) — cache (needs Redis mock), pubsub (needs Redis mock), expand config/schema service tests
