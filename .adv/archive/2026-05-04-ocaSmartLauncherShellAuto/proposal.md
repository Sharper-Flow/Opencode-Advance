## Proposal

Replace the `reload` muscle memory with an automatic shell-environment refresh, and surface plugin drift at the natural decision points — without touching the `oca` (no-args) entry point that Pattern B owns.

The change covers two loosely-coupled capabilities sharing one underlying primitive (the staleness stamp). They are bundled because they share state and one design conversation produces a cleaner result than two sequential ones.

### Scope alignment with Pattern B

This change explicitly defers to `patternBSessionTopology` (Pattern B Change #1) on:

- Bare `oca` (no-args) entry semantics — owned by Pattern B.
- Session topology (Pattern A vs Pattern B) — owned by Pattern B.
- Per-pane state schema bumps (`SchemaVersion`, `WindowID`, `ProjectSlug`) — owned by Pattern B.
- Smart get-or-create of tmux project sessions — owned by Pattern B.

This change does not introduce any new bare-`oca` behavior, does not modify `cmd/oca/pane.go`'s pane-state schema, and does not change any existing `oca session *` subcommand's contract. It composes with Pattern B by ensuring that whatever Pattern B's smart entry does, the shell environment it inherits is fresh.

### Capability 1 — shell auto-refresh stamp

Replace the `reload` muscle memory with a stamp-driven precmd hook in the OCA managed shell block.

**In scope:**

- A canonical stamp file (path TBD in design; e.g. `$OCA_CACHE_DIR/env.stamp`) whose mtime is bumped by:
  - `oca apply` (any target)
  - `oca install`, `oca update`, `oca pin`
  - Plugin rebuild operations
- The OCA managed shell block adds a precmd hook (zsh) / `PROMPT_COMMAND` integration (bash) that:
  - On each prompt, compares stamp mtime to a per-shell cached marker.
  - If the stamp is newer, silently re-sources the OCA managed block (and any sibling OCA-emitted env files), updates the cached marker, prints at most a single short notice (or nothing, configurable).
  - Never re-executes user-owned shell config — only the OCA-managed surface.
- The hook is opt-out-able via a `stack.toml` flag (default on).
- Behavior must be identical for zsh and bash users at the user-visible level, even though the hook implementation differs.

**Constraint:** the hook must be cheap. A noticeable per-prompt cost is unacceptable; it must be a single fast filesystem stat in the steady state.

### Capability 2 — plugin update awareness

Surface plugin drift at the natural decision points, without slowing down the common case.

**In scope:**

- An explicit `oca update --check` (or equivalently-named) read-only mode that:
  - Performs a fast `git fetch` (with hard timeout) on declared `[plugins.*]` checkouts.
  - Reports one of: "all plugins current", "N plugins behind upstream — run `oca update`", or "probe timed out / offline — skipped".
  - Never modifies plugin checkouts.
- Optional shell-prompt surfacer (gated by a `stack.toml` flag; default off):
  - Composes with Capability 1's stamp/hook plumbing.
  - On the first prompt of a new shell, if a cached drift result exists from a recent check, prints a one-line notice.
  - Cache is short (TTL configurable; default order-of-minutes) so repeated shell starts do not re-fetch.
- `stack.toml` gains a small launcher/update config block (name TBD in design) with:
  - `update_probe = "off" | "passive" | "warn"` (default `"off"`).
  - Probe timeout setting.
  - Cache TTL setting.

**Constraint:** the probe is informational only. It must never modify plugin checkouts, never block a shell prompt or `oca apply`, and must time out cleanly on slow remotes.

## Success Criteria

1. **`reload` becomes obsolete.** After this change ships and the user has run `oca install` / `oca apply` once, opening any new shell automatically picks up the newest OCA-managed environment without the user typing `reload`. Verifiable by editing the OCA-managed block (or running `oca apply`) in one terminal and opening a fresh prompt in another — the new env is live on the first prompt.

2. **No prompt regression.** Steady-state prompt latency with the auto-refresh hook installed is indistinguishable from prompt latency without it (single fast stat per prompt).

3. **No re-source of user config.** The auto-refresh hook re-sources only the OCA managed block and any OCA-emitted env files. User-owned `~/.zshrc` / `~/.bashrc` content is not re-executed by the hook.

4. **Drift visibility on demand.** `oca update --check` (or equivalent) reports drift status against `[plugins.*]` upstreams without modifying any checkout.

5. **Optional drift surfacer is opt-in and bounded.** With the default `update_probe = "off"`, there is zero added latency at shell start. With `update_probe = "warn"`, an offline machine or unreachable remote degrades gracefully (probe times out, shell continues, message says "skipped").

6. **No collision with Pattern B.** This change does not modify `cmd/oca/pane.go`, does not change `$XDG_STATE_HOME/oca/panes/*.json` schema, does not introduce a bare `oca` (no-args) command, and does not alter any `oca session *` subcommand's contract.

7. **Spec coverage.** Each capability ships with its own capability spec under `.adv/specs/`, with Given/When/Then requirements covering the stamp-and-hook contract and the probe budget/timeout semantics.

8. **Worktree-clean uninstall.** `oca uninstall` removes the new precmd hook from the managed block cleanly, leaving zero shell function residue.

## Constraints

- **Must not modify pane state schema.** Pattern B owns the schema bump. No `SchemaVersion`, `WindowID`, `ProjectSlug` work in this change.
- **Must not pre-claim CLI surface owned by Pattern B.** No bare `oca`, no `oca attach` / `oca pick` / `oca resume` definitions.
- **Must respect OCA dev-isolation.** All new code paths honor `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_CACHE_DIR`, and `XDG_STATE_HOME` overrides so tests do not touch live user state.
- **Must work with both zsh and bash.** Auto-refresh hook semantics are identical at the user level; per-shell implementation differences are internal.
- **Must not require Temporal.** Pure filesystem + subprocess; no workflow dependency.
- **Must remain ADV-clean.** Nothing in this change touches Advance-owned files; the managed shell block keeps its existing ownership boundary.
- **Hook overhead budget.** Single `stat(2)` per prompt in the steady state. No subprocess fork, no network call, no file read except the stamp itself when the mtime changes.
- **Probe budget.** `oca update --check` runs all plugin probes in parallel with a single hard timeout for the whole operation. Per-probe timeout is also bounded.

## Out of Scope

- **Smart `oca` entry point.** Owned by `patternBSessionTopology` (Pattern B #1).
- **Session topology, tmux session naming, window-per-worktree.** Owned by Pattern B.
- **Pane state schema additions.** Owned by Pattern B.
- **Auto-rebuild of plugins.** `oca update` already rebuilds; the probe in this change is read-only.
- **Cross-machine sync of stamp/cache.** Local-only.
- **Watchdog / activity tracking changes.** `oca watchdog` keeps its current contract.
- **Status bar additions.** Possible follow-up; not part of this change.
- **Eviction / GC of old pane state files.** Separate hygiene concern; out of scope.