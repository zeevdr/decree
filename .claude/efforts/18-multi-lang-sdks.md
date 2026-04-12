# Multi-Language SDKs

**Status:** Complete (Python + TypeScript shipped)
**Started:** 2026-04-09
**Completed:** 2026-04-12

---

## Strategy

Separate repos per language, proto published to BSR. Each SDK is independently versioned and released.

| SDK | Repo | Package | Status | Effort |
|-----|------|---------|--------|--------|
| Python | `zeevdr/decree-python` | `opendecree` (PyPI) | v0.1.0 shipped | `18-python-sdk.md` |
| TypeScript | `zeevdr/decree-typescript` | `@opendecree/sdk` (npm) | v0.1.0 shipped | `23-typescript-sdk.md` |

Proto source of truth stays in `zeevdr/decree`, published to BSR: `buf.build/opendecree/decree`.

## Key Decisions

- Separate repos (independent release cycles, CI, package managers)
- Proto via BSR (decouples from main repo file structure)
- `@grpc/grpc-js` for TypeScript (official gRPC, not Connect)
- `grpcio` for Python (official gRPC)
- Both SDKs: ConfigClient + ConfigWatcher, no AdminClient in v0.1.0
- Both SDKs: OIDC trusted publishing (PyPI + npm)
- Python: sync + async APIs. TypeScript: async-only (Node.js is async-first)
