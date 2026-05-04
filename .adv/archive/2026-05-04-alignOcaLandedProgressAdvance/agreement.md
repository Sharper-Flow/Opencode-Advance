# Agreement

## Objectives

1. Align OCA docs with current Advance ownership/state reality: in-repo `.adv/specs` plus valid `.adv/archive` bundles, external mutable state elsewhere.
2. Add OCA health diagnostics for new OCA-facing Advance prerequisites: stale worker heartbeat and `gh` CLI auth for mesh.
3. Reconcile OCA-owned worktree guidance so ADV-managed worktrees are explicitly Advance-owned.
4. Remove stale active guidance (`openchad`, checkout cleanup, old worktree aliases as primary names, transitional Advance wording).
5. Preserve OCA/Advance boundary: OCA surfaces environment state and docs; Advance owns ADV workflow behavior.

## Acceptance Criteria

1. AGENTS.md architecture and ownership matrix documents `.adv/archive/` bundle directories as Advance-owned in-repo artifacts while keeping external ADV state clear.
2. AGENTS.md/STATUS/NEXT_STEPS use landed wording for archived Advance changes and document heartbeat env vars: `ADV_WORKER_HEARTBEAT_STALE_MS`, `ADV_WORKER_HEARTBEAT_INTERVAL_MS`.
3. `internal/health/` warns when ADV worker lock v2 has stale `last_heartbeat` with live PID; v1/missing lock files degrade gracefully.
4. `internal/health/` dependency checks warn when `gh` CLI is missing or unauthenticated.
5. `assets/skills/worktree/SKILL.md` becomes a thin redirect/generic defer to `adv-worktree`, uses `adv_worktree_*` names, removes `git checkout` cleanup guidance, and removes active `openchad` references.
6. `.adv/archive` doctor behavior is bundle-aware: archive subdirs containing `change.json` are valid; empty or non-bundle archive residue can warn.
7. No exact `/adv-coordinate` references remain; historical `adv-coordinated-*` proposal filenames are allowed.
8. `go test ./...`, `go vet ./...`, and `gofmt -d .` pass.

## Constraints

- Do not modify Advance-owned assets from this OCA change.
- Do not add `related_repos` schema handling to OCA; Advance owns that project config.
- Do not change `internal/temporal/supervise.go` worker ownership semantics; OCA manages Temporal server, not ADV worker lifecycle.
- Warnings must be non-fatal unless existing OCA health semantics require fail.

## Decisions

### User Decisions

- `.adv/archive` warnings: warn non-bundles only; valid archive bundles with `change.json` should not warn.
- `gh` health noise: warn if `gh` is missing or unauthenticated.
- Worktree ownership: ADV owns ADV-managed worktrees; OCA skill should defer to `adv-worktree` rather than duplicate lifecycle guidance.

### Agent Decisions (LBP)

- Use `dependencies` health scope for `gh` CLI check and update CLI supported scope list if needed.
- Implement heartbeat warning in `adv-runtime` health because it is ADV runtime state, not Temporal server supervision.
- Scan all project dirs under ADV state root for worker locks, matching existing multi-project adv-runtime behavior.
- Keep OCA `worktree` skill as thin redirect/defer guidance; do not delete asset because OCA skills health expects source asset.
- Track ADV-owned `adv-worktree` stale wording as follow-up evidence, not as an OCA file mutation.

## Deferred Questions

None.

## Investment Snapshot

Investment: 0 tasks / 0 retries / ~4 min / tier: auto