# tmux Server Process Safety

## Incident

**Date:** 2026-04-28
**Impact:** All ~20 active opencode sessions killed simultaneously, requiring manual restart.

An AI agent ran `kill 3140` to clean up a "stale" tmux session. PID 3140 was the tmux **server process** (started by `tmux new-session -d` three days prior). Killing it destroyed all sessions managed by that server — including active work.

**Root cause:** The tmux server PID is visually indistinguishable from a regular child process in `ps aux` output. Nothing in the process list marks it as "tmux server — do not signal directly."

## Architecture

```
tmux new-session -d -s "oca-repo-0"
        │
        ▼
   tmux server PID (long-lived)
        │
        ├── session: oca-repo-0 (pane: opencode)
        ├── session: oca-repo-1 (pane: opencode)
        └── session: oca-repo-2 (pane: opencode)
```

The first `tmux new-session -d` on a socket creates the server process. That process persists as long as any session exists. Killing the server kills **all** sessions on that socket — there is no per-session isolation at the OS signal level.

### opencodeadvance vs open-chad

| Aspect | open-chad | opencodeadvance |
|--------|-----------|-----------------|
| Socket | Default (shared with user tmux) | Named (`-L oca`, isolated) |
| Server PID visible in `ps` | Yes — looks like a regular `tmux new-session` command | Yes — same |
| Socket isolation | **No** — killing server kills ALL user tmux sessions | **Yes** — killing server kills only OCA sessions |

opencodeadvance's named socket (`-L oca`) limits blast radius to OCA-managed sessions only. open-chad uses the default socket, so killing its server kills all tmux sessions including non-open-chad ones.

## Recommended Fixes

### F1 — `remain-on-exit off` on session creation (HIGH)

**Status:** open-chad implements this. opencodeadvance does not.

When opencode crashes or is killed, the tmux pane lingers in a "dead" state with `remain-on-exit on` (tmux default). This wastes memory and creates confusing `ps` output that agents may try to clean up.

**Implementation:** After `tmux new-session`, set:

```bash
tmux -L <socket> set-option -t <session> remain-on-exit off
```

In Go (`session.go Create`):

```go
// After successful new-session:
args := []string{"-L", m.socket, "set-option", "-t", name, "remain-on-exit", "off"}
subprocess.Run(ctx, subprocess.Cmd{Name: m.tmuxPath, Args: args, Timeout: sessionTimeout})
```

**Effect:** When opencode exits (crash, kill, graceful shutdown), the pane closes, the window closes, and the session auto-destroys. No orphan sessions accumulate.

### F2 — Auto-reap stale sessions on launch (HIGH)

**Status:** open-chad implements this (inline on every launch). opencodeadvance has `oca session reap` but does not auto-run it.

Orphaned sessions (from crashed opencode, SSH disconnects, etc.) accumulate if `remain-on-exit` is on or if the session was detached before the process died.

**Implementation:** In the `oca` CLI entry point (or `oca session new`), run reaping as a fire-and-forget background step:

```go
// In main or session.New handler:
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    m.Reap(ctx, reapThreshold) // kill unattached sessions older than threshold
}()
```

**Configuration:** Default threshold 4 hours (matches open-chad). Configurable via `stack.toml [session] reaper_age`.

**Open-chad implementation** for reference (`bin/openchad:128-146`):
```bash
_OC_STALE_AGE=${OC_STALE_AGE:-14400}  # 4 hours
tmux list-sessions -F '#{session_name} #{session_attached}' 2>/dev/null \
| grep '^oc-' \
| while read _sname _attached; do
    [ "$_attached" != "0" ] && continue
    # ... age check via embedded epoch ...
    tmux kill-session -t "$_sname"
done
```

opencodeadvance's reaper should use `#{session_activity}` (tmux native) rather than parsing session names — already implemented in `oca session reap`, just needs to be called automatically.

### F3 — Agent guard documentation (MEDIUM)

**Status:** Neither project documents this for AI agents.

AI agents (opencode, Claude, Copilot) that run shell commands can and will `kill` PIDs they identify as "stale." The tmux server PID is especially vulnerable because:

1. It shows up in `ps aux` as `tmux new-session -d -s ...` (looks like a stale command)
2. It has high elapsed time (days/weeks)
3. Nothing marks it as "critical infrastructure"

**Implementation:** Add to project AGENTS.md or agent instructions:

```markdown
## tmux Process Safety

- NEVER `kill` a tmux PID directly. Use `tmux kill-session -t <name>`.
- The tmux server process is the parent of all sessions on a socket.
  Killing it destroys ALL sessions — including active work.
- To identify the server: `tmux -L <socket> list-sessions` — if sessions
  exist, the server is running and must not be killed.
- To clean up stale sessions: `oca session reap --age=4h` (opencodeadvance)
  or `oc-killall` (open-chad). These use per-session `kill-session`.
```

### F4 — Process tree validation before kill (LOW)

**Status:** Neither project implements this.

A defensive check before any `kill` of a tmux-related PID:

```bash
# Before killing any PID, check if it's a tmux server
if pstree -p "$pid" 2>/dev/null | grep -q tmux; then
    echo "WARN: PID $pid has tmux children. Refusing to kill."
    echo "Use 'tmux kill-session -t <name>' instead."
    exit 1
fi
```

This could be wrapped into a `bin/oc-kill` helper that agents are instructed to use instead of raw `kill`.

## Comparison: Current State

| Protection | open-chad | opencodeadvance |
|------------|-----------|-----------------|
| `remain-on-exit off` | ✓ (set on every session) | ✗ (not set) |
| Auto-reap on launch | ✓ (inline, 4h threshold) | ✗ (manual `oca session reap`) |
| Named socket isolation | ✗ (default socket, shared) | ✓ (`-L oca`, isolated) |
| `oc-killall` / `oca killall` | ✓ (per-session kill) | ✓ (per-session kill) |
| Agent guard docs | ✗ | ✗ |
| Process tree validation | ✗ | ✗ |

## Priority

1. **F1** (remain-on-exit) — prevents orphan accumulation, small change
2. **F2** (auto-reap) — cleans up existing orphans, leverages existing reaper code
3. **F3** (agent guard docs) — prevents the class of mistake that caused the incident
4. **F4** (process tree validation) — defense in depth, lowest urgency

## Related

- `docs/design/tmux-session-reconnaissance.md` — session inspection commands
- `internal/session/session.go` — Go session manager
- `cmd/oca/session.go` — CLI session commands including `reap`
- open-chad `bin/openchad:128-146` — inline stale reaper reference
- open-chad `bin/oc-killall` — selective session kill reference
