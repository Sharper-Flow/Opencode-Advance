# Roadmap

<!-- adv-triage generated: 2026-05-11T23:50:00Z | DO NOT EDIT MANUALLY -->
<!-- Source of truth: GitHub Project #4 owned by @Sharper-Flow -->

Regenerate with `/adv-triage --execute`. Manual edits are overwritten.

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
| #18 | OCA doctor: surface Advance canaries + environment-specific checks (S1) | 7 | 5 | 6 | 3 | 6.0 | priority:low,tech-debt |
| #31 | Surface Advance worker restart exhaustion in OCA doctor + status bar | 6 | 4 | 5 | 3 | 5.0 | feature,priority:medium |
| #24 | Replace adv_status.sh with oca adv-status Go subcommand (O2) | 8 | 7 | 8 | 5 | 4.6 | feature,priority:high |
| #9 | Add OCA handoff for Advance self-update rebuilds (S7) | 8 | 3 | 8 | 5 | 3.8 | feature |
| #23 | TMUX boundary refactor — move ~2,073 LOC from Advance to OCA (O1) | 9 | 7 | 8 | 8 | 3.0 | feature,priority:high |
| #19 | Permission-first config for OCA-owned environment agents (S3) | 5 | 3 | 5 | 3 | 2.7 | feature,priority:medium |
| #22 | Resume hint reprint in outer terminal (S6) | 4 | 3 | 4 | 3 | 1.8 | feature,priority:low |
| #27 | Pattern B session topology + smart oca entry (Session #0) | 8 | 6 | 7 | 8 | 1.8 | feature,priority:high |
| #20 | MCP tool suite profiles and per-agent exposure (S4) | 5 | 3 | 5 | 5 | 1.7 | feature,priority:low |
| #29 | tmux-resurrect integration + concurrency warning polish (Session #2) | 5 | 3 | 4 | 3 | 1.7 | feature,priority:medium |
| #28 | Graceful opencode session hibernation (Session #1) | 7 | 5 | 7 | 8 | 1.3 | feature,priority:high |

## Tech Debt (by WSJF, descending)

| # | Title | V | TC | RROE | E | WSJF | Labels |
|---|-------|---|----|------|---|------|--------|
| #25 | Read session debt from Advance instead of duplicating scan (O4) | 5 | 3 | 5 | 2 | 4.0 | priority:medium,tech-debt |

## Deferred / Unscored

(none)

## Triage Run Summary

- Run timestamp: 2026-05-11T23:50:00Z
- Sources scanned: gh (15 open / 12 closed-this-session), agenda (0), wisdom (0), notes (0), changes (0), todos (0)
- Issues closed this run: 9
  - #13 (M3), #14 (M2), #15 (M4), #16 (M5) — shipped as `installOcaUmbrellaPlugin` + `fixBareOcaApplyLifecycleParity` + `finalizeMustQueueTriage3` + `close4PreExistingDataCoverage`
  - #17 (M6) — already closed prior to this run
  - #33, #34, #35, #36, #37 — refiled to `Sharper-Flow/Advance#112-#116` (ADV plugin bugs, per cross-repo boundary)
- Issues opened this run: 0 (Advance#111 is upstream-only, filed earlier this session for sync-global tool drift)
- Items deferred: 0
- Critical path to v1.0.0: **#30** (operator-driven cutover + tag publication)
- Boundary doc: docs/proposals/2026-05-08-cross-repo-boundary-audit.md
- Recently archived changes contributing to this regen:
  - `finalizeMustQueueTriage3` (2026-05-11): closed M4+M5+M6 cleanup; M2 supersede marker
  - `close4PreExistingDataCoverage` (2026-05-11): closed the 4 migrator data-coverage gaps; acceptance gate met (`oca migrate from-open-chad → oca debug validate` returns ok)
  - `installOcaUmbrellaPlugin` (2026-05-04 / archived 2026-05-11): registered OCA umbrella plugin in operator's opencode.json
