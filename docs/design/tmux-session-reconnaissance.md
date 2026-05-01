# tmux Session Reconnaissance

Reference for inspecting running opencode(advance) instances in tmux. Basis for future remote management features (web portal, SSH via tailscaled).

## Session Structure

Each opencode(advance) instance creates one tmux session with one window and one pane. Sessions live on a dedicated named socket (`-L oca`) — separate from the user's default tmux server, so OCA's session lifecycle never affects unrelated tmux sessions.

**Naming convention (Phase 4+):** `oca-<repo-slug>-<n>`

| Component | Source | Example |
|-----------|--------|---------|
| `oca-` | Fixed namespace prefix (`internal/session/session.go:25`) | `oca-` |
| `repo-slug` | Working directory's repo basename | `opencodeadvance` |
| `n` | Next available integer per repo slug | `0`, `1`, `2`, ... |

Full example: `oca-opencodeadvance-0`, `oca-pokeedge-2`.

> **Historical note:** earlier development snapshots in this repo (and the open-chad sibling) used `oc-{unix_epoch}-{parent_pid}` naming. Those samples below predate the Phase 4 session manager and are kept only for archival comparison — current implementation produces the `oca-<slug>-<n>` form above.

## Reconnaissance Commands

All commands target the OCA-managed socket via `-L oca`. Drop the `-L oca` flag only when you intentionally want to inspect the user's default tmux server.

### 1. List running OpenCode processes

```bash
ps -o pid,ppid,lstart,tty,args -C opencode 2>&1
```

Maps PID to PTY. Cross-reference `pid` with tmux `pane_pid` to link processes to sessions.

Sample output (open-chad era — historical):

```
PID    PPID                  STARTED TT       COMMAND
1035244   44240 Fri Apr 24 00:37:24 2026 pts/0    opencode
1175719   44240 Fri Apr 24 00:52:36 2026 pts/2    opencode
1193255   44240 Fri Apr 24 00:55:23 2026 pts/3    opencode
2973939   44240 Fri Apr 24 08:36:57 2026 pts/4    opencode
3218260   44240 Fri Apr 24 09:22:42 2026 pts/10   opencode
3307763   44240 Fri Apr 24 09:34:13 2026 pts/14   opencode
3620686   44240 Fri Apr 24 10:10:36 2026 pts/7    opencode
```

### 2. List tmux sessions

```bash
tmux -L oca list-sessions 2>&1
```

Filter by `oca-` prefix to isolate OCA sessions. All sessions typically show `attached` since each instance owns its session.

Sample output (illustrative, current naming):

```
oca-opencodeadvance-0: 1 windows (created Fri May 01 09:14:02 2026) (attached)
oca-pokeedge-0: 1 windows (created Fri May 01 09:36:11 2026) (attached)
oca-pokeedge-1: 1 windows (created Fri May 01 10:02:48 2026) (attached)
oca-advance-0: 1 windows (created Fri May 01 10:10:36 2026) (attached)
```

### 3. Map panes to sessions and commands

```bash
tmux -L oca list-panes -a -F '#{session_name} #{window_name} #{pane_pid} #{pane_current_command}' 2>&1
```

Primary query — full session/agent/change mapping in one pass.

| Format field | Meaning |
|---|---|
| `session_name` | Instance identity (`oca-<slug>-<n>`) |
| `window_name` | Agent emoji + name + active change title |
| `pane_pid` | OpenCode process PID (matches `ps` output) |
| `pane_current_command` | Should be `opencode` |

Sample output (open-chad era naming — historical, kept for emoji-status reference):

```
oc-1777005213-1034886 🟥 PW 1035244 opencode
oc-1777006125-1175361 🟥 Flowcalc 1175719 opencode
oc-1777006292-1192953 🟥 Pokeedge · Uuid Final Cutover 1193255 opencode
oc-1777034218-2973568 🟥 PW · Frontend Uuid Final Cutover 2973939 opencode
oc-1777036963-3217992 🟥 SS 3217260 opencode
oc-1777037654-3307304 🟨 Advance 3307763 opencode
oc-1777039837-3620387 🟥 Advance · Complete Temporal Only Migration 3620686 opencode
```

## Window Title Format

ADV status markers set the tmux window title:

```
{emoji} {agent_name} · {change_title}
```

| Emoji | Status |
|-------|--------|
| 🟥 | `[ADV:ATTN]` — user attention needed |
| 🟨 | `[ADV:TOOLING]` — tool/sub-agent in flight |
| 🟩 | `[ADV:WORK]` — actively working |
| 🟦 | `[ADV:SKILL_CREATED]` |
| 🟪 | `[ADV:REFLECTION]` |

When no change is active: `{emoji} {shortname}` only.

## Future Feature Mapping

These reconnaissance primitives support planned remote management features.

### Web Portal

| Portal feature | Recon command | Notes |
|---|---|---|
| Session list | `tmux -L oca list-sessions` | Parse `oca-` prefix |
| Session detail | `tmux -L oca list-panes -a -F '...'` | Window name = agent + change |
| Process health | `ps -o pid,lstart,tty,args -C opencode` | Cross-ref pane PID |
| Live output stream | `tmux -L oca capture-pane -t {session} -p` | Poll or pipe to WebSocket |
| Session kill | `tmux -L oca kill-session -t {session}` | Portal action; `oca session kill` is the supported wrapper |

### SSH + Tailscaled

```bash
tailscale ssh user@host
tmux -L oca attach -t oca-opencodeadvance-0
```

Portal can proxy the PTY stream instead of requiring direct tmux attach.

### API Serialization Sketch

```json
{
  "sessions": [
    {
      "id": "oca-opencodeadvance-0",
      "created": "2026-05-01T09:14:02Z",
      "pid": 1035244,
      "agent": "PW",
      "change": null,
      "status": "attn",
      "attached": true
    },
    {
      "id": "oca-advance-0",
      "created": "2026-05-01T10:10:36Z",
      "pid": 3620686,
      "agent": "Advance",
      "change": "Complete Temporal Only Migration",
      "status": "attn",
      "attached": true
    }
  ]
}
```

Parsing: session name for identity, window name split on `·` for agent/change, emoji for status.

## Session Lifecycle Management (Planned)

Optimized startup and cleanup of opencode(advance) sessions is a planned feature area. Key concerns:

### Startup Optimization

| Concern | Description |
|---------|-------------|
| Cold start latency | Time from `tmux new-session` to agent ready. Profile: config parse, plugin load, MCP server startup, index warmup |
| Warm resume | Reattaching to existing session vs spawning new. Fast path when session already exists |
| Parallel boot | Multiple `oca session` calls concurrently — coordinate session creation, avoid races on shared resources (plugin checkout, MCP ports) |
| Index pre-warm | lgrep semantic/symbol index built on first query or at session start. Tradeoff: eager (slower boot, faster first search) vs lazy (faster boot, slower first search) |

### Cleanup / Teardown

| Concern | Description |
|---------|-------------|
| Graceful shutdown | Signal opencode(advance) process → drain in-flight requests → checkpoint state → exit. Avoid orphan tmux sessions |
| Orphan detection | Sessions with no live OpenCode process (crashed, killed, SSH disconnect). Recon: pane PID not in `ps` output |
| Resource reclamation | MCP server processes, temp files in `$XDG_RUNTIME_DIR`, stale worktrees, index locks |
| ADV state consistency | In-progress change workflows left in non-terminal states. Detect on next session start or portal health check |

### Reconnaissance for Cleanup

```bash
# Find orphan sessions (tmux session exists but no opencode process)
tmux -L oca list-panes -a -F '#{session_name} #{pane_pid}' \
  | grep '^oca-' \
  | while read sess pid; do
      ps -p "$pid" -o pid= >/dev/null 2>&1 || echo "ORPHAN: $sess (pid $pid gone)"
    done
```

`oca session reap` (`internal/session/session.go`) is the supported wrapper — it filters by the `oca-` prefix on the OCA socket and uses `#{session_activity}` rather than name parsing.

### Portal Implications

| Portal action | Lifecycle phase | Notes |
|---|---|---|
| `POST /sessions` | Startup | Spawn tmux session + opencode, return session ID |
| `DELETE /sessions/{id}` | Cleanup | Graceful signal → timeout → force kill → tmux kill-session |
| `GET /sessions` | Recon | List all, flag orphans |
| `POST /sessions/cleanup` | Cleanup | Batch orphan reclamation |

## Name Styling: opencode(advance)

Preferred prose styling: **opencode(advance)**.

**Do NOT use in:**

| Context | Use instead | Reason |
|---------|-------------|--------|
| Markdown link text | `OpenCode Advance` | Parentheses clash with `[text](url)` syntax |
| Shell commands / scripts | `opencode-advance` or `oca` | Parens create subshell in bash |
| Go code / identifiers | `oca` | Parens invalid in identifiers |
| Git branch names | `opencode-advance` | Some tools choke on parens |
| URLs / slugs | `opencode-advance` | Parens percent-encoded, ugly |
| Regex / grep patterns | `opencode.advance` or literal-escaped | Parens are regex metacharacters |

**Safe in:** prose, headings, comments, design docs, presentation decks.

## Notes

- All sessions show `(attached)` because each opencode(advance) instance owns its tmux session — this is expected, not a bug.
- The `oca-` prefix on the named `-L oca` socket is the filter boundary for distinguishing OCA-managed sessions from user-created tmux sessions on the default socket.
- Pane PID and `ps` PID are the same value — direct cross-reference, no indirection needed.
- See `docs/design/tmux-server-safety.md` for blast-radius rules — never `kill` a tmux server PID directly; always go through `tmux -L oca kill-session` or `oca session kill`.
