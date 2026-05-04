# Design

## Goal

Bring OCA's docs, health checks, and OCA-owned worktree guidance in line with current Advance behavior without duplicating or mutating Advance-owned assets.

## Key Decisions

### KD-A — Docs and ownership matrix are source-alignment work, not behavior changes

Update `AGENTS.md`, `STATUS.md`, and `NEXT_STEPS.md` to reflect landed Advance work:

- `.adv/specs/` remains valid in-repo Advance state.
- `.adv/archive/{bundle}/change.json` bundle directories are valid in-repo archive artifacts after `addAgentMeshAndInRepoArchive`.
- Mutable runtime state still primarily lives under `$XDG_DATA_HOME/opencode/plugins/advance/{project-id}/`; in-repo archive is a durable source artifact, not live mutable state.
- `boundParentProjectWorkflow` and `repairTemporalMigrationDebt` are landed, not aspirational.
- Heartbeat env vars are documented: `ADV_WORKER_HEARTBEAT_STALE_MS`, `ADV_WORKER_HEARTBEAT_INTERVAL_MS`.

### KD-B — Bundle-aware `.adv/archive` diagnosis belongs in OCA health, but must be simple and non-destructive

Add or extend a health check that scans the current repo-local `.adv/` directory.

Classification:
- Valid: `.adv/specs/`.
- Valid: `.adv/archive/*/change.json` bundle directories.
- Warn: `.adv/changes`, `.adv/db`, `.adv/agenda*`.
- Warn: `.adv/archive/` exists but contains no bundle subdirectory with `change.json`, or contains non-bundle residue.
- Never delete or migrate files; hint points to Advance cleanup tooling (`adv_migrate_cleanup` dry-run/execute from an ADV-capable session).

Implementation seam: existing `CheckADVPlugin` already validates Advance checkout and ADV state dir, making it a likely home for local ADV state warnings. Add `health.Options.ProjectRoot` if needed, with `os.Getwd()` fallback so CLI compatibility remains intact.

### KD-C — `gh` CLI check extends dependency health

Extend `CheckDependencies` with a third check:

1. Probe `gh --version` for availability.
2. Probe `gh auth status` for authentication.
3. Return `StatusWarn` for missing or unauthenticated; `StatusPass` for authenticated.

Rationale: agent mesh depends on `gh` for issue creation/scan. Missing `gh` degrades mesh, but does not break core OCA/ADV operation, so warn rather than fail.

Implementation seam: use subprocess abstraction, not raw shell. Update doctor scope help/completion so `dependencies` is discoverable if it remains the host scope.

### KD-D — Worker heartbeat diagnosis belongs in `adv-runtime`, not `temporal`

Do not modify `internal/temporal/supervise.go`; OCA supervises the Temporal dev server, not ADV workers.

Add an `advruntime` helper such as `WorkerLockScanner`:

- Input: ADV state root (`advruntime.DefaultADVStateRoot()`), stale duration (default 60s or env `ADV_WORKER_HEARTBEAT_STALE_MS`), now, PID liveness function.
- Scan **all** project subdirectories matching `{advRoot}/{projectId}/worker.lock`, consistent with existing multi-project `adv-runtime` behavior.
- Parse v2 if `last_heartbeat` is present.
- V1 fallback: no `last_heartbeat` → pass/info; no warning.
- Missing lock files: no finding.
- Stale heartbeat + PID alive: warning with project id, owner pid, and remediation hint (`adv_temporal_worker_restart` / restart OpenCode if needed).
- Dead PID: warning with cleanup/restart hint, but no reclaim by OCA.

Integrate in `CheckAdvRuntime` after workflow/task-queue classifier so queue health and lock health are both visible in `oca doctor --scope adv-runtime`.

### KD-E — OCA worktree skill becomes a thin redirect/generic defer

Update `assets/skills/worktree/SKILL.md` using validator Option A:

- Keep file so OCA skills health still has a source asset.
- State: this skill is superseded for ADV-managed worktrees by Advance's `adv-worktree` skill; load `skill("adv-worktree")` for ADV flows.
- If generic git worktree guidance remains, keep it short and non-conflicting.
- Use `adv_worktree_create/delete/cleanup/triage` as primary names when mentioning ADV tools; aliases are backward compatibility only.
- Remove `git checkout` cleanup guidance; use `git -C "$MAIN" merge --ff-only` invariant if examples remain.
- Remove active `openchad`/`oc switch` references.

### KD-F — Do not edit ADV-owned `adv-worktree` from OCA

Discovery found the global ADV-owned `adv-worktree` skill also has stale `openchad` wording. This change records it as follow-up evidence only. Any mutation belongs in Advance.

### KD-G — No exact `/adv-coordinate` cleanup needed

Exact search found no `/adv-coordinate` references. Historical `adv-coordinated-session-marker` filenames are not stale command refs and should remain unless separately refactored.

## Validator Resolution

Independent validator verdict: CAUTION; resolved before gate completion by adding explicit Scope/Success Criteria, selecting thin worktree redirect, specifying all-project worker scan, and specifying archive bundle heuristic.