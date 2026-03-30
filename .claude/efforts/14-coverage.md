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
| sdk/configclient | 58.0% | 62.3% | 75% | in progress |
| cmd/decree | 57.8% | 57.8% | 65% | not started |
| internal | 41.1% | 46.4% | 65% | in progress |

## Completed

### Tier 1 — Quick Wins
- [x] internal/telemetry — ConfigFromEnv, envBool, AnyMetrics
- [x] sdk/configclient — With* builders, withAuth metadata/bearer paths
- [x] sdk/configwatcher — New, With* options, typed field registration, parsers, typedValueToString, Value.close, channel overflow

### Tier 2 — Services
- [x] internal/audit — QueryWriteLog, GetFieldUsage, GetTenantUsage, GetUnusedFields, UUID helpers, proto conversion
- [x] sdk/adminclient — With* builders, withAuth, all proto conversion helpers

### Tier 3 — Harder
- [x] internal/server — New, Serve, GracefulStop, IsServiceEnabled, SetServiceHealthy
- [x] sdk/adminclient — schema CRUD, tenant CRUD, locks, audit queries, usage stats, import/export, pagination, error mapping, all ServiceNotConfigured paths

## Remaining

### Still needed to reach targets
- [ ] sdk/configclient (62.3→75%) — SetTime, SetDuration setters, more error path coverage
- [ ] cmd/decree (57.8→65%) — CLI handlers with mocked gRPC
- [ ] internal (46.4→65%) — cache (mock Redis), pubsub (mock Redis), expand config service tests
