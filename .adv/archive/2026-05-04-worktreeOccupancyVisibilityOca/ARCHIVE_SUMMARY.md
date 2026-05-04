# Archive: worktree occupancy visibility for OCA and ADV sessions

**Change ID:** worktreeOccupancyVisibilityOca
**Archived:** 2026-05-04T13:09:05.518Z
**Created:** 2026-05-04T07:21:35.699Z

## Tasks Completed

- ✅ Go occupancy model + parser (internal/occupancy/): define PaneStateV1/V2 structs, Parse() for permissive v1/v2 JSON parsing, GroupBy() to group by project/worktree, liveness classification (active/stale/unknown), reconciliation logic with tmux pane cross-check. No external dependencies beyond stdlib.
- ✅ Go occupancy tests (internal/occupancy/): unit tests for v1/v2 parse (valid, partial, malformed), grouping by project/worktree, stale detection (pane gone vs active), malformed record counting, reconciliation precedence (tmux > pane state > git), edge cases (empty dir, mixed v1/v2). TDD red phase first.
- ✅ TypeScript plugin v2 state enrichment (plugins/oca/src/): extend session.created handler in index.ts to write v2 pane-state with schemaVersion:2, startedAt, lastSeenAt, socket, paneID, sessionName, windowID, windowName, agent (best-effort from env), gitRoot, gitCommonDir, projectId, worktreePath, worktreeBranch (best-effort). Preserve backwards compat: write v2 JSON but v1 fields remain. Update session.deleted to handle v2 cleanup.
- ✅ TypeScript plugin v2 tests (plugins/oca/src/): test v2 write produces correct schema, partial fields when env vars missing, v1 backwards compat on read, session.deleted cleans up v2, atomic write behavior. Add/update tests alongside existing watchdog.test.ts.
- ✅ CLI `oca occupancy` command (cmd/oca/occupancy.go): implement occupancy subcommand with --all, --json, --socket, --path, --status, --pane flags. Human output groups active occupants by project/worktree with warnings for multiple occupants. JSON output exposes full local operator data. --status --pane is narrow single-pane compact output (e.g. "1× trunk", "2× occupied ⚠", "?"). Uses internal/occupancy package for parsing/grouping/reconciliation.
- ✅ CLI occupancy golden/output tests (cmd/oca/occupancy_test.go): golden file tests for human output grouping, JSON output schema, --status --pane compact output, --all shows stale records, --path filtering, malformed record counting in warnings. Test with mock pane state dirs.
- ✅ Status bar integration (lib/status_bar.sh): update to pass pane id and call `timeout 0.5s oca occupancy --status --pane <pane-id>` for compact occupancy segment. Suppress stderr. Render empty segment on failure/timeout. Preserve existing branch/ADV/Temporal output.
- ✅ Status bar shell test (tests/shell/): test timeout/fail-closed behavior for `oca occupancy --status --pane`. Verify empty output on missing state file, correct compact output on valid state, timeout kills slow invocation. Uses bash test framework.
- ✅ Advance `lastSeenAt` + occupancy marker (cross-repo: /home/jrede/dev/oc-plugins/advance): add `lastSeenAt` field to SessionListEntry in plugin/src/tools/session/index.ts. Add worktree occupancy marker: when >1 session shares a worktree, emit `[ADV:WORKTREE_OCCUPANCY] N sessions share this worktree. Nominal 1:1 violated; continuing allowed.` Count-only, no peer PID/path/branch. Graceful degradation when Temporal unavailable.
- ✅ Cross-surface integration verification: run `go test ./...` for OCA, build plugin (`cd plugins/oca && bun build`), run Advance scoped tests for session/occupancy changes. Verify no regressions. Report pass/fail.

## Specs Modified

