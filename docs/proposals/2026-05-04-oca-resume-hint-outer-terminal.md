# S6: Resume Hint Visible in Outer Terminal

**Priority:** SHOULD (S6)
**Date:** 2026-05-04
**Owner:** OCA (`bin/oc`, session lifecycle)
**Status:** Draft

---

## Problem

When a user runs `oc` to launch OpenCode inside a tmux session, then uses `/exit`
to quit, OpenCode prints a resume hint (e.g. `opencode --resume <session-id>`)
to the tmux pane. The tmux session then exits/destroys, taking the pane buffer
with it. The outer terminal — where the user originally ran `oc` — sees nothing.

Result: the user has no way to know how to resume their session.

## Why

- Resume hint is OpenCode's built-in UX for session continuity.
- `bin/oc` wraps OpenCode in tmux for multiplexing but discards the exit output.
- Fix belongs in `bin/oc` (OCA-owned shell wrapper), not in OpenCode core.

## Possible Approaches

| # | Approach | Tradeoff |
|---|----------|----------|
| 1 | `tmux capture-pane -p` before session kill, grep for resume hint, reprint | Fragile parsing; may capture noise |
| 2 | OpenCode writes resume info to known state file on `/exit`; `bin/oc` reads and prints post-tmux | Clean, needs OpenCode hook or plugin |
| 3 | `bin/oc` reads latest session from OpenCode state dir post-exit, prints unconditionally | No OpenCode change needed; always prints even for clean exits |
| 4 | Hybrid: capture-pane + state-file fallback | Most robust, more code |

## Success Criteria

- After `oc` → `/exit`, the outer terminal shows a usable resume command.
- Hint is accurate (matches the session the user just exited).
- No noise for users who don't need resume.

## Out of Scope

- Changing how OpenCode generates the resume hint.
- New tmux plugin or hook infrastructure beyond `bin/oc`.
