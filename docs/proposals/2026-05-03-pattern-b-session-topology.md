# Pattern B Session Topology + Per-Window Status Decode + Smart `oca` Entry

**Status:** Proposal — ready for `/adv-proposal` · **Date:** 2026-05-03 (updated 2026-05-03 post-reconnaissance)
**Target repo:** `opencodeadvance` (OCA)
**Change ID (suggested):** `patternBSessionTopology`
**Parent decision lock:** [`2026-05-03-session-and-resource-architecture.md`](./2026-05-03-session-and-resource-architecture.md) §9 + §10
**Split position:** Change #1 of 7 — unblocks #2, #3
**Estimated effort:** 3–6 days
**Prerequisites:** ADV #6 (sync prompt fix) shipped; OCA #0 (umbrella plugin install) shipped — pane state writes depend on the OCA plugin being loaded.
**Sequence:** Ship after #6 and #0. All other OCA Pattern B work depends on this.

---

## Adaptation Note (read first)

This proposal carries the **decision-locked** direction from the parent doc. When `/adv-proposal` and `/adv-discover` run on this change, the discovery phase MUST:

1. Re-read parent §9.1 (Q11, Q12, Q13) and §9.2 (Q1–Q10) — do not relitigate locked decisions.
2. Re-read parent §10.1 ("OCA Pattern B (#1) — discovery items") and treat each item as a discovery objective. Adapt the design to findings; do not skip.
3. Re-read parent §3.1 (industry consensus) and §3.1.4 (Pattern B advantages) for context.
4. Re-verify the empirical findings recorded in parent §9.1 Q11 (opencode + ADV use same project-id) before any project-id-related code change.

Discovery findings may refine the design but MUST NOT reverse decision-locked answers without explicit operator re-approval.

---

## TL;DR

Replace OCA's current "one tmux session per `oca` invocation" topology (Pattern A) with **session-per-project + window-per-worktree** (Pattern B). Add per-window ADV status marker decode in the tmux status bar so 8+ concurrent agents are scannable without switching. Reshape the CLI surface so `oca` (no args) from inside a project root smartly attaches-or-creates the project session.

---

## Problem Statement

At the operator's actual workload (8+ concurrent agents per project, 4–5 active projects on a 42 GB WSL2 ceiling), Pattern A produces:

- `tmux ls` clutter: `oca-advance-0..3, oca-scratch-0..1, oca-pokeedge-0..2` instead of intentional project boundaries.
- No way to scan agent state across 8 concurrent windows without clicking each Windows Terminal tab.
- Switching agents on the same project requires hunting across WT tabs (Q13b grounding: this is the current pain).
- Each `oca` launch incurs Vision/Temporal singleton init contention per session, not per project.
- Session lifecycle is "throwaway wrapper" not "durable project context" — defeats reaping clarity, dashboard rendering, and per-project tmux config.
- Industry consensus (tmuxinator, tmuxp, Anthropic Agent Teams docs, multi-agent AI tmux trend, Dec 2025–Jan 2026) is unambiguously session-per-project. Pattern A is not LBP.

---

## Success Criteria

- [ ] `oca` invoked from inside a git project root attaches to the project session if it exists, else creates a session named after the project with a single `trunk` window running opencode.
- [ ] Each project session supports multiple windows: 1 `trunk` (cwd = main checkout) + N change windows (cwd = each active ADV worktree, name = ADV change-id slug).
- [ ] Window auto-creation hook fires when ADV creates a worktree, opening the corresponding window in the project session.
- [ ] tmux status bar renders per-window decode of `[ADV:WORK/ATTN/BLOCKED/IDLE]` markers using emoji (`🟩 🟥 🟪 ⬜ 🟨 🟦`), format: `0:🟩 1:🟥 2:🟪 ...`.
- [ ] Top tmux status row replaces decorative branding/filler with an orientation strip: project/session, branch or worktree, active ADV change/task, gate/phase/progress, health, and time.
- [ ] `[session].mode = "per-invocation"` in stack.toml restores Pattern A as escape valve.
- [ ] Default behavior is Pattern B day one (no opt-in flag required).
- [ ] All existing OCA Phase 4 session lifecycle commands (`oca session list`, `oca session attach`) continue to work; add `oca` as the smart entry point.
- [ ] `tmux ls` shows project names (`advance`, `opencodeadvance`), not slugs (`oca-advance-0`).
- [ ] Closing all windows via `/exit` from each opencode auto-destroys the session; no `oca session kill` required for normal flow.
- [ ] Pattern B passes the WSL2 + Windows Terminal force-close orphan-protection test (parent §3.1.3).

---

## Out of Scope

- Graceful hibernation (covered by change #2; this change must surface 💤 marker in status bar but does not implement detection or `/exit` automation).
- Idle worker reaper (covered by change #4 in ADV repo).
- `[ADV:WARN]` → `[ADV:COORDINATED]` marker reframe (covered by change #5 in ADV repo; this change reads existing markers as-is).
- tmux-resurrect integration (covered by change #3).
- OCA dashboard rendering changes (separate track per `docs/notes/2026-05-01-dashboard-architecture-research.md`).
- Pattern A removal (kept indefinitely as escape valve per Q1 resolution).
- opencode `--session <id>` resume (touched by change #2 only).
- Cross-host federation (parent §7 out-of-scope).
- WSL2 memory bump (parent §7 out-of-scope).

---

## Acceptance Criteria (operator-verifiable)

1. `cd ~/dev/opencodeadvance && oca` opens a tmux session named `opencodeadvance` with a `trunk` window running opencode in `~/dev/opencodeadvance`.
2. From inside that opencode, running `/adv-prep <change-id>` and creating a worktree results in a new tmux window named after the change-id slug, cwd = worktree path, running opencode.
3. `tmux -L oca ls` shows session names matching git project names, no `-N` numeric suffixes.
4. With 8 windows in one session, the tmux status bar displays per-window emoji + index for all 8 within the bar's width budget at typical terminal width (≥160 cols).
5. Setting `[session].mode = "per-invocation"` in stack.toml and restarting `oca` restores per-invocation session creation indistinguishably from current Pattern A.
6. Force-closing the Windows Terminal tab does not orphan the opencode process (tmux server intercepts SIGHUP; behavior identical to Pattern A per parent §3.1.3).

---

## Constraints

- MUST NOT modify ADV plugin behavior; window auto-create hook calls into OCA from ADV (or via filesystem watch on `~/.local/share/opencode/worktree/<projectId>/change/`).
- MUST preserve OCA Phase 4 session lifecycle commands.
- MUST emit pane state JSON in existing `$XDG_STATE_HOME/oca/panes/` schema (extension allowed for window/session linkage; backwards-compatible reads). Schema bump must include `SchemaVersion` field for forward/backward compat. Proposed schema bump (recorded post-reconnaissance, 2026-05-03):
  ```go
  type paneState struct {
      SchemaVersion int    `json:"schemaVersion"`            // NEW: 2 (legacy = absent or 1)
      SessionID     string `json:"sessionID"`                // unchanged
      Directory     string `json:"directory"`                // unchanged
      Ts            int64  `json:"ts"`                       // unchanged

      // Pattern B additions:
      WindowID    string `json:"windowID,omitempty"`         // tmux %@ id
      WindowName  string `json:"windowName,omitempty"`       // change-id slug or "trunk"
      ProjectSlug string `json:"projectSlug,omitempty"`      // project name = session name
  }
  ```
  Hibernation fields (`HibernationState`, `KeepAlive`, `LastActivityTs`) added by OCA #2; this change leaves room for them but does not implement.
- MUST coordinate schema bump with OCA umbrella plugin (`plugins/oca/src/watchdog.ts`) — both writer (plugin) and reader (Go CLI) must understand the new fields. Plugin install is a hard prerequisite via OCA #0.
- Status bar render budget: stay ≤ 80 chars in right-aligned region at 160-col terminal.
- Project name derivation: prefer `git rev-parse --show-toplevel` basename; fall back to `oca` config override.
- Window naming: ADV change-id slug verbatim (no truncation); long names tolerated by tmux but truncated in status bar render only.

---

## Discovery Agenda (per parent §10.1 + post-reconnaissance additions)

The discovery phase MUST address each of these before design:

1. **Smart `oca` get-or-create semantics:**
   - Inside any git repo, no existing OCA session → create + attach (happy path).
   - Inside any git repo, existing OCA session → attach.
   - Outside any git repo → list existing sessions OR error with helpful message? Decide.
   - Inside a git repo that ADV doesn't recognize as a project → still create OCA session? Depends on whether ADV project-id is required.
2. **Pattern A escape valve scope:** stack.toml `[session].mode = "per-invocation"` — global, per-project, both? Note: operator currently has no `stack.toml` (only `stack.example.toml`) — decide whether escape valve depends on stack-driven config or is also overridable via env var / CLI flag.
3. **Window auto-creation hooks:** new OCA tool that ADV plugin calls? Filesystem watch on worktree directory? Both? Specify race condition handling for simultaneous worktree creation.
4. **Trunk window cwd handling:** what if the operator is on a feature branch (not default branch) in trunk checkout when first launching `oca`? Trunk window cwd = wherever HEAD points; do not force-checkout default branch.
5. **Per-action blast radius:** confirm whether any operator-facing destructive op (e.g. forced session close, accidental window kill) needs a confirmation prompt at ≥2 windows.
6. **Existing CLI surface integration (post-recon, 2026-05-03):** `cmd/oca/session.go` already ships `newSessionNewCmd`, `newSessionAttachCmd`, `newSessionSwitchCmd`, `newSessionKillCmd`, `newSessionKillallCmd`, `newSessionRestartCmd`, `newSessionReapCmd`. Pattern B is mostly an *extension* of these, not a rewrite. Decide: add `--project` flag to `newSessionNewCmd` for get-or-create, or new top-level `oca` (no-arg) command that wraps "session new or attach by project"?
7. **Existing pane-state schema integration (post-recon):** current schema in `cmd/oca/pane.go:18-23` is `{sessionID, directory, ts}`. Plugin writer in `plugins/oca/src/watchdog.ts:55` calls `atomicWriteJSON`. Coordinate Go-side reader + plugin-side writer for new fields.
8. **Existing watchdog integration (post-recon):** plugin already supports a watchdog that respawns TUI on idle (different goal from hibernation #2 which exits + parks). Decide whether Pattern B's window state needs to coordinate with the existing watchdog's pane-state additions (e.g. `watchdog: { enabled, bump_count, status }` field already in plugin's `PaneState`).

---

## Locked Design Direction (from parent §9)

| Aspect | Decision (locked) |
|---|---|
| Topology | Pattern B: session per project, multiple windows per session |
| Window naming | ADV change-id slug verbatim; trunk window = `trunk` |
| Status bar | Per-window decode `0:🟩 1:🟥 ...` with index + emoji |
| Entry point | `oca` (no args) from project root = smart get-or-create |
| Rollout | Default-on day one; `[session].mode = "per-invocation"` escape valve |
| Closure lifecycle | `/exit` from each opencode; session auto-destroys when last window closes |
| Project-id basis | git root-commit SHA (matches both opencode + ADV; verified 2026-05-03) |

---

## Risks

| Risk | Mitigation |
|---|---|
| Operator muscle memory expects per-WT-tab session | Default-on with escape valve gives 1-config rollback path. |
| Status bar overflows at 8+ windows | Render budget enforced; long change-id slugs truncated in display only. |
| ADV worktree creation race vs OCA hook | Discovery item; specify deterministic ordering or idempotency. |
| Pattern A consumers (scripts, CI) break | Pattern A escape valve preserves; document migration in CHANGELOG. |

---

## Companion files to update on archive

- `docs/design/tmux-session-reconnaissance.md` — add Pattern B section
- `docs/design/stack-toml-schema.md` — add `[session].mode`
- `docs/design/cli-surface.md` — `oca` smart entry, retire `oca session new <project>` from primary docs
- `docs/proposals/phases.md` — Phase 8.1 entry
