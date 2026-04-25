# Capability: phase4-completion

Phase 4 completion spec covering session lifecycle commands, OCA_REPO_ROOT injection, status bar wiring, theme management, and integration test coverage. Extends `phase4-foundation` (which covers session create/list, shell helpers, boot splash, and isolation).

## Requirements

- `rq-p4c-session-lifecycle01` — `oca session attach <name>` attaches to an existing OCA tmux session via `syscall.Exec`, replacing the current process. Requires exactly one argument. Uses the session manager's socket.
- `rq-p4c-session-switch01` — `oca session switch <name>` switches the current tmux client to a different OCA-managed session via `tmux switch-client`. Requires exactly one argument.
- `rq-p4c-session-kill01` — `oca session kill <name>` destroys a specific OCA tmux session. Requires exactly one argument. Outputs JSON when `--output json` is set.
- `rq-p4c-session-killall01` — `oca session killall` destroys all OCA-managed sessions on the current socket. Reports count killed. Outputs JSON when `--output json` is set.
- `rq-p4c-session-restart01` — `oca session restart <name>` kills and recreates a session with the same name and working directory. Polls to verify the old session is dead before recreating (100ms interval, 2s timeout). Re-injects OCA_REPO_ROOT after recreation.
- `rq-p4c-session-reap01` — `oca session reap [--dry-run] [--age <duration>]` kills OCA-managed sessions with no recent activity. Enforces a 5-minute minimum age. Skips attached sessions. Dry-run shows what would be reaped without killing. Outputs JSON when `--output json` is set.
- `rq-p4c-repo-root-injection01` — `oca session new` and `oca session restart` inject `OCA_REPO_ROOT` into the tmux global environment via `Manager.SetGlobalEnv`. The value is resolved by walking up from cwd to find `lib/boot_splash.sh`. Failure to set is non-fatal (warning to stderr) — status bar degrades gracefully when unset.
- `rq-p4c-set-global-env01` — `Manager.SetGlobalEnv(ctx, key, value)` uses `tmux setenv -g` via the subprocess runner. It is a separate method from `Create()` to keep concerns separated and allow reuse.
- `rq-p4c-status-bar-wiring01` — `assets/themes/obsidian.tmux.conf` invokes `lib/status_bar.sh` via tmux `#()` format expansion using `$OCA_REPO_ROOT/lib/status_bar.sh`. Row 0 shows session name, git branch, ADV state, host, and clock. Row 1 shows window list and LLM gauges.
- `rq-p4c-theme-list01` — `oca theme list` scans `assets/themes/` for `.json` files and reports name + description. Outputs JSON when `--output json` is set.
- `rq-p4c-theme-apply01` — `oca theme apply <name>` loads a theme JSON file, resolves color references from `defs` object, and outputs the resolved palette. The theme struct uses `Name`, `Description`, and `Colors` fields.
- `rq-p4c-integration-tests01` — CLI integration tests exist in `tests/session_integration_test.go` for: session new (auto-name, custom-name, JSON, no-writes-to-live-config), session list (text, JSON, empty), session kill, session killall, session restart, session reap (dry-run), and OCA_REPO_ROOT injection verification. All use real tmux with isolated sockets.
