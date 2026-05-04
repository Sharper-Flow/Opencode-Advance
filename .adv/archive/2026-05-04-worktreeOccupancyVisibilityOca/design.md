# Design: Worktree Occupancy Visibility for OCA and ADV Sessions

## Design Summary

Implement a visibility layer, not an enforcement layer.

- OCA records pane/session/worktree metadata in backwards-compatible pane-state v2 JSON.
- OCA exposes `oca occupancy` to group panes by project/worktree and warn on multiple active occupants.
- OCA tmux status shows compact current-pane occupancy context.
- Advance exposes privacy-safe occupancy hints from its existing Temporal session registry.
- No chat/message content participates in occupancy.
- Pattern B session-per-project/window-per-worktree remains separate and out of scope.

Independent validator verdict: `CAUTION`. Required changes are incorporated below.

## Architecture

### Plane 1 — OCA pane-state authority for local UI

OCA keeps current pane-keyed state path:

```text
$XDG_STATE_HOME/oca/panes/<socket>/<paneId>.json
```

Legacy v1 remains valid:

```json
{ "sessionID": "...", "directory": "...", "ts": 1710000000000 }
```

New v2 is best-effort and additive:

```json
{
  "schemaVersion": 2,
  "sessionID": "...",
  "directory": "...",
  "ts": 1710000000000,
  "startedAt": 1710000000000,
  "lastSeenAt": 1710000000000,
  "socket": "oca",
  "paneID": "%42",
  "sessionName": "oca-opencodeadvance-0",
  "windowID": "@7",
  "windowName": "worktreeOccupancyVisibilityOca",
  "agent": "adv",
  "gitRoot": "/home/jrede/dev/opencodeadvance",
  "gitCommonDir": "/home/jrede/dev/opencodeadvance/.git",
  "projectId": "<root-commit-sha>",
  "worktreePath": "/home/jrede/dev/opencodeadvance",
  "worktreeBranch": "trunk"
}
```

### Writer vs reader responsibility

| Field group | Plugin writes? | CLI derives/enriches? | Notes |
|---|---:|---:|---|
| `schemaVersion`, `sessionID`, `directory`, `ts`, `startedAt`, `lastSeenAt` | yes | validates | Compatibility core |
| `socket`, `paneID` | yes | validates | From `TMUX`, `TMUX_PANE` |
| `sessionName`, `windowID`, `windowName` | best-effort | yes via tmux | Plugin may omit if tmux query fails |
| `agent` | best-effort | no/unknown | From explicit env/config only; never inferred from chat |
| `gitRoot`, `gitCommonDir`, `projectId`, `worktreePath`, `worktreeBranch` | best-effort | yes via bounded git calls | Plugin writes at session creation; CLI fills gaps on full listing |

Compatibility rules:

- Missing `schemaVersion` means v1.
- v1 records still support `oca pane restart-tui`.
- Partial v2 records are valid.
- Malformed records are skipped and counted.

### Plane 2 — OCA occupancy aggregation

Add `internal/occupancy` package.

Responsibilities:

1. Discover records under `$XDG_STATE_HOME/oca/panes/<socket>/*.json`.
2. Parse v1/v2 permissively.
3. Reconcile with live tmux panes when available.
4. Enrich missing git metadata from `directory` or live `pane_current_path` using bounded subprocess calls.
5. Group by `(projectId || gitRoot || directory)` and `(worktreePath || directory)`.
6. Mark multiple active occupants in one worktree as an occupancy warning.

### Reconciliation precedence

When sources disagree, use this order:

1. **Live tmux pane reality** — pane exists/current path/session/window from `tmux list-panes`; highest authority for local UI.
2. **Advance Temporal registry** — authority for ADV session/worktree state when available to ADV-side tools/markers.
3. **Pane-state file** — local persisted hint; used for restart/status enrichment and stale diagnostics.
4. **Git enrichment** — derived metadata; never overrides live tmux cwd when present.

Conflict rules:

- Pane state exists, tmux pane gone → `stale`.
- tmux pane exists, no pane state → `active` with `unknown` metadata; warn plugin/state absent.
- pane-state `directory` differs from `pane_current_path` → prefer `pane_current_path`; mark `directoryDrift=true` in JSON.
- Advance registry shows session but no OCA pane state → ADV marker may still count it; OCA CLI does not invent an OCA pane.
- Temporal unavailable → OCA CLI still works from tmux + pane state; ADV enrichment omitted.

### Plane 3 — OCA CLI

Add primary command:

```bash
oca occupancy [--all] [--json] [--socket <name>] [--path <dir>] [--status] [--pane <id>]
```

Default human output groups active occupants only, hides stale records, and reports counts:

```text
Project: opencodeadvance  cdae139e...
  Worktree: trunk  /home/jrede/dev/opencodeadvance
    ✓ %42  oca-opencodeadvance-0:@7  adv  last-seen 12s

  Worktree: change/worktreeOccupancyVisibilityOca  /.../change/worktreeOccupancyVisibilityOca
    ⚠ %81  oca-opencodeadvance-0:@9  adv  last-seen 4s
    ⚠ %94  oca-opencodeadvance-1:@1  build last-seen 8s
    warning: 2 active sessions share this worktree

Warnings: 1 malformed record skipped, 2 stale records hidden (use --all)
```

JSON output exposes local operator data, including local branch/worktree fields, because it is an explicit local CLI surface. It does not read chat content.

`--status --pane <id>` is intentionally narrow:

- reads only the target pane state plus live pane metadata if cheap.
- no cross-pane aggregation.
- no git subprocess.
- prints compact current-pane segment or empty output.

Examples:

```text
1× trunk
2× occupied ⚠
?
```

### Plane 4 — tmux status integration

Update tmux config to pass pane id:

```tmux
set -g status-format[0] "#(bash $OCA_REPO_ROOT/lib/status_bar.sh row0 '#{session_name}' '#{pane_current_path}' '#{pane_id}')"
```

Update `lib/status_bar.sh`:

- preserve existing branch/ADV/Temporal output.
- call compact status path: `timeout 0.5s oca occupancy --status --pane <pane-id>`.
- suppress stderr.
- render empty segment on failure.

### Plane 5 — Advance privacy-safe agent visibility

Advance remains authority for ADV session/worktree registry.

Required Advance contribution:

1. Add `lastSeenAt` to `SessionListEntry` peer projection.
2. Add worktree occupancy marker based on internal registry comparison:

```text
[ADV:WORKTREE_OCCUPANCY] 2 sessions share this worktree. Nominal 1:1 violated; continuing allowed.
```

3. Marker exposes count only. No peer PID, no full peer path, no branch names, no chat content.
4. If Temporal/session registry is unavailable, omit marker or emit degraded count-free status.

### Branch/privacy policy

- `worktreeBranch` is **not** added to peer-facing Advance output.
- `worktreeBranch` may appear in local OCA CLI output and current-pane tmux/status context because the local operator already has access to that worktree/branch and the existing status bar already displays the current pane git branch.
- ADV peer markers stay count-only.

## Decisions

| Question | Decision | Rationale |
|---|---|---|
| State scope | Pane-keyed state + project/worktree metadata | Fits existing restart/status-bar path; aggregation happens in CLI |
| 1:1 policy | Visibility only, no lock | Preserves ADV multi-session support |
| CLI shape | New `oca occupancy` | Clear noun for “who is parked where”; avoids overloading existing `session list` |
| Stale records | Hidden by default, counted; `--all` shows | Keeps default output actionable while preserving diagnostics |
| Status path | Single-pane only, no git subprocess | Protects status bar latency budget |
| Chat logs | Never read | Privacy and stability |
| Pattern B | Out of scope | This change is narrower foundation |
| Advance peer fields | Add `lastSeenAt`; do not add `worktreeBranch` | Avoid privacy regression |

## Implementation Plan Shape

Planning should synthesize tasks in this order:

1. OCA tests for pane-state v1/v2 parse and occupancy grouping.
2. OCA pane-state TypeScript schema/writer enrichment.
3. OCA Go occupancy model + parser + tmux/git enrichment.
4. OCA CLI `oca occupancy` human/json/status output.
5. OCA status bar integration.
6. OCA docs/instructions update.
7. Advance cross-repo privacy-safe session projection and occupancy marker.
8. Cross-surface verification.

## Verification Strategy

- Go unit tests for occupancy parser/grouping/liveness classification.
- Go CLI golden/output tests for human and JSON output.
- TypeScript plugin tests for state write/merge behavior.
- Shell/status-bar test proving timeout/fail-closed behavior.
- Advance tests for session projection privacy and occupancy marker.
- Full repo verification: `go test ./...` for OCA; scoped Advance test target plus broader package test if feasible.

## Validator Resolution

Independent validator returned `CAUTION` with three required design fixes:

1. Resolve `worktreeBranch` privacy policy.
2. Specify reconciliation precedence.
3. Delineate plugin writer vs CLI reader responsibilities.

All three are resolved above. No validator `CONFLICT`, no contract-compromise risk remains.