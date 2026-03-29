# Go Public Checklist

**Status:** In Progress
**Started:** 2026-03-29

---

## Goal

Prepare the repo for public release on GitHub. Ensure it's clean, professional, and ready for community consumption.

## Must-Do (before flipping to public)

- [x] **Scan for secrets** — gitleaks: no leaks found (46 commits scanned)
- [x] **LICENSE** — Apache 2.0 verified
- [x] **README review** — badges, power tools section, architecture diagram fix
- [x] **Module paths** — all 8 modules confirmed under `github.com/zeevdr/decree`
- [x] **Remove GOPRIVATE** — not set anywhere, nothing to remove
- [ ] **Clean git history** — squash noisy CI/Docker fix commits
- [ ] **GitHub repo settings** — description, topics, website URL

## Should-Do (before or shortly after)

- [ ] **Tag v0.1.0** — first release
- [ ] **GitHub Actions** — verify GITHUB_TOKEN has packages:write for ghcr.io
- [ ] **Branch protection** — require PR reviews, require CI to pass
- [x] **Issue templates** — bug report + feature request
- [x] **SECURITY.md** — vulnerability reporting instructions
- [x] **Code of Conduct** — Contributor Covenant v2.1 (by reference)

## Nice-to-Have

- [x] **README badges** — CI status, Go version, license
- [ ] **Social preview** — repo card image for link sharing
- [ ] **GitHub Discussions** — enable for Q&A and community
- [ ] **Example repo** — separate repo with runnable examples
- [ ] **Blog post / announcement** — introduce OpenDecree

## Implementation Order

1. ~~Scan for secrets~~ done
2. ~~License + module path confirmation~~ done
3. Clean git history (in progress)
4. ~~README final review~~ done
5. GitHub repo settings
6. Flip to public
7. Tag v0.1.0
8. Verify pkg.go.dev, ghcr.io, CI