# Wishlist — Future Work

**Status:** Backlog

---

Items roughly prioritized. Not committed to — this is a living list of ideas.

## Infrastructure

- [x] ~~**Helm chart**~~ — done: deploy/helm/decree with full env var support, secrets, ingress, OTel
- [x] ~~**Wire in-memory storage**~~ — done: `STORAGE_BACKEND=memory` in cmd/server
- [ ] **Usage stats recording** — async batched read tracking (deferred from AuditService)
- [ ] **Docker layer caching** — cache-from/to in CI Docker build steps
- [ ] **Internal coverage** — cache/pubsub need Redis mocks to reach 65%+

## Contrib Integrations

- [ ] **contrib/viper** — viper remote provider backed by configclient
- [ ] **contrib/koanf** — koanf provider for the Koanf config library
- [ ] **contrib/envconfig** — adapter that populates struct fields from config values

## SDK Enhancements

- [ ] **Retry/backoff on configclient** — automatic retry with backoff for transient gRPC errors
- [ ] **configwatcher write-through** — allow writes via watcher that optimistically update local values
- [ ] **configwatcher field groups** — register a struct and auto-map fields by tag
- [ ] **SDK code generation** — generate typed config structs from schema definitions

## Service Features

- [ ] **Webhook notifications** — HTTP callbacks on config changes (alternative to gRPC streaming)
- [ ] **Config diffing API** — server-side diff between two config versions
- [ ] **Schema migration assistant** — tooling to help migrate config values when schema changes
- [ ] **Multi-environment promotion** — promote config from dev → staging → prod with approval workflow
- [ ] **Config templates** — default config values defined at schema level, applied on tenant creation

## CLI Phase 4 — Polish

- [x] ~~**Shell completion**~~ — done: bash, zsh, fish, powershell (cobra built-in + flag completions)
- [ ] **Man page generation**
- [x] ~~**Goreleaser**~~ — done: builds server + CLI for linux/mac/windows (amd64/arm64)
- [ ] **Homebrew formula** — tap repo with goreleaser-generated formula

## Observability

- [ ] **Grafana dashboard template** — pre-built dashboard for OTel metrics
- [ ] **Alerting rules** — Prometheus rules for DB pool exhaustion, high error rates

## Documentation

- [ ] **Architecture decision records (ADRs)** — formalize key decisions into ADR format
- [ ] **Social preview** — repo card image for link sharing
- [ ] **Blog post / announcement** — introduce OpenDecree
