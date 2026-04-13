# Python SDK (`opendecree`)

**Status:** Code complete — awaiting PyPI publish + branch protection
**Started:** 2026-04-10
**Last Updated:** 2026-04-12
**Repo:** `zeevdr/decree-python`
**PyPI:** `opendecree`
**Python:** 3.11+

---

## Goal

A production-quality Python SDK for OpenDecree that covers config reads, writes, and live change subscriptions. Both sync and async APIs. Vanilla tooling only — standard, widely-adopted libraries.

## Scope (v0.1.0)

| Feature | Description |
|---------|-------------|
| ConfigClient (sync) | `get(t, f)` → str, `get(t, f, type)` → typed via @overload. set, set_many, set_null, get_all. Context manager. `watch()` factory. |
| AsyncConfigClient | Same overloaded API with async/await. Async context manager. |
| ConfigWatcher (sync) | `client.watch(t)` context manager → `watcher.field(path, type, default)` → `WatchedField[T]`. `.value` property, `__bool__`. |
| AsyncConfigWatcher | Same but asyncio-native. Async context manager, async iterator for changes. |
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

### Pythonic Patterns

- **Context managers** for all resources with lifecycles (clients, watchers)
- **`@overload` + generics** for typed `get()` — one method, mypy-verified return types
- **`.value` property** on watched fields instead of `.get()` method
- **`__bool__`** on watched fields for natural conditionals (`if enabled:`)
- **`@dataclass(frozen=True, slots=True)`** for all return types — immutable, fast (3.11+)
- **`client.watch()`** factory method — watcher inherits connection/auth from client

### ConfigClient (sync)

```python
from opendecree import ConfigClient
from datetime import timedelta

# Context manager for clean channel lifecycle
with ConfigClient("localhost:9090", subject="myapp") as client:
    # get() with no type → str (default)
    val = client.get("tenant-id", "payments.fee")                    # → str

    # get() with type argument → typed return (via @overload)
    retries = client.get("tenant-id", "payments.retries", int)       # → int
    enabled = client.get("tenant-id", "payments.enabled", bool)      # → bool
    rate    = client.get("tenant-id", "payments.fee_rate", float)    # → float
    timeout = client.get("tenant-id", "payments.timeout", timedelta) # → timedelta

    # Set (always string — server handles type conversion)
    client.set("tenant-id", "payments.fee", "0.5%")

    # Bulk operations
    all_config = client.get_all("tenant-id")  # → dict[str, str]
    client.set_many("tenant-id", {"a": "1", "b": "2"}, description="batch update")

    # Null
    client.set_null("tenant-id", "payments.fee")

    # Nullable get — returns None instead of raising on null values
    val = client.get("tenant-id", "payments.fee", str, nullable=True)  # → str | None

    # Server compatibility check
    client.check_compatibility()  # raises IncompatibleServerError if mismatch
```

**Type signature (overloaded):**

```python
@overload
def get(self, tenant_id: str, field_path: str) -> str: ...
@overload
def get(self, tenant_id: str, field_path: str, type: type[int]) -> int: ...
@overload
def get(self, tenant_id: str, field_path: str, type: type[float]) -> float: ...
@overload
def get(self, tenant_id: str, field_path: str, type: type[bool]) -> bool: ...
@overload
def get(self, tenant_id: str, field_path: str, type: type[timedelta]) -> timedelta: ...
@overload
def get(self, tenant_id: str, field_path: str, type: type[str], nullable: bool = ...) -> str | None: ...
```

Supported types: `str`, `int`, `float`, `bool`, `timedelta`. Conversion from the proto TypedValue happens in the SDK.

### AsyncConfigClient

```python
from opendecree import AsyncConfigClient

async with AsyncConfigClient("localhost:9090", subject="myapp") as client:
    val     = await client.get("tenant-id", "payments.fee")            # → str
    retries = await client.get("tenant-id", "payments.retries", int)   # → int
    await client.set("tenant-id", "payments.fee", "0.5%")
```

Same `get()` overload pattern. Async context manager (`__aenter__`/`__aexit__`).

### ConfigWatcher (sync)

Watcher created via `client.watch()` — inherits connection and auth. Context manager auto-starts/stops.

```python
from opendecree import ConfigClient

with ConfigClient("localhost:9090", subject="myapp") as client:
    with client.watch("tenant-id") as watcher:
        # Register fields with type + default (generic)
        fee     = watcher.field("payments.fee", float, default=0.01)
        enabled = watcher.field("payments.enabled", bool, default=False)
        name    = watcher.field("payments.name", str, default="")

        # .value property — always-fresh, thread-safe, typed
        print(fee.value)          # 0.025 (float)
        print(enabled.value)      # True (bool)

        # __bool__ — fields are truthy/falsy based on their value
        if enabled:
            print(f"Processing at rate {fee.value}")

        # Change callbacks
        @fee.on_change
        def fee_changed(old: float, new: float):
            print(f"Fee changed: {old} → {new}")

        # Or iterate changes (blocking)
        for change in fee.changes():
            print(change)
    # watcher auto-stops on exit
```

### AsyncConfigWatcher

```python
from opendecree import AsyncConfigClient

async with AsyncConfigClient("localhost:9090", subject="myapp") as client:
    async with client.watch("tenant-id") as watcher:
        fee = watcher.field("payments.fee", float, default=0.01)

        print(fee.value)  # always-fresh

        if fee:  # __bool__ works
            print("fee is set")

        async for change in fee.changes():  # async iterator
            print(f"{change.old_value} → {change.new_value}")
    # auto-stops on exit
```

### WatchedField[T]

```python
class WatchedField(Generic[T]):
    @property
    def value(self) -> T: ...           # always-fresh, thread-safe
    def __bool__(self) -> bool: ...     # truthy based on value (False/0/""/"0" → False)
    def on_change(self, fn: Callable[[T, T], None]) -> None: ...  # decorator
    def changes(self) -> Iterator[Change[T]]: ...                  # sync: blocking iterator
    # async variant:
    def changes(self) -> AsyncIterator[Change[T]]: ...             # async iterator
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
@dataclass(frozen=True, slots=True)
class ConfigValue:
    field_path: str
    value: str            # raw string value
    checksum: str
    description: str

@dataclass(frozen=True, slots=True)
class Change:
    field_path: str
    old_value: str | None
    new_value: str | None
    version: int
    changed_by: str

@dataclass(frozen=True, slots=True)
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

### Phase 1: Scaffold + Stubs — COMPLETE (2026-04-10)

- [x] Create repo `zeevdr/decree-python`
- [x] `sdk/pyproject.toml` with setuptools, minimal deps (grpcio + protobuf)
- [x] `Makefile` with targets: generate, lint, format, typecheck, test, build (Docker-based)
- [x] Proto generation: fetch from BSR, generate with grpcio-tools + mypy-protobuf
- [x] Commit generated stubs to `sdk/src/opendecree/_generated/`
- [x] `.gitattributes`, `.python-version`, `py.typed`
- [x] Empty `__init__.py` with version + public API stubs
- [x] CI workflow: `.github/workflows/ci.yml` (3.11/3.12/3.13, lint+typecheck+test)
- [x] Publish workflow: `.github/workflows/publish.yml` (OIDC trusted publishing)

### Phase 2: ConfigClient — sync — COMPLETE (2026-04-10)

- [x] `_channel.py` — channel factory (insecure/TLS, keepalive options)
- [x] `_interceptors.py` — auth metadata interceptor (sync + async)
- [x] `errors.py` — exception hierarchy + gRPC error mapping
- [x] `types.py` — ConfigValue, Change, ServerVersion dataclasses
- [x] `_retry.py` — exponential backoff with jitter (sync + async)
- [x] `_convert.py` — TypedValue conversion (str, int, float, bool, timedelta)
- [x] `_stubs.py` — lazy stub loading, shared response processing
- [x] `client.py` — ConfigClient with overloaded `get()`, context manager, `watch()` factory
- [x] Tests: client, errors, convert, retry, types (73 tests, 85% coverage)

Note: `_compat.py` (VersionService check) deferred — not blocking v0.1.0.

### Phase 3: AsyncConfigClient — COMPLETE (2026-04-10)

- [x] `async_client.py` — AsyncConfigClient (mirrors sync API, per-call metadata)
- [x] Tests for async client (69 tests, 84% coverage)

### Phase 4: ConfigWatcher — sync — COMPLETE (2026-04-10)

- [x] `watcher.py` — ConfigWatcher context manager via `client.watch()`
  - WatchedField[T] with `.value`, `__bool__`, `on_change`, `changes()`
  - Background thread with auto-reconnect and exponential backoff
  - Initial snapshot loading before subscription
- [x] Tests for watcher lifecycle, reconnection, type conversion (91 tests, 86% coverage)

### Phase 5: AsyncConfigWatcher — COMPLETE (2026-04-12)

- [x] `async_watcher.py` — AsyncConfigWatcher async context manager
  - AsyncWatchedField[T] with `.value`, `__bool__`, `on_change`, async `changes()`
  - Background asyncio task with auto-reconnect
- [x] `test_async_watcher.py` — 16 tests
- [x] Fix: reconnect backoff on normal stream end, CI deps + mypy fixes

### Phase 6: Docs + Distribution — COMPLETE (2026-04-12, manual steps remain)

- [x] `README.md` — badges (CI, PyPI, Python, license), install, quickstart
- [x] Publish workflow configured (`.github/workflows/publish.yml`, OIDC)
- [x] `docs/quickstart.md` — install, typed reads, watch, error handling
- [x] `docs/configuration.md` — all client options (auth, TLS, retry, errors)
- [x] `docs/watching.md` — WatchedField[T], callbacks, iterators, reconnect
- [x] `docs/async.md` — async client + watcher, sync vs async comparison
- [x] `CHANGELOG.md` — v0.1.0 entry
- [x] Governance: SECURITY.md, CODE_OF_CONDUCT.md, CONTRIBUTING.md
- [x] GitHub templates: bug report, feature request, PR template
- [x] PyPI keywords in pyproject.toml
- [ ] **PyPI trusted publisher setup** (add pending publisher on pypi.org + GitHub Environment)
- [ ] **Branch protection on main** (require PR + CI checks)
- [ ] **Tag v0.1.0** → triggers PyPI publish

## Key Decisions

1. **Minimal runtime deps** — only `grpcio` + `protobuf`. Nothing else ships to users.
2. **Python 3.11+** — 3.10 loses security support soon
3. **grpcio (not betterproto)** — official, stable, C-core performance, sync+async in one package
4. **setuptools build** — comes with Python, PEP 621 pyproject.toml, no plugins. Version is a string, not dynamic.
5. **Maximize code generation from protos** — grpcio-tools + mypy-protobuf generate stubs; wrapper code is thin
6. **ruff for linting+formatting** — single dev tool, replaces black+isort+flake8
7. **`@dataclass(frozen=True, slots=True)` return types** — immutable, fast, don't expose proto messages
8. **Both sync + async** — sync as primary, async mirrors the same API
9. **Overloaded `get()` with generics** — `get(t, f)` → str, `get(t, f, int)` → int. One method, mypy-verified.
10. **`client.watch()` factory** — watcher inherits connection/auth, context manager for lifecycle
11. **`WatchedField[T]`** — `.value` property, `__bool__`, `on_change` callback, `changes()` iterator
12. **Context managers everywhere** — clients and watchers use `with`/`async with` for clean lifecycle
13. **PyPI trusted publishing** — OIDC, no API tokens
14. **Usage docs in Python repo, concepts link to main docs** — avoid duplication
15. **Client reports supported proto/server version** — `check_compatibility()` call + constants
16. **Stubs committed to repo** — same pattern as Go generated code
17. **Repo is `decree-python` not `decree-sdk-python`** — room for contrib packages later

## Verification

```bash
make generate             # regenerate proto stubs from BSR
make lint                 # ruff check + ruff format --check
make typecheck            # mypy sdk/src/
make test                 # pytest with coverage
make build                # python -m build -C sdk/ (sdist + wheel)
pip install -e sdk/       # dev install for manual testing
```
