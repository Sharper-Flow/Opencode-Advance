# Roadmap

<!-- adv-triage generated: 2026-05-09T02:00:00Z | DO NOT EDIT MANUALLY -->
<!-- Source of truth: GitHub Project #4 owned by @Sharper-Flow -->

Regenerate with `/adv-triage --execute`. Manual edits are overwritten.

## Bugs (by priority)

### Critical
(none)

### High
| # | Title | Labels |
|---|-------|--------|
| #15 | OCA instruction assets are real files, not templates (M4) | bug |

### Medium
| # | Title | Labels |
|---|-------|--------|
| #11 | oca maintain execute self-blocks when run from OpenCode | bug |
| #21 | Legacy in-repo ADV state doctor warning (S5) | bug |

### Low
(none)

## Features (by WSJF, descending)

| # | Title | V | TC | RROE | E | WSJF | Labels |
|---|-------|---|----|------|---|------|--------|
| #30 | Phase 8: v1.0 release candidate validation + tag publication | 10 | 10 | 10 | 3 | 10.0 | feature |
| #13 | OCA umbrella plugin install (M3) | 9 | 9 | 9 | 3 | 9.0 | feature |
| #14 | Bare oca apply lifecycle parity (M2) | 9 | 8 | 8 | 5 | 5.0 | feature |
| #16 | Starter and migration stack validity (M5) | 7 | 6 | 7 | 3 | 4.7 | feature |
| #24 | Replace adv_status.sh with oca adv-status Go subcommand (O2) | 8 | 7 | 8 | 5 | 4.6 | feature |
| #31 | Surface Advance worker restart exhaustion (moved from ADV #70) | 6 | 4 | 5 | 3 | 5.0 | feature |
| #9 | Add OCA handoff for Advance self-update rebuilds (S7) | 8 | 3 | 8 | 5 | 3.8 | feature |
| #23 | TMUX boundary refactor — move ~2,073 LOC from Advance to OCA (O1) | 9 | 7 | 8 | 8 | 3.0 | feature |
| #18 | Context budget audit + instruction loading diet (S2) | 7 | 5 | 6 | 5 | 2.3 | feature |
| #22 | Resume hint reprint in outer terminal (S6) | 4 | 3 | 4 | 3 | 1.8 | feature |
| #27 | Pattern B session topology + smart oca entry (Session #0) | 8 | 6 | 7 | 8 | 1.8 | feature |
| #19 | Permission-first agent configuration (S3) | 5 | 3 | 5 | 5 | 1.7 | feature |
| #20 | MCP tool suite profiles and per-agent exposure (S4) | 5 | 3 | 5 | 5 | 1.7 | feature |
| #29 | tmux-resurrect integration + concurrency warning polish (Session #2) | 5 | 3 | 4 | 3 | 1.7 | feature |
| #28 | Graceful opencode session hibernation (Session #1) | 7 | 5 | 7 | 8 | 1.3 | feature |

## Tech Debt (by WSJF, descending)

| # | Title | V | TC | RROE | E | WSJF | Labels |
|---|-------|---|----|------|---|------|--------|
| #25 | Read session debt from Advance instead of duplicating scan (O4) | 5 | 3 | 5 | 2 | 4.0 | tech-debt |
| #17 | OCA skill guidance refresh (M6) | 6 | 4 | 6 | 3 | 3.2 | tech-debt |
| #26 | Document opencode.json section ownership contract (O3) | 5 | 3 | 4 | 2 | 3.0 | tech-debt |
| #18 | OCA runtime doctor canaries (S1) | 7 | 5 | 6 | 5 | 2.3 | tech-debt |

## Deferred / Unscored

(none)

## Triage Run Summary

- Run timestamp: 2026-05-09T02:00:00Z
- Sources scanned: gh (22), agenda (0), wisdom (0), notes (0), changes (0), todos (0)
- Issues opened this run: 1 (#31, moved from ADV #70)
- Field assignments this run: 3 (#25 reframed, #26 reframed, #31 added)
- Items deferred: 0
- Boundary doc: docs/proposals/2026-05-08-cross-repo-boundary-audit.md (revised under "Advance must work standalone; OCA enhances")
