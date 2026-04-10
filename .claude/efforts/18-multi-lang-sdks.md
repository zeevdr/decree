# Multi-Language SDKs

**Status:** In Progress (Python SDK detailed, TS SDK planned)
**Started:** 2026-04-09

---

## Strategy

Separate repos per language, proto published to BSR. Each SDK is independently versioned and released.

| SDK | Repo | Package | Status | Effort |
|-----|------|---------|--------|--------|
| Python | `zeevdr/decree-python` | `opendecree` (PyPI) | Detailed | `18-python-sdk.md` |
| TypeScript | `zeevdr/decree-sdk-ts` | `@opendecree/sdk` (npm) | Planned | (below) |

Proto source of truth stays in `zeevdr/decree`, published to BSR: `buf.build/opendecree/decree`.

## TypeScript SDK (`zeevdr/decree-sdk-ts`)

**Stack:** `@connectrpc/connect` + `@bufbuild/protobuf` (buf ecosystem, gRPC + HTTP)
**Package:** `@opendecree/sdk` on npm

### Wrapper API
- `ConfigClient` — get, getAll, set, setMany, setNull + typed (getInt, getBool, getFloat)
- `AdminClient` — createSchema, getSchema, listSchemas, publishSchema, createTenant, getTenant, listTenants, lockField, unlockField
- Auth interceptor (subject, role, tenantId, bearerToken)
- Error mapping (ConnectError codes → NotFoundError, LockedError, AlreadyExistsError, etc.)

### Work Items
- [ ] Scaffold repo with package.json, tsconfig.json
- [ ] `buf.gen.yaml` pulling from BSR
- [ ] Generate stubs into `src/gen/`
- [ ] Implement `ConfigClient` wrapper
- [ ] Implement `AdminClient` wrapper
- [ ] Auth interceptor
- [ ] Error mapping
- [ ] Tests (vitest with mocked Connect transport)
- [ ] README with install + usage examples
- [ ] CI (GitHub Actions: lint, test, build)
- [ ] npm publish setup

## Key Decisions
- Separate repos (independent release cycles, CI, package managers)
- Proto via BSR (decouples from main repo file structure)
- Connect for TS (supports both gRPC and HTTP transport)
- Official grpcio for Python (most widely adopted)
- Python SDK detailed in its own effort file (`18-python-sdk.md`)
