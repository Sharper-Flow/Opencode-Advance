# Advance Plugin Impact Check — 2026-05-02

Purpose: capture recent and in-progress Advance plugin work that may affect OpenCode Advance (OCA).

## Checked sources

- Local Advance checkout: `/home/jrede/dev/oc-plugins/advance`
- Recent git history and worktrees
- Advance ADV change list for draft/pending/active changes
- OCA docs/code references to Advance-owned assets, Temporal, and sync delegation

## Findings

### High impact: `repairTemporalMigrationDebt`

Status at check time: draft, execution pending, 8/15 tasks done.

Relevant changes:

- Adds `adv_migrate_cleanup` to detect and remove legacy in-repo `.adv/{changes,db,agenda*}` and non-bundle `.adv/archive` residue while preserving `.adv/specs/` and valid archive bundles.
- Adds `adv_change_diagnose` for disk-vs-Temporal divergence inspection.
- Adds `target_path` support to recovery/sweep tools.
- Adds `_healthSnapshot` to `adv_status` for closed/source-dir leak detection.

OCA impact:

- Direct. Earlier dry-runs against this repo reported legacy `.adv/changes` plus `.adv/archive` entries that now require bundle-aware classification.
- Do not remove manually. Use landed Advance cleanup tooling, then approve execute cleanup with backup/commit only after dry-run review.
- Candidate OCA doctor enhancement: warn when legacy `.adv/{changes,db,agenda*}` or non-bundle `.adv/archive` residue exists in repo, but never treat `.adv/specs/` or valid archive bundles as legacy.

### High impact: `boundParentProjectWorkflow`

Status at check time: draft, accepted, release pending, 17/17 tasks done.

Relevant changes:

- Adds Temporal worker singleton behavior.
- Bounds parent `projectWorkflow` summary growth.
- Adds archive purge and bulk-close disk cleanup.
- Changes `adv_temporal_worker_restart` to fire-and-forget; verify via `adv_status` / `adv_temporal_diagnose` after a short wait.

OCA impact:

- OCA Temporal health/doctor logic should not assume one worker per OpenCode session.
- Status bar / doctor should prefer service reachability and workflow health over process-count heuristics.
- No immediate OCA code change required unless a doctor check starts inspecting Advance worker process counts.

### Medium impact: dirty Advance checkout

Status at check time: Advance `trunk` was ahead of `origin/trunk` and had uncommitted edits plus untracked `migrate-cleanup` files.

OCA impact:

- Avoid pinning Advance to the local checkout state until dirty work is archived/merged.
- `oca update/apply` can surface dirty-checkout friction depending stack/ref/pin flow.

### Medium impact: Task guard parallelism

Observed in uncommitted Advance edits:

- Top-level primary agents may spawn up to 3 concurrent subagents.
- Subagents remain blocked from spawning nested subagents.

OCA impact:

- Aligns with current ADV policy. No OCA code impact found.
- Avoid documenting “3-4 parallel subagents” as an Advance guarantee; canonical cap is moving to 3.

## No impact found

- No recent `scripts/sync-global.sh` contract change found.
- OCA already treats `scout.md` and `refine.md` as stale Advance assets, not current shipped assets.
- No OCA references to `adv_migrate_cleanup` existed before this note.

## Follow-up checklist

- [ ] Run landed `adv_migrate_cleanup` dry-run against OCA again.
- [ ] If dry-run still reports legacy state, approve execute mode with backup/commit.
- [ ] Add OCA doctor warning for legacy in-repo ADV state if this becomes a recurring issue.
- [ ] Review OCA Temporal doctor/status logic for assumptions about worker process count after `boundParentProjectWorkflow` lands.
