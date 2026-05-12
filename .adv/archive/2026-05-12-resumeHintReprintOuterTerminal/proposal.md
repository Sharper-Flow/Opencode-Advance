## Cross-Project Origin

This change was created as a follow-up from **opencode-advance**.

| Field | Value |
|-------|-------|
| Source project | opencode-advance |
| Source path | `/home/jrede/dev/opencodeadvance` |

> **Note:** The originating project should be consulted for context on why this change is needed.

## What Changes

OCA's session-attach path will gain a post-exit hook that reprints the most relevant `opencode --session <id>` resume guidance to the **outer terminal** (the shell that invoked `oca session attach <name>`) once the tmux client detaches because its last opencode pane closed cleanly.

Today both attach paths (`cmd/oca/session.go::newSessionAttachCmd` → `internal/session/session.go::Manager.Attach` via `syscall.Exec`, and `lib/session_lifecycle.sh::oca_session_attach` via `exec tmux …`) fully replace the calling process. No code runs after tmux returns, so opencode's resume hint — printed inside the destroyed pane buffer — is unreachable.

The change replaces the `syscall.Exec` (Go) and `exec tmux` (bash) replacement-style invocations with a launch-and-wait pattern:

1. Spawn `tmux attach` as a subprocess
2. Await tmux exit
3. Classify the exit (clean client detach vs. external kill vs. error)
4. On clean detach: query opencode's session list, identify the most-recently-updated session for the workdir, print a one-line resume hint to stdout

Process-replacement semantics (signal forwarding, terminal control) must be preserved while tmux runs; the hint emit is fast (single subprocess + JSON parse + one print) and runs only after tmux has fully released the terminal.

## Success Criteria

- [ ] When `oca session attach <name>` exits because the last opencode pane closed cleanly, the outer terminal receives a printed line of the form:
  ```
  💤 Resume: opencode --session <id>  (project: <dir>)
  ```
- [ ] The session ID is correct: queried from `opencode session list --format json` post-exit, filtered by `directory == <session-workdir>`, sorted by `updated` desc, taking first.
- [ ] No hint emitted when:
  - the session was killed externally via `tmux kill-session` (intentional cleanup)
  - `opencode session list` returns no session matching the workdir (legitimate empty state)
  - the user used `oca session kill <name>` (explicit intent to discard)
  - stdout is not a TTY (CI/script context)
- [ ] Works whether the user is inside an outer tmux session or a plain shell.
- [ ] Honors `NO_COLOR` env (plain output, no emoji/ANSI when set).
- [ ] Unit-testable: the session-ID lookup is separated from the print path so the lookup can be mocked and exit-classification logic verified without spawning tmux.
- [ ] Signal forwarding (Ctrl-C, SIGTERM, SIGWINCH) to the tmux subprocess matches current `syscall.Exec` behavior — no regressions to keybinding latency or resize handling.
- [ ] `go test ./...` passes; `go vet ./...` and `gofmt -d .` clean.
- [ ] Bash test under `tests/shell/` covers the session_lifecycle.sh wrapper path.

## Scope

### In Scope

- `cmd/oca/session.go::newSessionAttachCmd` — replace `mgr.Attach(...)` exec semantics with launch-and-wait + post-exit hook
- `internal/session/session.go::Manager.Attach` — refactor from `syscall.Exec` to `exec.CommandContext` + `Wait`; preserve signal/PTY semantics
- New helper (Go) for: clean-vs-killed exit classification, opencode session-list query, JSON parse + filter + sort, hint formatting
- `lib/session_lifecycle.sh::oca_session_attach` — replace `exec tmux` with subprocess invocation that defers to the Go helper for hint emit
- Tests:
  - `internal/session/*_test.go` — unit tests for exit classifier, session-list parser, hint formatter (mockable subprocess seam)
  - `cmd/oca/session_test.go` — integration test for the attach command's post-exit emit
  - `tests/shell/` — bash test covering the wrapper-script path
- Sentinel mechanism so `oca session kill` can mark "intentional kill — suppress hint" (file under `OCA_CACHE_DIR` keyed by session name, cleaned on read)

### Out of Scope

- Changing how OpenCode generates or formats its resume hint
- New tmux plugin, hook, or `wait-for` infrastructure beyond OCA-owned attach path
- Hibernation logic (GH #28, deferred to v1.1)
- Backporting to openchad's `bin/oc` (OCA replaces openchad at v1.0 tag)
- Multi-session "pick which to resume" UX — single most-recent session is sufficient for v1
- Resume hints for sessions launched outside `oca session attach` (e.g. user runs `tmux attach` directly)
- Changes to `opencode session list --format json` output schema (consumed as-is)

## Affected Code

| File | Role | Change shape |
|------|------|--------------|
| `cmd/oca/session.go` | CLI entry — `oca session attach` | Restructure RunE to await Manager.Attach return, then call hint emitter |
| `internal/session/session.go` | `Manager.Attach` | Swap `syscall.Exec` → `exec.CommandContext` + signal forwarding loop |
| `internal/session/` (new) | Resume-hint helper module | New file: exit classifier, session lookup, formatter |
| `lib/session_lifecycle.sh` | `oca_session_attach` | Replace `exec tmux` with invocation that re-enters Go binary for post-exit hint |
| `cmd/oca/session_kill*.go` (new or existing) | `oca session kill` command path | Write sentinel file before killing so hint emit is suppressed |
| `internal/session/session_test.go` | Unit tests | Add classifier + lookup + formatter tests |
| `cmd/oca/session_test.go` | Command-level tests | Mock manager + opencode-list to verify emit path |
| `tests/shell/session_attach_*.sh` (new) | Bash integration | Verify wrapper-script path |

## Related Repositories

| Repo | Role | Required |
|------|------|----------|
| `opencode-advance` (this repo) | Primary — owns OCA session lifecycle | Yes |

No cross-repo work. Advance is unaffected (this is purely OCA enhancement layer per the AGENTS.md ownership map).

## Constraints

- **Process-control parity:** the new launch-and-wait code must forward SIGINT, SIGTERM, SIGWINCH, SIGTSTP/SIGCONT to the tmux subprocess and propagate tmux's exit status. Must not regress terminal raw-mode handling.
- **No new build deps:** Go stdlib only (`os/exec`, `os/signal`, `syscall`, `encoding/json`).
- **opencode binary discovery:** silently skip emit if `opencode` is not on PATH — never fail the attach flow because of a missing hint.
- **Schema fragility:** treat `opencode session list --format json` output defensively. Parse into a minimal struct with only `id`, `directory`, `updated`. Unknown/missing fields fall back to no-emit, never crash.
- **TTY discipline:** detect `isatty(stdout)`; suppress emit in pipes/CI.
- **`NO_COLOR` compliance:** when set, output is plain ASCII (no emoji, no ANSI).
- **Cross-cutting test-resource guardrails:** any test that spawns tmux must use a private socket and clean up; bash tests follow `tests/shell/` conventions.

## Impact

| Area | Impact |
|------|--------|
| User UX | Eliminates "lost session ID" friction post-`/exit`. Direct fix for the stale-session-debt accumulation pattern OCA's session-debt scanner already tracks. |
| OCA cutover (v1.0) | Removes a known UX regression that would otherwise carry over from openchad — this is on the v1.0 SHOULD list (S6). |
| OpenCode core | None. No changes requested or required. |
| Advance plugin | None. |
| Existing tests | `internal/session/session_test.go` and `cmd/oca/session_test.go` need new cases; existing assertions preserved. |
| Performance | Negligible — one extra subprocess + JSON parse per attach exit. No impact on attach latency or in-session responsiveness. |
| Backward compat | None broken. Bash entry point (`lib/session_lifecycle.sh`) keeps the same callable interface; only its internal exec strategy changes. |

## Context

- Parent design doc: [`docs/proposals/2026-05-04-oca-resume-hint-outer-terminal.md`](../../../docs/proposals/2026-05-04-oca-resume-hint-outer-terminal.md) (S6 in SHOULD queue)
- Upstream issue: Sharper-Flow/Opencode-Advance#22
- Related-but-shipped (verified 2026-05-12 during prior discovery, already closed):
  - GH #11 — `oca maintain` self-blocks; shipped in `internal/maintain/session_gate.go`
  - GH #21 — legacy ADV state doctor warning; shipped in `internal/health/advance.go::checkLocalADVState`
- Related session-DB context: `~/.local/share/opencode/opencode.db` is the durable session store; `opencode session list --format json` is the documented query surface (parent doc §3.5.2).
- Pattern reference: openchad's `bin/oc` exhibits the same UX bug today; this fix is the OCA-side equivalent.

## Discovery Agenda

Unresolved unknowns to validate during `/adv-discover`:

1. **Signal-forwarding semantics under `exec.CommandContext`** — Need to verify Go's stdlib subprocess spawning preserves the signal model `syscall.Exec` provides today (Ctrl-C, SIGWINCH propagation, SIGTSTP/SIGCONT job control). May need explicit `signal.Notify` + forwarding loop. Risk: terminal raw-mode handoff could regress. Resolution: `/adv-discover` to spike a minimal launch-and-wait prototype against tmux and verify keybinding latency + resize behavior.
2. **`opencode session list --format json` schema stability** — Parent doc §3.5.2 describes `{id, title, directory, updated, projectId}` but no version contract. Check OpenCode's source/docs for breaking-change history; decide on schema-pinning approach (defensive parse vs. version assertion).
3. **Exit-code distinction: clean detach vs. external kill vs. error** — tmux returns `0` for normal client detach; what does it return when the session is killed via `tmux kill-session` from another client while we're attached? Need empirical verification — may require a sentinel file approach as a reliable signal regardless of tmux exit code.
4. **Sentinel mechanism for `oca session kill`** — Where to persist the "intentional kill" marker (`$OCA_CACHE_DIR`?), schema (one file per session name?), atomic-write + cleanup-on-read semantics, race conditions with concurrent attach.
5. **TTY detection portability** — `isatty()` via `os/term.IsTerminal(int(os.Stdout.Fd()))` works on Linux/macOS; verify behavior under outer-tmux (TTY allocated to outer pane) and under nested SSH.
6. **Opencode-binary discovery** — `exec.LookPath("opencode")`? Does OCA already cache this somewhere (config rendering, plugin install)? Check `internal/install/` and `internal/plugin/` for existing path-resolution patterns to reuse.
7. **Cross-project session disambiguation** — When the user has multiple opencode sessions in the same workdir (e.g. concurrent agents), is "most-recently-updated" always the correct one? Consider: does opencode's `updated` timestamp reflect the last user interaction or any background activity? May need fallback if multi-match becomes common.
8. **Bash wrapper coordination** — `lib/session_lifecycle.sh::oca_session_attach` is invoked both directly (script callers) and indirectly via the Go CLI. Need to decide whether the Go binary's attach path replaces it entirely, or whether the bash entry point shells out to a new `oca session __post-attach-hint` subcommand.
9. **Test infrastructure for tmux subprocess** — `tests/shell/` conventions for spinning up isolated tmux servers; existing fixtures in `internal/session/session_test.go` for socket isolation. Verify the test pattern scales to launch-and-wait validation.

## Risks

| Risk | Mitigation |
|---|---|
| `opencode session list --format json` schema changes | Defensive parse — only consume `id`, `directory`, `updated`; treat parse error as "no hint", log and continue |
| Multiple sessions share same workdir | Take most-recently-updated; document the tiebreak in user-facing docs; consider future flag if collisions prove common |
| User explicitly killed; hint would be misleading | Sentinel file written by `oca session kill` before tmux kill; helper checks + deletes sentinel before emit |
| Outer terminal is not interactive (CI, scripts) | Honor `NO_COLOR`; skip emit when stdout is not a TTY |
| opencode binary not on PATH | `exec.LookPath` returning error → skip emit silently |
| Signal forwarding regression | Discovery spike + integration test with tmux subprocess; explicit signal-handler tests |
| tmux exit code ambiguity | Sentinel-file approach is the source of truth for "intentional kill"; tmux exit code is a hint only |
| Race: concurrent kill-while-attached from another tmux client | Treat unknown exit conditions as "skip emit" to avoid misleading output |

## Approach

> **Note:** Approach is the working hypothesis carried into discovery/design. The Discovery Agenda above flags items that may revise it.

1. Replace `syscall.Exec` in `internal/session/session.go::Manager.Attach` with a launch-and-wait pattern using `exec.CommandContext` + signal-forwarding loop. Preserve PTY/terminal-control semantics.
2. Add post-exit hook in `cmd/oca/session.go::newSessionAttachCmd` that:
   - Detects clean-vs-killed exit (sentinel file + tmux exit code)
   - Queries `opencode session list --format json` (subprocess)
   - Filters by workdir, sorts by `updated` desc, takes first
   - Emits formatted hint (`NO_COLOR`-aware) to stdout
3. Update `lib/session_lifecycle.sh::oca_session_attach` to invoke the Go binary's attach path (which now owns the post-exit hook) rather than `exec tmux` directly. Keeps the bash interface stable for existing callers.
4. Implement sentinel-write in `oca session kill` to suppress hint on intentional kill.
5. Tests — unit (classifier, parser, formatter), command-level (mocked Manager + mocked opencode-list), bash integration (`tests/shell/`).