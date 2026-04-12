# TypeScript SDK (`@opendecree/sdk`)

**Status:** Complete
**Started:** 2026-04-12
**Completed:** 2026-04-12
**Repo:** `zeevdr/decree-typescript`
**npm:** `@opendecree/sdk` (org: `opendecree`)
**Node.js:** 20+

---

## Goal

A production-quality TypeScript SDK for OpenDecree that covers config reads, writes, and live change subscriptions. Async-only (Node.js is fundamentally async). Vanilla tooling, high coverage.

## Scope (v0.1.0)

| Feature | Description |
|---------|-------------|
| ConfigClient | `get(t, f)` → string, `get(t, f, Number)` → number. set, setMany, setNull, getAll. |
| ConfigWatcher | `client.watch(t)` → `watcher.field(path, Number, { default })` → `WatchedField<T>`. EventEmitter + async iteration. |
| Auth | Metadata headers (x-subject, x-role, x-tenant-id) and Bearer token via interceptors |
| Error mapping | gRPC status codes → typed Error subclasses |
| Retry | Exponential backoff with jitter for transient errors |
| Compatibility | `client.serverVersion` + `client.checkCompatibility()` |
| Dispose | `Symbol.dispose` support + `close()` for cleanup |

### NOT in v0.1.0

- AdminClient (schema/tenant management) — use CLI or REST
- Tools (diff, validate, seed, dump) — use CLI
- Browser support — use REST gateway
- Contrib packages (Express, Fastify middleware) — later

## Naming

| Aspect | Value |
|--------|-------|
| npm package | `@opendecree/sdk` |
| npm org | `opendecree` |
| Import | `import { ConfigClient } from '@opendecree/sdk'` |
| Repo | `zeevdr/decree-typescript` |

## Tech Stack

| Concern | Tool | Why |
|---------|------|-----|
| Language | TypeScript 5.5+ | Strict mode, emit ESM + declarations |
| Runtime | Node.js 20+ | 20 is current LTS, 18 already EOL |
| Modules | ESM-only | Modern standard, no CJS complexity |
| gRPC | @grpc/grpc-js | Official, pure JS, maintained by gRPC team |
| Proto gen | buf + ts-proto | Already use buf, ts-proto generates idiomatic TS |

**Runtime deps: `@grpc/grpc-js` only.** Generated proto code is committed.

### Dev tools (not shipped to users)

| Concern | Tool | Why |
|---------|------|-----|
| Build | tsc | Standard, no bundler needed for a library |
| Lint + format | Biome | Single tool, fast (like ruff for Python) |
| Test | vitest | Modern standard, fast, built-in coverage + mocking |
| Proto gen | buf + ts-proto (Docker) | Consistent with Go/Python repos |
| Publish | npm provenance (OIDC) | No tokens, same as PyPI trusted publishing |

## Project Structure

```
decree-typescript/
├── src/                          # TypeScript source
│   ├── index.ts                  # Public API barrel export
│   ├── client.ts                 # ConfigClient
│   ├── watcher.ts                # ConfigWatcher + WatchedField<T>
│   ├── errors.ts                 # Error classes
│   ├── types.ts                  # Interfaces (readonly, plain objects)
│   ├── retry.ts                  # Exponential backoff with jitter
│   ├── channel.ts                # gRPC channel factory
│   ├── compat.ts                 # Version compatibility checking
│   ├── convert.ts                # TypedValue ↔ native conversion
│   └── generated/                # ts-proto generated code (committed)
│       └── centralconfig/
│           └── v1/
├── test/                         # Tests (*.test.ts suffix)
│   ├── client.test.ts
│   ├── watcher.test.ts
│   ├── errors.test.ts
│   ├── retry.test.ts
│   ├── convert.test.ts
│   └── compat.test.ts
├── docs/                         # Usage docs
│   ├── quickstart.md
│   ├── configuration.md
│   └── watching.md
├── contrib/                      # Future: Express, Fastify middleware
├── package.json                  # Metadata, scripts, deps (replaces Makefile + pyproject.toml)
├── tsconfig.json                 # TypeScript strict config
├── biome.json                    # Lint + format config
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                # lint, typecheck, test (Node 20/22/24)
│   │   └── publish.yml           # npm publish with provenance on tag
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   └── feature_request.md
│   └── PULL_REQUEST_TEMPLATE.md
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
├── CODE_OF_CONDUCT.md
├── LICENSE                       # Apache 2.0
├── README.md
└── .gitignore
```

### Key structural decisions

- **No `sdk/` wrapper** — npm publishes from root, unlike Python's `sdk/` subdirectory
- **No Makefile** — `npm run` scripts in package.json (TS convention)
- **No Docker for dev tools** — all tools are npm packages, `npm install` gives exact versions
- **Docker only for proto generation** — buf runs in Docker (same as other repos)
- **`test/` not `tests/`** — Node.js convention
- **`*.test.ts` suffix** — TS convention (not `test_*.ts` prefix)
- **`dist/` is build output** — gitignored, tsc emits here
- **Flat `src/`** — no nested namespace directories (not needed in TS)
- **`contrib/` at root** — placeholder, restructure to npm workspaces when first contrib is added

## Public API Design

### TypeScript-native patterns

- **Generics with runtime converters** — `get(t, f, Number)` → `number` (not overloads)
- **EventEmitter** for watcher changes — built into Node.js, zero deps
- **`for await...of`** for async change iteration
- **`Symbol.dispose`** for automatic cleanup (Node 22+, graceful fallback)
- **Interfaces, not classes** for return types — plain objects, zero runtime cost
- **`readonly` properties** — immutable by convention (TS equivalent of frozen dataclass)
- **camelCase** everywhere — `fieldPath`, not `field_path`
- **Async-only** — no sync API (Node.js is fundamentally async)

### ConfigClient

```typescript
import { ConfigClient } from '@opendecree/sdk';

// Explicit cleanup
const client = new ConfigClient('localhost:9090', { subject: 'myapp' });
try {
  // Default: string
  const fee = await client.get('tenant-id', 'payments.fee');

  // Typed via runtime converter
  const retries = await client.get('tenant-id', 'payments.retries', Number);
  const enabled = await client.get('tenant-id', 'payments.enabled', Boolean);

  // Set
  await client.set('tenant-id', 'payments.fee', '0.5%');

  // Bulk
  const all = await client.getAll('tenant-id');
  await client.setMany('tenant-id', { a: '1', b: '2' }, { description: 'batch' });

  // Null
  await client.setNull('tenant-id', 'payments.fee');

  // Nullable get
  const val = await client.get('tenant-id', 'payments.fee', String, { nullable: true });

  // Compatibility
  const version = await client.serverVersion;
  await client.checkCompatibility();
} finally {
  client.close();
}

// Or with Symbol.dispose (Node 22+, TS 5.2+)
{
  using client = new ConfigClient('localhost:9090', { subject: 'myapp' });
  const fee = await client.get('tenant-id', 'payments.fee');
} // auto-disposed
```

### ConfigWatcher

```typescript
import { ConfigClient } from '@opendecree/sdk';

const client = new ConfigClient('localhost:9090', { subject: 'myapp' });
const watcher = client.watch('tenant-id');

// Register fields before starting
const fee = watcher.field('payments.fee', Number, { default: 0.01 });
const enabled = watcher.field('payments.enabled', Boolean, { default: false });

await watcher.start();

// .value — always fresh
console.log(fee.value);  // 0.025

// EventEmitter pattern
fee.on('change', (oldVal, newVal) => {
  console.log(`Fee changed: ${oldVal} → ${newVal}`);
});

// Async iteration
for await (const change of fee) {
  console.log(`${change.oldValue} → ${change.newValue}`);
}

await watcher.stop();
client.close();
```

### Client Options

```typescript
interface ClientOptions {
  // Auth (metadata headers)
  subject?: string;        // x-subject
  role?: string;           // x-role (default: "superadmin")
  tenantId?: string;       // x-tenant-id
  token?: string;          // Bearer token (overrides metadata)

  // Connection
  insecure?: boolean;      // plaintext, default true
  credentials?: grpc.ChannelCredentials;

  // Behavior
  timeout?: number;        // default RPC timeout in ms (default: 10000)
  retry?: RetryConfig | false;  // false to disable
}
```

### Error Hierarchy

```typescript
class DecreeError extends Error {
  readonly code?: grpc.status;
}
class NotFoundError extends DecreeError {}
class AlreadyExistsError extends DecreeError {}
class InvalidArgumentError extends DecreeError {}
class LockedError extends DecreeError {}
class ChecksumMismatchError extends DecreeError {}
class PermissionDeniedError extends DecreeError {}
class UnavailableError extends DecreeError {}
class IncompatibleServerError extends DecreeError {}
class TypeMismatchError extends DecreeError {}
```

### Return Types (interfaces, not classes)

```typescript
interface ConfigValue {
  readonly fieldPath: string;
  readonly value: string;
  readonly checksum: string;
  readonly description: string;
}

interface Change {
  readonly fieldPath: string;
  readonly oldValue: string | null;
  readonly newValue: string | null;
  readonly version: number;
  readonly changedBy: string;
}

interface ServerVersion {
  readonly version: string;
  readonly commit: string;
}

interface RetryConfig {
  maxAttempts?: number;      // default: 3
  initialBackoff?: number;   // ms, default: 100
  maxBackoff?: number;       // ms, default: 5000
  multiplier?: number;       // default: 2
  retryableCodes?: grpc.status[];
}
```

## Implementation Phases

### Phase 1: Scaffold + Stubs (day 1)

- [ ] Create repo `zeevdr/decree-typescript`
- [ ] `package.json` with name, version, type: module, exports, scripts
- [ ] `tsconfig.json` with strict mode, ESM output, declaration emit
- [ ] `biome.json` config
- [ ] `vitest.config.ts` with coverage
- [ ] Proto generation: buf + ts-proto via Docker
- [ ] Commit generated stubs to `src/generated/`
- [ ] `.gitattributes`, `.gitignore`
- [ ] Empty `index.ts` with version + public API stubs
- [ ] CI workflow: lint (biome), typecheck (tsc), test (vitest, Node 20/22/24)
- [ ] Publish workflow: npm provenance on `v*` tags
- [ ] Add repo to GitHub Project

### Phase 2: ConfigClient (days 2-3)

- [ ] `channel.ts` — channel factory (insecure/TLS, keepalive)
- [ ] `errors.ts` — error hierarchy + gRPC status mapping
- [ ] `types.ts` — ConfigValue, Change, ServerVersion interfaces
- [ ] `retry.ts` — exponential backoff with jitter
- [ ] `convert.ts` — TypedValue ↔ native conversion
- [ ] `compat.ts` — VersionService call + version comparison
- [ ] `client.ts` — ConfigClient with typed get(), Symbol.dispose
- [ ] Tests for all client methods (mocked stubs)

### Phase 3: ConfigWatcher (days 4-5)

- [ ] `watcher.ts` — ConfigWatcher + WatchedField<T>
  - EventEmitter for change callbacks
  - `for await...of` async iteration
  - Auto-reconnect with exponential backoff
  - Initial snapshot loading
- [ ] Tests for watcher lifecycle, reconnection, type conversion

### Phase 4: Docs + Distribution (day 6)

- [ ] `README.md` — badges, install, quickstart
- [ ] `docs/quickstart.md`, `configuration.md`, `watching.md`
- [ ] `CHANGELOG.md` — v0.1.0 entry
- [ ] Governance: SECURITY.md, CODE_OF_CONDUCT.md, CONTRIBUTING.md
- [ ] Issue templates, PR template
- [ ] npm provenance setup (OIDC — add to npm org)
- [ ] Branch protection on main
- [ ] Tag v0.1.0, verify `npm install @opendecree/sdk` works

## Key Decisions

1. **Async-only** — no sync API; Node.js is fundamentally async
2. **ESM-only** — modern standard, no CJS dual-package complexity
3. **Node 20+** — 18 is EOL, 20 is current LTS
4. **`@grpc/grpc-js`** — official gRPC library, same reasoning as grpcio for Python
5. **buf + ts-proto** — consistent with Go/Python, idiomatic TS output (interfaces not classes)
6. **tsc for build** — standard, no bundler needed for a library this size
7. **Biome for lint + format** — single tool (like ruff for Python)
8. **vitest for testing** — modern standard, fast, built-in coverage
9. **Interfaces for return types** — plain readonly objects, not class instances
10. **Runtime converters for typed get()** — `get(t, f, Number)` not `get<number>(t, f)`
11. **EventEmitter for watcher** — built into Node.js, TS-native pattern
12. **`for await...of`** for async change iteration
13. **`Symbol.dispose`** for auto-cleanup (graceful fallback to `close()`)
14. **camelCase** — TS convention, not snake_case or PascalCase for properties
15. **npm provenance** — OIDC, no tokens (same pattern as PyPI trusted publishing)
16. **`@opendecree/sdk`** scoped package — npm org `opendecree` created
17. **No Docker for dev tools** — npm packages, not system binaries
18. **`contrib/`** at root — placeholder, restructure to workspaces when needed

## Differences from Python SDK

| Aspect | Python | TypeScript |
|--------|--------|------------|
| Sync + async | Both (separate classes) | Async-only (one client) |
| Module system | N/A | ESM-only |
| Return types | `@dataclass(frozen=True)` | `interface` with `readonly` |
| Watcher callbacks | `@field.on_change` decorator | `field.on('change', fn)` EventEmitter |
| Change iteration | `for change in field.changes()` | `for await (const change of field)` |
| Resource cleanup | `with` context manager | `Symbol.dispose` + `close()` |
| Typed gets | `@overload` + type arg | Generic + runtime converter (`Number`) |
| Dev tools | Docker (ruff, mypy, pytest) | npm (biome, tsc, vitest) |
| Build | Makefile + Docker | `npm run` scripts |
| Package scope | `opendecree` (unscoped) | `@opendecree/sdk` (scoped) |

## Verification

```bash
npm run lint              # biome check
npm run build             # tsc → dist/
npm test                  # vitest with coverage
npm run test:coverage     # coverage report
npm pack                  # create tarball for inspection
```
