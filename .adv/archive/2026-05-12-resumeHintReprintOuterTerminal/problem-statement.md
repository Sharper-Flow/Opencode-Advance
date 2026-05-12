## Problem Statement

When a user attaches an OCA-managed tmux session containing an opencode pane and exits opencode via `/exit`:

1. opencode prints a resume hint (`opencode --session <id>` or similar) into the tmux pane
2. The pane closes (opencode process exits)
3. tmux destroys the session (single-window auto-cleanup)
4. The outer terminal (shell that ran `oca session attach <name>`) returns control but the resume hint is gone with the destroyed pane buffer

Result: the user has no actionable way to know how to resume the conversation they just exited. They forget the session ID, lose conversational continuity, and start a fresh session — accumulating session-DB rows + losing context.

This is GitHub issue Sharper-Flow/Opencode-Advance#22 (S6 in the SHOULD queue), with associated proposal at `docs/proposals/2026-05-04-oca-resume-hint-outer-terminal.md`.

### Acceptance criteria (from issue #22)

- [ ] After session exit, the outer terminal shows a usable resume command
- [ ] Hint is accurate (matches the session the user just exited)
- [ ] Works in both tmux and non-tmux environments
- [ ] Hint includes session name and project path
- [ ] No hint emitted when session was explicitly killed (kill is intentional cleanup)

### Verified current state (2026-05-12)

- `internal/session/session.go::Create` creates detached tmux session at workingDir; does not launch opencode (operator runs opencode inside attached pane manually OR via openchad today)
- `lib/session_lifecycle.sh::oca_session_attach` uses `exec tmux -L oca attach -t <name>` — shell is replaced; no post-exit hook is currently possible from this path
- `opencode session list --format json` returns sessions with `{id, title, directory, updated, projectId}` per parent doc §3.5.2 — well-defined query surface for "latest session in this project"
- `~/.local/share/opencode/opencode.db` is durable SQLite session store

### Two other items originally bundled into this "quick bug-fix sweep" Change A were verified already-shipped during discovery 2026-05-12:

- Sharper-Flow/Opencode-Advance#11 (oca maintain self-blocks): shipped in `internal/maintain/session_gate.go` with distinct `CALLER_OPENCODE_PROCESS` vs `ACTIVE_OPENCODE_PROCESS` codes + tests. Closed.
- Sharper-Flow/Opencode-Advance#21 (legacy ADV state doctor warning): shipped in `internal/health/advance.go::checkLocalADVState` with 4 dedicated test cases. Closed.

Change A is therefore single-item.

## Why this matters for v1.0

The operator's cutover sequence relies on tmux+opencode session lifecycle being usable. Today on openchad, opencode's resume hint is eaten by tmux pane destruction; same UX bug will manifest on OCA's Pattern B post-cutover unless we fix it.

Without this fix, post-cutover sessions accumulate as orphaned opencode session DB entries because users forget the IDs and start fresh sessions. This is exactly the "stale session debt" problem OCA's session-debt scanner already tracks.

## Out of scope

- Changing how OpenCode generates the resume hint
- New tmux plugin or hook infrastructure beyond OCA-owned session.go + session_lifecycle.sh
- Hibernation logic (#28 deferred to v1.1)
- Backporting to openchad's `bin/oc` (OCA replaces openchad at v1.0 tag)