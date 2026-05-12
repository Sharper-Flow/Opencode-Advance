# Archive: Replace adv_status.sh with oca adv-status Go subcommand (GH #24)

**Change ID:** replaceAdvStatusShOcaAdvStatus
**Archived:** 2026-05-12T21:23:42.208Z
**Created:** 2026-05-12T19:03:08.006Z

## Tasks Completed

- ✅ Create `internal/advstatus/` package with types, constants, and core reader functions
  > Created internal/advstatus/ package: status.go (types, constants, formatters), change.go (FindActiveChanges, SummarizeChange, ActiveChangeSummary), temporal.go (TemporalHealthProbe, parseTemporalAddress, probeTCP). 22 tests passing with 4 golden files.
- ✅ Create `cmd/oca/adv_status.go` cobra command with --query, --path, --project flags
  > Created cmd/oca/adv_status.go with newAdvStatusCmd — 5 query modes, inherited --output + local --json, registered in root.go. Added internal/advstatus/queries.go with BranchSafety, WorkspaceState, WorkspaceLookup using advruntime.ProjectWorkspaceStates().
- ✅ Update `lib/status_bar.sh` to call `oca adv-status` instead of shell functions
  > Updated lib/status_bar.sh: removed source adv_status.sh, replaced 4 function calls with oca adv-status --query invocations, replaced _oca_adv_snapshot_read + jq pipeline with workspace-lookup query.
- ✅ Delete `lib/adv_status.sh` and update shell tests
  > Task completed
- ✅ Update spec `oca-workspace-projection` to reference Go binary as data source
  > Task completed
- ✅ Full integration verification: go tests, shell tests, binary behavior
  > Task checkpoint completed

## Specs Modified

