---
name: issue
description: Create a GitHub issue with proper labels, size, and milestone. Ensures consistent issue tracking.
user-invocable: true
argument-hint: <description of the issue>
allowed-tools: Bash(gh *)
---

Create a well-structured GitHub issue from a description.

## Steps

1. Parse `$ARGUMENTS` to understand the issue
2. Determine labels from the available set:

   **Type** (pick one):
   - `bug` — something broken
   - `enhancement` — new feature or improvement
   - `docs` — documentation only
   - `discovery` — research/design exploration

   **Area** (pick all that apply):
   - `server`, `sdk`, `cli`, `proto`, `ci`, `infra`, `docs`
   - `python-sdk`, `typescript-sdk`, `admin-gui`

   **Size** (pick one):
   - `size: S` — quick win, a few hours or less
   - `size: M` — moderate, a day or two, clear scope
   - `size: L` — larger effort, multiple days, design decisions needed

3. Determine milestone — match to an open milestone if the issue fits one, otherwise leave unassigned
4. Draft the issue title (concise, under 70 chars) and body
5. Show the plan and ask for confirmation before creating:

   ```
   Title: <title>
   Labels: enhancement, server, size: M
   Milestone: Hardening
   ---
   <body preview>
   ```

6. Create: `gh issue create --title "..." --body "..." --label "..." --milestone "..."`
7. Report the issue URL

## Body format

```markdown
## Description
<1-3 sentences explaining the what and why>

## Acceptance criteria
- [ ] <concrete, testable items>
```

## Rules
- ALWAYS ask for confirmation before creating
- Do NOT assign (solo maintainer)
- Keep title under 70 characters
- Every issue gets a type label, area label, and size label
