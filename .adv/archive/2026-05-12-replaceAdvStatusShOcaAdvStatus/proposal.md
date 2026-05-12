# Replace adv_status.sh with oca adv-status Go Subcommand

## Why

Shell reads of `change.json`, `snapshot.json`, `temporal.env` couple OCA to ADV's internal schema. Go subcommand is typed, testable, and degrades gracefully when schema evolves.

## What Changes

### New files
- `internal/advstatus/status.go` — core types and readers: `ActiveChange`, `ChangeSummary`, `TemporalHealth`, `WorkspaceState`, `BranchSafety`
- `internal/advstatus/status_test.go` — unit tests with golden files
- `cmd/oca/adv_status.go` — cobra command `adv-status` with `--format text|json` and `--query active-change|worktrees|temporal-health|branch-safety`

### Modified files
- `lib/status_bar.sh` — replace 5 shell function calls with `oca adv-status --format text --query <name>` invocations; remove `source adv_status.sh`
- `lib/adv_status.sh` — delete (or reduce to 3-line thin wrapper calling `oca adv-status` for backward compat)

### Reused existing code
- `internal/advruntime/paths.go` — `DefaultADVStateRoot()` for state dir resolution
- `internal/advruntime/workspace_projection.go` — `ProjectWorkspaceStates()` for snapshot.json parsing and `WorktreeWorkspaceState` types

## Success Criteria

- [ ] `oca adv-status --format text --query active-change` replaces `oca_adv_active_summary` output
- [ ] `oca adv-status --format json` produces machine-readable JSON output
- [ ] `--query temporal-health` probes Temporal reachability (replaces `oca_adv_temporal_health`)
- [ ] `--query branch-safety <path> <project-id>` returns trunk guard glyph (replaces `oca_status_branch_safety`)
- [ ] `--query worktrees <project-id>` returns workspace state glyphs (replaces `oca_status_workspace_state`)
- [ ] `adv_status.sh` deleted or reduced to thin wrapper
- [ ] All existing `go test ./...` tests pass
- [ ] New code covered by unit tests with golden files

## Affected Code

- `lib/adv_status.sh` — deleted or gutted
- `lib/status_bar.sh` — caller updated to use Go binary
- `internal/advstatus/` — new package (reads ADV state via existing `advruntime` helpers)
- `cmd/oca/adv_status.go` — new cobra command

## Constraints

- Must run fast enough for tmux `status-interval` (≤50ms per invocation)
- Must work without `jq` installed (Go-native JSON parsing)
- Must degrade gracefully when ADV state dir doesn't exist (empty output, exit 0)
- Must handle missing `temporal.env` without error (silent no-op, matching current behavior)
- No Temporal SDK dependency — probe with TCP connect only (matching current `/dev/tcp` behavior)
- Cannot import Temporal client packages — this is a lightweight CLI, not a Temporal operator

## Impact

- tmux status bar behavior unchanged (same glyphs, same conditions)
- `adv_status.sh` callers: if deleted, any direct script users break; if thin-wrapped, backward compat preserved
- No ADV plugin changes required — reads the same state files

## Risks

- Go binary cold-start latency vs shell script (mitigated: `oca` binary already in PATH, Go startup ~5ms)
- `snapshot.json` format change by ADV signal cutover (same risk as current shell, but Go types catch schema drift at compile time)

## Validation Plan

- Unit tests for each query mode with golden file output
- Integration test: run `oca adv-status` against fixture ADV state dirs
- Manual: verify tmux status bar renders identically before/after

---

## Discovery Findings

### Discovery Checklist

| # | Step | Result |
|---|------|--------|
| 1 | Skill Discovery | PASS — no Go CLI / ADV state skills found. No core-domain gap (internal refactor). |
| 2 | Prior Research Extension | PASS — no prior research packs (`docs/*-prep.md`, `temp/*.md`). Cited: `docs/proposals/2026-05-08-cross-repo-boundary-audit.md` § O2. New finding: `status_bar.sh` also calls `_oca_adv_snapshot_read` directly (line 102) for window glyph enrichment — not just through the 5 public functions. |
| 3 | Conflict & Related-Work Scan | PASS — no active changes overlap. 40 archived changes, none modifying `adv_status.sh`. NO_TASKS/NO_DELTAS warnings expected (pre-prep). No agenda overlap. |
| 4 | Edge Case Investigation | PASS — see Edge Cases section. |
| 5 | Design Question Depth | PASS — see Open Design Questions section. |
| 6 | Draft Spec Deltas | PASS — see Draft Spec Deltas section. |
| 7 | P25 Related-Pattern Scan | PASS — see Related Pattern Scan section. |
| 8 | LBP Check | PASS — see LBP Check section. |

### Skills Considered
- Scanned `~/.config/opencode/skills/` — no skills matching Go CLI, ADV state, or shell-to-Go migration domains.
- No core-domain gap — this is an internal refactor using existing patterns.

### Extends
- **Cited:** `docs/proposals/2026-05-08-cross-repo-boundary-audit.md` § O2 — establishes the principle (Advance owns state format, OCA must not couple to internals) and action plan (Go subcommand, delete shell parser).
- **New finding:** `status_bar.sh` line 102 calls `_oca_adv_snapshot_read` directly for workspace status enrichment of window glyphs. This is a 6th call site not listed in the proposal's "5 shell function calls". The Go subcommand needs a `--query workspace-lookup` mode or the window glyph enrichment must call `--query worktrees` and build its own lookup table in shell.

### Conflict Scan
- No active changes overlap with this scope.
- 40 archived changes — none modified `adv_status.sh` after initial creation.
- `adv_change_validate`: passed. NO_TASKS/NO_DELTAS warnings expected pre-prep.
- Agenda: empty.

### Current State
- **`lib/adv_status.sh`**: 442 LOC, 15 shell functions. Reads `change.json` (jq), `snapshot.json` (jq), `temporal.env` (grep). Caches results with 10s TTL file-based cache in `$OCA_CACHE_DIR/adv_status`.
- **`internal/advruntime/`**: Go package with typed readers for ADV state. `ProjectWorkspaceStates()` already reads `snapshot.json` → `WorktreeWorkspaceState` structs. `DefaultADVStateRoot()` resolves the ADV state directory. No Go code reads `change.json` or `temporal.env`.
- **`lib/status_bar.sh`**: Sources `adv_status.sh`. Calls 5 public functions + 1 internal function (`_oca_adv_snapshot_read`) directly:
  1. `oca_adv_active_summary` (line 224) — find first active change, format as `{shortId}:{gate}`
  2. `oca_status_branch_safety` (line 216) — trunk guard ⚡ glyph
  3. `oca_status_workspace_state` (line 234) — highest-priority workspace state glyph
  4. `oca_adv_temporal_health` (line 258) — TCP probe → `T:✓` / `T:✗`
  5. `_oca_adv_snapshot_read` (line 102) — raw snapshot JSON for window glyph enrichment
  6. `_oca_adv_has_active_changes` (called by `oca_status_branch_safety` internally)
- **Shell tests**: `tests/shell/adv_status_test.sh` (162 LOC, 8 test cases) + `tests/shell/status_bar_test.sh` (645 LOC, 20+ test cases including branch safety, workspace state, window glyph enrichment). Both source `adv_status.sh` directly.
- **Spec**: `oca-workspace-projection` (3 requirements: projection reads, session enrichment, status bar glyphs). Current spec covers `snapshot.json` reading but not `change.json` or `temporal.env`.

### Edge Cases

1. **Missing `jq`**: Shell functions degrade silently (return `adv:?` or empty). Go subcommand has no jq dependency — JSON parsing is native. **Edge case**: Go binary must still return empty string (not error) when `change.json` is malformed — matching current `jq -r ... 2>/dev/null` behavior.

2. **Multiple ADV projects**: `oca_adv_active_summary` scans all project dirs when no specific dir given. Go version must also scan `$XDG_DATA_HOME/opencode/plugins/advance/*/changes/` — not just a single project.

3. **Temporal env IPv6 brackets**: Shell strips `[]` from bracketed addresses (line 243-246). Go TCP probe must handle `[::1]:7233` format — use `net.Dial` with host:port, which handles this natively.

4. **Cache vs on-demand**: Shell caches with 10s TTL. Go binary invoked fresh each tmux refresh (~10s interval). Each invocation reads files fresh — no persistent cache needed if file reads are <50ms (they are — local SSD JSON files).

5. **Window glyph snapshot enrichment**: `_oca_status_window_glyphs_from_list` builds a changeId→status lookup from raw snapshot JSON. In Go world, this becomes either:
   - (a) `oca adv-status --query workspaces-lookup <project-id>` returning `changeId:status` lines, parsed in shell
   - (b) Entire window glyph logic moves to Go (larger scope, likely a future change)

### Open Design Questions

**DQ1: Should `adv_status.sh` be deleted or thin-wrapped?**
- **Trust model:** User decision (backward compat)
- **Blast radius:** Any script directly sourcing `adv_status.sh` breaks on delete
- **Alternatives:** (a) Delete entirely — cleanest, removes shell test burden; (b) 3-line wrapper sourcing `oca adv-status` — backward compat for direct callers
- **Recommendation:** Delete. Shell tests for `adv_status.sh` should be replaced by Go tests. `status_bar_test.sh` should be updated to call `oca adv-status` or have its adv_status tests removed.

**DQ2: How does `status_bar.sh` call the Go binary?**
- **Trust model:** Agent decision (implementation detail)
- **Blast radius:** tmux status bar breaks if invocation pattern is wrong
- **Alternatives:**
  - (a) `oca adv-status --query active-change --format text` for each query — 4-5 separate invocations per refresh
  - (b) `oca adv-status --format json` once, parse in shell with jq — defeats purpose
  - (c) `oca adv-status --query all` single invocation returning structured output shell can parse
- **Recommendation:** (a) separate invocations — each is ~5ms, total ~25ms, well within 50ms budget. Clean separation, no jq dependency in `status_bar.sh`.

**DQ3: What about the window glyph snapshot enrichment?**
- **Trust model:** Agent decision (implementation detail)
- **Blast radius:** Window glyphs lose workspace state suffix if not handled
- **Alternatives:**
  - (a) Add `--query workspace-lookup <project-id>` returning `changeId:status` TSV
  - (b) Use `--query worktrees <project-id>` and build lookup in shell from the richer output
  - (c) Move entire window glyph rendering to Go (future scope)
- **Recommendation:** (a) `--query workspace-lookup` — minimal, direct replacement for the `_oca_adv_snapshot_read` + jq pipeline. Shell parses simple TSV, no jq needed.

### Draft Spec Deltas

**Δ1: New spec `oca-adv-status-cli`** (capability for the Go subcommand)

| ID | Title | Priority |
|---|---|---|
| rq-advcli-active-change | `--query active-change` returns first active change summary | must |
| rq-advcli-json-format | `--format json` returns structured JSON output | must |
| rq-advcli-temporal-health | `--query temporal-health` probes Temporal via TCP | must |
| rq-advcli-branch-safety | `--query branch-safety <path> <pid>` returns trunk guard glyph | must |
| rq-advcli-workspace-state | `--query worktrees <pid>` returns highest-priority workspace glyph | must |
| rq-advcli-workspace-lookup | `--query workspace-lookup <pid>` returns changeId:status TSV | must |
| rq-advcli-graceful | Returns empty output + exit 0 when ADV state unavailable | must |
| rq-advcli-performance | Completes in ≤50ms on local SSD | must |

**rq-advcli-active-change G/W/T:**
- Given: ADV state dir has project with active (non-archived, non-closed) change
- When: `oca adv-status --query active-change --format text`
- Then: Output is `{shortId}:{currentGate}` matching current shell format

**rq-advcli-temporal-health G/W/T:**
- Given: `temporal.env` exists with `ADV_TEMPORAL_ADDRESS=127.0.0.1:1`
- When: `oca adv-status --query temporal-health --format text`
- Then: Output is `T:✗` (unreachable)

**rq-advcli-graceful G/W/T:**
- Given: No ADV state directory exists
- When: `oca adv-status --query active-change --format text`
- Then: Output is empty, exit code 0

### Related Pattern Scan
- **Similar pattern:** `internal/advruntime/workspace_projection.go` — already reads `snapshot.json` with Go-native JSON parsing. This is the exact pattern to follow.
- **Similar pattern:** `internal/health/advance.go` — reads `change.json` in archive dirs. Uses `os.ReadFile` + `json.Unmarshal`. Same pattern needed for active changes.
- **Similar pattern:** `internal/render/temporal.go` — renders `temporal.env`. Go code reads env vars; the `adv_status.sh` reads the rendered file. Go subcommand should read the same file.
- **No similar pattern for TCP health probe** — the shell's `/dev/tcp` probe is unique. Go equivalent: `net.DialTimeout("tcp", addr, 1*time.Second)`.

### LBP Check
- **Direction:** Replace shell with Go — **matches LBP**. Go is the project's primary language. Typed JSON parsing > jq string processing. Compile-time schema drift detection > silent failures.
- **External-solution check:** Not applicable — purely internal refactor. No external alternatives viable.

### AMBIGUITY ANALYSIS — no ambiguity findings. Coverage: B:C F:C S:C M:C

- **B (Boundaries):** Clear — replace shell functions, keep output format identical. Out-of-scope: window glyph rendering logic stays in shell (just data source changes).
- **F (Functional Scope):** Clear — 5 query modes + workspace lookup, text + json formats.
- **S (Completion Signals):** Clear — shell tests pass → Go tests pass → tmux bar renders identically.
- **M (Missing Information):** Clear — all state files documented, all callers identified, all edge cases understood.

### Recommended Objectives

1. Create `internal/advstatus` package with typed readers for `change.json` and `temporal.env`, reusing `advruntime.ProjectWorkspaceStates` for `snapshot.json`
2. Create `cmd/oca/adv_status.go` cobra command with `--format text|json`, `--query active-change|worktrees|temporal-health|branch-safety|workspace-lookup`
3. Update `lib/status_bar.sh` to call `oca adv-status` instead of shell functions; remove `source adv_status.sh`
4. Delete `lib/adv_status.sh` entirely
5. Replace `tests/shell/adv_status_test.sh` with Go tests; update `tests/shell/status_bar_test.sh` to remove adv_status-specific tests
6. Update spec `oca-workspace-projection` rq-ocawp-statusBar01 to reference Go binary as data source