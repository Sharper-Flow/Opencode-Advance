## Problem

`lib/adv_status.sh` (442 LOC, 15 shell functions) reads ADV internal state files (`change.json`, `snapshot.json`, `temporal.env`) with `jq` to produce status bar output. This is fragile coupling to ADV's internal schema — if ADV's state format changes (e.g., signal cutover from file-backed to Temporal-native), the shell parsing breaks silently.

The `internal/advruntime` Go package already provides typed, testable readers for these same state files (`ProjectWorkspaceStates`, `DefaultADVStateRoot`, `paths.go`). The shell script duplicates this logic without type safety, testability, or graceful schema evolution.

Three callers depend on `adv_status.sh`:
1. `lib/status_bar.sh` — sources it directly, calls 5 functions for tmux status rendering
2. tmux `status-interval` — refreshes every 10s via shell invocation
3. CLI `adv_status.sh summary|find|parse` — direct script execution

The shell script's caching layer (10s TTL file-based) is also duplicated — the Go binary can compute on-demand fast enough for status bar refresh, or use an in-process cache.