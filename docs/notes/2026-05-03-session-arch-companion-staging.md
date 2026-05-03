# Session Architecture: Companion Doc + Spec Delta Staging

**Date:** 2026-05-03
**Purpose:** Pre-staged content for the 7-change session-architecture work. The `/adv-apply` phase of each change should pull from this file rather than redrafting from scratch. All content is **draft**; discovery/design phases may refine before application.

**Parent decision lock:** [`../proposals/2026-05-03-session-and-resource-architecture.md`](../proposals/2026-05-03-session-and-resource-architecture.md)

**Related proposals:**
- [`../proposals/2026-05-03-adv-sync-prompt-ref-fix.md`](../proposals/2026-05-03-adv-sync-prompt-ref-fix.md) (#6, BLOCKER)
- [`../proposals/2026-05-03-oca-plugin-install.md`](../proposals/2026-05-03-oca-plugin-install.md) (#0, PREREQ)
- [`../proposals/2026-05-03-pattern-b-session-topology.md`](../proposals/2026-05-03-pattern-b-session-topology.md) (#1)
- [`../proposals/2026-05-03-graceful-hibernation.md`](../proposals/2026-05-03-graceful-hibernation.md) (#2)
- [`../proposals/2026-05-03-tmux-resurrect-warning-polish.md`](../proposals/2026-05-03-tmux-resurrect-warning-polish.md) (#3)
- [`../proposals/2026-05-03-adv-idle-worker-reaper.md`](../proposals/2026-05-03-adv-idle-worker-reaper.md) (#4)
- [`../proposals/2026-05-03-adv-coordinated-session-marker.md`](../proposals/2026-05-03-adv-coordinated-session-marker.md) (#5)

---

## Change #6 — ADV sync-global single-ref prompt fix

**Repo:** ADV plugin (`oc-plugins/advance`). All assets live in ADV repo, not OCA.

**Spec deltas (target spec: existing or new in ADV repo):**

Likely extends an existing ADV spec covering `sync-global.sh` behavior. Discovery phase should locate the matching spec; if none exists, create `.adv/specs/sync-global-prompt-refs/spec.md` in ADV repo.

```markdown
### rq-syncglobal-singleref01: Single-file prompt-ref expansion

`sync-global.sh --fix` produces single-file concatenated prompt parts that
OpenCode's `{file:...}` resolver can interpolate.

**Given** the canonical `adv.md` and `providers/{provider}.md` files exist
under `~/.config/opencode/agent-parts/advance/`
**When** `bash scripts/sync-global.sh --fix` runs
**Then** `~/.config/opencode/agent-parts/advance/adv-{provider}.md` exists for
each configured provider, containing canonical body + provider hint joined

**Given** the new concatenated file exists
**When** `opencode debug agent adv-{provider}` is invoked
**Then** the resolved `prompt` field contains stable section markers from
canonical `adv.md` (`## Step 3: Gate Machine`, `## ADV State Access Policy`,
`## Output Contract`) AND the matching provider hint marker
(`<!-- PROVIDER_HINT:{provider} -->`)
AND does NOT contain `[ADV:PROVIDER_STUB_UNEXPANDED]`

### rq-syncglobal-stalecheck01: Staleness detection

`sync-global.sh --check` flags stale concatenated files relative to canonical
sources.

**Given** canonical `adv.md` was modified after the concatenated
`adv-{provider}.md` was generated
**When** `bash scripts/sync-global.sh --check` runs
**Then** check exits non-zero and reports the staleness via mtime or content
hash compare

**Given** legacy multi-`{file:...}` form is present in `opencode.json`
`agent.adv-{provider}.prompt`
**When** `bash scripts/sync-global.sh --check` runs
**Then** check flags the legacy form as drift

**Given** the post-fix state is correct
**When** `bash scripts/sync-global.sh --check` runs
**Then** check passes cleanly
```

**Companion docs to update (in ADV repo):**

- `docs/provider-agent-assembly.md` § Design Principles point 3 — rewrite to describe single-concatenated-file mechanism
- `docs/provider-agent-assembly.md` § Sync Behavior § Generation step 6 — rewrite the JSON example to single ref
- `docs/provider-adv-smoke-checklist.md` — update verification steps

---

## Change #0 — OCA umbrella plugin install

**Spec deltas:**

Extends existing spec `.adv/specs/oca-plugin-session-tracker/spec.md`. Adds:

```markdown
### rq-opst-plugin-registered01: Plugin registered in opencode.json

The OCA umbrella plugin is registered in the operator's global
`opencode.json` `plugin` array.

**Given** the OCA repository at `~/dev/opencodeadvance` with
`plugins/oca/dist/index.js` built
**When** the install procedure is followed (manual edit or future
`oca apply --target plugins`)
**Then** `~/.config/opencode/opencode.json` `plugin` array contains an
entry for the OCA umbrella plugin

**Given** the OCA plugin is registered
**When** OpenCode is restarted and a new session starts in any tmux pane
with `TMUX_PANE` set
**Then** `~/.local/state/oca/panes/<socket>/<paneId>.json` is created
within 5 seconds of session creation

**Given** the plugin is registered and active
**When** opencode session is deleted
**Then** the corresponding pane state file is removed
```

**Companion docs to update:**

`docs/design/architecture.md` — add a brief operator-facing note under "Plugin Management" or similar:

```markdown
### OCA Umbrella Plugin

`plugins/oca/` contains the OCA-side plugin (TypeScript, bun-built) that runs
inside opencode and provides:

- Per-pane session-state JSON written to `$XDG_STATE_HOME/oca/panes/<socket>/<paneId>.json`
- Watchdog primitive for idle-pane TUI respawn
- Future hooks for hibernation (#2) and Pattern B window tracking (#1)

**Install:** add the absolute path to `plugins/oca` in the operator's
`opencode.json` `plugin` array. Future stack.toml-driven deployment will
register the plugin automatically.
```

`docs/design/cli-surface.md` § `oca pane` section — add a line confirming the
plugin is required for pane state to be written:

```markdown
> **Plugin requirement:** `oca pane restart-tui` reads pane state written by
> the OCA umbrella plugin (`plugins/oca/`). If the plugin is not registered
> in `opencode.json`, the command falls back to `opencode --continue` (loses
> session-id targeting). See change #0 (`installOcaUmbrellaPlugin`) for the
> registration procedure.
```

---

## Change #1 — Pattern B session topology + status decode + smart `oca` entry

**Spec deltas:**

Likely creates new spec `.adv/specs/pattern-b-session-topology/spec.md`.

```markdown
# Pattern B Session Topology

Capability spec for session-per-project tmux topology with multi-window per
project.

## Requirements

### rq-patb-smartentry01: Smart `oca` entry from project root

Running `oca` (no args) inside a git repository attaches to or creates the
project's tmux session.

**Given** the current directory is a git repository with project slug `<slug>`
AND no tmux session named `<slug>` exists on the OCA socket
**When** `oca` is invoked with no args
**Then** a new tmux session named `<slug>` is created with one window named
`trunk`, opencode launched in that window with cwd = project root, and the
operator is attached

**Given** a tmux session named `<slug>` already exists on the OCA socket
**When** `oca` is invoked with no args from a directory inside the same project
**Then** the operator is attached to the existing `<slug>` session without
creating a new window

**Given** the current directory is NOT a git repository
**When** `oca` is invoked with no args
**Then** the command exits with a helpful error suggesting either `cd <repo>`
or `oca session list` (exact wording = discovery item)

### rq-patb-windowperworktree01: Window per ADV worktree

When ADV creates a worktree for a change, OCA opens a corresponding tmux
window in the project session.

**Given** an active OCA session for project `<slug>` and an ADV worktree
created for change `<change-id-slug>`
**When** the worktree creation hook fires (mechanism = discovery item)
**Then** a new window is added to the `<slug>` session with name
`<change-id-slug>` and cwd = the worktree path

**Given** an ADV worktree is deleted
**When** the worktree deletion hook fires
**Then** the corresponding window is closed (or marked for cleanup based on
operator's keep-alive setting per #2)

### rq-patb-statusdecode01: Per-window status decode

The tmux status bar surfaces ADV markers from each window using emoji.

**Given** a project session with N windows, each running opencode emitting
`[ADV:WORK|ATTN|BLOCKED|IDLE]` markers in tab titles
**When** the operator views the tmux status bar
**Then** the bar displays per-window decode in format `0:🟩 1:🟥 2:🟪 ...`
with index + emoji per window, within the bar's render budget at terminal
width ≥ 160 cols

### rq-patb-paneschemabump01: Pane state schema bump

The pane-state JSON schema is extended for window/session linkage with
backward compatibility.

**Given** an existing pane state file with legacy schema (`{sessionID,
directory, ts}`, no `schemaVersion` field)
**When** the OCA Go CLI reads it
**Then** read succeeds, treats as `schemaVersion: 1`, with new fields
absent

**Given** the OCA plugin is writing new state for a pane
**When** the file is created
**Then** it contains `schemaVersion: 2` plus new fields `windowID`,
`windowName`, `projectSlug` populated from the tmux + project context

### rq-patb-escapevalve01: Pattern A escape valve

Operators can disable Pattern B and revert to per-invocation session
creation.

**Given** stack.toml or future env var `[session].mode = "per-invocation"`
**When** `oca` is invoked
**Then** behavior is identical to current Pattern A (each invocation
creates a fresh `oca-<slug>-<n>` session)
```

**Companion doc updates:**

`docs/design/tmux-session-reconnaissance.md` — add a Pattern B section after the existing Pattern A description:

```markdown
## Pattern B: Session-per-Project (post-v1)

Adopted via change `patternBSessionTopology`. Replaces per-invocation Pattern
A as the default while preserving Pattern A as an escape valve.

**Naming convention:** session name = git repo slug (no `oca-` prefix, no
`-<n>` suffix). Examples: `opencodeadvance`, `pokeedge`, `advance`.

**Window structure:** each project session has:

| Window | Name | cwd |
|---|---|---|
| 0 | `trunk` | Project's main checkout |
| 1+ | `<change-id-slug>` (per ADV change) | The change's ADV worktree path |

**Switching:** `Ctrl+B 0–9` for direct window selection; `Ctrl+B w` for
window tree. `oca` from inside the project root smartly attaches.

**Status bar:** Per-window decode `0:🟩 1:🟥 2:🟪 ...` of `[ADV:*]` markers,
rendered within bar width budget.
```

`docs/design/cli-surface.md` — add `oca` (no-arg) entry as the primary surface
under a new "## Smart Entry" section near the top:

```markdown
## `oca` (no args, smart entry)

When run inside a git repository, `oca` (no args) attaches to or creates the
project's tmux session, opening opencode in a `trunk` window if creating.
This is the primary entry point under Pattern B.

| Scenario | Result |
|---|---|
| Inside git repo, no existing project session | Create session named after repo, `trunk` window opens opencode |
| Inside git repo, existing project session | Attach to existing session |
| Outside any git repo | Helpful error: suggests `cd <repo>` or `oca session list` |

`oca session new` and friends remain available for explicit per-invocation
control under the Pattern A escape valve.
```

`docs/design/stack-toml-schema.md` — add `[session].mode` field to the
`[session]` section table:

```markdown
| `mode` | string | `"per-project"` (default), `"per-invocation"` (Pattern A escape valve) |
```

---

## Change #2 — Graceful opencode session hibernation

**Spec deltas:**

New spec `.adv/specs/graceful-session-hibernation/spec.md`.

```markdown
# Graceful Session Hibernation

Capability spec for opencode session hibernation via graceful `/exit` + resume
via `opencode --session <id>`.

## Requirements

### rq-hib-idledetect01: Idle detection conjunction

A pane is considered idle when ALL of the following are true: no keystrokes
for the configured threshold, no inflight bash/tool calls, no Temporal
workflows in `Running` state for the session.

**Given** a pane with `paneState.lastActivityTs` older than
`[hibernation].idle_minutes`
AND `opencode session list --format json` shows no active tool calls for the
session
AND `ListWorkflowExecutions(Running)` returns no workflows tagged with the
session's project ID
**When** the OCA hibernation watcher polls
**Then** the pane is classified `idle` and queued for hibernation

**Given** a pane meets time-idle criteria but has an inflight bash call
**When** the watcher polls
**Then** the pane remains `active` (NOT hibernated)

### rq-hib-gracefulexit01: Graceful exit + session-id capture

OCA exits the opencode session via `/exit` and captures the session ID.

**Given** a pane queued for hibernation
**When** the watcher sends `tmux send-keys -t <pane> '/exit' Enter`
**Then** opencode exits cleanly and persists state to its session DB

**Given** opencode has exited
**When** OCA queries `opencode session list --format json`
**Then** the captured session ID is the most-recent-updated entry whose
`directory` matches the pane's workdir

**Given** opencode is hung and does not respond to `/exit` within the grace
period (default = discovery item, ~30 sec)
**When** the watcher escalates
**Then** SIGTERM is sent; if still hung after additional grace, SIGKILL is
sent only after WAL fsync confirmation on the session DB

### rq-hib-placeholder01: Hibernated pane placeholder

After graceful exit, the tmux window persists with a placeholder shell.

**Given** a pane has been hibernated with captured `sessionID = ses_xxx`
**When** the operator views the window
**Then** the pane shows `💤 hibernated · session ses_xxx · press R to resume`

**Given** the operator presses `R` (or configured keybind) in the placeholder
**When** the keybind fires
**Then** OCA spawns `opencode --session <id>` in the same pane and the
session resumes

### rq-hib-keepalive01: Per-pane keep-alive opt-out

Operators can lock individual panes to prevent hibernation.

**Given** a pane and the operator presses `Ctrl+B H`
**When** the keybind fires
**Then** `paneState.keepAlive` is toggled, the status bar shows `🔒` while
locked, and the watcher skips the pane regardless of idle state
```

**Companion doc updates:**

`docs/design/stack-toml-schema.md` — add `[hibernation]` section:

```markdown
## `[hibernation]`

| Field | Type | Default | Purpose |
|---|---|---|---|
| `enabled` | bool | `true` | Master switch for hibernation |
| `idle_minutes` | int | `60` | Idle threshold before graceful exit |

Idle detection requires AND of: no keystrokes for `idle_minutes`, no
inflight tool calls, no `Running` Temporal workflows for the session.

Per-pane opt-out via tmux keybind `Ctrl+B H` (status indicator: 🔒).
```

`docs/design/cli-surface.md` — add subsection under `oca pane`:

```markdown
### Pane lifecycle (with hibernation)

| State | Display | Triggered by |
|---|---|---|
| Active | (no decoration) | Default while opencode running |
| Hibernated | `💤 hibernated · session <id> · press R to resume` | Idle for `[hibernation].idle_minutes` |
| Locked | `🔒` in status bar | `Ctrl+B H` keybind toggle |

Resume from hibernated state via `R` keypress in the placeholder OR from
dashboard row click (v1.2+).
```

`docs/notes/2026-05-XX-hibernation-cold-start-measurement.md` — placeholder
note for discovery-phase measurement output:

```markdown
# Hibernation Cold-Start Measurement

Captured during `/adv-discover` phase of `gracefulSessionHibernation`.

## Pre-measurement baselines (2026-05-03, partial)

Subprocess overheads against operator's 3.4 GB opencode.db:

| Operation | Time |
|---|---|
| `opencode --version` | 0.61s |
| `opencode session list --format json` | 2.23s |
| `opencode debug agent adv-claude` | 2.34s |

Inferred lower bound for `opencode --session <id>` resume: ~2.2s.

## Full TUI cold-start measurement (TBD)

To be filled in by `/adv-discover`. Procedure:

1. Identify a hibernated session ID via `opencode session list --format json`.
2. Run `time opencode --session <id> --print-logs --log-level INFO 2>&1 | head` in a fresh pane.
3. Record P50, P95 across 5 runs.
4. Document any cold/warm tier difference.

Target: P95 ≤ 5 sec. If exceeded, consider WAL checkpoint cadence or
session DB compaction.
```

---

## Change #3 — tmux-resurrect + warning polish

**Spec deltas:**

Light — extends existing OCA tmux/theme specs. Likely under
`.adv/specs/phase4-foundation/` or new `.adv/specs/tmux-resurrect-integration/`.

```markdown
### rq-tmuxres-keybind01: Manual save/restore keybinds

OCA's tmux config registers tmux-resurrect with manual save/restore
keybinds.

**Given** OCA's rendered tmux config is loaded
**When** the operator presses `Ctrl+B Ctrl+s`
**Then** tmux-resurrect saves current session state (window layout,
window names, cwd per pane) to disk

**Given** a saved tmux-resurrect state exists
**When** the operator presses `Ctrl+B Ctrl+r`
**Then** tmux-resurrect restores window layout + cwd, but does NOT
auto-launch opencode in restored windows

### rq-tmuxres-noContinuum01: Continuum not installed

tmux-continuum is explicitly not loaded due to WSL2 focus-bug (issue #148).

**Given** OCA's rendered tmux config is loaded
**When** `tmux show-options -g | grep continuum` is run
**Then** no continuum-related options are present

### rq-markerdecode-coordinated01: Forward-compat marker parser

OCA's status-bar renderer recognizes both `[ADV:WARN]` (yellow) and future
`[ADV:COORDINATED]`-class markers (green) without breaking on unknown
`[ADV:*]` shapes.

**Given** ADV emits `[ADV:WARN] Shared-tree conflict: ...`
**When** OCA renders the status bar
**Then** the marker shows in yellow/warn color

**Given** ADV emits `[ADV:PEER_SESSIONS] N peer session(s)` (existing info
marker)
**When** OCA renders the status bar
**Then** the marker shows in green/info color (no warn)

**Given** ADV emits an unknown `[ADV:NEWMARKER] ...`
**When** OCA renders the status bar
**Then** the marker shows in default/neutral color (graceful degradation)
```

**Companion doc updates:**

`docs/design/theme.md` — add a "Marker Decode Color Map" subsection:

```markdown
## Status Bar Marker Decode

OCA's tmux status bar renders ADV-emitted `[ADV:*]` markers using color +
emoji per class:

| Marker class | Emoji | Color | Severity |
|---|---|---|---|
| `[ADV:WORK]` | 🟩 | green | info |
| `[ADV:ATTN]` | 🟥 | red | needs user |
| `[ADV:BLOCKED]` | 💀 | red | doom-loop |
| `[ADV:IDLE]` | ⬜ | gray | idle |
| `[ADV:PEER_SESSIONS]` | (count badge) | green | info |
| `[ADV:WARN]` | ⚠ | yellow | warn |
| Unknown `[ADV:*]` | (neutral) | default | unknown — graceful degrade |
```

`assets/tmux/` — new file `resurrect.tmux.conf` (or vendored sources) — see
proposal §Companion files for path decision (TPM vs vendoring).

`templates/tmux.conf.block.gotmpl` — append the resurrect plugin load block.

---

## Change #4 — ADV idle Temporal worker reaper

**Repo:** ADV plugin. Specs live in ADV repo.

**Spec deltas (target spec in ADV repo):**

Likely extends an existing ADV worker/temporal spec. Discovery should locate
or create at `.adv/specs/idle-worker-reaper/spec.md`.

```markdown
### rq-idlereaper-shutdown01: Idle shutdown after threshold

Worker-lock owner shuts down its Temporal worker after the configured idle
threshold with no Running workflows.

**Given** a project's Temporal worker has been running for ≥
`worker.idle_shutdown_minutes` (default 60) with no inbound tool calls
AND `ListWorkflowExecutions(Running)` returns no workflows for the project's
task queues
AND no activity is mid-execution
**When** the reaper polls
**Then** the worker shuts down via `releaseWorkerLockGraceful` (calls
existing `releaseWorkerLock` after the inflight guard passes); the plugin
remains as a Temporal client only

### rq-idlereaper-respawn01: Respawn on next call

After idle shutdown, the next ADV tool call re-acquires the lock and
respawns the worker.

**Given** a project with no active worker (lock released by reaper)
**When** any opencode session on the project invokes an ADV tool
**Then** `acquireWorkerLock` succeeds (canonical doesn't exist), the worker
is spawned, and the tool call completes within the documented latency
budget (~3-5 sec target)

### rq-idlereaper-inflightguard01: Inflight guard

Reaper MUST NOT shut down worker mid-workflow or mid-activity.

**Given** a project with an active workflow execution in `Running` state
**When** idle threshold elapses
**Then** the reaper does NOT shut down; worker remains active

### rq-idlereaper-hysteresis01: Hysteresis prevents flapping

Reaper shuts down the worker at most once per hour to avoid rapid
on/off cycles.

**Given** worker was reaped at time T
**When** worker is respawned and goes idle again at T+30min
**Then** reaper does NOT shut down again until T+60min minimum
```

**Companion doc updates (in ADV repo):**

- `docs/idle-worker-shutdown.md` — new design note (referenced in proposal §Companion files)
- `plugin/src/temporal/worker-multi.ts` — implementation (not doc)
- `plugin/src/temporal/worker-lock.ts` — extension (`releaseWorkerLockGraceful`)
- `ADV_INSTRUCTIONS.md` — minor note if operator-facing semantics warrant

---

## Change #5 — ADV peer-session topology distinction

**Repo:** ADV plugin.

**Spec deltas (target spec in ADV repo):**

Likely extends `.adv/specs/multi-session-coordination/` or similar. Discovery
should locate.

```markdown
### rq-peertopo-distinct01: Distinct-worktree case (info only)

When ≥2 peers detected, all on distinct worktrees, only the existing info
marker emits.

**Given** 8 OCA sessions in same project, each with opencode in its own ADV
worktree (Pattern B's expected operating point)
**When** peer detection runs
**Then** `[ADV:PEER_SESSIONS] 8 peer session(s) active in this project.`
emits at info level
AND no `[ADV:WARN]` emits

### rq-peertopo-conflict01: Shared-tree case (warn emit)

When ≥2 peers detected on the same working tree, an additional warn marker
identifies the conflict.

**Given** 2 OCA sessions in the same trunk checkout (no worktree switch)
**When** peer detection runs
**Then** `[ADV:PEER_SESSIONS] 2 peer session(s) active in this project.`
emits at info level
AND `[ADV:WARN] Shared-tree conflict: PIDs <a,b> on tree <path>. Git
operations from any of those affect all.` emits at warn level

### rq-peertopo-cache01: Topology cache TTL

`git worktree list` invocations are cached to avoid fanout under heavy
peer-detection load.

**Given** ≥2 peers detected
**When** peer detection runs 5 times within `coordination_topology_cache_ttl_seconds`
(default 30)
**Then** `git worktree list` is invoked at most once per unique workdir
across the cache window (cache hits prevent fanout)
```

**Companion doc updates (in ADV repo):**

- `ADV_INSTRUCTIONS.md § Multi-Session Coordination` — document new warn marker
- `docs/peer-session-topology.md` — new design note

---

## /adv-apply Phase Usage

For each change, the apply phase should:

1. **Reference this staging file** for spec delta + companion doc starting points.
2. **Refine during /adv-discover** — discovery phase findings may invalidate or refine these drafts. This file is a **starting point**, not a contract.
3. **Apply spec deltas** via the change's deltas.json (ADV mechanism). Discovery + design phases own the final spec content.
4. **Apply companion doc updates** as part of `/adv-apply` task implementation. Each change's "Companion files to update on archive" header lists the canonical target paths.

**Bonus, all changes:** update `docs/proposals/phases.md` with archive-time notes (lines reflecting "shipped" status) per the existing pattern (see Phase 0–8 entries).

---

## Stop Point

After this file is staged, the only remaining work that does not require ADV #6 to ship is:

- Light cleanup (e.g. minor proposal polish if review feedback comes in)
- Reading ADV repo source for additional reaper/topology context
- Verifying assumptions against changing OCA codebase (git status / test baseline)

Everything **gate-driven** (`/adv-proposal`, `/adv-discover`, `/adv-design`, `/adv-prep`, `/adv-apply`, `/adv-review`, `/adv-harden`, `/adv-archive`) **requires a fresh OpenCode session with the working ADV agent (post-#6)** to run cleanly. Workaround C in this session is sufficient for *reading* and *drafting* but is not a substitute for the gate machine.
