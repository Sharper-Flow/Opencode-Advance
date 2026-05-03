# OCA Umbrella Plugin Installation (Prerequisite for #1, #2)

**Status:** Proposal — ready for `/adv-proposal` · **Date:** 2026-05-03
**Target repo:** `opencodeadvance` (OCA)
**Change ID (suggested):** `installOcaUmbrellaPlugin`
**Parent decision lock:** [`2026-05-03-session-and-resource-architecture.md`](./2026-05-03-session-and-resource-architecture.md) — discovered as prerequisite during pre-work reconnaissance (2026-05-03)
**Split position:** **Change #0 of 7 — PREREQUISITE for #1, #2.** Must ship before Pattern B (#1) or hibernation (#2) work can rely on pane state.
**Estimated effort:** 1–2 hours
**Sequence:** Ship after ADV #6 (sync prompt fix). Ship before OCA #1.

---

## Adaptation Note (read first)

This proposal was discovered during reconnaissance prep work for the OCA 5-change split. It is not in the parent decision-lock doc but is a **prerequisite** for OCA #1 and #2. When `/adv-proposal` and `/adv-discover` run on this change, the discovery phase MUST:

1. Re-verify the empirical finding: the OCA umbrella plugin (`plugins/oca/`) source exists in the OCA repo but is NOT registered in the operator's `opencode.json` plugin array as of 2026-05-03.
2. Confirm the operator has no `stack.toml` today — they're running pre-stack-migration with hand-edited `opencode.json`. Decide whether this change also pulls forward stack.toml adoption or stays manual.
3. Decide between two install paths (Option A vs B in §Resolution Options); Option A (manual edit) is the recommended LBP fix for an immediate prerequisite; Option B (stack.toml-driven) is broader scope.

---

## TL;DR

The OCA umbrella plugin (`plugins/oca/`, package `@sharperflow/oca-plugin`) provides session-pane state tracking and the watchdog primitive that OCA changes #1 (Pattern B) and #2 (hibernation) build on. Plugin source exists in the OCA repo but is not registered in the operator's `opencode.json` plugin array — `~/.config/opencode/plugins/` and `~/.local/state/oca/panes/` both don't exist for this user. Add the plugin to opencode.json, verify pane state starts being written, and unblock #1/#2.

---

## Problem Statement

### Verified empirical state (2026-05-03)

- Plugin source: `~/dev/opencodeadvance/plugins/oca/{src/index.ts, dist/index.js, package.json}` — exists, package name `@sharperflow/oca-plugin`, version 0.1.0.
- `~/.config/opencode/opencode.json` `plugin` array contains 5 entries (ADV, claude-max, morph-fast-apply, vision, opencode-openai-codex-auth) — **NOT including the OCA umbrella plugin**.
- `~/.config/opencode/plugins/` directory does not exist → no opencode-managed install.
- `~/.local/state/oca/panes/` directory does not exist → no pane state being written.
- `~/dev/opencodeadvance/stack.toml` does not exist → operator has no stack-driven config (only `stack.example.toml` reference).
- `oca` Go CLI is presumably installed (consistent with operator workflow), but the **plugin** half of OCA — the part that runs inside opencode and writes pane state — is not loaded.

### Impact on OCA #1 + #2

- **OCA #1 (Pattern B):** the proposed pane-state schema bump (`WindowID`, `WindowName`, `ProjectSlug`) extends the schema in `cmd/oca/pane.go`. But the writer is the plugin (`plugins/oca/src/watchdog.ts`). If the plugin isn't loaded, no state is being written, and the OCA Go CLI's `oca pane restart-tui` falls back to `opencode --continue` (verified in pane.go:67-69).
- **OCA #2 (hibernation):** the proposed `Ctrl+B H` keep-alive opt-out persists per-pane state. Same plugin dependency. Without the plugin, hibernation has no per-pane state to read/write.

Both #1 and #2 implicitly assume the plugin is loaded. This change makes that assumption explicit and addresses it before downstream work begins.

---

## Success Criteria

- [ ] `~/.config/opencode/opencode.json` `plugin` array contains an entry pointing to the OCA umbrella plugin (path or npm-style identifier — discovery item).
- [ ] After restart, the plugin loads cleanly (no error in opencode session start).
- [ ] When operator launches an OCA tmux session and starts opencode in a pane, `~/.local/state/oca/panes/<socket>/<paneId>.json` is created with the canonical `paneState` schema (`sessionID`, `directory`, `ts`).
- [ ] `oca pane restart-tui` (existing command) reads the state file and resumes via `opencode -s <session-id>` (not the `--continue` fallback).
- [ ] No regression in existing 5 plugins (ADV, claude-max, morph-fast-apply, vision, codex-auth).
- [ ] Install procedure is documented + idempotent (re-running doesn't break anything).

---

## Out of Scope

- Migrating operator to a `stack.toml`-driven config (could be done later as separate change).
- Implementing `oca apply` against a `stack.toml` for plugin registration (separate scope; this change is a one-shot fix).
- Schema bump for Pattern B (covered by OCA #1).
- Watchdog activation / opt-in defaults (covered by hibernation #2).
- New OCA plugin features (only existing plugin install).

---

## Acceptance Criteria (operator-verifiable)

1. Verify pre-state: `python3 -c "import json; print(len(json.load(open('/home/jrede/.config/opencode/opencode.json'))['plugin']))"` returns 5; `~/.local/state/oca/panes/` does not exist.
2. Apply the install (per chosen Option A or B from §Resolution Options).
3. Verify post-state: plugin array now contains 6 entries including OCA umbrella plugin path.
4. Restart OpenCode entirely.
5. Open a fresh OCA session via `oca` (or `oca session new`); start opencode in the pane.
6. Verify `~/.local/state/oca/panes/<socket>/<paneId>.json` exists and contains valid JSON matching the `paneState` schema.
7. Run `oca pane restart-tui --force` from inside the pane; verify it resumes via `opencode -s <session-id>` (not `--continue`).
8. No regression: spawn a test ADV change via `/adv-status` and confirm ADV plugin still works correctly.

---

## Constraints

- MUST NOT remove or modify any of the 5 existing registered plugins.
- MUST NOT break any existing opencode session.
- MUST be reversible — operator can remove the plugin entry and revert to current state if needed.
- MUST handle JSONC comments in opencode.json correctly (current file has commented sections).
- Install path SHOULD prefer absolute path to local checkout (matches pattern of other 5 plugins) over npm package install (which would require publishing the OCA plugin first).

---

## Resolution Options Considered

| Option | Approach | Verdict |
|---|---|---|
| **A — Manual opencode.json edit** | Add `"/home/jrede/dev/opencodeadvance/plugins/oca"` to the `plugin` array. Match existing pattern (other 5 plugins all use absolute paths). 5-minute fix. | **Recommended** for the immediate prerequisite. |
| B — Pull forward stack.toml adoption | Create `~/dev/opencodeadvance/stack.toml` declaring all 6 plugins; run `oca apply` to render `opencode.json`. Larger scope (operator's full config becomes stack-driven). Overlaps with v1.0 migration story. | **Defer** — out of scope for this prerequisite change. File as separate post-v1 work. |
| C — Publish + npm install | Publish `@sharperflow/oca-plugin` to npm; install via `opencode plugin <module>` command. Requires publishing infrastructure decisions. | **Defer** — too much scope for a prerequisite. |
| D — Skip — redesign #1/#2 to not need plugin | Pattern B + hibernation could in theory work via tmux-only state. Major redesign; loses pane-state JSON benefits + watchdog primitive. | Rejected — defeats the point of the OCA plugin existing. |

---

## Discovery Agenda

The discovery phase MUST address:

1. **Build state:** is `plugins/oca/dist/index.js` current relative to `src/`? If not, run `bun build` first. Verify build is reproducible.
2. **Plugin path style:** match other 5 plugins (absolute path to local checkout). Confirm by listing existing plugin entries.
3. **JSONC comment preservation:** if opencode.json has comments, ensure JSON edit doesn't strip them. Use a JSONC-aware tool or manual edit with verification.
4. **No-stack-toml acknowledgment:** confirm operator is OK with manual opencode.json edit for this change (vs pulling forward stack.toml adoption). Document decision.
5. **Watchdog activation defaults:** plugin checks `OCA_WATCHDOG_ENABLED=1` env var. Decide whether to enable by default at install time, or leave opt-in. Most likely defer to hibernation #2.

---

## Risks

| Risk | Mitigation |
|---|---|
| Plugin load failure breaks opencode session start | Verify plugin loads cleanly in a test session before declaring success; revert on failure. |
| JSONC comments stripped during edit | Use comment-preserving edit (e.g. manual Edit tool with exact strings) or back up + diff. |
| Plugin version mismatch with @opencode-ai/plugin host API | Plugin pkg.json declares `^1.0.0` peer dep — verify host's plugin API version matches. |
| Operator runs `oca apply` later (with new stack.toml) and overwrites manual edit | Document the manual edit so future stack.toml migration includes the OCA plugin entry. |

---

## Composition with other changes

- **Prerequisite for OCA #1 (Pattern B)** — pane state schema bump assumes plugin is writing state.
- **Prerequisite for OCA #2 (hibernation)** — keep-alive persistence assumes plugin.
- **Independent of OCA #3, ADV #4, ADV #5, ADV #6.**

---

## Companion files to update on archive

- Operator's `~/.config/opencode/opencode.json` (one-shot edit)
- `docs/design/architecture.md` — note OCA umbrella plugin install in operator-facing docs
- `docs/proposals/phases.md` — record install as prerequisite for the post-v1 session-architecture work
- Optional: `stack.example.toml` already references the plugin? Verify and update if not.
