# tmux-resurrect Manual Persistence + Concurrency Warning Posture Polish

**Status:** Proposal — ready for `/adv-proposal` · **Date:** 2026-05-03
**Target repo:** `opencodeadvance` (OCA)
**Change ID (suggested):** `tmuxResurrectWarningPolish`
**Parent decision lock:** [`2026-05-03-session-and-resource-architecture.md`](./2026-05-03-session-and-resource-architecture.md) §3.6 + §3.8 + §9
**Split position:** Change #3 of 5 — small composable polish, can run in parallel with #2
**Estimated effort:** ~1 day
**Sequence:** Ship after Pattern B (#1) lands; can run in parallel with hibernation (#2).

---

## Adaptation Note (read first)

This proposal carries the **decision-locked** direction from the parent doc. When `/adv-proposal` and `/adv-discover` run on this change, the discovery phase MUST:

1. Re-read parent §3.6 (tmux persistence layer model) — state durability is already three layers deep (Temporal, opencode session DB, ADV external state); tmux-resurrect is a layout convenience, not a state-durability foundation.
2. Re-read parent §3.6 — tmux-continuum is **deferred** due to WSL2 focus-bug ([tmux-continuum#148](https://github.com/tmux-plugins/tmux-continuum/issues/148)); only manual `tmux-resurrect` is in scope.
3. Re-read parent §3.8 (concurrency warning posture) and §9.2 Q8 — the marker reframe itself is **deferred to ADV** (change #5); this OCA change only handles status-bar polish for whatever marker ADV emits.

Discovery findings may refine the design but MUST NOT reverse decision-locked answers without explicit operator re-approval.

---

## TL;DR

Bundle two small composable polish items into one change:

1. **tmux-resurrect (manual):** ship `Ctrl+B Ctrl+s` save / `Ctrl+B Ctrl+r` restore for window layout + working directories. Manual only; no auto-save (continuum deferred per WSL2 focus-bug).
2. **Concurrency warning posture polish:** OCA-side rendering of whatever ADV emits — when ADV ships the `[ADV:COORDINATED]` marker (change #5), OCA decodes it as green/info in the status bar instead of yellow/warn. Until then, OCA passes ADV's existing `[ADV:WARN]` through as-is.

---

## Problem Statement

**tmux-resurrect:** Pattern B durable project sessions are valuable, but a tmux server crash, host restart, or WSL2 reboot loses window layouts. Operators have to rebuild the project session window topology by hand. tmux-resurrect is mature, well-tested, and restores window layouts + cwd. The known WSL2 limitation only affects continuum (auto-save); manual save works correctly.

**Warning posture:** ADV emits `[ADV:WARN] Concurrent OpenCode sessions detected` today. At Pattern B + 8 agents, this fires constantly and the operator habituates to ignoring it. The marker semantic itself is ADV-domain (change #5), but OCA owns the status-bar rendering and can prepare the decode logic for the new marker without waiting for ADV to ship.

---

## Success Criteria

- [ ] tmux-resurrect installed and configured by `oca apply` (TPM-managed or vendored).
- [ ] Default keybinds: `Ctrl+B Ctrl+s` save, `Ctrl+B Ctrl+r` restore.
- [ ] Save captures: window layouts, window names, cwd per pane, no opencode in-memory state (parent §3.6: that's covered by opencode session DB + change #2 hibernation resume).
- [ ] Restore reproduces the project session structure (1 trunk + N change worktree windows) but does NOT auto-launch opencode in each window — operator decides per-window.
- [ ] OCA status-bar renderer recognizes both `[ADV:WARN]` (existing, yellow) and `[ADV:COORDINATED]` (future from change #5, green) marker classes.
- [ ] When ADV ships `[ADV:COORDINATED]`, OCA renders it without warning visual; until then, `[ADV:WARN]` rendering unchanged.
- [ ] tmux-continuum is **NOT** installed (parent §3.6 + §9 lock).

---

## Out of Scope

- tmux-continuum auto-save (deferred per WSL2 focus-bug).
- OCA-native polling alternative to continuum (parent §3.6 mentions `oca-resurrect` daemon as future option; out of scope for v1).
- Auto-relaunch of opencode in restored windows (operator chooses per-window).
- ADV-side marker reframe (covered by change #5 in ADV repo).
- Cross-host / cross-device session restore (parent §7 out-of-scope).

---

## Acceptance Criteria (operator-verifiable)

1. After running `oca apply`, `Ctrl+B Ctrl+s` saves current tmux state to disk.
2. After tmux server restart (or WSL2 reboot), `Ctrl+B Ctrl+r` restores the project session structure with windows, names, and cwds intact.
3. Restored windows show placeholder shells in correct cwds; opencode is not auto-launched (operator runs it manually or uses `/exit`-then-resume from change #2).
4. tmux status bar correctly displays `[ADV:WARN]` markers in yellow today; `[ADV:COORDINATED]` in green when ADV change #5 lands (forward-compatible parsing).
5. tmux-continuum is not present in the rendered tmux config; verified by `tmux show-options -g | grep continuum` returning empty.

---

## Constraints

- MUST NOT install tmux-continuum.
- MUST NOT auto-launch opencode in restored windows (avoids conflicting with change #2 hibernation resume UX).
- Status-bar marker decode MUST be forward-compatible: handle unknown `[ADV:*]` markers gracefully (default rendering, not crash).
- TPM (Tmux Plugin Manager) installation MUST be optional — vendoring is acceptable if TPM is not desired.

---

## Discovery Agenda

The discovery phase MUST address:

1. **TPM vs vendoring:** OCA already manages tmux config; vendoring tmux-resurrect into `assets/tmux/` may be cleaner than TPM dependency. Decide.
2. **Restore + hibernation interaction:** restored windows have placeholder shells; if change #2 hibernation captured a session ID for the original window, can restore re-attach? Probably not (session ID is per-pane state lost on tmux server crash); document the limitation.
3. **`[ADV:WARN]` consumer audit:** confirm any OCA tooling that currently parses `[ADV:WARN]` does not break when ADV ships `[ADV:COORDINATED]`. Forward-compat test.
4. **WSL2 focus-bug evidence:** verify [tmux-continuum#148](https://github.com/tmux-plugins/tmux-continuum/issues/148) is still open before final exclusion; if closed/fixed, revisit scope.

---

## Locked Design Direction (from parent §9)

| Aspect | Decision (locked) |
|---|---|
| tmux-resurrect | Ship manual save/restore |
| tmux-continuum | Do not ship (WSL2 focus-bug) |
| Marker reframe | OCA renders forward-compatibly; ADV owns the marker semantic (change #5) |

---

## Risks

| Risk | Mitigation |
|---|---|
| tmux-resurrect dependency churn | Vendor option in discovery if TPM is brittle. |
| Restored window cwd points to deleted worktree | Resurrect handles missing-dir gracefully (falls back to home); worst case operator restarts the window. |
| Operator confuses restore with hibernation resume | Documentation makes the distinction explicit (parent §3.6 layered model); keybind reference clarifies. |

---

## Companion files to update on archive

- `assets/tmux/` — add `resurrect.tmux` config block or vendored sources
- `templates/tmux.conf.block.gotmpl` — add resurrect plugin load
- `docs/design/theme.md` — document marker decode color map
- `docs/proposals/phases.md` — Phase 8.3 entry
