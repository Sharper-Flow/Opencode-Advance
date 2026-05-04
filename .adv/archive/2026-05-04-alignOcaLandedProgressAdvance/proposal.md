# Proposal

## Problem

Advance has landed and in-progress changes that shift OCA-facing ownership boundaries, archive layout, worker liveness signals, command surfaces, and health prerequisites. OCA docs and health checks must be realigned so OCA does not duplicate Advance-owned behavior or misdiagnose valid Advance state.

## Scope

In scope:
- `AGENTS.md`, `STATUS.md`, `NEXT_STEPS.md`
- OCA-owned `assets/skills/worktree/SKILL.md`
- OCA health checks under `internal/health/` and helper code under `internal/advruntime/` if needed
- OCA doctor scope list/help if dependency scope exposure needs update
- Tests for changed health/docs behavior

Out of scope:
- Mutating Advance-owned assets, including global `adv-worktree`
- Running `adv_migrate_cleanup` or deleting ADV state
- Stack schema changes
- Changing Temporal dev-server supervision semantics

## Success Criteria

- OCA docs accurately describe `.adv/specs` and `.adv/archive` in-repo ownership, plus external ADV state
- OCA health checks warn for stale ADV worker heartbeat and missing/unauthenticated `gh` CLI without hard-failing normal use
- OCA-owned worktree skill defers ADV-managed worktree lifecycle to Advance and removes stale `openchad`/checkout guidance
- OCA docs no longer use aspirational wording for landed Advance changes
- Bundle-aware `.adv/archive` handling avoids false legacy warnings for valid archive bundles
- `go test ./...`, `go vet ./...`, and `gofmt -d .` pass

## Discovery Findings

Discovery completed and agreement persisted. See `agreement.md` for approved objectives/AC.

Key evidence:
- `AGENTS.md:103` says `.adv/` is specs-only; Advance now writes `.adv/archive/` bundles in-repo.
- `AGENTS.md:217-218`, `STATUS.md:10,60`, `NEXT_STEPS.md:55-56,112` use stale/pre-landing wording for Advance repair and worker changes.
- `internal/health/dependency.go` checks Node and Temporal CLI only; no `gh` check.
- `internal/health/adv_runtime.go` checks Temporal workflows/worktrees but not `worker.lock` heartbeat.
- `assets/skills/worktree/SKILL.md:17,54-58,96,103` uses old aliases, `git checkout`, and `openchad` wording.
- `project.json:4-8` includes `.adv/specs`, `.adv/archive`, `.adv/db`; docs must distinguish valid specs/archive bundles from mutable residue.
- Exact `/adv-coordinate` command refs absent; only `adv-coordinated-session-marker` historical proposal filenames found.

Discovery decisions:
- `.adv/archive` doctor behavior: warn non-bundles only.
- `gh` CLI health: warn if missing/unauthenticated.
- ADV-managed worktrees belong to Advance; OCA worktree skill becomes a thin redirect/generic defer to `adv-worktree`.