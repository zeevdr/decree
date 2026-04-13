# Examples & Demos

**Status:** Planning
**Started:** 2026-04-09
**Updated:** 2026-04-13

---

## Goal

Provide runnable examples at two levels: per-repo SDK examples for specific use cases, and a `decree-demos` repo with full end-to-end solution examples.

## Two-Level Approach

### 1. Per-Repo SDK Examples

Each SDK repo gets its own `examples/` directory with language-specific use cases:

- **decree** (Go SDK) — configclient, configwatcher, adminclient examples
- **decree-python** — sync/async client, watcher, FastAPI integration
- **decree-typescript** — client, watcher, Next.js integration

These live alongside the SDK code so they stay in sync with API changes and serve as living documentation.

### 2. decree-demos (Full Solution Examples)

A separate `zeevdr/decree-demos` repo with end-to-end scenarios that demonstrate the full platform:

- Single schema, single tenant (simplest case)
- Multi-tenant with shared schema
- Schema evolution and tenant migration
- Config-as-code with CI/CD
- Real-time config with watcher
- Multi-language (same scenario in Go, Python, TypeScript)

Each demo includes a docker-compose.yml, seed files, and step-by-step instructions.

## Work Items

### Per-repo examples
- [ ] Go SDK examples (decree repo)
- [ ] Python SDK examples (decree-python repo)
- [ ] TypeScript SDK examples (decree-typescript repo)

### decree-demos repo
- [ ] Repo scaffold with shared docker-compose.yml
- [ ] Single tenant quickstart demo
- [ ] Multi-tenant demo
- [ ] Schema evolution demo
- [ ] Config-as-code demo
- [ ] curl/REST walkthrough

## Dependencies

- All SDKs shipped (done)
- REST gateway (done)
