# Session & Resource-Sharing Architecture

**Status:** Proposal — **decision-locked 2026-05-03**, ready for ADV split (5 changes across OCA + ADV repos). Drafting deferred. See §9.  
**Date:** 2026-05-03  
**Author:** Drafted via OCA + ADV holistic analysis  
**Audience:** OCA design + ADV plugin design + operator workflow

## TL;DR

Optimize OCA's session topology and resource-sharing model for the operator's actual workflow: **8+ concurrent agents per project, 4–5 active projects, 42 GB WSL2 ceiling on 64 GB host.**

| Decision | Direction |
|---|---|
| Session topology | **Adopt "session-per-project" (Pattern B)** as default; multi-window per session for concurrent agents. |
| Per-window status | Decode `[ADV:WORK/ATTN/BLOCKED/IDLE]` markers into the tmux status bar so 8 windows can be scanned without switching. |
| Cross-project visibility | Continue OCA dashboard v1.1 per locked direction; Pattern B makes rows cleaner. |
| RAM hibernation | **Graceful `/exit` + `opencode --session <id>` resume.** SIGSTOP rejected — six documented blockers (see §3.5). |
| Worker sharing | **Already shipped** — `rq-workerSingleton01` gives one worker per project regardless of agent count. |
| Idle worker reaper | File as ADV plugin change. Marginal win but composes with hibernation. |
| WSL2 memory cap | **Leave at 42 GB.** Operator-confirmed deliberate ceiling. Hibernation reclaims headroom; cap stays. |
| Persistence | tmux-resurrect (manual) — yes. tmux-continuum — defer in WSL2 due to focus-bug ([#148](https://github.com/tmux-plugins/tmux-continuum/issues/148)). |
| Cross-device monitoring | OCA dashboard v2.0 web terminal (xterm.js + WebSocket PTY). |

The dominant single lever for headroom under the 42 GB ceiling is **graceful hibernation**: when an agent has been idle for N minutes, OCA gracefully exits it, captures its session ID, and parks the window with a "press R to resume" placeholder. opencode supports `--session <id>` natively. Memory math: 50% idle of 40 agents reclaims ~16 GB.

## 1. Context

### 1.1 Operator workflow (verified)

- **Host:** 64 GB RAM, 16-core desktop, Windows + WSL2 Ubuntu.
- **WSL2 cap:** 42 GB (deliberate, see `.wslconfig`). Operator-confirmed: this is the intended ceiling, not a config oversight. Architecture must work within it.
- **Concurrent agents:** typically 8+ on the same project, often 4–5 projects active simultaneously.
- **Launcher (today):** `oc` → openchad → `tmux new-session -d -s oc-<epoch>-<pid>` → opencode. One detached tmux session per `oc` invocation. Each Windows Terminal tab attaches to its own session.
- **Launcher (in-flight):** OpenCode Advance (`oca`) Go CLI replaces openchad. Already ships session lifecycle (Phase 4), Temporal supervision (Phase 6.5), MCP/Vision config rendering (Phase 1), pane state tracking (Phase 4).

### 1.2 Verified current sharing model

| Component | Process count | Shared by | Status |
|---|---|---|---|
| Vision MCP daemon | 1 host-wide | All opencode sessions | ✓ Shipped |
| MCP servers (23) | 1 per server, fixed ports 6275–6310 | All opencode sessions | ✓ Shipped |
| Temporal dev server | 1 host-wide on `:7233` | All ADV plugins | ✓ Shipped |
| **Temporal worker** | **1 per project** | **All opencode sessions on the same project** | **✓ Shipped — `rq-workerSingleton01`** |
| Workflow runtime | 1 per worker | All workflows for that project | ✓ Shipped |
| ADV plugin runtime | 1 per opencode session | — (intrinsically per-session) | Cannot be shared |
| tmux server | 1 (`-L oca` socket) | All OCA sessions | ✓ Shipped |
| tmux session | 1 per `oca session new` (today) | — | Pattern A today |

**Critical correction to mental model:** "Each opencode runs its own Temporal worker" is **wrong**. ADV's `worker-lock.ts` (`plugin/src/temporal/worker-lock.ts`, spec `rq-workerSingleton01`) implements a filesystem-locked singleton. First plugin instance acquires `worker.lock`; all subsequent instances on the same project become "Temporal clients only" — they skip worker spawn entirely. Project ID = `git rev-list --max-parents=0 HEAD` (root commit SHA), so worktrees of the same repo correctly share one worker.

Live process tree confirms: 4 opencode sessions running, 2 worker processes — one per project (advance, scratch). Working as designed.

### 1.3 Memory math at 42 GB ceiling

Empirical per-process baselines:

| Process | RSS | Note |
|---|---|---|
| opencode (active) | ~800 MB | varies 600 MB–1 GB during heavy tool calls |
| opencode (idle, post-warmup) | ~600 MB | |
| Temporal worker (per project) | ~350 MB | one per project regardless of agent count |
| Vision daemon | ~38 MB | shared host-wide |
| Temporal server | ~210 MB | shared host-wide |
| tmux server | ~7 MB | shared host-wide |

Realistic concurrent scenarios:

| Scenario | Total | % of 42 GB |
|---|---|---|
| 1 project × 8 agents | ~7.0 GB | 17% |
| 2 projects × 8 agents | ~13.8 GB | 33% |
| 3 projects × 8 agents | ~20.6 GB | 49% |
| 4 projects × 8 agents | ~27.5 GB | 65% |
| 5 projects × 8 agents | ~34.4 GB | **82%** — page-cache pressure starts |
| 6 projects × 8 agents | ~41.2 GB | **98%** — swap |

Practical ceiling without RAM-saving levers: ~4 active projects × 8 agents.

## 2. Component & Lever Inventory

| # | Lever | Status | Memory win at 42 GB | UX win | Effort | Verdict |
|---|---|---|---|---|---|---|
| 1 | Pattern B session topology | Not shipped | none | **Critical** at 8-agent scale | 2–4 days | **Critical** |
| 2 | Per-window ADV status decode | Not shipped | none | **Critical** at 8-agent scale | 1–2 days | **Critical** |
| 3 | OCA dashboard v1.1 | In flight | none | **Critical** | 2–3 wk (in flight) | **Critical** |
| 4 | **Graceful hibernation** (`/exit` + `--session`) | Not shipped | **~16 GB at 50% idle of 40 agents** | medium | 2–4 days | **High** |
| 5 | tmux-resurrect (manual) | Not shipped | none | medium (recovery) | 0.5 day | Medium-High |
| 6 | Idle worker reaper (ADV plugin) | Not shipped | ~350 MB × idle projects | low | 1–2 days | Medium |
| 7 | Concurrent-session warning posture | Default warning | none | medium (noise) | 0.5 day | Medium |
| 8 | Worker / Vision concurrency observability | Not measured | none | low | 0.5 day | Medium |
| 9 | OCA dashboard v2.0 (web terminal) | Roadmapped | none | **High** (cross-device) | 3–4 wk | High (long-term) |
| ✗ | ~~SIGSTOP/SIGCONT hibernation~~ | Rejected | — | — | — | **Killed by research** (§3.5) |
| 10 | Worker pool sharing | **Already shipped** | — | — | — | ✓ `rq-workerSingleton01` |
| 11 | tmux-continuum (auto-save) | Not shipped | none | low | low | **Defer in WSL2** ([#148](https://github.com/tmux-plugins/tmux-continuum/issues/148)) |

## 3. Lever Analysis

### 3.1 Session Topology — Pattern A vs Pattern B

**Pattern A (today):** every `oc <project>` invocation creates a fresh `oc-<epoch>-<pid>` (openchad) or `oca-<slug>-<n>` (OCA) tmux session. Each Windows Terminal tab attaches to its own session. 1 session : 1 window : 1 pane.

**Pattern B (proposed):** one tmux session per project (`advance`, `scratch`, `pokeedge`). Multiple windows per session, one opencode per window. One Windows Terminal tab attaches to a project's session. Cross-project navigation via WT tabs.

#### 3.1.1 Industry consensus

| Source | Year | Recommendation |
|---|---|---|
| Jess Archer, "Managing Development Environments with Tmux and Tmuxinator" | 2019 | "Create a separate session for each project, with each window dedicated to a specific process." |
| tmuxinator README + ecosystem | ongoing | session-per-project; `mux start <project>` is get-or-create |
| tmuxp docs | current | sessions are project-scoped, declarative YAML |
| dev.to "How do I work on multiple projects simultaneously" | 2020 | "Manage multiple projects using sessions and create multiple terminal windows" |

No surfaced source advocates per-invocation tmux sessions as a deliberate pattern.

#### 3.1.2 Multi-agent AI trend (last 12 months — LBP-relevant)

| Source | Date | Pattern |
|---|---|---|
| Kaushik Gopal, "Forking subagents in an AI coding session with tmux" | Dec 2025 | tmux as orchestration substrate; pane-per-agent |
| Dariusz Parys, "Claude Code Multi-Agent tmux Setup" | 2025 | parallel agents in tmux panes |
| Reddit r/commandline "Tmux workflow for multiple AI coding agents?" | Jan 2026 | "splitting tmux panes" baseline |
| Anthropic, "Orchestrate teams of Claude Code sessions" (official docs) | current | team lead + worker sessions in tmux |
| `pchalasani/claude-code-tools`, `affaan-m/everything-claude-code` | current | boss-worker / dmux pane orchestration |

Direction: **session-per-project + windows/panes-per-agent** is the published default for multi-agent AI coding workflows.

#### 3.1.3 Signal propagation (WSL2 + Windows Terminal + tmux)

Verified from tmux internals (unix.SE 568847), [microsoft/terminal#9914](https://github.com/microsoft/terminal/issues/9914):

| Event | Pattern A | Pattern B |
|---|---|---|
| User clicks WT tab X | SIGHUP → bash → tmux client; tmux **server** survives (daemonized); session detached | Identical — tmux server survives; project session detached; all windows keep running |
| User runs `/exit` in opencode | opencode exits → pane closes → window closes → session auto-destroys (only window) → client exits | opencode exits → pane closes → window closes; **session persists** (other windows still open) |
| Force-close terminal ([#9914](https://github.com/microsoft/terminal/issues/9914)) | wsl.exe SIGKILL'd; tmux server unaffected | Identical |
| `tmux kill-window` | SIGHUP → opencode → clean exit | Identical |
| `tmux kill-session` | All windows SIGHUP'd | All N opencodes on project SIGHUP'd at once (higher per-action blast radius — mitigate with confirmation prompt) |
| tmux server crash | All sessions lost | Same — same blast radius |

Both patterns equally protect against force-close orphans because both use tmux. The difference is **what a "session" represents** — disposable wrapper (A) vs durable project context (B).

#### 3.1.4 Pattern B advantages summarized

| Dimension | Winner |
|---|---|
| `tmux ls` readability | B (`advance`, `scratch`) vs A (`oca-advance-0..3`) |
| Switch agents on same project | B (`Ctrl+B 0–9` instant; `Ctrl+B w` window tree) |
| Closing a project entirely | B (one cmd, kills children cleanly) |
| Reap policy clarity | B (sessions are intentional; no "is this orphan or just detached?" ambiguity) |
| Per-project tmux config (status bar, hooks) | B (sessions are durable; A's are throwaway) |
| Boot animation cost | B (once per project, not per window) |
| Vision/Temporal singleton init races | B (per project, one launch) |
| OCA dashboard row rendering | B (1 row per project; A clutters with `slug-0..N`) |
| Multi-agent ergonomics (industry trend) | B (native tmux ops; A is awkward) |
| Implementation cost today | A (zero — Pattern A is shipped; B requires OCA work) |
| Backwards-compat with openchad muscle memory | A |

No dimension shows a hard blocker for B. UX/coherence wins are decisive at 8-agent scale.

### 3.2 Per-Window ADV Status Decode

At 8 windows per project, switching into each to learn its status is impractical. The tmux status bar must show per-window markers.

opencode + ADV already emit `[ADV:WORK/ATTN/BLOCKED/IDLE]` in tab titles. tmux can read window titles via `#W` and color them via `window-status-format`. OCA's status-bar renderer (`internal/session/`, Phase 4 shipped) can include a per-window decode.

Render sketch:

```
[advance]  0:🟩 review-A  1:🟥 apply-B  2:🟪 reflect-C  3:⬜ trunk  4:🟨 plan-D  5:🟩 ...
```

Implementation: OCA Phase 8 polish item. All primitives shipped (status bar, pane state JSON, ADV marker convention).

### 3.3 OCA Dashboard

Direction is locked per `docs/notes/2026-05-01-dashboard-architecture-research.md`:

| Phase | Capability | Cost |
|---|---|---|
| v1.1 | Read-only unified table: cross-project changes + sessions + Temporal/worker health | 2–3 wk |
| v1.2 | Per-row controls (gate approve, retry, cancel) + local switch-session (Approach A: tmux switch-client) | 1–2 wk |
| v1.3 | Local "open in new terminal window" (Approach B) + token auth | 1 wk |
| v2.0 | Web terminal (xterm.js + WebSocket PTY broker) + Tailscale-friendly bind + cross-host federation | 3–4 wk |

Stack: Go HTTP server + `//go:embed` HTML/Tailwind/Datastar; SSE for state, WebSocket for terminal PTY; tmux access via control mode (`tmux -L oca -C`).

Composes with the other levers:

- **Pattern B:** Dashboard naturally renders one row per project; A would clutter the table.
- **Hibernation:** Dashboard surfaces "5 agents active, 3 hibernated" per project; resume-from-row buttons at v1.2.
- **Web terminal (v2.0):** Cross-device session resume — phone/tablet/laptop attaches to the same tmux session via xterm.js. tmux supports multiple clients; browser attach doesn't displace the local terminal client.

### 3.4 Worker Sharing — Already Shipped (Foundation)

Spec: `rq-workerSingleton01`. Implementation: `plugin/src/temporal/worker-lock.ts` (282 lines).

Mechanism:

1. First opencode on a project tries to acquire `worker.lock` via atomic O_EXCL file create.
2. On success: this opencode spawns the Temporal worker.
3. Subsequent opencodes on the same project read the lock, verify owner PID via `process.kill(pid, 0)`, become "Temporal clients only".
4. On owner exit: lock is released via best-effort rename+remove. Next opencode reclaims via stale-PID detection (ESRCH).
5. Project ID = git root commit SHA, so all worktrees of the same repo correctly share one worker.

**Verification task (P7):** confirm rapid lock-handover under Pattern B at 8-agent scale. Read of `worker-lock.ts` suggests reclaim handles PID liveness correctly; need to validate respawn path lives in `worker-multi.ts` orchestrator. Treat as observability pass, not design change.

### 3.5 Hibernation — SIGSTOP Rejected, Graceful Adopted

#### 3.5.1 SIGSTOP/SIGCONT — six documented blockers

| # | Source | Finding |
|---|---|---|
| 1 | [anthropics/claude-code#43177](https://github.com/anthropics/claude-code/issues/43177) (Apr 2026) | "When a stdio-type MCP server process dies or disconnects, Claude Code marks it as failed and never attempts reconnection." Generalizes to MCP-client class; opencode is the same family. |
| 2 | [modelcontextprotocol/csharp-sdk#726](https://github.com/modelcontextprotocol/csharp-sdk/issues/726) (Aug 2025) | "The SSE MCP Server automatically disconnects when there is no interaction for a certain period, and the MCP Client is unaware of this." Vision serves many MCP via SSE/HTTP. |
| 3 | [Cursor forum, Feb 2026](https://forum.cursor.com/t/http-mcp-server-becomes-unresponsive-after-repeated-sse-stream-disconnects/152243) | "HTTP MCP server becomes unresponsive after repeated SSE stream disconnects" — even reconnect attempts fail to recover. |
| 4 | [temporalio/sdk-typescript#119](https://github.com/temporalio/sdk-typescript/issues/119) (open since 2021) | Client reconnection configuration is still an open feature request; default reconnect handling is fragile. |
| 5 | [unix.SE #202104](https://unix.stackexchange.com/questions/202104/) | While stopped, "there's no such thing as a timeout in the process. There are usually no timeouts in the network stack either" — but counterparties time out, leading to silent application-level death. |
| 6 | [microsoft/WSL#12747](https://github.com/microsoft/WSL/issues/12747) (Mar 2025) | WSL2 has hibernate-related networking issues layered on top. |

**Failure mode:** SIGSTOP for >a few minutes → counterparties (Vision MCP servers, Temporal server) drop the stopped client's connections → SIGCONT resumes the Node process → opencode appears alive but MCP/Temporal calls fail mysteriously → user has to restart anyway, losing in-memory conversation. **Worst possible outcome:** silent, partial, requires restart that we tried to avoid.

**Verdict:** SIGSTOP hibernation is rejected. Documentation is unambiguous and convergent across multiple ecosystems.

#### 3.5.2 Graceful hibernation (adopted)

[opencode CLI docs (May 2026)](https://opencode.ai/docs/cli/) confirm native session resume:

```bash
opencode --continue              # resume last session
opencode --session <id>          # resume specific session by ID
opencode --session <id> --fork   # fork while resuming
```

Session state durable in `~/.local/share/opencode/opencode.db` (SQLite). ADV change state durable in Temporal (per-project external store) and is independent of any single opencode session.

Mechanism:

1. OCA detects window idle (no opencode tool activity, no keystrokes) for N min via existing pane-state tracking.
2. OCA sends graceful exit: `tmux send-keys -t <pane> '/exit' Enter`.
3. opencode persists final state to its session DB and exits cleanly.
4. OCA captures the session ID (from opencode's exit message, or post-exit query against opencode.db by mtime+pid).
5. Window remains in tmux with placeholder shell + status: `💤 hibernated · session abc123 · press R to resume`.
6. On user `R` keypress (or click in dashboard): OCA spawns `opencode --session <id>` in same pane.
7. opencode resumes from session DB; ADV plugin re-attaches to project worker (or respawns it via lock if idle reaper killed it).

Memory math at scale (5 projects × 8 agents, 50% hibernated):

| State | RAM |
|---|---|
| 40 active opencode | ~32 GB |
| 20 active + 20 hibernated | ~16 GB |
| **Savings** | **~16 GB** |

Dominant lever for headroom under the 42 GB ceiling.

**Default:** auto on idle, default-on, per-session opt-out flag. Settable via tmux keybind (`Ctrl+B H` toggle) with status indicator (🔒). Default idle threshold **60 minutes** (tunable in stack.toml). Idle detection requires: no keystrokes for N min AND no inflight tool calls AND no Temporal workflows in `Running` state for this session — never kills mid-work.

**Risks (manageable, none documented as blockers):**

| Risk | Mitigation |
|---|---|
| Session ID capture timing | **Verified trivial (2026-05-03):** `opencode session list --format json` returns `[{id, title, updated, created, projectId, directory}]`. OCA queries by `directory == workdir`, sorts by `updated` desc, takes first. No tmux-output interception needed. |
| Cold-start latency on resume (~3–5 sec) | Acceptable for 30-min-idle threshold; pre-warm option for "next likely target" |
| User in middle of conversation gets exited | Idle detection uses last keystroke + last tool call timestamps; conservative N (default 30 min) |
| State drift if ADV worker died during hibernation | Worker singleton + lock-reclaim already handles this; resume reacquires |
| Graceful exit fails (opencode hung) | Fallback: SIGTERM, then SIGKILL after grace period; session DB has last persisted state |

### 3.6 tmux Persistence

**tmux-resurrect:** Manual save/restore via `Ctrl+B Ctrl+s` / `Ctrl+B Ctrl+r`. Mature. Restores window layouts + working directories. Does NOT restore opencode in-memory state — but graceful hibernation already covers that via session DB.

**tmux-continuum:** Auto-save resurrect every N minutes; auto-restore on tmux server start. **Known WSL2 bug** ([#148](https://github.com/tmux-plugins/tmux-continuum/issues/148), Mar 2025): auto-save only fires when terminal window is in focus. Manual save still works.

**Critical insight: state durability is already three layers deep:**

| Layer | What it preserves | Recovery |
|---|---|---|
| Temporal workflows (per project) | Change state, gate progress, task graph, evidence, audit trail | Via `adv_change_show` |
| ADV external state directory | Specs, archive, agenda, wisdom | On disk |
| opencode session DB | Conversation history, session ID | `opencode --session <id>` |
| tmux-resurrect | Window layout, cwd | Convenience layer |

ADV's "resume by changeId" pattern is the real session-resume primitive. tmux-resurrect is a layout convenience.

**Recommendation:** ship tmux-resurrect (manual). Defer continuum until WSL2 bug resolves OR build OCA-native polling alternative (`oca-resurrect` daemon writing layouts via tmux control mode — already in-process for the dashboard).

### 3.7 Idle Worker Reaper

Re-elevated from "demoted" because at 42 GB ceiling every ~350 MB matters when several projects are open.

After N minutes of no Temporal activity (no inbound tool calls, no inflight workflows), worker-lock owner shuts down the worker, releases the lock, stays as a Temporal client only. On next ADV tool call, any opencode on the project re-acquires the lock and respawns the worker.

| Aspect | Detail |
|---|---|
| RAM win | ~350 MB per idle project. With 3 idle projects: ~1 GB. |
| Cost | First-call-after-idle latency: ~few seconds (Temporal SDK + Node startup + connection establish + workflow registration). |
| Inflight guard | Don't shutdown if `ListWorkflowExecutions(Running)` returns non-empty. |
| Hysteresis | Default 60 min idle; only shut down once per hour. |
| Config | `worker.idle_shutdown_minutes` in plugin config; default 60, 0 = disabled. |
| Implementation locus | `worker-multi.ts` + `worker-lock.ts`; ~1–2 days work + tests. |

Composes with hibernation: once all opencodes on a project are hibernated, no client traffic → reaper kicks in → worker dies → project is at zero RAM cost until first user input.

### 3.8 Concurrent-Session Warning Posture

Today ADV emits `[ADV:WARN] Concurrent OpenCode sessions detected in this project (PIDs: a,b,c). Git operations from any session affect all.` This is correct safety messaging at 1–2 sessions but constant noise at 8.

**Two posture options:**

| Option | Approach |
|---|---|
| Suppress when expected | `coordination_mode = "expected"` in stack.toml; warning silenced when ≥2 concurrent sessions match |
| Upgrade indicator | Compute git-worktree topology: "8 sessions detected · 8 worktrees · no shared mutation surface · OK" |

**Recommendation:** the upgrade-indicator approach. Inspect git-worktree-list for each detected session; if all are on distinct branches with distinct working trees, render as "coordinated" green; if any two share a working tree, render as red warning. ADV worktree policy enforces per-change isolation, so the green path is the expected case.

### 3.9 Worker / Vision Concurrency Observability (One-Time)

ADV worker uses default Temporal SDK config (no `maxConcurrent*` overrides). Defaults:

| Setting | Default | At 8 agents per project |
|---|---|---|
| `maxConcurrentActivityTaskExecutions` | 100 | OK |
| `maxConcurrentWorkflowTaskExecutions` | 40 | OK |
| `maxConcurrentLocalActivityExecutions` | 100 | OK |
| `maxConcurrentActivityTaskPolls` | 5 | OK |
| `maxConcurrentWorkflowTaskPolls` | 5 | OK |

Defaults are sized for hundreds of concurrent workflows per worker; 8 is well within capacity. **Verification task:** instrument `adv_investment_report` or add a small `worker.metrics()` probe to confirm no queue saturation under real workload. Likely a non-issue.

Vision daemon: 8 agents × 23 MCP servers in worst case = up to 184 parallel calls. Slot-grouped servers (playwright variants) explicitly designed for concurrent users via slot-routing — already correct architecture. Stateless RPC servers (kagi, context7, lgrep, firecrawl) handle concurrent calls fine. **Verification task:** under real 8-agent load, run `vision_metrics` and check for any single-server bottleneck.

## 4. Pros / Cons / Tradeoffs Matrix

Single decision-maker table. Comparing today's posture against the proposed posture (Pattern B + status decode + dashboard + graceful hibernation + resurrect + idle reaper).

| Dimension | Today (Pattern A) | Proposed (B + hibernation + dashboard + reaper) |
|---|---|---|
| Headroom for 5 projects × 8 agents | 82% utilization (page-cache pressure) | ~50% (with 50% hibernated) |
| Headroom for 4 projects × 8 agents | 65% (tight) | ~40% (comfortable) |
| Switch between agents on same project | Hunt across N WT tabs | `Ctrl+B 0–9` instant; `Ctrl+B w` tree |
| See state of all 8 agents at once | None | tmux status bar scan + dashboard table |
| Cross-device monitoring | SSH only | Dashboard v2.0 web terminal (any browser) |
| `tmux ls` readability | `oca-advance-0..3, oca-scratch-0..1` | `advance, scratch, pokeedge` |
| Closing a project | N tab closes; hope reaper handles orphans | `oca session kill advance` (with confirm at ≥2 windows) |
| Orphan opencode after WT close | Possible (4h reap window) | Project session intentionally persists |
| Resume after laptop restart | Manual restart × 8 (lose conversations) | tmux-resurrect for layout + `opencode --session <id>` per window |
| Worker count (8 agents on 1 project) | 1 (already shared) | 1 |
| RAM per idle agent | ~600 MB | 0 MB (hibernated) |
| RAM per idle project (no agents active) | ~350 MB worker still running | 0 MB (worker reaped) |
| First-call-after-idle latency | none | ~3–5 sec (worker respawn) + ~3–5 sec (opencode resume if hibernated) |
| Industry alignment / LBP | Niche per-invocation pattern | Canonical session-per-project (tmuxinator/tmuxp/Coder/Anthropic patterns) |
| Multi-agent AI workflow ergonomics | Awkward | Native to current ecosystem direction |
| Boot animation cost | Per WT tab (~600 ms) | Per project tab (amortized over windows) |
| Vision/Temporal singleton init races | Per-launch contention | Per-project, one launch |
| Discoverability for new users | Low | High (dashboard) |
| Failure modes (tmux server crash) | Same | Same |
| Implementation cost | Zero (today) | OCA: medium; ADV: small (idle reaper) |
| Cognitive coherence | "session = throwaway wrapper" | "session = my work on project X" |
| Operability — kill/restart/inspect | Per opencode | Per project, with dashboard view |
| Per-action blast radius | One opencode | All N opencodes in project (mitigate via confirm) |
| Backwards compat with current muscle memory | Identical | Different — shipped as opt-in then default-on |

No dimension shows the proposed posture worse on substance. Three dimensions (per-action blast radius, latency on resume, implementation cost) are the legitimate cons; all are mitigated.

## 5. Recommendation

Adopt the proposed posture. Sequence by value-per-day:

| Phase | Action | Repo | Effort |
|---|---|---|---|
| P1 | **Pattern B session topology** — `[session].mode = "per-project"`, `oca session new --project <slug>` (get-or-create), `oca session attach <project>`, multi-window support, pane state schema bump | OCA Phase 8 | 2–4 days |
| P2 | **Per-window ADV status decode** in tmux status bar | OCA Phase 8 | 1–2 days |
| P3 | **OCA dashboard v1.1** — read-only unified table | OCA (in flight) | 2–3 wk |
| P4 | **Graceful hibernation** — idle detection + `/exit` + session-id capture + resume | OCA + small ADV cooperation | 2–4 days |
| P5 | **tmux-resurrect** (manual) + **concurrent-session warning posture upgrade** | OCA stack.toml + ADV plugin | 1 day combined |
| P6 | **Idle worker reaper** | ADV plugin | 1–2 days |
| P7 | **Worker / Vision concurrency observability pass** | ADV + Vision | 0.5 day |
| P8 | **OCA dashboard v1.2 + v2.0** | OCA roadmap | per existing plan |

P1, P2 unlock day-one UX wins for 8-agent workflow. P3 is killer feature for cross-project view. P4 is dominant RAM lever. P6 is small composing win. P8 is the long-term unifier (cross-device).

## 6. Open Questions

1. **Pattern B retirement of Pattern A:** ship as opt-in for one minor version, then flip default? Or default-on day one with `mode = "per-invocation"` escape valve?
2. **Pattern B window naming:** with 8 ADV changes per project, names matter. ADV change ID (`tk-xxxx` format)? Truncated change summary? Both?
3. **Status bar density:** top-line summary `[advance: 5W 2A 1B]` or full per-window decode `0:🟩 1:🟥 2:🟩 ...` or both?
4. **Pattern B session kill confirmation:** silent for one-window sessions, prompt at ≥2 windows?
5. **Pattern B auto-population:** when `oca session new advance` creates a fresh session, open one window (auto-launched opencode) or two (one opencode, one free shell)?
6. **Hibernation idle threshold default:** 30 min OK? Configurable via `[hibernation].idle_minutes`?
7. **Hibernation per-session opt-out UX:** tmux keybind, dashboard toggle, or `oca pane keep-alive`?
8. **Concurrent-session warning upgrade target:** reuse `[ADV:WARN]` marker with new content, or introduce `[ADV:COORDINATED]` distinct marker?
9. **Dashboard density at 5 × 8 = 40 rows:** compact (Linear, ~24 px) or comfortable (Vercel, ~40 px)? Compact almost certainly wins at this density.
10. **Cross-worktree workflow integration:** should `oca session new --worktree <change-id>` auto-create a window in the project's session tied to that worktree? Composes Pattern B with ADV worktree policy.

## 7. Out of Scope (deliberate)

- **WSL2 memory bump.** Operator-confirmed: 42 GB cap is intentional architectural ceiling. Hibernation reclaims headroom; cap stays.
- **SIGSTOP/SIGCONT process-pause hibernation.** Killed by research (§3.5).
- **tmux-continuum auto-save.** Deferred until WSL2 focus-bug ([#148](https://github.com/tmux-plugins/tmux-continuum/issues/148)) resolves or OCA-native alternative ships.
- **Cross-host federation in v1.x.** Per locked dashboard direction; pulled in by v2.0 web terminal long-term.
- **Replacement of opencode session DB.** Out of scope; durable layer already exists.

## 8. Companion Files to Update

If this proposal is accepted:

- `docs/design/tmux-session-reconnaissance.md` — add Pattern B section
- `docs/design/stack-toml-schema.md` — add `[session].mode`, `[hibernation]`, `[worker].idle_shutdown_minutes` fields
- `docs/design/cli-surface.md` — `oca session new --project`, `oca session attach <project>`, `oca session kill-window <project>:N`, `oca pane keep-alive`
- `docs/notes/2026-05-01-dashboard-architecture-research.md` — note Pattern B interaction with row rendering
- `docs/proposals/phases.md` — add Phase 8.x entries P1–P7

ADV plugin docs (separate repo, `~/dev/oc-plugins/advance/`):

- New design note: `docs/idle-worker-shutdown.md`
- Or directly file as `/adv-proposal "ADV idle worker reaper"` in the ADV repo

## Appendix A: Verified Architecture Reads

| File / spec | Confirms |
|---|---|
| `~/dev/open-chad/bin/openchad` (lines 213–256) | Pattern A today: `oc-$(date +%s)-$$`, `tmux new-session -d -s <name> ... opencode`, `remain-on-exit off` |
| `~/dev/opencodeadvance/internal/session/session.go:57-83` | `Manager.Create` runs `tmux new-session -d -s <name> -c <workingDir>` on `-L oca` socket |
| `~/dev/opencodeadvance/cmd/oca/session.go:60-77, 168` | `NextSessionName` derives `oca-<repoSlug>-<n>`; `--name` flag exists, no `--project` get-or-create today |
| `~/dev/opencodeadvance/cmd/oca/pane.go:18-23, 112-117` | `paneState` schema: `sessionID`, `directory`, `ts`. Per-pane state in `$XDG_STATE_HOME/oca/panes/<socket>/<paneID>.json` |
| `~/dev/opencodeadvance/internal/session/session.go:313-380` | `ReapCandidates` uses `#{session_activity}`; no detached-tab vs orphan distinction |
| `~/dev/oc-plugins/advance/plugin/src/temporal/worker-lock.ts` (282 lines) | `rq-workerSingleton01` — atomic O_EXCL filesystem lock, PID-liveness reclaim, project-keyed by root commit SHA |
| `~/dev/oc-plugins/advance/plugin/src/utils/project-id.ts:20-44` | `getProjectId()` = `git rev-list --max-parents=0 HEAD` |
| `~/dev/oc-plugins/advance/plugin/src/temporal/out-of-process-worker.ts:1-13` | "Migration from per-queue children to a single shared child reduces process overhead" — within-project queue sharing already shipped |
| `~/dev/opencodeadvance/docs/notes/2026-05-01-dashboard-architecture-research.md` | Dashboard direction locked: Datastar + server-rendered + SSE; v1.1–v2.0 phased plan |
| Live `ps`/`free` snapshot 2026-05-03 | 4 opencode + 2 worker procs (one per project) confirms `worker-lock` working as designed |
| `/mnt/c/Users/jrede/.wslconfig` | `memory=42GB`, deliberate operator ceiling on 64 GB host |

## Appendix B: Hibernation Research Trail

Six citations rejecting SIGSTOP/SIGCONT, one confirming the graceful-exit alternative:

| # | URL | Status | Date |
|---|---|---|---|
| 1 | https://github.com/anthropics/claude-code/issues/43177 | Open bug | Apr 2026 |
| 2 | https://github.com/modelcontextprotocol/csharp-sdk/issues/726 | Open bug | Aug 2025 |
| 3 | https://forum.cursor.com/t/http-mcp-server-becomes-unresponsive-after-repeated-sse-stream-disconnects/152243 | Field report | Feb 2026 |
| 4 | https://github.com/temporalio/sdk-typescript/issues/119 | Open feature request | 2021–present |
| 5 | https://unix.stackexchange.com/questions/202104/ | Reference | 2015 (durable answer) |
| 6 | https://github.com/microsoft/WSL/issues/12747 | Open bug | Mar 2025 |
| ✓ | https://opencode.ai/docs/cli/ | Official docs | May 2026 — confirms `--session <id>` and `--continue` |

## Appendix C: Industry Pattern Citations

Pattern B (session-per-project) consensus:

- https://jessarcher.com/articles/managing-development-environments-with-tmux-and-tmuxinator/ (2019) — canonical statement
- https://github.com/tmuxinator/tmuxinator — tooling
- https://tmuxp.git-pull.com/ — declarative YAML alternative
- https://medium.com/@johanjohansson_63760/how-to-use-tmux-and-tmuxinator-efficiently-f58ccdd46406 (2022)
- https://martinwood.org/managing-multiple-projects-with-tmuxinator (2020)
- https://dev.to/robusgauli/how-do-i-work-on-multiple-projects-simultaneously-without-losing-my-mind-5h3o (2020)

Multi-agent AI + tmux trend (LBP-relevant, last 12 months):

- https://kau.sh/blog/agent-forking/ (Dec 2025)
- https://www.dariuszparys.com/claude-code-multi-agent-tmux-setup/ (2025)
- https://news.ycombinator.com/item?id=46401588 (2025) — Tmux-CLI delegation tool
- https://github.com/pchalasani/claude-code-tools — boss-worker pattern
- https://github.com/affaan-m/everything-claude-code — dmux pane manager
- https://code.claude.com/docs/en/agent-teams — Anthropic official Agent Teams pattern

WSL2 + Windows Terminal + tmux signal handling:

- https://github.com/microsoft/terminal/issues/9914 (Apr 2021) — orphaned processes when WT force-closes
- https://unix.stackexchange.com/questions/568847/tmux-kill-window-doesnt-kill-child-processes (2020) — tmux SIGHUP semantics

## Appendix D: Rejected Alternatives

| Alternative | Why rejected |
|---|---|
| Bump WSL2 to 56 GB | Operator-confirmed deliberate ceiling at 42 GB. Hibernation is the right lever, not config bump. |
| SIGSTOP/SIGCONT process pause | Six documented blockers across MCP/Temporal/WSL2 ecosystems (§3.5). Silent failure mode. |
| Per-opencode separate Temporal worker (status quo) | Already not the case — `rq-workerSingleton01` shipped. Mental model correction. |
| One mega tmux session for all projects | Conflates project boundaries; defeats Pattern B's whole UX gain. |
| tmux-continuum auto-save | WSL2 focus-bug ([#148](https://github.com/tmux-plugins/tmux-continuum/issues/148), open March 2025). Defer. |
| Forking agents into panes within one window (vs separate windows) | Reddit r/commandline Jan 2026: "doesn't really scale past 4 panes". 8 agents need windows, not panes. |

---

## 9. Decision Lock (2026-05-03)

All §6 open questions + 3 architectural critique items resolved in operator-facing decision round. Proposal moves from "pre-decision" to "decision-locked, ready for ADV split."

### 9.1 Architectural critique resolutions

| ID | Topic | Resolution |
|---|---|---|
| Q11 | opencode `projectId` vs ADV `projectId` reconciliation | **No work needed.** Empirical verification (2026-05-03): both schemes resolve to identical hashes (`opencode debug scrap` returned `cdae139e16c8cbaa4ef2ebf35dda326e826e93bc` for `~/dev/opencodeadvance`; `git rev-list --max-parents=0 HEAD` returned the same). opencode's `projectId` IS the git root-commit SHA. The hypothesized inconsistency was a misread of the proposal. |
| Q12 | Mid-tool-call hibernation safety | **Hard skip.** Idle detection requires AND of: no keystrokes for N min, no inflight bash/tool calls (queried via opencode tool-call ledger), no Temporal workflows in `Running` state for this session. Never kills mid-work. Conservative; safety > RAM reclaim. |
| Q13 | Pattern B × ADV worktrees | **Window-per-worktree, named by change-id slug.** Project session structure: 1 `trunk` window (cwd = main checkout) + N change windows (cwd = each active worktree, name = ADV change-id slug, e.g. `hardenOcaAgainstAdvRuntime`). ADV worktree creation auto-spawns the corresponding window in the project session. At peak load: ~9 windows per heavily-loaded project. Grounded in operator pattern (Q13a): "every active change gets its own worktree." |

### 9.2 §6 open questions resolutions

| Q | Topic | Resolution |
|---|---|---|
| 1 | Pattern B rollout | **Default-on day one + escape valve.** `[session].mode = "per-invocation"` in stack.toml restores Pattern A. Bold cutover, fast payoff. |
| 2 | Window naming | **Change-id slug** (e.g. `hardenOcaAgainstAdvRuntime`). Trunk window = `trunk`. Stable, matches worktree dir name. |
| 3 | Status bar density | **Per-window decode** `0:🟩 1:🟥 2:🟩 ...`. Each window shows index + emoji marker. |
| 4 | Session kill confirmation | **N/A.** `oca session kill` is not part of the operator's lifecycle. Closure happens via `/exit` from inside opencode (window auto-closes; session auto-destroys when last window closes). Raw `tmux kill-session` remains available for edge recovery. |
| 5 | New session entry point | **`oca` (no args) from project root = smart get-or-create.** If project session exists, attach. If not, create session named after project, open `trunk` window with opencode launched. No explicit `oca session new <project>` for normal flow. |
| 6 | Hibernation idle threshold | **60 min default**, configurable via `[hibernation].idle_minutes` in stack.toml. |
| 7 | Hibernation opt-out UX | **tmux keybind toggle + status indicator.** `Ctrl+B H` toggles keep-alive on current window; status bar shows 🔒 when locked. Fast, visible, in-context. |
| 8 | Concurrent-session warning marker | **Defer to ADV plugin design.** OCA-side opens issue in ADV repo; OCA does not own the marker semantics. |
| 9 | Dashboard row density | **Comfortable (~40 px) like Vercel.** ~25 rows per screen at 1080p; scroll for rest. |
| 10 | Cross-worktree integration | **Resolved by Q13.** `oca session new --worktree <change-id>` is supplanted by automatic window creation when ADV creates the worktree. |

### 9.3 Five-change split

Work splits into **5 ADV changes** across **2 repositories** (OCA + ADV), plus an existing in-flight track (dashboard).

| # | Change | Repo | Phase(s) | Effort | Notes |
|---|---|---|---|---|---|
| 1 | Pattern B + per-window status decode + smart `oca` entry | OCA | P1+P2+CLI | 3–6 days | Highest UX value; unblocks the rest. Single coherent change covering session topology, status decode, and CLI surface (smart `oca` from project dir). |
| 2 | Graceful hibernation | OCA | P4 | 2–4 days | Highest RAM value (~16 GB at 50% idle). Depends on change #1 shipped (status bar must surface 💤 marker). |
| 3 | tmux-resurrect (manual) + minor warning posture polish | OCA | P5 | 1 day | Small composable change. Can ride alongside or after #1. |
| 4 | Idle worker reaper | ADV plugin | P6 | 1–2 days | ADV-repo-side. Composes with hibernation: when all opencodes on a project hibernate, no client traffic → reaper kicks in → worker dies → project at zero RAM cost until next user input. |
| 5 | Concurrent-session marker reframe (`[ADV:WARN]` → `[ADV:COORDINATED]` semantics) | ADV plugin | Q8 | 0.5–1 day | ADV-repo-side. Filed as ADV-domain decision per Q8 resolution. OCA opens an issue. |

Outside this split: **OCA dashboard v1.1 / v1.2 / v2.0** (P3 / P8) continues on its existing track per `docs/notes/2026-05-01-dashboard-architecture-research.md`. Pattern B materially improves dashboard rendering (one row per project) but is not a hard prerequisite.

### 9.4 Sequencing

| Order | Change | Rationale |
|---|---|---|
| 1 | OCA Pattern B (#1) | Unblocks everything. Day-one UX win at 8-agent scale. |
| 2 | OCA hibernation (#2) | Dominant RAM lever; needs #1's status surface. |
| 3 | OCA resurrect + warning (#3) | Small polish; can run in parallel with #2 if desired. |
| 4 | ADV idle reaper (#4) | ADV-repo work; parallelizable with OCA work. Lands once worktree-shared-worker invariant is verified at 8-agent scale. |
| 5 | ADV marker reframe (#5) | Quality-of-life cleanup; lands any time after Pattern B is in operator hands. |

Drafting via `/adv-proposal` is **deferred**. This decision lock captures the full plan; individual change proposals will be created when work is ready to start.

---

## 10. Items for /adv-discover Phase

Critique gaps that survived decision-lock — these are investigation items for the discovery phase of each change, not redesign blockers.

### 10.1 OCA Pattern B (#1) — discovery items

- **Smart `oca` get-or-create semantics:** what happens when invoked outside any git repo? Outside any known project? Inside a git repo but no existing OCA session?
- **Pattern A escape valve UX:** stack.toml `[session].mode = "per-invocation"` — global, per-project, both?
- **Window auto-creation hooks:** exact ADV plugin handshake when a worktree is created. New OCA tool that the ADV plugin calls? File-watch on `~/.local/share/opencode/worktree/<projectId>/change/`? Both? Race conditions on simultaneous worktree creation?
- **Trunk window cwd handling:** what if the operator is on a feature branch in trunk checkout when first launching `oca`? Trunk window = wherever the main checkout HEAD points, not necessarily the default branch.
- **Per-action blast radius mitigation:** §3.1.3 mentions "confirmation prompt" for kill at ≥2 windows. Per Q4 resolution, kill is not the lifecycle — confirm whether ANY operator-facing destructive op needs the prompt.

### 10.2 OCA hibernation (#2) — discovery items

- **Cold-start latency measurement:** §3.5.2 claims "~3–5 sec" for `opencode --session <id>` resume against a 3.4 GB session DB. Needs empirical measurement with real session of typical size before committing the UX promise.
- **Graceful exit failure path:** §3.5.2 mentions "SIGTERM, then SIGKILL after grace period." Need WAL fsync confirmation on SIGTERM before SIGKILL escalation, else session DB partial-write → resume corrupts. Spec the grace period and escalation ladder.
- **Session DB single-writer contention:** 8 concurrent agents on same project all write to one SQLite DB (currently 3.4 GB, WAL 4 MB). Measure write latency under load; consider WAL checkpoint cadence.
- **Power management interaction:** WSL2 + Windows sleep/wake cycles vs hibernated sessions. Investigate whether hibernated tmux windows survive a host sleep cycle correctly.
- **Telemetry / opt-out persistence:** `Ctrl+B H` keep-alive toggle state — persisted where? Per-tmux-session ephemeral, or per-pane in `$XDG_STATE_HOME/oca/panes/`?
- **Idle-reaper × hibernation race:** if reaper kills worker while a hibernated agent resumes, first call eats both worker respawn (~3–5s) AND opencode resume (~3–5s). Quantify combined latency; consider pre-warm hooks.

### 10.3 ADV idle worker reaper (#4) — discovery items

- **Inflight guard exhaustiveness:** `ListWorkflowExecutions(Running)` covers active workflows; verify it also catches activities mid-flight, signal handlers, in-flight queries. Missing dimension = worker shutdown mid-work.
- **Lock-handover under 8-agent load (P7):** confirm rapid lock-handover when the worker-lock owner exits while peer opencode sessions immediately reclaim. Read of `worker-lock.ts` suggests reclaim handles PID liveness correctly; needs validation under real concurrent load.
- **Vision daemon worst-case fanout:** 8 agents × 23 MCP servers = up to 184 parallel calls. `vision_metrics` baseline under simulated load; identify any single-server bottleneck before declaring observability pass complete.

### 10.4 ADV marker reframe (#5) — discovery items

- **`[ADV:WARN]` consumers:** any tooling parses the existing marker? OCA status decode? Dashboard? Migration plan if downstream consumers exist.
- **Worktree topology check:** the green path needs `git worktree list` per detected session; specify cache strategy (every emit? cached for N seconds?) to avoid 8 × `git worktree list` invocations per status update.

### 10.5 Cross-cutting

- **`oca` invoked outside tmux entirely:** CI, scripts, headless flows. Pattern B implication — does `oca run` (if added) bypass tmux? Out of scope for #1 but worth flagging.
- **Multi-version coexistence during rollout:** Pattern A ↔ Pattern B coexistence period. Default-on day one (Q1 resolution) means short coexistence window, but operators on older OCA may have stale `oc-<epoch>-<pid>` sessions when upgrading. Migration script or graceful detection.

| Dashboard SPA stack (React/Vite/etc.) | Already rejected in dashboard architecture research note; same logic applies. |
