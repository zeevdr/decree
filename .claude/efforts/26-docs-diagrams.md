# Documentation Diagrams

**Status:** Phase 1 complete, Phase 2 skipped, Phase 3 open
**Started:** 2026-04-13
**Updated:** 2026-04-13

---

## Goal

Add Mermaid diagrams to documentation where visual representation improves understanding. Replace ASCII art with proper diagrams, and add new diagrams for concepts that currently lack visual aids.

## Approach

- Use Mermaid fenced code blocks (renders natively on GitHub)
- Keep simple one-liner ASCII flows where they work (e.g. `A → B → C`)
- No external tooling or build steps required

## Diagram Opportunities

### Phase 1: Replace existing ASCII art — DONE (#114)

All 5 ASCII diagrams replaced with Mermaid:
- schemas-and-fields.md — `stateDiagram` for lifecycle
- overview.md — `flowchart` for mental model + architecture (with separated Storage, Cache, Pub/Sub backends)
- subscriptions.md — `sequenceDiagram` for change propagation
- versioning.md — `flowchart` for delta resolution
- tenants.md — `graph` with distinct node shapes per entity type

Also refreshed: added multi-tenant access control to tenants.md, auth mention in architecture overview.

### Phase 2: Skipped — not needed (#103 closed)

Reviewed auth.md, deployment.md, and observability.md. These are reference-style docs where tables and code blocks communicate better than diagrams. Forcing Mermaid here would just restate the tables in a worse format.

### Phase 3: Nice to have

| Doc | Concept | Diagram Type |
|-----|---------|-------------|
| concepts/typed-values.md | Value transformation across layers | `flowchart` |
| README.md | SDK module relationships | `graph` |
| sdk.md | SDK package overview | `graph` |
| usecases/config-as-code.md | Git-based vs runtime config flow | `flowchart` |
