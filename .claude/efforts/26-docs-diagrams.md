# Documentation Diagrams

**Status:** Planning
**Started:** 2026-04-13

---

## Goal

Add Mermaid diagrams to documentation where visual representation improves understanding. Replace ASCII art with proper diagrams, and add new diagrams for concepts that currently lack visual aids.

## Approach

- Use Mermaid fenced code blocks (renders natively on GitHub)
- Keep simple one-liner ASCII flows where they work (e.g. `A → B → C`)
- No external tooling or build steps required

## Diagram Opportunities

### Phase 1: Replace existing ASCII art (high value)

| Doc | Current | Diagram Type |
|-----|---------|-------------|
| concepts/schemas-and-fields.md | ASCII lifecycle `Create → Update → Publish → Assign` | `stateDiagram` — draft/published states, immutability |
| concepts/overview.md | ASCII flow `Schema → Tenant → Config → Subscribe` | `flowchart` — core workflow + architecture |
| concepts/subscriptions.md | ASCII write → storage → pub/sub → stream | `sequenceDiagram` — change propagation |
| concepts/versioning.md | Text-based version delta examples | `flowchart` — delta resolution stacking |
| concepts/tenants.md | ASCII tree of schema → versions → tenants | `erDiagram` or `graph` — entity relationships |

### Phase 2: Add new diagrams (missing visuals)

| Doc | Concept | Diagram Type |
|-----|---------|-------------|
| concepts/auth.md | Auth flow: request → extract claims → role check → access | `flowchart` — decision tree |
| server/deployment.md | Docker Compose topology: app → PG + Redis | `graph` — deployment architecture |
| server/observability.md | Nested trace spans: gRPC → Redis → DB | `sequenceDiagram` — span hierarchy |

### Phase 3: Nice to have

| Doc | Concept | Diagram Type |
|-----|---------|-------------|
| concepts/typed-values.md | Value transformation across layers | `flowchart` |
| README.md | SDK module relationships | `graph` |
| sdk.md | SDK package overview | `graph` |
| usecases/config-as-code.md | Git-based vs runtime config flow | `flowchart` |
