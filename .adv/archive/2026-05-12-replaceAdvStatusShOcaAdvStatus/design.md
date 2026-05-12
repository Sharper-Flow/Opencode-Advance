# Design: Replace adv_status.sh with oca adv-status Go Subcommand

## Architecture

### New Package: `internal/advstatus`

```
internal/advstatus/
├── status.go          # Types, readers, query functions
├── status_test.go     # Unit tests with golden file fixtures
├── change.go          # change.json reader + active change finder
├── temporal.go        # temporal.env reader + TCP health probe
└── testdata/          # Golden file fixtures
    ├── active-change-text.golden
    ├── active-change-json.golden
    ├── temporal-health-reachable.golden
    ├── temporal-health-unreachable.golden
    ├── branch-safety-unsafe.golden
    ├── workspace-state-setup-failed.golden
    ├── workspace-lookup.golden
    └── fixture-state/  # Simulated ADV state dirs
        ├── project-a/
        │   ├── changes/change-1/change.json
        │   └── snapshot.json
        └── temporal.env
```

### Types

```go
// QueryMode enumerates the --query values.
type QueryMode string

const (
    QueryActiveChange    QueryMode = "active-change"
    QueryTemporalHealth  QueryMode = "temporal-health"
    QueryBranchSafety    QueryMode = "branch-safety"
    QueryWorktrees       QueryMode = "worktrees"
    QueryWorkspaceLookup QueryMode = "workspace-lookup"
)

// ChangeSummary is the parsed summary of a single change.
type ChangeSummary struct {
    ID          string `json:"id"`
    ShortID     string `json:"shortId"`
    CurrentGate string `json:"currentGate"`
    Status      string `json:"status"`
}

// TemporalHealth is the result of a Temporal TCP probe.
type TemporalHealth struct {
    Reachable bool   `json:"reachable"`
    Address   string `json:"address,omitempty"`
    Glyph     string `json:"glyph,omitempty"` // "T:✓" or "T:✗"
}

// BranchSafetyResult is the trunk guard assessment.
type BranchSafetyResult struct {
    Unsafe bool   `json:"unsafe"`
    Branch string `json:"branch"`
    Glyph  string `json:"glyph,omitempty"` // tmux-colored ⚡ or empty
}

// WorkspaceStateResult is the highest-priority workspace state glyph.
type WorkspaceStateResult struct {
    Status string `json:"status,omitempty"`
    Glyph  string `json:"glyph,omitempty"` // tmux-colored glyph or empty
}

// WorkspaceLookupEntry is a single changeId:status pair.
type WorkspaceLookupEntry struct {
    ChangeID string `json:"changeId"`
    Status   string `json:"status"`
}
```

### Constants

```go
// DefaultBranches are branch names considered "default" for trunk guard.
// Must match adv_status.sh:332 (main|master|trunk|develop).
var DefaultBranches = map[string]bool{
    "main":    true,
    "master":  true,
    "trunk":   true,
    "develop": true,
}

// ActionableWorktreeStatuses are worktree states that indicate active work.
// Terminal states (merged, stale, deleted) must NOT trigger branch safety warning.
// Must match adv_status.sh:317.
var ActionableWorktreeStatuses = map[string]bool{
    "active":        true,
    "idle":          true,
    "materializing": true,
    "setup_failed":  true,
    "pending_delete": true,
    "unmaterialized": true,
}
```

### Core Functions

```go
// FindActiveChanges scans ADV state dir for non-archived, non-closed changes.
// Returns change IDs sorted by directory modification time (most recent first).
func FindActiveChanges(advRoot string) ([]string, error)

// SummarizeChange reads change.json and returns a ChangeSummary.
// Returns nil (no error) if change.json missing or unreadable (graceful degradation).
func SummarizeChange(changeDir string) (*ChangeSummary, error)

// ActiveChangeSummary finds the first active change across all projects
// and returns its summary. Returns nil when none found.
func ActiveChangeSummary() (*ChangeSummary, error)

// TemporalHealthProbe reads temporal.env from the cache dir and probes the
// address via TCP. Uses render.CacheDir() for path resolution (matching
// adv_status.sh:211-212 which reads $OCA_CACHE_DIR/temporal.env).
// Returns empty TemporalHealth (not error) when temporal.env missing.
func TemporalHealthProbe() (*TemporalHealth, error)

// BranchSafety checks if a path is on a default branch while ADV has
// active worktree records. Returns colored tmux glyph when unsafe.
// Uses DefaultBranches map and ActionableWorktreeStatuses filter.
func BranchSafety(panePath, projectID string) (*BranchSafetyResult, error)

// WorkspaceState returns the highest-priority workspace state glyph for a project.
// Priority: setup_failed > stale > pending_delete > merged > (active/idle = empty).
func WorkspaceState(projectID string) (*WorkspaceStateResult, error)

// WorkspaceLookup returns all worktree registry entries as changeId:status pairs.
func WorkspaceLookup(projectID string) ([]WorkspaceLookupEntry, error)
```

### Command: `cmd/oca/adv_status.go`

Registered as `rootCmd.AddCommand(newAdvStatusCmd(state))` — top-level `oca adv-status` (not nested under `oca adv`).

Uses root's persistent `--output text|json` flag (inherited via `state.output`, matching `occupancy.go`, `adv.go`, `doctor.go`). No separate `--format` flag. Also accepts `--json` shorthand like `occupancy.go:70`.

```
oca adv-status --query <mode> [--output text|json] [--json] [--path <dir>] [--project <id>]
```

| Flag | Purpose | Required | Source |
|------|---------|----------|--------|
| `--query` | Query mode: active-change, temporal-health, branch-safety, worktrees, workspace-lookup | Yes | New |
| `--output` | Output format: text (default), json | No | Inherited from root (`root.go:84`) |
| `--json` | Shorthand for `--output json` | No | Local flag (like `occupancy.go:70`) |
| `--path` | Pane path (for branch-safety) | For branch-safety | Local flag |
| `--project` | Project ID (for branch-safety, worktrees, workspace-lookup) | For worktrees/workspace-lookup | Local flag |

**Text output**: query-specific formatted string (glyphs, TSV, summary). Empty string + exit 0 when no data.
**JSON output**: structured JSON envelope `{query, result}`.

### Data Flow

```
status_bar.sh (tmux #() every 10s)
  → oca adv-status --query active-change --output text
      → advstatus.ActiveChangeSummary()
          → os.ReadDir(advRoot/*/changes/) → find non-archived
          → os.ReadFile(change.json) → json.Unmarshal → find first pending gate

  → oca adv-status --query temporal-health --output text
      → advstatus.TemporalHealthProbe()
          → render.CacheDir() + "/temporal.env" → parse ADV_TEMPORAL_ADDRESS
          → net.DialTimeout("tcp", addr, 1s)

  → oca adv-status --query branch-safety --path <p> --project <id> --output text
      → advstatus.BranchSafety(path, projectID)
          → git rev-parse (exec.Command) → is DefaultBranch?
          → advruntime.ProjectWorkspaceStates(projectID) → has ActionableWorktreeStatuses?

  → oca adv-status --query worktrees --project <id> --output text
      → advstatus.WorkspaceState(projectID)
          → advruntime.ProjectWorkspaceStates(projectID) → find highest-priority status → map to glyph

  → oca adv-status --query workspace-lookup --project <id>
      → advstatus.WorkspaceLookup(projectID)
          → advruntime.ProjectWorkspaceStates(projectID) → extract changeId:status pairs
```

### status_bar.sh Changes

Replace:
```bash
# Before (sourced adv_status.sh)
adv_state=$(oca_adv_active_summary 2>/dev/null)
safety=$(oca_status_branch_safety "$pane_path" "$project_id" 2>/dev/null)
ws_state=$(oca_status_workspace_state "$project_id" 2>/dev/null)
temporal_health=$(oca_adv_temporal_health 2>/dev/null)

# After (calls Go binary, using --output text explicitly)
adv_state=$(oca adv-status --query active-change --output text 2>/dev/null)
safety=$(oca adv-status --query branch-safety --path "$pane_path" --project "$project_id" --output text 2>/dev/null)
ws_state=$(oca adv-status --query worktrees --project "$project_id" --output text 2>/dev/null)
temporal_health=$(oca adv-status --query temporal-health --output text 2>/dev/null)
```

For window glyph enrichment, replace the `_oca_adv_snapshot_read` + jq pipeline:
```bash
# Before
snapshot=$(_oca_adv_snapshot_read "$project_id" 2>/dev/null)
ws_lookup=$(printf '%s' "$snapshot" | jq -r '...' 2>/dev/null)

# After
ws_lookup=$(oca adv-status --query workspace-lookup --project "$project_id" 2>/dev/null)
# ws_lookup is now "changeId:status" lines — no jq needed for parsing
```

### Shell Test Changes

- **Delete** `tests/shell/adv_status_test.sh` entirely (all test logic moves to Go)
- **Update** `tests/shell/status_bar_test.sh`:
  - Remove adv_status sourcing (line 8-9)
  - Remove branch-safety tests (lines 287-597) — now covered by Go tests
  - Remove workspace-state tests (lines 383-568) — now covered by Go tests
  - Remove temporal-health tests (lines 79-100, 255-262) — now covered by Go tests
  - Keep window glyph tests — update to use `oca adv-status --query workspace-lookup` instead of `_oca_adv_snapshot_read`
- **Update** `tests/integration/integration_test.sh`:
  - Remove `lib/adv_status.sh` from sourceability check (line 12)
  - Remove `tests/shell/adv_status_test.sh` from test runner (line 58)

### Spec Delta

Update `oca-workspace-projection` spec:
- **rq-ocawp-statusBar01**: Add clause that the Go binary (`oca adv-status`) is the data source for tmux status bar. Shell functions are removed.

### File Inventory

| File | Action | LOC (est.) |
|------|--------|-----------|
| `internal/advstatus/status.go` | New | ~200 |
| `internal/advstatus/change.go` | New | ~100 |
| `internal/advstatus/temporal.go` | New | ~80 |
| `internal/advstatus/status_test.go` | New | ~300 |
| `internal/advstatus/testdata/*` | New | ~50 |
| `cmd/oca/adv_status.go` | New | ~150 |
| `cmd/oca/root.go` | Modify (1 line: AddCommand) | +1 |
| `lib/status_bar.sh` | Modify (replace 6 call sites, remove source) | -10, +10 |
| `lib/adv_status.sh` | Delete | -442 |
| `tests/shell/adv_status_test.sh` | Delete | -162 |
| `tests/shell/status_bar_test.sh` | Modify (remove adv_status tests, update window glyph tests) | -300, +30 |
| `tests/integration/integration_test.sh` | Modify (remove adv_status.sh from list, remove test runner line) | -2 |
| `lib/README.md` | Modify (remove adv_status.sh reference) | -1 |
| `assets/instructions/temp_directory.md` | Modify (update cache file table) | -2, +2 |
| `.adv/specs/oca-workspace-projection/spec.md` | Modify (add Go binary reference) | +5 |

**Net**: ~830 new LOC (Go + tests), ~900 deleted LOC (shell). Overall reduction with higher quality.

## Error Handling

| Scenario | Shell behavior | Go behavior |
|----------|---------------|-------------|
| ADV state dir missing | Empty output, exit 0 | Empty output, exit 0 |
| `change.json` malformed | `jq` fails silently → empty | `json.Unmarshal` error → log + empty |
| `temporal.env` missing | Empty output | Empty output |
| Temporal unreachable | `T:✗` (TCP timeout) | `T:✗` (`net.DialTimeout` 1s) |
| Not on default branch | Empty (no warning) | Empty (no warning) |
| `jq` not installed | Degraded output | N/A (no jq dependency) |
| Go binary not in PATH | N/A | Shell `2>/dev/null` catches → empty |

## Performance Budget

| Query | Shell (current) | Go (target) |
|-------|-----------------|-------------|
| active-change | ~20ms (jq + file reads) | <10ms (native JSON) |
| temporal-health | ~5ms (TCP probe, cached) | ~5ms (TCP probe, no cache) |
| branch-safety | ~15ms (jq + git) | <10ms (advruntime + git) |
| worktrees | ~10ms (jq) | <5ms (advruntime) |
| workspace-lookup | ~10ms (jq) | <5ms (advruntime) |
| **Total per refresh** | ~60ms | ~25ms |

No persistent cache needed — each Go invocation is a fresh process that reads files on local SSD (<5ms per read).

## Key Design Decisions

1. **Top-level command (`oca adv-status`)** not nested under `oca adv` — called from tmux status bar where brevity matters, and semantically distinct from `oca adv recover` (operator tooling).

2. **Use inherited `--output` + local `--json`** — matches existing cobra pattern (`root.go:84` persistent `--output`, `occupancy.go:70` `--json` shorthand). No separate `--format` flag.

3. **Separate `internal/advstatus` package** — distinct concern from `advruntime` (which is Temporal health models, recovery planning, session debt). `advstatus` is about surfacing ADV state for the status bar.

4. **Reuse `advruntime.ProjectWorkspaceStates()`** for all snapshot.json reads — avoids duplicating the typed parser. `advstatus` adds change.json reading and temporal.env reading on top.

5. **Reuse `render.CacheDir()`** for temporal.env path resolution — same 4-tier fallback chain as the shell's `$OCA_CACHE_DIR/temporal.env`.

6. **No persistent cache** — Go binary startup + file reads total <10ms. tmux refresh is every 10s. Caching adds complexity for no measurable benefit.

7. **`--output text` returns pre-formatted tmux glyphs** — including `#[fg=#E5A649]` escapes. This matches the current shell behavior exactly and avoids shell-side formatting logic.

8. **`--query workspace-lookup` returns TSV** — `changeId:status\n` lines. Shell parses with `grep`/`cut` (no jq needed). Direct replacement for the `_oca_adv_snapshot_read` + jq pipeline in window glyph enrichment.

9. **Explicit `DefaultBranches` and `ActionableWorktreeStatuses` constants** — enumerated in code, not heuristic. Must match shell filter sets exactly (main|master|trunk|develop and active|idle|materializing|setup_failed|pending_delete|unmaterialized).

## Validator Resolution Log

| # | Finding | Resolution |
|---|---------|-----------|
| 1 | Use `--output` not `--format` | ✓ Fixed: inherit root `--output` + local `--json` shorthand |
| 2 | Specify temporal.env path resolution | ✓ Fixed: use `render.CacheDir()` + "/temporal.env" |
| 3 | Enumerate actionable worktree statuses | ✓ Fixed: `ActionableWorktreeStatuses` constant map |
| 4 | Enumerate default branch names | ✓ Fixed: `DefaultBranches` constant map |