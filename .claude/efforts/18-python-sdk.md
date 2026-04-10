# Python SDK (`opendecree`)

**Status:** Planning
**Started:** 2026-04-10
**Repo:** `zeevdr/decree-python`
**PyPI:** `opendecree`
**Python:** 3.11+

---

## Goal

A production-quality Python SDK for OpenDecree that covers config reads, writes, and live change subscriptions. Both sync and async APIs. Vanilla tooling only — standard, widely-adopted libraries.

## Scope (v0.1.0)

| Feature | Description |
|---------|-------------|
| ConfigClient (sync) | get, get_all, set, set_many, set_null + typed getters (get_int, get_bool, get_float, get_string, get_duration) |
| AsyncConfigClient | Same API but async/await |
| ConfigWatcher (sync) | Register fields, background thread, .get() for current value, callbacks for changes |
| AsyncConfigWatcher | Same but asyncio-native, async iterator for changes |
| Auth | Metadata headers (x-subject, x-role, x-tenant-id) and Bearer token injection via interceptors |
| Error mapping | gRPC StatusCode → typed Python exceptions (NotFoundError, LockedError, etc.) |
| Retry | Exponential backoff with jitter for transient errors (UNAVAILABLE, DEADLINE_EXCEEDED) |
| Compatibility | Reports supported proto/server version, optional check_compatibility() call |

### NOT in v0.1.0

- AdminClient (schema/tenant management) — use CLI or REST
- Tools (diff, validate, seed, dump) — use CLI
- Snapshot reads — add later if demanded
- Optimistic concurrency (CAS) — add later

## Naming

| Aspect | Value |
|--------|-------|
| PyPI package | `opendecree` |
| Import | `import opendecree` |
| Repo | `zeevdr/decree-python` |
| SDK source | `sdk/src/opendecree/` |
| Generated stubs | `sdk/src/opendecree/_generated/` |

The repo is `decree-python` (not `decree-sdk-python`) because it may also contain contrib packages (e.g., `contrib/viper`, `contrib/fastapi`) in the future. The SDK lives under `sdk/`.

## Tech Stack

| Concern | Tool | Version | Why |
|---------|------|---------|-----|
| Runtime | Python | ≥3.11 | 3.10 loses security support soon |
| gRPC | grpcio | ≥1.68,<2 | Official, C-core, sync+async |
| Protobuf | protobuf | ≥5.29,<6 | Official runtime |

**That's it for runtime deps. Two packages.**

### Dev/build tools (not shipped to users)

| Concern | Tool | Why |
|---------|------|-----|
| Build | setuptools | Comes with Python, PEP 621, no plugins |
| Linting | ruff | Replaces black+isort+flake8, single tool |
| Type checking | mypy | Standard |
| Testing | pytest + pytest-asyncio | Standard |
| Proto gen | grpcio-tools + mypy-protobuf | Official Python generation path |
| Publishing | PyPI trusted publishing (OIDC) | No API tokens |

## Project Structure

```
decree-python/
├── sdk/                             # SDK package (published to PyPI as opendecree)
│   ├── src/
│   │   └── opendecree/
│   │       ├── __init__.py          # version, public API re-exports
│   │       ├── py.typed             # PEP 561 marker
│   │       ├── client.py            # ConfigClient (sync)
│   │       ├── async_client.py      # AsyncConfigClient
│   │       ├── watcher.py           # ConfigWatcher (sync, background thread)
│   │       ├── async_watcher.py     # AsyncConfigWatcher (asyncio)
│   │       ├── _channel.py          # channel factory, keepalive, TLS config
│   │       ├── _interceptors.py     # auth metadata interceptors (sync + async)
│   │       ├── _retry.py            # exponential backoff with jitter
│   │       ├── _compat.py           # server version check via VersionService
│   │       ├── errors.py            # exception hierarchy (public)
│   │       ├── types.py             # dataclass wrappers: ConfigValue, Change, etc.
│   │       └── _generated/          # proto stubs (committed, linguist-generated)
│   │           ├── __init__.py
│   │           └── centralconfig/
│   │               └── v1/
│   │                   ├── __init__.py
│   │                   ├── types_pb2.py / .pyi
│   │                   ├── config_service_pb2.py / .pyi
│   │                   ├── config_service_pb2_grpc.py / .pyi
│   │                   └── version_service_pb2.py / .pyi
│   ├── tests/
│   │   ├── conftest.py              # shared fixtures, mock stubs
│   │   ├── test_client.py
│   │   ├── test_async_client.py
│   │   ├── test_watcher.py
│   │   ├── test_async_watcher.py
│   │   ├── test_retry.py
│   │   ├── test_errors.py
│   │   └── test_compat.py
│   ├── docs/
│   │   ├── quickstart.md            # install + first get/set
│   │   ├── configuration.md         # client options, auth, TLS
│   │   ├── watching.md              # watcher usage + patterns
│   │   └── async.md                 # async client + watcher usage
│   ├── pyproject.toml               # hatchling + hatch-vcs, PyPI metadata
│   └── CHANGELOG.md                 # Keep a Changelog format
├── contrib/                         # future: contrib packages (fastapi, etc.)
├── Makefile                         # top-level: generate, lint, test, build
├── LICENSE                          # Apache 2.0
├── README.md                        # repo overview, links to sdk/ and contrib/
├── .gitattributes                   # mark _generated as linguist-generated
├── .github/
│   └── workflows/
│       ├── ci.yml                   # lint, type-check, test (3.11/3.12/3.13)
│       └── publish.yml              # PyPI trusted publishing on tag
└── .python-version                  # 3.11
```

## Public API Design

### ConfigClient (sync)

```python
from opendecree import ConfigClient

# Create client — context manager for clean channel lifecycle
with ConfigClient("localhost:9090", subject="myapp") as client:
    # String get/set (any type as string)
    val = client.get("tenant-id", "payments.fee")
    client.set("tenant-id", "payments.fee", "0.5%")

    # Typed getters — return native Python types
    retries = client.get_int("tenant-id", "payments.retries")
    enabled = client.get_bool("tenant-id", "payments.enabled")
    rate = client.get_float("tenant-id", "payments.fee_rate")
    timeout = client.get_duration("tenant-id", "payments.timeout")  # → timedelta

    # Bulk operations
    all_config = client.get_all("tenant-id")  # → dict[str, str]
    client.set_many("tenant-id", {"a": "1", "b": "2"}, description="batch update")

    # Null
    client.set_null("tenant-id", "payments.fee")

    # Nullable getter
    val = client.get_nullable("tenant-id", "payments.fee")  # → str | None

    # Server compatibility check
    client.check_compatibility()  # raises IncompatibleServerError if mismatch
```

### AsyncConfigClient

```python
from opendecree import AsyncConfigClient

async with AsyncConfigClient("localhost:9090", subject="myapp") as client:
    val = await client.get("tenant-id", "payments.fee")
    await client.set("tenant-id", "payments.fee", "0.5%")
    retries = await client.get_int("tenant-id", "payments.retries")
```

### ConfigWatcher (sync)

```python
from opendecree import ConfigClient, ConfigWatcher

with ConfigClient("localhost:9090", subject="myapp") as client:
    watcher = ConfigWatcher(client, "tenant-id")

    # Register fields with type + default
    fee = watcher.float_field("payments.fee", default=0.01)
    enabled = watcher.bool_field("payments.enabled", default=False)

    watcher.start()  # background thread, loads snapshot + subscribes

    # Always-fresh reads (thread-safe)
    print(fee.get())       # 0.025
    print(enabled.get())   # True

    # Change callbacks
    @fee.on_change
    def fee_changed(old: float, new: float):
        print(f"Fee changed: {old} → {new}")

    # Or iterate changes
    for change in fee.changes():  # blocking iterator
        print(change)

    watcher.stop()
```

### AsyncConfigWatcher

```python
from opendecree import AsyncConfigClient, AsyncConfigWatcher

async with AsyncConfigClient("localhost:9090", subject="myapp") as client:
    watcher = AsyncConfigWatcher(client, "tenant-id")
    fee = watcher.float_field("payments.fee", default=0.01)

    await watcher.start()

    print(fee.get())  # always-fresh, thread-safe

    async for change in fee.changes():  # async iterator
        print(f"{change.old_value} → {change.new_value}")

    await watcher.stop()
```

### Client Options

```python
ConfigClient(
    target="localhost:9090",
    *,
    # Auth (metadata headers)
    subject: str | None = None,     # x-subject
    role: str = "superadmin",       # x-role
    tenant_id: str | None = None,   # x-tenant-id
    token: str | None = None,       # Bearer token (alternative to metadata)
    # Connection
    insecure: bool = True,          # skip TLS (default for dev)
    tls_credentials: grpc.ChannelCredentials | None = None,
    # Behavior
    timeout: float = 10.0,          # default RPC timeout (seconds)
    retry: RetryConfig | None = RetryConfig(),  # exponential backoff
)
```

### RetryConfig

```python
@dataclass
class RetryConfig:
    max_attempts: int = 3
    initial_backoff: float = 0.1    # seconds
    max_backoff: float = 5.0        # seconds
    multiplier: float = 2.0
    retryable_codes: tuple[grpc.StatusCode, ...] = (
        grpc.StatusCode.UNAVAILABLE,
        grpc.StatusCode.DEADLINE_EXCEEDED,
    )
```

### Error Hierarchy

```python
class DecreeError(Exception):           # base
class NotFoundError(DecreeError): ...
class AlreadyExistsError(DecreeError): ...
class InvalidArgumentError(DecreeError): ...
class LockedError(DecreeError): ...           # field is locked
class ChecksumMismatchError(DecreeError): ... # optimistic concurrency
class PermissionDeniedError(DecreeError): ...
class UnavailableError(DecreeError): ...
class IncompatibleServerError(DecreeError): ...  # version mismatch
class TypeMismatchError(DecreeError): ...        # wrong type getter
```

### Types (dataclass wrappers)

```python
@dataclass(frozen=True)
class ConfigValue:
    field_path: str
    value: str            # raw string value
    checksum: str
    description: str

@dataclass(frozen=True)
class Change:
    field_path: str
    old_value: str | None
    new_value: str | None
    version: int
    changed_by: str

@dataclass(frozen=True)
class ServerVersion:
    version: str          # e.g., "0.3.1"
    commit: str
```

### Version Compatibility

```python
# Exposed as module-level constants
opendecree.SUPPORTED_SERVER_VERSION  # ">=0.3.0,<1.0.0"
opendecree.PROTO_VERSION             # "v1" (centralconfig.v1)
opendecree.__version__               # "0.1.0" (SDK version from git tag)

# Runtime check
client.check_compatibility()
# → hits VersionService.GetServerVersion()
# → compares against SUPPORTED_SERVER_VERSION
# → raises IncompatibleServerError if out of range
# → logs warning if close to upper bound
```

## Documentation Strategy

The Python repo holds **usage-focused docs** (quickstart, API patterns, async guide). For detailed concepts (schemas, typed values, versioning, auth model), link to the main decree docs:

```
docs/quickstart.md          → "For full concepts, see https://decree.dev/concepts/"
docs/configuration.md       → "For auth model details, see https://decree.dev/concepts/auth/"
docs/watching.md            → "For subscription internals, see https://decree.dev/concepts/subscriptions/"
```

The README has a short "Getting Started" section and links to both local docs/ and the main site.

## Implementation Phases

### Phase 1: Scaffold + Stubs (day 1)

- [ ] Create repo `zeevdr/decree-python`
- [ ] `sdk/pyproject.toml` with setuptools, minimal deps (grpcio + protobuf)
- [ ] `Makefile` with targets: generate, lint, format, typecheck, test, build
- [ ] Proto generation: fetch from BSR, generate with grpcio-tools + mypy-protobuf
- [ ] Commit generated stubs to `sdk/src/opendecree/_generated/`
- [ ] `.gitattributes`, `.python-version`, `py.typed`
- [ ] CI workflow (lint + typecheck + test matrix)
- [ ] Empty `__init__.py` with version + public API stubs

### Phase 2: ConfigClient — sync (days 2-3)

- [ ] `_channel.py` — channel factory (insecure/TLS, keepalive options)
- [ ] `_interceptors.py` — auth metadata interceptor (sync)
- [ ] `errors.py` — exception hierarchy + gRPC error mapping
- [ ] `types.py` — ConfigValue, Change, ServerVersion dataclasses
- [ ] `_retry.py` — exponential backoff with jitter
- [ ] `_compat.py` — VersionService call + version comparison
- [ ] `client.py` — ConfigClient with all methods
- [ ] Tests for all client methods (mock stubs)
- [ ] Tests for error mapping, retry, interceptors

### Phase 3: AsyncConfigClient (day 4)

- [ ] `_interceptors.py` — async auth interceptor
- [ ] `async_client.py` — AsyncConfigClient (mirrors sync API)
- [ ] Tests for async client

### Phase 4: ConfigWatcher — sync (days 5-6)

- [ ] `watcher.py` — ConfigWatcher with background thread
  - Register typed fields (string, int, float, bool, duration)
  - Start: load snapshot + subscribe to stream
  - Thread-safe `.get()` on field values
  - Change callbacks via `@field.on_change`
  - Blocking iterator via `field.changes()`
  - Auto-reconnect with backoff on stream failure
- [ ] Tests for watcher lifecycle, reconnection, type conversion

### Phase 5: AsyncConfigWatcher (day 7)

- [ ] `async_watcher.py` — AsyncConfigWatcher
  - Same API but asyncio-native
  - `async for change in field.changes()`
  - `await watcher.start()` / `await watcher.stop()`
- [ ] Tests for async watcher

### Phase 6: Docs + Distribution (day 8)

- [ ] `README.md` — install, quickstart, link to docs
- [ ] `docs/quickstart.md` — first get/set in <5 min
- [ ] `docs/configuration.md` — all client options
- [ ] `docs/watching.md` — watcher patterns
- [ ] `docs/async.md` — async usage guide
- [ ] `CHANGELOG.md` — v0.1.0 entry
- [ ] PyPI trusted publisher setup
- [ ] `publish.yml` workflow
- [ ] Tag v0.1.0, verify PyPI publish

## Key Decisions

1. **Minimal runtime deps** — only `grpcio` + `protobuf`. Nothing else ships to users.
2. **Python 3.11+** — 3.10 loses security support soon
3. **grpcio (not betterproto)** — official, stable, C-core performance, sync+async in one package
4. **setuptools build** — comes with Python, PEP 621 pyproject.toml, no plugins. Version is a string, not dynamic.
5. **Maximize code generation from protos** — grpcio-tools + mypy-protobuf generate stubs; wrapper code is thin and derives types/methods from proto definitions
6. **ruff for linting+formatting** — single dev tool, replaces black+isort+flake8
7. **Dataclass return types** — don't expose proto messages in public API
8. **Both sync + async** — sync as primary, async mirrors the same API
9. **Watcher uses background thread (sync) / asyncio task (async)** — mirrors Go pattern
10. **PyPI trusted publishing** — OIDC, no API tokens
11. **Usage docs in Python repo, concepts link to main docs** — avoid duplication
12. **Client reports supported proto/server version** — `check_compatibility()` call + constants
13. **Stubs committed to repo** — same pattern as Go generated code
14. **Repo is `decree-python` not `decree-sdk-python`** — room for contrib packages later

## Verification

```bash
make generate             # regenerate proto stubs from BSR
make lint                 # ruff check + ruff format --check
make typecheck            # mypy sdk/src/
make test                 # pytest with coverage
make build                # python -m build -C sdk/ (sdist + wheel)
pip install -e sdk/       # dev install for manual testing
```
