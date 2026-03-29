# CLI Tool

**Status:** Phase 1+2 Complete (power tools remaining)

---

## Completed

26 commands across 6 groups (schema, tenant, config, watch, lock, audit) + version + gen-docs. Own Go module. 17 unit tests. `--publish` flag on schema import. `--mode` flag on config import (merge/replace/defaults).

## Remaining: Phase 3 — Power Tools

- [ ] `ccs docs generate` — schema → markdown docs
- [ ] `ccs diff` — config version diffing
- [ ] `ccs validate` — offline YAML validation
- [ ] `ccs seed` — bootstrap: schema + tenant + config from YAML in one command
- [ ] `ccs dump` — full tenant backup (schema + config + locks)

## Phase 4 — Polish (wishlist)

- [ ] Shell completion (bash, zsh, fish — cobra built-in)
- [ ] Man page generation
- [ ] Homebrew formula / goreleaser
