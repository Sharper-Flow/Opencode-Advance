## Problem

Two operator pains around OCA shell environment freshness and plugin update awareness. Both are independent of session topology and ship cleanly without touching the bare `oca` entry point (which is owned by `patternBSessionTopology` per the decision-locked architecture in `docs/proposals/2026-05-03-session-and-resource-architecture.md`).

### Scope clarification (post-conflict-scan)

The original proposal also covered a third capability (smart `oca` launcher resolving by `$PWD`). That capability has been **removed from this change** because `docs/proposals/2026-05-03-pattern-b-session-topology.md` (Pattern B Change #1) is the primary owner of the bare-`oca` entry semantics. This change conforms to Pattern B's standard: it does not touch the `oca` (no-args) entry point, does not re-define session lookup semantics, and does not pre-claim CLI surface that Pattern B will design.

The remaining two pains are independent of Pattern B and can ship before, after, or alongside it without collision.

### Pain B — the `reload` ritual

Every time the user starts a new shell, they reflexively run `reload` (re-source `~/.zshrc`) on the assumption that *something* in their environment may have changed since last shell start — a plugin updated, a new alias landed, a config block was rewritten. Today:

- The OCA shell profile managed-block contains only PATH wiring. There is no staleness signal, no precmd hook, and no auto re-source mechanism.
- `oca apply`, plugin git pulls, and plugin rebuilds happen out-of-band from the shell. The shell has no way to know its loaded environment is older than what is on disk.
- Users defensively run `reload` "just in case." It works, but the cost is a manual ritual on every shell start.

Net effect: a manual cargo-cult command gates every session start.

### Pain C — plugin update blindness

After exit and resume, the user has no idea if their plugins (Advance, OCA umbrella plugin, etc.) have moved upstream. Today:

- `[plugins.*]` checkouts are local git clones; `oca update` exists but is opt-in and silent until invoked.
- The user finds out plugins are stale only when something breaks at runtime (a new ADV tool is missing, a sync prompt mismatches, a hook fires unexpectedly).
- There is no signal at the natural decision points (shell start, after `oca apply`, after `oca install`).

Net effect: silent drift between local plugin checkouts and upstream, surfaced only by failure modes.

### Why both belong in one change

Both pains share the same plumbing: a stamp file written by `oca apply` / `oca update` / plugin operations, and a consumer (shell hook for Pain B, status surfacer for Pain C) that reads the stamp to decide what to do. Bundling them lets the stamp design serve both consumers cleanly, without forcing a follow-up change to retrofit the second consumer onto a stamp that was designed for only the first.

### Why this matters now

OCA has the foundations to fix both: `oca apply` is the chokepoint for config writes, the shell managed-block already exists, and `[plugins.*]` already track git remotes for `oca update`. The missing pieces are the staleness signal, the auto-refresh hook, and the drift-surfacer wiring.