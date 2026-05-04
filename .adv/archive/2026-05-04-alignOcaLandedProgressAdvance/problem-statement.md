## Problem

Advance has landed and in-progress changes that shift OCA-facing ownership boundaries, archive layout, worker liveness signals, command surfaces, and health prerequisites. OCA docs and health checks must be realigned so OCA does not duplicate Advance-owned behavior or misdiagnose valid Advance state.

## Success Criteria

- OCA docs accurately describe `.adv/specs` and `.adv/archive` in-repo ownership, plus external ADV state
- OCA health checks warn for stale ADV worker heartbeat and missing/unauthenticated `gh` CLI without hard-failing normal use
- OCA-owned worktree skill defers ADV-managed worktree lifecycle to Advance and removes stale `openchad`/checkout guidance
- OCA docs no longer use aspirational wording for landed Advance changes
- Bundle-aware `.adv/archive` handling avoids false legacy warnings for valid archive bundles
- `go test ./...`, `go vet ./...`, and `gofmt -d .` pass

## Scope

In scope:
- `AGENTS.md`, `STATUS.md`, `NEXT_STEPS.md`
- OCA-owned `assets/skills/worktree/SKILL.md`
- OCA health checks under `internal/health/` and helper code under `internal/advruntime/` if needed
- OCA doctor scope list/help if dependency scope exposure needs update
- Tests for the changed health/docs behavior

Out of scope:
- Mutating Advance-owned assets, including global `adv-worktree`
- Running `adv_migrate_cleanup` or deleting ADV state
- Stack schema changes
- Changing Temporal dev-server supervision semantics