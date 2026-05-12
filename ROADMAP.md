# Roadmap

<!-- adv-triage generated: 2026-05-12T00:30:00Z | DO NOT EDIT MANUALLY -->
<!-- Source of truth: GitHub Project #4 owned by @Sharper-Flow -->

Regenerate with `/adv-triage --execute`. Manual edits are overwritten.

## v1.0 critical path (in order)

Sequence locked 2026-05-11 after TMUX-scope research (workload target = 15 simultaneous active; real RAM bottleneck = Sharper-Flow/Advance#117 worker singleton leak).

| Order | # | Title | Effort | Notes |
|---|---|-------|--------|-------|
| 1 | #11 | oca maintain execute self-blocks when run from OpenCode | ~half day | Trivial fix (filter self-PID) |
| 2 | #21 | S5 Legacy in-repo ADV state doctor warning | ~half day | Doctor check, no auto-delete |
| 3 | #22 | S6 Resume hint reprint in outer terminal | ~half day | UX gap; tmux pane goes blank now |
| 4 | #18 + #31 | Doctor canary surface (incl. worker-leak surface for Advance#117) | ~3 days | Same surface; bundle as one change |
| 5 | #24 | O2 Replace adv_status.sh with `oca adv-status` Go subcommand | ~3 days | Foundation for tmux status bar; depends on Advance#104 for cleanest read surface |
| 6 | #23 | O1 TMUX boundary refactor (~2,073 LOC migration to OCA) | 1–2 weeks | Largest piece; cross-repo coordination (Advance-side delete phase filed separately) |
| 7 | #29 | tmux-resurrect + concurrency warning polish | ~1 day | Composable; can run in parallel with #23 |
| 8 | #25 | O4 Read session debt from Advance (blocked: Advance#104) | ~1 day | Trivial OCA work; gated on Advance shipping stable read surface |
| 9 | #9 | S7 OCA handoff for Advance self-update rebuilds | 1–2 days | Advance#40 closed → unblocked |
| 10 | #30 | Phase 8 v1.0 release candidate + tag publication | ~half day | The release ritual |

**Total estimate**: ~3 weeks of focused OCA work.

**Cutover model**: Operator stays on openchad daily-driver throughout development. Cutover happens at v1.0 tag.

## Deferred to v1.1 (conditional or follow-up)

| # | Title | Defer reason |
|---|-------|--------------|
| #28 | Session #1 — Graceful opencode session hibernation | Conditional on RAM still hurting after worker-leak fix (Advance#117) + Pattern B cutover. UX details will be designed against real usage data, not paper specs. |
| #19 | S3 Permission-first config for OCA-owned environment agents | OpenCode permission API still evolving; risk of premature lock-in |
| #20 | S4 MCP tool suite profiles and per-agent exposure | Context-budget optimization, not blocking |

## Upstream-blocked (Advance-side dependencies)

| OCA # | Blocked on | Status |
|---|---|---|
| #25 | Sharper-Flow/Advance#104 (Expose stable ADV read surface for OCA O2) | Open upstream; OCA work waits |
| (n/a) | Sharper-Flow/Advance#117 (Worker singleton broken — ~2.2 GB RAM waste) | **Filed 2026-05-11**; OCA #18 expanded to surface in doctor |

## Bugs (by priority)

### Critical
(none)

### High
(none)

### Medium
| # | Title | Labels |
|---|-------|--------|
| #11 | oca maintain execute self-blocks when run from OpenCode | bug,priority:medium |
| #21 | Legacy in-repo ADV state doctor warning (S5) | bug,priority:medium |

### Low
(none)

## Features (by WSJF, descending)

| # | Title | V | TC | RROE | E | WSJF | Labels |
|---|-------|---|----|------|---|------|--------|
| #30 | Phase 8: v1.0 release candidate validation + tag publication | 10 | 10 | 10 | 3 | 10.0 | feature,priority:critical |
| #18 | OCA doctor: surface Advance canaries + worker-leak + environment checks (S1, expanded 2026-05-11) | 8 | 6 | 7 | 3 | 7.0 | priority:low,tech-debt |
| #31 | Surface Advance worker restart exhaustion in OCA doctor + status bar | 6 | 4 | 5 | 3 | 5.0 | feature,priority:medium |
| #24 | Replace adv_status.sh with oca adv-status Go subcommand (O2) | 8 | 7 | 8 | 5 | 4.6 | feature,priority:high |
| #9 | Add OCA handoff for Advance self-update rebuilds (S7) | 8 | 3 | 8 | 5 | 3.8 | feature |
| #23 | TMUX boundary refactor — move ~2,073 LOC from Advance to OCA (O1) | 9 | 7 | 8 | 8 | 3.0 | feature,priority:high |
| #22 | Resume hint reprint in outer terminal (S6) | 4 | 3 | 4 | 3 | 1.8 | feature,priority:low |
| #29 | tmux-resurrect integration + concurrency warning polish (Session #2) | 5 | 3 | 4 | 3 | 1.7 | feature,priority:medium |

## Tech Debt (by WSJF, descending)

| # | Title | V | TC | RROE | E | WSJF | Labels |
|---|-------|---|----|------|---|------|--------|
| #25 | Read session debt from Advance instead of duplicating scan (O4) | 5 | 3 | 5 | 2 | 4.0 | priority:medium,tech-debt |

## Deferred / Unscored

| # | Title | Status |
|---|-------|--------|
| #19 | Permission-first config for OCA-owned environment agents (S3) | Deferred to v1.1 |
| #20 | MCP tool suite profiles and per-agent exposure (S4) | Deferred to v1.1 |
| #27 | Pattern B session topology + smart oca entry (Session #0) | **Stale issue; shipped 2026-05-04 as `patternBSessionTopologyOne`. Close.** |
| #28 | Graceful opencode session hibernation (Session #1) | Deferred to v1.1; conditional reopen |

## Triage Run Summary

- Run timestamp: 2026-05-12T00:30:00Z
- Sources scanned: gh (10 open after closes), agenda (0), wisdom (0), notes (0), changes (0), todos (0)
- Issues closed this session: 9 (#13, #14, #15, #16, #17 shipped; #33-#37 refiled to Advance #112-#116)
- Issues deferred this session: 1 (#28 to v1.1 with conditional reopen criteria)
- Issues to close as stale: 1 (#27 Pattern B already shipped)
- Issues opened upstream: 2 (Sharper-Flow/Advance#111 sync-global drift; Sharper-Flow/Advance#117 worker singleton broken)
- v1.0 critical path locked: 10 items, ~3 weeks
- Cutover model: stay on openchad until tag; cutover at v1.0 release
- Boundary doc: docs/proposals/2026-05-08-cross-repo-boundary-audit.md
- Recently archived changes:
  - `finalizeMustQueueTriage3` (2026-05-11)
  - `close4PreExistingDataCoverage` (2026-05-11)
