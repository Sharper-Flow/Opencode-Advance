# Design

## Architecture Overview

`oca maintain` is an offline operator command. It does not try to hot-reload OpenCode or Temporal workers. It makes runtime code safe for the *next* session by running only when current sessions/workers are quiescent.

Authority split:

```text
OCA CLI = offline orchestration, session/process gate, git merge/rebuild/cleanup UX
Git = branch, worktree, merge, dirty-state truth
Advance standalone scripts = ADV state eligibility + registry reconciliation surfaces
Temporal = health signal only in v1; Worker Versioning deferred
OpenCode session = consumer of next-session plugin marker, never mutated live
```

Validator CAUTION resolutions incorporated:

- Agreement now persists explicit AC.
- Full verification evidence is defined for v1 as archived ADV state with `release` gate status `done`.
- Existing OCA primitives are reused: `internal/advruntime/*`, `cmd/oca/occupancy.go`, `internal/session/*`, plugin build/update helpers.
- Advance standalone inspect output must include `schema_version`.
- `oca adv recover --dry-run` overlap is addressed by sharing report/scanner code and making `maintain` the execute-capable offline command.

## Key Decisions

### 1. New command: `oca maintain`

Register a new root command in `cmd/oca/root.go`:

```text
oca maintain [--dry-run] [--execute]
             [--include-cleanup]
             [--include-merge]
             [--include-rebuild]
             [--project <path>]
```

Defaults:

- no flag or `--dry-run`: plan only, no mutation
- `--execute`: permit safe mutations after hard gates pass
- text output by default; existing `--output json` emits stable JSON

Core packages:

```text
cmd/oca/maintain.go
internal/maintain/plan.go
internal/maintain/session_gate.go
internal/maintain/git.go
internal/maintain/plugin_drift.go
internal/maintain/rebuild.go
internal/maintain/cleanup.go
internal/maintain/advance_state.go
internal/maintain/report.go
```

### 2. Hard active-state gate

Before any mutation in `--execute`, `oca maintain` checks:

1. OpenCode processes (`opencode`) on the local machine.
2. OCA tmux sessions/windows for the project.
3. ADV/Temporal worker processes for the project queue.
4. Temporal poller presence for the project queue when Temporal is reachable.

Reuse existing primitives:

- `cmd/oca/occupancy.go` live pane/session helpers for OpenCode/OCA occupancy.
- `internal/session` tmux session manager.
- `internal/advruntime.NewWorkerLockScanner()` for worker lock/process facts.
- `internal/advruntime.NewWorkflowClassifier()` / Temporal task queue probing for poller facts.

If any active state exists, execute exits non-zero before mutation and reports blockers. Dry-run reports blockers but stays read-only.

No auto-kill path in v1. This follows Temporal docs: worker shutdown may interrupt in-flight Workflow Tasks/Activities unless graceful shutdown completes, and ADV has no Worker Versioning deployment model yet.

### 3. Maintenance plan model

`internal/maintain` produces a deterministic plan:

```go
type Plan struct {
  SchemaVersion int
  GeneratedAt time.Time
  ProjectRoot string
  DefaultBranch string
  SessionGate GateReport
  PluginDrift []PluginDrift
  MergeCandidates []MergeCandidate
  Rebuilds []RebuildAction
  CleanupCandidates []CleanupCandidate
  TemporalHealth []HealthFinding
  Actions []Action
  Blockers []Blocker
}
```

Plan actions have stable IDs (`merge:<changeId>`, `rebuild:advance`, `cleanup:<branch>`) and explicit `wouldRun`, `executeAllowed`, `blockedBy`, and `evidence` fields.

### 4. Existing `oca adv recover` relationship

`oca adv recover --dry-run` remains read-only recovery planning. `oca maintain` becomes the offline execute-capable maintenance command.

To avoid duplication:

- Move reusable `collectReport()` logic from `cmd/oca/adv.go` into an internal package or expose a shared collector.
- Reuse `internal/advruntime` scanners for session debt, worker locks, workflow queues, and worktree census.
- `maintain` may embed the `adv recover` report as a health/recovery section.

### 5. Advance state boundary: standalone scripts, not MCP tools

`oca maintain` must run when OpenCode is closed, so it cannot rely on ADV MCP tools. Advance adds a standalone script surface under `scripts/maintenance/`:

```text
scripts/maintenance/inspect.mjs
scripts/maintenance/reconcile-worktree.mjs
```

`inspect.mjs` returns JSON with `schema_version: 1`:

```json
{
  "schema_version": 1,
  "project_id": "...",
  "changes": [
    {
      "change_id": "...",
      "status": "archived",
      "release_gate": "done",
      "full_verification": {
        "source": "release_gate",
        "verified": true,
        "summary": "release gate complete"
      },
      "branch": "change/...",
      "worktree_path": "...",
      "cleanup_eligibility": "eligible|blocked_*"
    }
  ]
}
```

`reconcile-worktree.mjs` updates ADV registry only for already-proven git facts after OCA removes a worktree. If unavailable, OCA reports registry reconciliation as a follow-up and does not mutate ADV state directly.

This keeps OCA from parsing live external ADV state schemas directly and avoids OpenCode cached plugin code.

### 6. Verified merge policy

A branch is eligible only when all are true:

1. Advance inspect says change is archived and release gate is `done`.
2. `full_verification.verified == true` with source `release_gate` for v1.
3. Main checkout is on default branch and clean.
4. Branch exists locally as `change/<changeId>` or explicit linked branch.
5. Branch is clean in its worktree, including untracked files.
6. `git merge --ff-only <branch>` can succeed.

Non-ff merges are skipped in v1 with a diagnostic: use PR/manual reconciliation. This preserves the existing ADV merge-before-delete protocol and trunk-is-prod.

### 7. Rebuild and build marker

After merging any branch that touches plugin runtime source (`plugin/src/**`, `plugin/package.json`, lockfile, build config, worker source), OCA rebuilds the configured plugin in the main checkout using existing plugin build commands from `stack.toml`.

After successful build, OCA writes a sidecar marker:

```json
{
  "schema_version": 1,
  "plugin": "advance",
  "source_root": "/home/.../oc-plugins/advance",
  "git_sha": "<HEAD>",
  "branch": "trunk",
  "built_at": "<RFC3339>",
  "dist_index": "plugin/dist/index.js",
  "worker_bundle": "plugin/dist/temporal/worker.js",
  "build_command_hash": "<hash>"
}
```

Default marker path:

```text
<advance-checkout>/plugin/dist/oca-build.json
```

OCA doctor and Advance runtime diagnostic read this marker.

### 8. Loaded runtime diagnostic in Advance

Advance adds a tool/diagnostic surface such as `adv_plugin_runtime_info` or extends `adv_status view=health` to report:

- plugin module URL/path loaded by OpenCode (`import.meta.url` / resolved package main)
- process start time
- loaded marker if available
- expected marker from configured plugin checkout if discoverable
- worker script path and marker if available
- caveat: host-loaded tool code requires OpenCode restart

This satisfies #40's requirement that a new session can tell what plugin dist/path it loaded.

### 9. Cleanup policy

Cleanup candidate eligible only when:

1. Branch merged into default branch.
2. Worktree clean, including untracked.
3. No process cwd under worktree.
4. No OCA/OpenCode tmux session references it.
5. Advance inspect surface marks it safe or unavailable-but-git-safe.

Execute performs:

```text
git worktree remove <path>
git branch -d <branch>
advance scripts/maintenance/reconcile-worktree.mjs --branch <branch> --status deleted
```

If reconcile script is unavailable/fails, git cleanup result is reported with a follow-up blocker for ADV registry reconciliation.

### 10. Temporal v1 scope

Temporal actions in v1 are health/report-only:

- report server reachable/unreachable
- report task queue pollers if reachable
- report worker-lock/process state if inspectable
- suggest `oca temporal start|status` or ADV repair tools

No `adv_temporal_worker_restart`, no forced worker lock reclaim, no Workflow Versioning mutation in v1.

## Implementation Strategy

### Phase A — OCA plan and session gate

Files:

- `cmd/oca/maintain.go`
- `internal/maintain/plan.go`
- `internal/maintain/session_gate.go`
- shared runtime report collector extracted from `cmd/oca/adv.go`
- tests under `tests/maintain_*` or package tests

Work:

- command registration
- dry-run JSON/text plan
- active-state scan using existing scanners
- no mutation in dry-run

### Phase B — plugin drift and rebuild marker

Files:

- `internal/maintain/plugin_drift.go`
- `internal/maintain/rebuild.go`
- `internal/plugin/build.go` if helper extraction needed
- `internal/health/plugin.go` marker check extension

Work:

- detect source newer/different than dist marker
- write `dist/oca-build.json`
- extend doctor plugin check to report marker freshness

### Phase C — Advance standalone inspect surface

Files in Advance:

- `scripts/maintenance/inspect.mjs`
- `scripts/maintenance/reconcile-worktree.mjs`
- tests for schema and corrupted issue URL validation
- optional `plugin/src/tools/plugin-runtime-info.ts`

Work:

- emit `schema_version: 1` archived/verified/branch eligibility JSON
- validate issue URLs on `adv_change_update_issues` to prevent recurrence
- expose runtime loaded-path/build-marker diagnostic

### Phase D — merge/rebuild execute path

Files:

- `internal/maintain/git.go`
- `internal/maintain/rebuild.go`
- command tests with fake git repos

Work:

- default-branch clean gate
- archived+release-done candidate filter
- `git merge --ff-only`
- rebuild after runtime-source merge
- idempotent rerun after partial build failure

### Phase E — conservative cleanup

Files:

- `internal/maintain/cleanup.go`
- `internal/advruntime/worktree_census.go` helper reuse if needed

Work:

- process-cwd check
- clean/merged/session-free checks
- git worktree remove + branch delete
- ADV reconcile script integration or follow-up report

## Verification Plan

OCA:

- Unit tests for plan generation, active gate, JSON shape, merge candidate filtering, build marker writing.
- Fake git integration tests for ff-only merge, conflict skip, cleanup skip.
- `go test ./...` full verification.

Advance:

- Tests for maintenance inspect JSON schema.
- Tests for issue URL validation in `adv_change_update_issues` to prevent recurrence.
- Tests for runtime diagnostic marker/path reporting.
- Existing `pnpm test` / targeted tool tests for touched files.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| OCA duplicates ADV state parsing | Use Advance standalone JSON inspect/reconcile scripts. |
| Existing recovery scanners duplicated | Extract/reuse `internal/advruntime` collector/scanners. |
| Active worker killed mid-task | Hard refuse; no auto-kill. |
| Unverified code reaches trunk | Archived + release gate complete + ff-only merge required. |
| Mtime lies about dist freshness | Git SHA build marker. |
| Cleanup deletes useful work | Require merged, clean, session-free, process-free. |
| Partial failure after merge before rebuild | Idempotent rerun detects stale/missing marker and retries rebuild. |

## Design Notes for #40

This design solves #40 by avoiding live reload entirely:

1. Fix lands and verifies in worktree.
2. Change reaches release/archived state.
3. All OpenCode/ADV sessions close.
4. `oca maintain --execute` verifies archived+release-done evidence, merges, rebuilds, writes marker.
5. Next OpenCode session loads main-checkout plugin dist and can report marker/path.

The design intentionally does not solve full Temporal Worker Versioning. It records the gap and leaves the path open for a later change.