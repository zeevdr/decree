---
name: issues
description: List open issues grouped by size (S/M/L) and milestone. Quick view of what to tackle next.
user-invocable: true
allowed-tools: Bash(gh *)
---

List all open issues grouped by size label, then by milestone.

## Steps

1. Fetch open issues: `gh issue list --state open --limit 100 --json number,title,milestone,labels`
2. Group by size label (`size: S`, `size: M`, `size: L`, unlabeled)
3. Within each size group, sort by milestone (milestoned first, then unassigned)

## Output format

```
## size: S (quick wins)
#132  Add make pre-commit target
#47   Social preview image                          [no milestone]

## size: M (moderate)
#110  Stress test Phase 1                           Stress Testing
#98   Security review                               Security Review

## size: L (larger efforts)
#32   contrib/viper: remote config provider          Ecosystem
#88   Admin GUI Phase 8: embed in Go                 Admin GUI

## Unlabeled
#999  Some new issue                                [no milestone]
```

Keep it compact — one line per issue, right-aligned milestone. No prose, just the list.
