# Go Public Checklist

**Status:** In Progress (pre-launch items remaining)
**Started:** 2026-03-29

---

## Completed

Secret scan, LICENSE, README review, module paths, git history, GitHub settings, repo flipped public, v0.1.0 tagged, branch protection, issue templates, SECURITY.md, Code of Conduct, Discussions, CI fixed. See `completed.md`.

## Pre-Launch (before announcing)

- [ ] **REST/HTTP Gateway** (effort 16) — REST/JSON API for all gRPC services
- [ ] **Admin GUI** (effort 17) — web UI for config/schema management (alpha)
- [ ] **TypeScript SDK** (effort 18) — npm package with thin wrapper
- [ ] **Python SDK** (effort 18) — PyPI package with thin wrapper
- [ ] **Examples repo** (effort 19) — runnable examples per language
- [ ] **BSR proto publishing** — buf push on release tags
- [ ] **Final README update** — add REST/GUI/SDK sections, update install instructions
- [ ] **Verify ghcr.io image push** — confirm release workflow pushes images

## Order

1. REST Gateway (effort 16)
2. BSR proto publishing
3. Admin GUI (effort 17) + TS SDK + Python SDK (parallel)
4. Examples repo (effort 19)
5. Final README + announce
