# Design

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│ cmd/oca/session.go::newSessionAttachCmd                  │
│                                                          │
│  1. Create Manager                                       │
│  2. mgr.AttachAndWait(ctx, name, sessionWorkdir)         │
│     ├── spawns tmux attach (exec.Command, fd passthrough)│
│     ├── signal forwarding loop (SIGINT/SIGTERM/SIGWINCH) │
│     └── returns (exit code, session-gone bool)           │
│  3. If clean exit && session gone:                       │
│     ├── checkKillSentinel(name) — skip if present        │
│     ├── queryOpenCodeSessions(workdir)                   │
│     ├── formatHint(sessionID, workdir)                   │
│     ├── emitToStdout(hint)                               │
│     └── emitToFile(hint, cacheDir)                       │
│  4. cleanupKillSentinel(name)                            │
└─────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────┐
│ cmd/oca/session.go::newSessionKillCmd         │
│                                               │
│  1. Create Manager                            │
│  2. writeKillSentinel(name, cacheDir)         │
│  3. mgr.Kill(ctx, name)                       │
└──────────────────────────────────────────────┘
```

New Go module: `internal/session/hint.go` — contains:
- `AttachAndWait(ctx, mgr, name, workdir, cacheDir)` — replaces `Manager.Attach`
- `checkKillSentinel(name, cacheDir) bool`
- `writeKillSentinel(name, cacheDir) error`
- `cleanupKillSentinel(name, cacheDir)`
- `queryOpenCodeSessions(workdir) (sessionID, error)`
- `formatHint(sessionID, workdir) string`
- `emitHint(hint, cacheDir) error` — stdout + file

## Key Decisions

### KD1: exec.Command with fd passthrough, not syscall.Exec

**Rationale:** `syscall.Exec` replaces the process — no Go code runs after tmux exits. `exec.Command` with `cmd.Stdin = os.Stdin; cmd.Stdout = os.Stdout; cmd.Stderr = os.Stderr` passes through the parent's file descriptors directly. tmux handles its own terminal setup (raw mode, alternate screen buffer) over these fds. No PTY library needed.

**Why not subprocess.Run:** `subprocess.Run` captures stdout/stderr via `bytes.Buffer` pipes — tmux cannot operate over pipes (needs a real terminal for raw mode and SIGWINCH).

**Why not syscall.ForkExec + Wait4:** Lower-level, no practical benefit over `exec.Command` for this use case. `exec.Command` provides the same fd passthrough with cleaner API and built-in context cancellation.

**Why not a PTY library (creack/pty, etc.):** Unnecessary dependency. tmux creates its own PTY for the inner shell. The outer process just needs to pass through its own terminal fds.

**TMUX env var handling:** Clear `TMUX` from the child environment to prevent tmux nesting issues when attaching from inside an existing tmux session. Pattern proven in `wuphf` and other Go tmux wrappers:
```go
cmd.Env = filterTMUX(os.Environ())
```

### KD2: File-based kill sentinel — required for intentional vs. natural death distinction

**Rationale:** The sentinel distinguishes two cases where the session is destroyed:
- **Natural opencode exit** (`/exit`) → session destroyed → **SHOULD emit hint** (user wants to resume later)
- **Intentional `oca session kill`** → session destroyed → **SHOULD NOT emit hint** (user explicitly discarded)

Both cases result in session-gone + exit-0, but the user intent differs. The sentinel encodes the "I chose to kill this" signal.

tmux session variables (`tmux setenv -t <name>`) are destroyed with the session — but we need the sentinel to survive past session destruction. A file under `$OCA_CACHE_DIR/kill-sentinel/` survives tmux lifecycle.

**Location:** `$OCA_CACHE_DIR/kill-sentinel/<session-name>` — consistent with OCA cache patterns, keyed by session name.

**Lifecycle:** Written by `oca session kill` before `tmux kill-session`. Consumed (read + deleted) by `AttachAndWait` post-exit. External kills without `oca session kill` are treated as natural deaths — hint may be emitted (harmless, user can ignore).

### KD3: Session-liveness check via tmux has-session

**Rationale:** After tmux attach returns exit 0, distinguish "session destroyed by opencode exit" from "user detached via Ctrl-B d". Check if the session still exists on the socket via `tmux has-session -t <name>`. If it exists → user detached → no hint. If it doesn't exist → session was destroyed → check sentinel → emit hint.

**Implementation:** Use `tmux has-session` (lighter than `list-sessions`).

### KD4: opencode session query via subprocess

**Rationale:** `opencode session list --format json` is the documented query surface. Parse into a minimal struct with only `id`, `directory`, `updated`. Filter by `directory == workdir`, sort by `updated` desc, take first.

**Error handling:** If `opencode` not on PATH → silent skip. If JSON parse fails → silent skip. If no matching session → silent skip. Never fail the attach flow.

### KD5: Bash wrapper delegates to Go binary

**Rationale:** `lib/session_lifecycle.sh::oca_session_attach` currently calls `exec tmux -L "$socket" attach -t "$name"`. Replace with `oca session attach "$name"` — the Go binary now owns the full attach + hint flow. Single implementation to maintain. Bash callers already have the Go binary on PATH (OCA is installed).

## Implementation Strategy

### Sequencing

1. **Hint module** (`internal/session/hint.go`) — pure functions with no tmux dependency: sentinel ops, opencode query, hint formatter, file emit. Fully unit-testable with mocked `exec.LookPath` and `exec.Command`.
2. **AttachAndWait** (`internal/session/session.go`) — refactor `Manager.Attach` from `syscall.Exec` to `exec.Command` + fd passthrough + signal forwarding + `cmd.Run()`. New method `AttachAndWait` to avoid breaking existing `Attach` callers during migration.
3. **CLI integration** (`cmd/oca/session.go`) — update `newSessionAttachCmd` to call `AttachAndWait` + hint emit. Update `newSessionKillCmd` to write sentinel.
4. **Bash wrapper** (`lib/session_lifecycle.sh`) — `oca_session_attach` calls `oca session attach "$name"`.
5. **Tests** — unit tests for hint module, integration tests for attach-and-emit flow.
6. **Spec update** — modify `rq-p4c-session-lifecycle01`.

### Signal Forwarding Strategy

During `cmd.Wait()` for tmux attach:

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH, syscall.SIGTSTP)
go func() {
    for sig := range sigCh {
        cmd.Process.Signal(sig)
    }
}()
err := cmd.Wait()
signal.Stop(sigCh)
```

Note: SIGWINCH is technically delivered to the foreground process group automatically by the kernel, but explicit forwarding is a harmless safety net.

### fd Passthrough Setup

```go
cmd := exec.Command(tmuxPath, "-L", socket, "attach", "-t", name)
cmd.Stdin = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Env = filterTMUX(os.Environ())
// No SysProcAttr needed — tmux handles terminal setup
```

### Hint Format

**TTY output:**
```
💤 Resume: opencode --session ses_xxx  (project: /path/to/dir)
```

**NO_COLOR set:**
```
Resume: opencode --session ses_xxx  (project: /path/to/dir)
```

**File output** (`$OCA_CACHE_DIR/last-session-hint`):
```
opencode --session ses_xxx --project /path/to/dir
```
File format uses `--project` flag for explicit cwd independence.

## LBP Analysis

**Pattern:** `exec.Command` with fd passthrough for terminal subprocess + post-exit hook.

**LBP status:** ✓ Standard Go pattern. Proven in production:
- `wuphf` (MIT): `cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr` + `cmd.Run()` for tmux attach-session
- `ccc` (MIT): same pattern
- `daytona` (AGPL-3.0): signal forwarding for SSH subprocess

No alternatives are simpler or more idiomatic:
- `syscall.Exec` → no post-exit code (current problem)
- `os/exec` with PTY → unnecessary dependency, tmux creates its own PTY
- `tmux wait-for` channel → requires tmux-specific hook setup, fragile
- `tmux capture-pane` → fragile parsing, may capture noise

## Affected Components

| Component | Change type | Risk |
|-----------|-------------|------|
| `internal/session/session.go` | Major refactor (Attach → AttachAndWait) | Medium — signal forwarding must match current behavior |
| `internal/session/hint.go` (new) | New file | Low — pure functions, fully testable |
| `cmd/oca/session.go` | Moderate (attach + kill paths) | Low — existing error handling patterns |
| `lib/session_lifecycle.sh` | Minor (exec → oca call) | Low — single line change |
| `.adv/specs/phase4-completion/spec.md` | Spec delta | None — documentation only |
| `internal/session/session_test.go` | New tests | None |
| `cmd/oca/session_test.go` | New tests | None |
| `internal/session/hint_test.go` (new) | New file | None |

## Risks / Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Signal forwarding regression (keybinding latency, resize) | High | Explicit signal-forwarding test with tmux subprocess; compare before/after latency |
| tmux exit code unreliable for kill detection | Medium | Sentinel file is primary signal; exit code is secondary; session-liveness check is tertiary |
| `opencode session list` schema change | Low | Defensive parse (only 3 fields); silent skip on error |
| Race: concurrent kill-while-attached | Low | Sentinel written before kill; read + delete after attach exits; conservative skip on ambiguity |
| Bash callers that source session_lifecycle.sh directly | Low | Wrapper function signature unchanged; internal implementation change only |
| TTY fd passthrough on non-Linux (macOS) | Low | `os.Stdin/Stdout/Stderr` are platform-agnostic; tmux handles platform-specific terminal setup |
| Nested tmux attach (TMUX env var) | Low | Clear TMUX env var from child process per proven pattern |

## Validator Result

Validator verdict: **CAUTION** — architecturally sound, two actionable items resolved inline:
1. Sentinel justification clarified (KD2 now explicitly states natural vs. intentional death distinction)
2. TMUX env var filtering added to fd passthrough setup (KD1)

Sources: `wuphf` (MIT, tmux attach pattern), `ccc` (MIT, tmux attach), `daytona` (signal forwarding), Go os/exec docs.