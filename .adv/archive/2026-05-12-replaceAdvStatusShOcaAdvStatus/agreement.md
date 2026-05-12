# Agreement

## Objectives

1. Replace `lib/adv_status.sh` (442 LOC shell + jq) with `oca adv-status` Go subcommand using typed readers
2. Delete `adv_status.sh` entirely — no backward compat wrapper
3. Update all callers (`status_bar.sh`, shell tests) to use Go binary
4. Separate invocations per query (~25ms total, no jq dependency in shell)
5. Include all 6 data queries: active-change, temporal-health, branch-safety, worktrees, workspace-lookup, plus JSON format

## Acceptance Criteria

1. `oca adv-status --query active-change --format text` returns `{shortId}:{currentGate}` for first active change, empty when none found
2. `oca adv-status --query temporal-health --format text` returns `T:✓` / `T:✗` via TCP probe, empty when `temporal.env` missing
3. `oca adv-status --query branch-safety <path> <project-id> --format text` returns `#[fg=#E5A649]⚡#[default]` when on default branch with active changes, empty otherwise
4. `oca adv-status --query worktrees <project-id> --format text` returns highest-priority workspace state glyph (✗/ѻ/✓/␡) with tmux escapes, empty when none
5. `oca adv-status --query workspace-lookup <project-id>` returns `changeId:status` TSV for all worktree registry entries
6. `oca adv-status --format json` returns structured JSON for any query mode
7. All invocations complete in ≤50ms on local SSD
8. Returns empty output + exit 0 when ADV state dir doesn't exist (graceful degradation)
9. `lib/adv_status.sh` deleted entirely
10. `lib/status_bar.sh` updated to call `oca adv-status` for all 6 data queries
11. `tests/shell/adv_status_test.sh` deleted; Go unit tests cover all query modes
12. `tests/shell/status_bar_test.sh` updated to remove adv_status-specific tests (branch safety, workspace state, temporal health)
13. All existing `go test ./...` pass; new `internal/advstatus` tests pass
14. Spec `oca-workspace-projection` rq-ocawp-statusBar01 updated to reference Go binary as data source

## Constraints

- No Temporal SDK dependency — TCP probe only
- No `jq` dependency in `status_bar.sh` after migration
- Must handle IPv6 bracketed addresses in `temporal.env`
- Must scan all ADV projects when no project-id given (active-change query)
- Terminal worktree states (merged, stale, deleted) must NOT trigger branch safety warning

## Avoidances

- No wrapper script — full delete of `adv_status.sh`
- No single combined query — separate invocations per data need
- No jq in `status_bar.sh` — Go binary returns pre-formatted text
- No window glyph rendering in Go — shell keeps glyph assembly, just data source changes

## Decisions

### User Decisions

1. **Delete adv_status.sh entirely** — no backward compat wrapper. All callers are in-repo.
2. **Separate invocations** — 4-5 calls to `oca adv-status --query X` per tmux refresh. ~25ms total, clean API.
3. **Include workspace-lookup now** — full replacement of all 6 call sites in one change. No feature gap.

### Agent Decisions (LBP)

1. **`--query workspace-lookup` returns TSV** — `changeId:status` lines, parseable in shell without jq. Minimal replacement for `_oca_adv_snapshot_read` + jq pipeline.
2. **No persistent cache in Go** — file reads on local SSD are <5ms. tmux refresh interval is 10s. No cache needed.
3. **Reuse `advruntime.ProjectWorkspaceStates()`** for snapshot.json reading — already typed and tested.
4. **New `internal/advstatus` package** for change.json + temporal.env reading — separate from `advruntime` (which focuses on Temporal health models).

## Deferred Questions

(none)

## Sign-Off

User approved AC via inline whitelist match.