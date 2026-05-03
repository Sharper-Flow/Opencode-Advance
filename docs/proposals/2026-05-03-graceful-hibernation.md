# Graceful Opencode Session Hibernation

**Status:** Proposal — ready for `/adv-proposal` · **Date:** 2026-05-03 (updated 2026-05-03 post-reconnaissance)
**Target repo:** `opencodeadvance` (OCA)
**Change ID (suggested):** `gracefulSessionHibernation`
**Parent decision lock:** [`2026-05-03-session-and-resource-architecture.md`](./2026-05-03-session-and-resource-architecture.md) §3.5 + §9 + §10
**Split position:** Change #2 of 7 — depends on #1 shipped (status bar must surface 💤 marker)
**Estimated effort:** 2–4 days
**Prerequisites:** ADV #6 (sync prompt fix), OCA #0 (umbrella plugin install), OCA #1 (Pattern B + status decode).
**Sequence:** Ship after Pattern B (#1) lands.

---

## Adaptation Note (read first)

This proposal carries the **decision-locked** direction from the parent doc. When `/adv-proposal` and `/adv-discover` run on this change, the discovery phase MUST:

1. Re-read parent §3.5.1 (six SIGSTOP blockers) — SIGSTOP/SIGCONT is **rejected** and MUST NOT be revisited as an alternative without new evidence.
2. Re-read parent §9.1 Q12 (mid-tool-call hibernation safety) — hard skip if any inflight tool/workflow is the locked default.
3. Re-read parent §9.2 Q6, Q7 — 60 min default idle, tmux keybind opt-out with status indicator.
4. Re-read parent §10.2 ("OCA hibernation (#2) — discovery items") and treat each item as a discovery objective. Adapt the design to findings; do not skip.
5. Re-verify the empirical finding in parent §3.5.2 risk row 1 (session-id capture via `opencode session list --format json`) before designing capture path.

Discovery findings may refine the design but MUST NOT reverse decision-locked answers without explicit operator re-approval.

---

## TL;DR

When an opencode session has been idle for 60+ minutes (no keystrokes, no inflight tool calls, no Running Temporal workflows for that session), OCA gracefully exits it via `/exit`, captures the session ID via `opencode session list --format json`, and parks the tmux window with a 💤 placeholder. Operator presses a key (e.g. `R`) to resume via `opencode --session <id>`. This is the dominant RAM lever under the 42 GB WSL2 ceiling — projected ~16 GB savings at 50% idle of 40 agents (parent §3.5.2 memory math).

---

## Problem Statement

At 5 projects × 8 agents = 40 concurrent opencode processes, RAM utilization hits 82% of the 42 GB WSL2 cap (~34.4 GB; parent §1.3). Page-cache pressure begins. 6th project pushes to 98% / swap. Most idle agents consume ~600 MB each but contribute zero work — they're parked waiting on user input or watching a long task.

SIGSTOP/SIGCONT process pause was investigated and rejected (six citations across MCP, Temporal, WSL2 ecosystems; parent §3.5.1) — counterparties drop the stopped client's connections, SIGCONT resumes a process whose MCP/Temporal calls then fail silently.

The graceful alternative is supported natively by opencode (`opencode --session <id>` + `--continue`; opencode CLI docs May 2026, parent §3.5.2). Session state is durable in `~/.local/share/opencode/opencode.db` (SQLite). ADV change state is durable in Temporal (per-project external store) independent of any single opencode session. The infrastructure exists; OCA just needs to orchestrate the lifecycle.

---

## Success Criteria

- [ ] OCA detects idle opencode windows using AND of: no keystrokes for ≥60 min, no inflight bash/tool calls (queried via opencode tool-call ledger), no Temporal workflows in `Running` state for that session.
- [ ] On idle detection, OCA sends graceful `/exit` to the opencode pane via `tmux send-keys`.
- [ ] OCA captures the session ID via `opencode session list --format json`, sorting by `updated` desc, matching by `directory == workdir`.
- [ ] tmux window persists post-exit with placeholder shell + status: `💤 hibernated · session <id-short> · press R to resume`.
- [ ] Pressing `R` (or configured keybind) in the placeholder spawns `opencode --session <id>` in the same pane.
- [ ] Operator can opt out per-window via `Ctrl+B H` keybind; status bar shows 🔒 when locked.
- [ ] Idle threshold configurable via `[hibernation].idle_minutes` in stack.toml (default 60).
- [ ] Hibernation is on by default; per-project disable via `[hibernation].enabled = false`.
- [ ] Cold-start latency for resume measured and documented (parent §10.2 discovery item).
- [ ] Graceful exit failure path specified and implemented: SIGTERM after grace period, SIGKILL only after WAL fsync confirmation.

---

## Out of Scope

- SIGSTOP/SIGCONT process-pause hibernation (rejected per parent §3.5.1).
- tmux-resurrect integration (covered by change #3).
- Idle worker reaper in ADV (covered by change #4 in ADV repo; this change composes with it but does not implement it).
- Pre-warm / predictive resume hooks (parent §3.5.2 mentions as option; deferred to follow-up).
- Cross-device hibernation (e.g. resume on a different machine — out of scope for v1; touched by dashboard v2.0 web terminal track).
- Replacement of opencode session DB (parent §7 out-of-scope).

---

## Acceptance Criteria (operator-verifiable)

1. Open an opencode in a project, leave idle 60+ minutes (or temporarily lower threshold for test), confirm window auto-hibernates with 💤 placeholder.
2. Press `R` in the placeholder; opencode resumes with full conversation history from session DB.
3. Mid-tool-call (e.g. running long bash command), threshold elapses → window does NOT hibernate; remains active.
4. With keybind `Ctrl+B H` toggled on, threshold elapses → window does NOT hibernate; status shows 🔒.
5. WSL2 + Windows host sleep cycle survives: hibernated window is still hibernated post-wake; resume works.
6. Memory measurement: at 8 agents × 50% hibernated, RSS reduction matches predicted ~3.2 GB (vs no hibernation) within ±20%.
7. Graceful-exit failure: opencode hung scenario simulated; SIGTERM sent after grace, SIGKILL only after WAL fsync verified.

---

## Constraints

- MUST NOT use SIGSTOP/SIGCONT under any condition.
- MUST query opencode tool-call ledger + Temporal `ListWorkflowExecutions(Running)` before declaring a session idle.
- MUST capture session ID via documented `opencode session list --format json` (no DB scraping).
- MUST persist keep-alive opt-out state per-pane in `$XDG_STATE_HOME/oca/panes/` (discovery item: confirm exact persistence layer).
- MUST handle session DB corruption gracefully — fallback message, no auto-delete.
- Cold-start latency budget: <10 sec for typical session size (target <5 sec; measure and document).

---

## Discovery Agenda (per parent §10.2)

The discovery phase MUST address each of these before design:

1. **Cold-start latency measurement** (partial baseline captured 2026-05-03; full TUI cold-start TBD): empirical baselines from non-TUI subprocess calls against the operator's real 3.4 GB opencode.db:
   - `opencode --version` = 0.61s (Node startup baseline)
   - `opencode session list --format json` = 2.23s (DB I/O ≈ +1.6s)
   - `opencode debug agent adv-claude` = 2.34s (config+plugin parse ≈ +1.7s)
   - **Inferred lower bound for `opencode --session <id>` resume:** ~2.2s minimum. Full TUI cold-start (with session loading + plugin init + UI render) likely **3-5s** matching parent §3.5.2 claim. Discovery phase MUST measure full TUI cold-start in a controlled session and document P50/P95.
2. **Graceful exit failure path:** specify SIGTERM grace period, WAL fsync verification, SIGKILL escalation ladder. Test session DB integrity post-SIGKILL.
3. **Session DB single-writer contention:** measure SQLite write latency at 8 concurrent agents writing to one DB (currently 4 MB WAL). Consider WAL checkpoint cadence.
4. **Power management interaction:** test hibernated tmux windows survive WSL2 + Windows host sleep cycle. Document any failure modes.
5. **Telemetry / opt-out persistence:** persist `Ctrl+B H` keep-alive state per-pane in `$XDG_STATE_HOME/oca/panes/<socket>/<paneId>.json` via the OCA plugin's `atomicWriteJSON`. Schema bump (post-recon, 2026-05-03) adds `KeepAlive bool` and `HibernationState string` (active|hibernated|locked) and `LastActivityTs int64` to the `paneState` struct. Survives tmux server restart because state is on-disk JSON. Coordinate schema bump with OCA #1 (which adds `WindowID/WindowName/ProjectSlug` fields). Plugin install via OCA #0 is hard prerequisite — without the plugin, no pane state is written.
6. **Idle-reaper × hibernation race:** when idle worker reaper (change #4) kills worker while a hibernated agent resumes, first call eats both worker respawn (~3–5s) AND opencode resume (~3–5s). Quantify combined latency; consider pre-warm hook.

---

## Locked Design Direction (from parent §9)

| Aspect | Decision (locked) |
|---|---|
| Mechanism | Graceful `/exit` + `opencode --session <id>` resume |
| SIGSTOP | Rejected; do not implement |
| Idle detection | AND of: no keystrokes, no inflight tools, no Running Temporal workflows for session |
| Idle threshold | 60 min default, `[hibernation].idle_minutes` in stack.toml |
| Opt-out UX | `Ctrl+B H` keybind toggle + 🔒 status indicator |
| Default | On; per-project disable available |
| Session ID capture | `opencode session list --format json`, match by `directory`, sort by `updated` desc |

---

## Risks

| Risk | Mitigation |
|---|---|
| Session DB partial-write on SIGKILL | WAL fsync verification before SIGKILL escalation; discovery item to spec ladder. |
| Cold-start latency exceeds budget | Measurement-first discovery; if >10s, consider session DB compaction or pre-warm. |
| Idle detection misses inflight Temporal work | AND-conjunctive check includes `ListWorkflowExecutions(Running)`; discovery item to verify exhaustiveness. |
| WSL2 sleep/wake breaks hibernated state | Power management discovery item; document workaround if needed. |
| Operator mistakenly resumes wrong session | Session ID + title displayed in 💤 placeholder for verification before resume. |

---

## Companion files to update on archive

- `docs/design/stack-toml-schema.md` — add `[hibernation].idle_minutes`, `[hibernation].enabled`
- `docs/design/cli-surface.md` — document `Ctrl+B H` keybind, 💤 placeholder UX
- `docs/proposals/phases.md` — Phase 8.2 entry
- `docs/notes/` — new note `2026-05-XX-hibernation-cold-start-measurement.md` capturing P50/P95 latency findings
