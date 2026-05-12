# Agreement

## Objectives

1. Replace OCA's session-attach process-replacement pattern with launch-and-wait, enabling post-exit code execution
2. After clean session exit, emit a resume hint to the outer terminal and a known file path
3. Suppress hint on intentional kill (via sentinel) and when session survives (tmux detach)
4. Maintain full terminal semantics (raw mode, SIGWINCH, signals) during attach
5. Update the codified spec (`rq-p4c-session-lifecycle01`) to reflect new behavior

## Acceptance Criteria

**AC1:** `oca session attach <name>` spawns `tmux attach` via `exec.Command` with `cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr` and `cmd.Run()`. After tmux exits, Go code runs the hint-emit logic.

**AC2:** On clean exit (tmux returns 0, session no longer exists on socket, no kill sentinel present for the session name):
- Print to stdout: `💤 Resume: opencode --session ses_xxx  (project: /path/to/dir)`
- Write to file: `$OCA_CACHE_DIR/last-session-hint` containing the same line (atomic write, 0600 perms)
- Honor `NO_COLOR`: when set, output is plain ASCII without emoji

**AC3:** No hint emitted when:
- Kill sentinel exists at `$OCA_CACHE_DIR/kill-sentinel/<session-name>` (written by `oca session kill` before tmux kill-session)
- Session still exists on the socket after tmux returns (user detached via Ctrl-B d)
- `opencode` binary not found on PATH (silent skip, attach succeeds)
- stdout is not a TTY (defensive guard — unreachable in normal attach flow)

**AC4:** `oca session kill <name>` writes sentinel file to `$OCA_CACHE_DIR/kill-sentinel/<session-name>` before issuing `tmux kill-session`. Sentinel is consumed (deleted) by the hint-emit path if present.

**AC5:** `lib/session_lifecycle.sh::oca_session_attach` delegates to `oca session attach <name>` (Go binary). The bash function becomes a thin wrapper.

**AC6:** Signal forwarding: Ctrl-C (SIGINT), SIGTERM, and SIGWINCH are forwarded to the tmux subprocess during attach. Ctrl-Z (SIGTSTP) behavior matches current `syscall.Exec` semantics. Keybinding latency and resize handling must not regress.

**AC7:** Unit tests cover:
- Exit classifier (clean exit vs. session-still-alive vs. sentinel-present)
- Session-liveness checker (mocked tmux list-sessions)
- Hint formatter (full command format, NO_COLOR handling)
- Sentinel write/read/cleanup (file operations)
- Session-list parser (opencode JSON parsing with defensive handling)

**AC8:** Integration tests cover:
- Attach → clean exit → hint emitted (mocked opencode session list)
- Attach → detach (session alive) → no hint
- Kill → sentinel written → subsequent attach-exit → hint suppressed
- `opencode` not on PATH → no hint, no error

**AC9:** Spec `rq-p4c-session-lifecycle01` updated to describe launch-and-wait + post-exit hint emit behavior. G/W/T scenarios cover: clean exit with hint, intentional kill suppression, detach suppression.

**AC10:** `go test ./...` passes. `go vet ./...` and `gofmt -d .` clean.

## Constraints

- C1: Go stdlib only — no PTY libraries, no external dependencies
- C2: `opencode session list --format json` schema consumed defensively — only `id`, `directory`, `updated` fields used; unknown/missing fields → no hint (never crash)
- C3: File-based sentinel under `$OCA_CACHE_DIR` — atomic write via temp file + rename, 0600 perms, cleanup-on-read
- C4: Hint file (`$OCA_CACHE_DIR/last-session-hint`) uses same atomic write pattern
- C5: Bash wrapper must remain callable with same interface (`oca_session_attach <name>`)
- C6: No changes to OpenCode core or Advance plugin
- C7: `subprocess.Run` cannot be used for tmux attach (piped I/O incompatible with terminal control)

## Avoidances

- DONT1: Do NOT use `subprocess.Run` for the attach subprocess (pipes break terminal control)
- DONT2: Do NOT add a PTY library dependency (tmux handles terminal setup)
- DONT3: Do NOT change how OpenCode generates its own resume hint
- DONT4: Do NOT emit hint for `oca session kill` path (intentional = no hint)
- DONT5: Do NOT create new tmux hooks, plugins, or `wait-for` channels

## Decisions

### User Decisions
- UD1: Hint format is full command — `opencode --session ses_xxx  (project: /path/to/dir)` — one-step copy-paste
- UD2: Hint written to both stdout and `$OCA_CACHE_DIR/last-session-hint` for tool/script consumption
- UD3: Non-TTY = silent skip (defensive guard; unreachable for `session attach` which requires TTY)

### Agent Decisions (LBP)
- AD1: Session-liveness check before emit — verify session gone from socket + no sentinel present (DQ1 option c)
- AD2: File-based sentinel under `$OCA_CACHE_DIR/kill-sentinel/` (DQ2 option a)
- AD3: Bash delegates to Go binary for attach flow (DQ3 option a)
- AD4: `exec.Command` + fd passthrough chosen over `syscall.ForkExec` — stdlib pattern, no PTY needed, tmux handles terminal setup

## Deferred Questions

None.

## Sign-Off

Awaiting user approval of acceptance criteria.