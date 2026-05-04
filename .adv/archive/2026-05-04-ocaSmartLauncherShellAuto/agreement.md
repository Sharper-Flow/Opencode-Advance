# Agreement

## Objectives

1. Replace the `reload` muscle memory with an automatic, near-zero-cost shell-environment refresh that silently re-sources OCA-managed env when `oca apply` / `oca install` / `oca update` / `oca pin` / plugin rebuilds bump a canonical staleness stamp.
2. Surface plugin drift on demand via an explicit read-only `oca update --check` (or equivalently-named) subcommand, without mutating any plugin checkout.
3. Make plugin drift opportunistically visible at shell start via an opt-in passive surfacer that displays cached drift results — never triggering a fresh fetch in the default config.
4. Conform to `patternBSessionTopology` (Pattern B Change #1) by deferring all bare-`oca` semantics, pane-state schema decisions, and session topology work to that change.

## Acceptance Criteria

**Auto-refresh hook:**

1. With `auto_refresh.enabled = true` (default) after `oca install`, opening a fresh shell loads the OCA-managed env. Running `oca apply` in another terminal causes the first new prompt in the original terminal to silently re-source the OCA env file. No notice printed by default.
2. With `auto_refresh.notice = "every"`, the same scenario prints a single short one-line notice on the first prompt after re-source.
3. With `auto_refresh.enabled = false`, the precmd hook is absent from the managed block; behavior matches pre-change baseline.
4. Steady-state prompt cost (stamp unchanged) = exactly one `stat(2)` call. No subprocess fork, no file read, no git call. Empirically indistinguishable from un-hooked prompt.
5. The hook handles non-interactive shells correctly (script, `ssh -c`): hook is a no-op when shell is not interactive.
6. The hook re-sources a generated sibling env file (path decided in `/adv-design`), not the user's `~/.zshrc` / `~/.bashrc` content.

**Update probe:**

7. `oca update --check` (or equivalent name decided in `/adv-design`) reports per-plugin drift status with values `current` / `behind <N>` / `skipped: <reason>`. No plugin checkout is mutated. Honors `--output text|json`.
8. `oca update --check` enforces both per-plugin and global hard timeouts; offline / unreachable plugins surface as `skipped: timeout` without hanging the command.
9. Default `update_probe = "passive"`: a new shell never triggers a fetch. If a recent `oca update --check` left a cached drift result and TTL has not expired, a one-line drift summary prints once on the first prompt; otherwise nothing.
10. With `update_probe = "off"`, no drift surfacer ever appears; explicit `oca update --check` is the only drift visibility.
11. With `update_probe = "warn"`, the shell-start surfacer additionally triggers a fresh probe on cache miss (with timeout); offline/timeout reports as a single "drift: skipped" line.

**Boundaries (no Pattern B collision):**

12. Pane state schema (`$XDG_STATE_HOME/oca/panes/*.json`) and `cmd/oca/pane.go` `paneState` struct are unchanged. No bare `oca` (no-args) command introduced. No `oca session *` subcommand contract changed.

**Lifecycle + spec coverage:**

13. `oca uninstall` removes the precmd hook from the managed block cleanly; no `_oca_*` shell-function residue remains in user scope after a fresh login.
14. New specs `oca-shell-auto-refresh` and `oca-update-probe` exist under `.adv/specs/` with the `rq-shellAutoRefresh01..06` and `rq-updateProbe01..04` Given/When/Then requirements as drafted in discovery.

## Constraints

- **Pattern B conformance.** Bare `oca` semantics, pane-state schema additions (`SchemaVersion`, `WindowID`, `ProjectSlug`), and session topology are owned by `patternBSessionTopology`. This change MUST NOT touch any of them.
- **Per-prompt budget.** Hook cost in steady state is one `stat(2)`; no subprocess fork, no network call, no file read except the stamp when its mtime advances. At 8+ shells per Pattern B project session, the budget is multiplied by shell count, so cost must remain effectively zero.
- **Probe safety.** All git ops route through `internal/subprocess` with the existing transport-confusion hardening (`protocol.file.allow=user`, `protocol.ext.allow=false`) and `GIT_TERMINAL_PROMPT=0` to refuse credential prompts. Probe must be read-only — `git fetch` is acceptable (writes refs to local) but no checkout, merge, or pull may run.
- **Test isolation.** All new code paths honor `OCA_CACHE_DIR`, `XDG_STATE_HOME`, `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR` overrides so tests do not touch live user state.
- **Shell parity.** Auto-refresh behavior at the user-visible level is identical for zsh and bash; per-shell implementation differences (zsh `add-zsh-hook precmd` vs bash `PROMPT_COMMAND`) are internal.
- **Managed-block atomicity.** Hook installation extends `internal/install/shellprofile.go:WriteBlock` content; existing `ErrBlockEdited` semantics preserved (refuse-on-edit unless `--force`).
- **No Temporal dependency.** Pure filesystem + subprocess. No workflow involvement.
- **ADV-clean.** No Advance-owned files modified. Managed-block ownership boundary unchanged.

## Avoidances

The following directions were considered and rejected:

- **Bare `oca` smart launcher.** Owned by Pattern B (#1). Pre-claiming this surface here would create a hard collision with a decision-locked architecture.
- **Hook re-evaluating the user's full `~/.zshrc` / `~/.bashrc`.** Risks executing user code repeatedly per prompt. Replaced by sibling env file (`~/.config/oca/env.sh`-style) sourced by hook (LBP).
- **Background daemon for periodic plugin probes.** Pushes complexity into a long-running process for marginal value. Rejected; explicit `oca update --check` + opt-in shell surfacer is sufficient.
- **`direnv` for OCA env management.** `direnv` is per-directory; OCA env is global. Not a fit.
- **Default `update_probe = "off"`.** Considered; user chose `"passive"` to make cached drift results visible without ever triggering a fetch on shell start. Trades a tiny first-prompt print for better drift visibility; preserves zero-fetch-on-shell-start latency guarantee.
- **Default `update_probe = "warn"`.** Considered; rejected because it would add fetch latency at shell start in the default config and violates the steady-state cost goal.
- **One-line notice on every refresh by default.** Rejected at user direction (Q3) due to multiplicative noise at 8+ shells per Pattern B session. Notice is opt-in via `auto_refresh.notice`.

## Decisions

### User Decisions

- **Q3 — Refresh notice visibility.** Silent default + opt-in notice via `auto_refresh.notice = "off" | "once" | "every"`. *Why it matters: at 8+ shells per Pattern B session, default-on notices multiply visually. Silent default keeps the noise floor at zero; opt-in lets users who want signal get it.*
- **Q4 — Plugin drift surfacing.** `oca update --check` (always available) plus opt-in shell surfacer (default off / passive per Q5). *Why it matters: explicit `--check` is unambiguous and zero-side-effect; the opt-in passive surfacer composes with the auto-refresh hook so users who want drift visibility at shell start can have it without a separate mechanism.*
- **Q5 — `update_probe` default.** `"passive"`. *Why it matters: zero added latency at shell start (no fetch ever), but if a recent `oca update --check` populated the cache, the shell shows that drift. This matches the operator's stated `reload` reflex (wanting awareness) without paying a per-shell fetch cost.*

### Agent Decisions (LBP)

- **Q1 — Stamp file location.** `$OCA_CACHE_DIR/env.stamp` (matches existing `apply.lock` location, respects `OCA_CACHE_DIR` test override).
- **Q2 — Re-source target.** Sibling generated env file under `~/.config/oca/` (exact filename decided in `/adv-design`); managed block in shell rc adds the precmd hook + sources that file once at install. Hook re-sources the env file, not the rc.
- **Q6 — Cache TTL for drift result.** Default 5 minutes (LBP-typical for short-lived caches); configurable via `stack.toml`.
- **Q7 — Same-shell apply trigger.** Trust mtime. Bumping the stamp from inside the shell that issued `oca apply` causes the next prompt to detect mtime advance and re-source. No special same-shell mechanism. Confirm in `/adv-design` that no race exists between the apply subprocess returning and the next prompt firing.
- **Q8 — `stack.toml` schema location.** Deferred to `/adv-design` after surveying current schema conventions. Likely `[oca.shell]` for the hook + `[oca.update_probe]` for the probe, or a unified `[oca]` block. No semantic decision deferred — only naming.

## Deferred Questions

- **Exact filename for the sibling OCA env file.** Bound to Q2 design choice; decided during `/adv-design`.
- **Exact `stack.toml` section names for the new flags.** Bound to Q8; decided during `/adv-design`.
- **Numeric per-prompt wall-clock budget (in milliseconds).** AC #4 commits to "indistinguishable from un-hooked"; specific number captured during `/adv-design` measurement and codified in the test matrix.
- **Empirical measurement of `git fetch` cost per plugin remote.** To set sane default per-plugin and global timeouts (AC #8). Captured during `/adv-design`.

## Sign-Off

User approved acceptance criteria inline at AC checkpoint (Phase 4.5.1) on 2026-05-04. Proposal updated post-conflict-scan to drop Capability 1 (smart launcher) at user direction ("the other [Pattern B] is the primary change, we need to conform to their standard"). All open questions resolved or explicitly deferred to `/adv-design`. Discovery gate ready to complete.