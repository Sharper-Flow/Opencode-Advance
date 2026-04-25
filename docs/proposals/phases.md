# Phase Sequencing

OpenCode Advance v1.0 is developed in sequential phases. Each phase is one or more ADV changes. Phase dependencies are strict: Phase N cannot start until Phase N-1 has been archived.

---

## Phase 0: Foundation + Brand

**Status:** Complete — delivered in archived change `phase0FoundationBrand` and merged to `trunk`.

**Goal:** Establish the repository foundation, lock the brand identity, and ship the minimal branded CLI/shell baseline needed for later phases.

**Estimate:** 3-5 days

**Deliverables:**

- Repository scaffolded and Go module initialized
- `cobra` dependency added and minimal `oca` command tree shipped
- Shared runtime brand assets + Go renderer under `brand/`
- `oca version` renders the branded wordmark + version output
- `lib/palette.sh`, `lib/wordmark.sh`, and `lib/boot_splash.sh` shipped with truecolor/256/mono + `NO_COLOR` behavior
- Brand documentation finalized (`docs/design/brand.md`, `wordmark.md`, `palette.md`, `theme.md`)
- CI + local verification cover `go vet`, `go test`, `go build`, and shell verification
- README / status / resume docs refreshed for the Phase 0 baseline

**Exit criteria:**

- [x] `go run ./cmd/oca version` prints the branded wordmark + version
- [x] `bash lib/boot_splash.sh` renders the minimal boot splash behavior
- [x] Color/no-color and terminal fallback behavior are verified in Go + shell tests
- [x] CI passes with Go + shell verification wired in

**Historical implementation reference:**

- Archived ADV change: `phase0FoundationBrand`
- Merge commit on `trunk`: `dc85b39 feat(phase0): establish branded CLI baseline`
- Use this shipped baseline as the reference point for all later phases

### Retrospective

Phase 0 established the branded CLI baseline and learned three durable lessons. Worktree operations must use absolute file paths to avoid targeting the original checkout instead of the worktree (ws-1ApJ_H). Shell color tests must explicitly clear inherited `COLORTERM` when asserting 256-color fallback, or truecolor-capable environments invalidate assertions (ws-8cww-_). CLI surface is testable through a small `commandOptions` struct injecting stdout/stderr and version metadata, avoiding a heavy application container (pw-rmPnXzHs).

---

## Phase 1: stack.toml + MCP apply

**Status:** Complete — delivered in archived change `phase1StackTomlParserMcpApply` and merged to `trunk`.

**Goal:** Parse `stack.toml`, validate it, and render the `[mcp.servers.*]` section into both `opencode.json` and `vision/servers.yaml`.

**Estimate:** 1-2 weeks

**Deliverables:**

- `internal/config/` — TOML parser, schema types, validation, variable resolution
- `internal/render/` — JSON merge, YAML writing, programmatic rendering
- `internal/health/mcp.go` — MCP server HTTP health checks
- `cmd/oca/apply.go` — `oca apply --target mcp`
- `cmd/oca/doctor.go` — `oca doctor --scope mcp`
- `cmd/oca/debug.go` — `oca debug plan`, `oca debug validate`
- Golden-file tests for every schema variant

**Exit criteria:**

- `oca apply --target mcp --dry-run` prints the render plan for the example stack.toml
- `oca apply --target mcp` writes the MCP fragment to an isolated test `opencode.json` atomically
- `oca doctor --scope mcp` checks each declared MCP server and reports pass/fail
- All 9 MCP servers from the user's real stack are represented in tests
- Isolated test config dir is used throughout (never touches `~/.config/opencode/`)

**Tasks (high-level):**

- tk-phase1-01: Define Stack, MCP, MCPServer struct types in `internal/config/types.go`
- tk-phase1-02: Implement TOML parser with BurntSushi/toml
- tk-phase1-03: Implement schema validation (required fields, value ranges)
- tk-phase1-04: Implement variable resolution ($HOME, ~/, {checkout}, env vars)
- tk-phase1-05: Implement idempotent JSON merge
- tk-phase1-06: Implement atomic file write (temp + rename)
- tk-phase1-07: Implement backup rotation
- tk-phase1-08: Implement programmatic `vision/servers.yaml` renderer
- tk-phase1-09: Implement programmatic `opencode.json` MCP fragment renderer
- tk-phase1-10: Implement `oca apply --target mcp` command
- tk-phase1-11: Implement `oca doctor --scope mcp` with HTTP health checks
- tk-phase1-12: Implement `oca debug plan` and `oca debug validate`
- tk-phase1-13: Write golden tests for stack.example.toml rendering
- tk-phase1-14: Write integration test for full mcp apply cycle

### Retrospective

Phase 1 delivered TOML parsing, MCP rendering, and the first `oca apply` target. Vision's stdio MCP servers defer process spawn to session-manager, not daemon startup — so "required server failed to start" is architecturally undetectable until the first session connects. Doctor is the right enforcement layer, not startup checks (ws-8zADsv). E2E lifecycle on changes with 30+ tasks is not single-session work; discovery + design + planning alone can consume 40–60% of output budget (ws-4fuYiD). The gap-audit pattern — running an explicit completeness scan after planning closes and before execution starts — found 16 real gaps including a critical render-rule ambiguity and missing concurrency protection (pw-_cO0efB0). `env_file` handling follows systemd `EnvironmentFile=` semantics adapted to declarative-config scope: record path at parse, soft-warn if missing, never read contents, keeping OCA secret-blind (pw-ol4U1yE7).

---

## Phase 2: Plugin + Instruction Management

**Status:** Complete — delivered in archived change `phase2PluginInstruction` and merged to `trunk`.

**Goal:** Manage plugin lifecycle (clone, build, pin, wire into opencode.json) and render the instructions list. Critically, delegate ADV asset sync to Advance's own sync-global.sh.

**Estimate:** 1 week

**Shipped deliverables:**

- `internal/subprocess/` — generic subprocess runner with timeout, signal handling, exit classification
- `internal/plugin/` — git clone/pull, npm source handling, SHA capture/pin, build command execution
- `internal/sync/advance.go` — subprocess invocation of `sync-global.sh --fix` with captured output
- `internal/render/` — `MergeArray` for plugin array merge, `WriteAtomic` (temp+rename), `AcquireApplyLock` (flock)
- `cmd/oca/pin.go` — `oca pin` writes current SHAs to stack.toml
- `cmd/oca/update.go` — `oca update` pulls latest (respecting pins)
- `cmd/oca/doctor.go` — expanded with `--scope plugins`, `--network` flag
- 28 tasks, 8 conventional commits (`6d32bdd..bb4ff36`), ~2,400 LOC net-new

**Historical implementation reference:**

- Archived ADV change: `phase2PluginInstruction` (archive dir: `.adv/archive/2026-04-20-phase2PluginInstruction/`)
- Commit range on `trunk`: `6d32bdd..bb4ff36`
- Subsequent delivered phase: **Phase 3.5** (skills + commands + formatters + toggles)

### Retrospective

Phase 2 shipped the plugin lifecycle, subprocess runner, and apply-lock primitives. The subprocess runner pattern is now canonical for all Phase 2+ callers: `exec.CommandContext` with timeout, `cmd.Cancel` for SIGTERM, `cmd.WaitDelay` for SIGKILL grace period, combined stdout+stderr, and a classified exit taxonomy (ws-m5thsp, promoted to pw-xQMO10LC). Four judgment-call resolutions established project conventions: `--frozen-lockfile` strict for pnpm, runtime XDG resolution for path-dependent policies, always-git-fetch on update, and local-only doctor by default with `--network` for upstream checks (ws-haIw5P, promoted to pw-arzbCyP5). Per-target backup suppression requires `WriteAtomic` itself to treat `maxBackups<=0` as "no backup at all" — not just the caller (ws-GcnRm8). Stale worktree stripping must resolve `$XDG_DATA_HOME` inside hot-path helpers, not at package init, or cached regex drift fails pruning (ws-5yhCxi). The `adv_change_update` tool does not expose `judgment_calls` as a typed parameter, so `/adv-prep` Phase J persists them in proposal.md text and `/adv-apply` Phase 1.5 reads them from there — functional but not schema-typed (ws-z_2J12).

---

## Phase 3: Providers + Permissions + LSP + Watcher + Diff

**Status:** Complete — delivered in archived change `phase3CoreOpencodeJsonCoverage` and merged to `trunk`.

**Goal:** Core `opencode.json` coverage: providers, permissions, watcher ignore globs, LSP, all-target apply, and drift detection. Agent rendering is explicitly out of scope for this phase. Skills, custom commands, formatters, and OpenCode-level toggles are deferred to Phase 3.5.

**Estimate:** 1 week

**Deliverables:**

- `internal/render/providers.go` — providers rendering (handles nested model/variants)
- `internal/render/permissions.go` — permissions rendering
- `internal/render/watcher.go` — watcher.ignore rendering
- `internal/render/lsp.go` — LSP rendering
- `cmd/oca/diff.go` — `oca diff` drift detection
- `cmd/oca/apply.go` supports all Phase 3 targets plus no-target composed apply
- Golden-file tests for every target

**Exit criteria:**

- `oca apply` with no `--target` applies all targets in dependency order
- `oca diff` shows drift for every configuration slice
- The complete `stack.example.toml` renders to a valid `opencode.json` that OpenCode can load
- `[agents]` remains deferred and existing `.agent.*` state is left untouched

**Tasks (high-level):**

- tk-phase3-01: Implement providers rendering (nested structure)
- tk-phase3-02: Implement permissions rendering (external_directory, bash, default, doom_loop)
- tk-phase3-03: Implement watcher.ignore rendering
- tk-phase3-04: Implement LSP rendering
- tk-phase3-05: Implement `oca diff` command
- tk-phase3-06: Implement target-aware composed apply (run all targets in order)
- tk-phase3-07: Golden/integration tests for each target and apply-all path
- tk-phase3-08: End-to-end test: stack.example.toml → opencode.json → OpenCode loads

**Historical implementation reference:**

- Archived ADV change: `phase3CoreOpencodeJsonCoverage`
- Merge commit on `trunk`: `91a1d0f feat(render): complete phase 3 core config coverage`
- Subsequent delivered phase: **Phase 3.5**

### Retrospective

Phase 3 completed core `opencode.json` coverage for providers, permissions, watcher, LSP, composed no-target apply, and `oca diff`. The most important scope correction was explicit: agents were removed from shipped Phase 3 and remain deferred. Composed apply now relies on a running in-memory document chain plus `NoRollback` behavior so earlier successful writes are preserved on mid-plan failure. Supporting docs and the example stack must treat Phase 3 as complete, treat Phase 3.5 as shipped, and point future work at Phase 4 rather than back at the archived core-render change.

---

## Phase 3.5: Skills + Commands + Formatters + OpenCode Toggles

**Status:** Complete — delivered in archived change `addPhase35ConfigCoverageSkills` and merged to `trunk`.

**Goal:** Complete declarative coverage of the remaining `opencode.json` surfaces: OCA-owned skill management, custom slash commands, code formatters, and OpenCode-level behavior toggles. Fills the gap between core config (Phase 3) and primary client/theme work (Phase 4).

**Estimate:** 3-4 days

**Deliverables:**

- `internal/render/skills.go` — skill asset copy from `assets/skills/` to target skills dir, respecting plugin-owned exclusions
- `internal/render/commands.go` — custom command rendering for `opencode.json` `.command`
- `internal/render/formatters.go` — formatter rendering for `opencode.json` `.formatter`
- `internal/render/toggles.go` — OpenCode-level toggle rendering (default_agent, share, snapshot, autoupdate, compaction, disabled/enabled_providers)
- `cmd/oca/apply.go` extended for skills, commands, formatters, and toggles targets
- `assets/skills/` populated with canonical copies of all OCA-owned skills
- Golden-file tests for each new target
- `oca doctor --scope skills` verifies OCA-owned skills are present and not overwriting plugin-owned skills

**Exit criteria:**

- `oca apply --target skills` copies OCA-owned skills to the target directory without touching ADV-owned skills
- `oca apply --target commands` renders custom slash commands into `opencode.json`
- `oca apply --target formatters` renders formatter config into `opencode.json`
- `oca apply --target toggles` renders OpenCode-level toggles into `opencode.json`
- All new targets work correctly in isolated test config directories
- `oca apply` with no `--target` applies all targets in dependency order, including Phase 3.5 targets
- `oca doctor --scope skills` detects skill ownership violations (OCA writing ADV-owned paths)

**Tasks (high-level):**

- tk-phase3.5-01: Implement skill asset copy logic (enumerate `assets/skills/`, skip plugin-provided categories)
- tk-phase3.5-02: Implement custom command rendering
- tk-phase3.5-03: Implement formatter rendering
- tk-phase3.5-04: Implement OpenCode toggles rendering (default_agent, share, snapshot, autoupdate, compaction, provider lists)
- tk-phase3.5-05: Populate `assets/skills/` with canonical OCA skill files (lgrep, mcp-selection, morph, prioritizer, worktree, caveman, caveman-commit, caveman-review)
- tk-phase3.5-06: Golden tests for each new target
- tk-phase3.5-07: Integration test: apply all targets together with full stack.example.toml
- tk-phase3.5-08: `oca doctor --scope skills` implementation and test

**Historical implementation reference:**

- Archived ADV change: `addPhase35ConfigCoverageSkills`
- Merge commit on `trunk`: `7d5ce85 chore: archive addPhase35ConfigCoverageSkills`
- Next recommended phase: **Phase 4**

### Retrospective

Phase 3.5 completed the remaining declarative config surfaces for OCA-owned skills, custom commands, formatters, and OpenCode toggles. Two durable lessons stood out. TDD compliance in ADV requires both red and green evidence entries — green-only evidence is insufficient for strict archive validation, even when tests pass (record red + green explicitly). Integration tests that exercise the compiled CLI must set `OCA_ASSETS_ROOT` to the repo `assets/` directory because the test binary runs from a temporary location and cannot infer asset-relative paths automatically.

---

## Phase 4: Primary Client UX + Theme

**Status:** Complete — delivered in archived changes `phase4FoundationSessionTheme` and `completePhase4SessionLifecycle`, merged to `trunk`.

**Goal:** current tmux-first client/session lifecycle, Obsidian theme, redesigned status bar, and new boot splash with animation. Builds on full config coverage from Phases 3 and 3.5. Same-host re-entry from other terminals/devices is supported, but remains secondary to normal local usage.

**Estimate:** 1-1.5 weeks

**Deliverables:**

- `assets/themes/obsidian.json` — primary client theme asset (current OpenCode/TUI target)
- `assets/themes/obsidian.tmux.conf` — tmux status bar theme
- `lib/status_bar.sh` — status bar renderers (row0: session + git + ADV state + host + clock; row1: window list + LLM gauges + date)
- `lib/boot_splash.sh` — full boot splash with indigo pulse animation (16-frame truecolor interpolation)
- `lib/adv_status.sh` — ADV state reader for status bar (reads change.json, finds active changes, summarizes gate progress)
- `lib/llm_gauge.sh` — LLM provider fuel gauge renderer (4 providers, colorized ASCII bars)
- `lib/session_lifecycle.sh` — session creation shell wrapper
- `cmd/oca/session.go` — `oca session new/list/attach/switch/kill/killall/restart/reap`
- `cmd/oca/theme.go` — `oca theme list/apply`
- `internal/session/session.go` — Manager with Create, List, NextSessionName, Attach, SwitchClient, Kill, KillAll, Restart, GetSessionByName, SetGlobalEnv
- tmux config block template (`templates/tmux.conf.block.gotmpl`)
- 11 CLI integration tests covering all session commands
- Shell tests for adv_status.sh, llm_gauge.sh, status_bar.sh in `tests/shell/`

**Exit criteria:**

- [x] `oca session new` creates a tmux session, shows the boot splash, and launches the configured primary client in the current directory
- [x] `oca session list` shows active sessions
- [x] `oca session attach` can re-enter a same-host OCA tmux session from another terminal/device without losing running work
- [x] Obsidian theme loads cleanly in the current primary client with all palette colors applied
- [x] Status bar shows: session title + ADV change (row 0 left), repo/branch/worktree (row 0 right), window name (row 1 left), metrics + LLM gauges + clock (row 1 right)
- [x] Boot splash renders wordmark with indigo+ pulse effect on truecolor terminals
- [x] Stale session reaper cleans up unattached `oca-*` sessions without deleting sessions a user would reasonably expect to resume
- [x] Remote re-entry relies on existing host access paths (local shell, SSH, Tailscale), not custom transport or OCA-managed auth
- [x] No synthwave edges, no color-cycling, no per-session randomized borders
- [x] OCA_REPO_ROOT injected into tmux global env during session creation for status bar script resolution

**Historical implementation reference:**

- Foundation archived change: `phase4FoundationSessionTheme`
- Completion archived change: `completePhase4SessionLifecycle`
- Next recommended phase: **Phase 5**

### Retrospective

Phase 4 shipped the full tmux-first client/session lifecycle across two ADV changes. The foundation change delivered significantly more scope than its task list suggested — session lifecycle commands (attach, switch, kill, killall, restart, reap), theme management, status bar with ADV state and LLM gauges, and boot splash animation were all implemented in the foundation change. The completion change fixed a critical runtime bug (OCA_REPO_ROOT not injected into tmux global env, making status bar blank), added CLI integration tests for all session commands, and created the completion spec. Key lesson: the `obsidian.tmux.conf` uses `$OCA_REPO_ROOT` in `#()` format expansions which must be injected via `tmux setenv -g` at session creation time — tmux `#()` runs in a fresh shell and cannot resolve the variable otherwise.

---

## Phase 5: Temporal Enablement

**Status:** Advance has shipped Temporal as its primary state backend. This phase wires OCA to manage the Temporal infrastructure that Advance now depends on. The `[temporal]` section in `stack.toml` is already parsed (Phase 2 pre-allocated the hooks); this phase makes it operational.

**Why this phase ships before installer/migration:** Advance already depends on Temporal. Every subsequent phase benefits from OCA managing it — the installer should install Temporal awareness, migration should produce a Temporal-ready stack, and v1.0 should ship with Temporal tooling. Waiting until after migration means shipping without Temporal support and retrofitting it later.

**Goal:** Wire OCA to manage the Temporal infrastructure that Advance now requires as its primary state backend. Advance runs two durable workflows (`changeWorkflow`, `projectWorkflow`) backed by Temporal, with file-based fallback when Temporal is unavailable. OCA's job: install/detect the Temporal CLI, supervise a local dev server when requested, propagate `ADV_TEMPORAL_*` env vars, wire the Temporal client bundle into the Advance plugin load, and verify the whole chain via `oca doctor --scope temporal`.

**Estimate:** 3-5 days

**Deliverables:**

- `internal/temporal/` — CLI detection, optional install flow, dev-server supervision, client-bundle resolution
- `internal/render/temporal.go` — renders `[temporal]` config into a form the Advance plugin can consume (e.g., env file, plugin-boot config fragment)
- `internal/health/temporal.go` — reachability, namespace existence, worker heartbeat (registered via the Phase 2 health-check registry)
- `cmd/oca/temporal.go` — `oca temporal {status,start,stop,restart,logs}` for the supervised dev server
- `cmd/oca/apply.go` — wire the reserved `--target temporal` surface (Phase 2 reserved; this phase implements)
- `cmd/oca/doctor.go` — wire the reserved `--scope temporal` surface (Phase 2 reserved; this phase implements)
- Env-file rendering: OCA writes `$OCA_CACHE_DIR/temporal.env` with `ADV_TEMPORAL_*` values; Advance plugin subprocess inherits them via the Phase 2 subprocess runner's explicit `env` parameter
- `stack.example.toml` updated: the `[temporal]` placeholder block from Phase 2 becomes a fully documented live example

**Exit criteria:**

- `oca apply --target temporal` with `[temporal].enabled = false` (default) is a no-op that reports "temporal disabled"
- `oca apply --target temporal` with `[temporal].enabled = true` and `[temporal].dev_server = false` validates reachability of the declared `address` and stops there
- `oca apply --target temporal` with `[temporal].enabled = true` and `[temporal].dev_server = true` detects or installs the Temporal CLI, supervises `temporal server start-dev` as a background process (PID file in `$OCA_CACHE_DIR`), and ensures the declared namespace exists
- `oca temporal status` reports: CLI path, dev-server PID (if any), reachability, namespace state, worker heartbeat
- `oca doctor --scope temporal` returns pass when all of the above are healthy; fails with actionable remediation otherwise
- When a plugin declares `temporal_bundle = "..."`, OCA passes it through the plugin-boot subprocess env so the Advance plugin's `createStore({ temporalBundle })` activates the overlay
- Non-loopback `address` without `allow_remote = true` fails fast with a clear error (mirrors Advance's own fail-fast policy)
- No agent-visible tool surface changes to Advance — this phase only wires the environment around the already-merged Advance code

**Tasks (high-level):**

- tk-phase5-01: Implement Temporal CLI detection (`temporal --version`); optional install via documented flow (brew / curl script), gated behind explicit `--install` flag — never silent
- tk-phase5-02: Implement dev-server supervisor (start, stop, restart, PID file, log capture to `$OCA_CACHE_DIR`)
- tk-phase5-03: Implement env-file rendering (`$OCA_CACHE_DIR/temporal.env`) with all `ADV_TEMPORAL_*` values; atomic write reusing Phase 1 render primitives
- tk-phase5-04: Implement reachability / namespace / worker-heartbeat checks via the Phase 2 health-check registry
- tk-phase5-05: Implement `oca temporal status/start/stop/restart/logs`
- tk-phase5-06: Implement `oca apply --target temporal`
- tk-phase5-07: Implement `oca doctor --scope temporal`
- tk-phase5-08: Wire `temporal_bundle` into the Advance plugin's boot env via the Phase 2 subprocess runner's `env` parameter
- tk-phase5-09: Integration test: `enabled = false` → no-op across all touchpoints
- tk-phase5-10: Integration test: `enabled = true` + `dev_server = true` → end-to-end local dev-server lifecycle (start, reach, create namespace, stop)
- tk-phase5-11: Integration test: non-loopback address without `allow_remote` → fail fast
- tk-phase5-12: Update `stack.example.toml` with a full `[temporal]` example (uncommented + documented)
- tk-phase5-13: `SETUP.md` / `README.md` — Temporal section explaining when to enable it and what OCA manages vs. what the user runs themselves

**Explicitly out of scope (defer or decline):**

- Managing a production Temporal cluster (this phase is local-dev / single-user)
- Worker-process supervision beyond reading its heartbeat (the worker runs inside the Advance plugin process; OCA does not spawn it)
- Temporal Cloud integration (only generic `address` + `allow_remote` are supported)

**Advance Temporal architecture (already shipped, for reference):**

Advance's Temporal integration uses two long-lived workflows:
- `changeWorkflow` — per-change state machine (tasks, gates, wisdom, artifacts, re-entry)
- `projectWorkflow` — per-project singleton (agenda, project-level wisdom, migration ledger)

Operations use Temporal queries (read) and updates (mutate) with deterministic handlers. Worker runs in-process on Node hosts, or as an out-of-process Node child on Bun hosts (via `ADV_NODE_PATH`). Continue-as-new prevents unbounded history at configurable thresholds. Env vars: `ADV_TEMPORAL_ADDRESS`, `ADV_TEMPORAL_NAMESPACE`, `ADV_TEMPORAL_ALLOW_REMOTE`, `ADV_NODE_PATH`, `ADV_DISABLE_TEMPORAL`.

---

## Phase 6: Installer + Shell Profile

**Goal:** `oca install` performs end-to-end first-time setup. Shell profile wiring (PATH, completions) is idempotent and reversible. Installer includes Temporal prerequisite checks and optional dev-server setup (leveraging Phase 5).

**Estimate:** 4-5 days

**Deliverables:**

- `cmd/oca/install.go` — full install flow
- `internal/install/` — prerequisite checks, shell profile management
- `templates/shell_profile.block.gotmpl`
- `cmd/oca/completion.go` — shell completion generation (bash, zsh, fish)
- `cmd/oca/uninstall.go` — removes managed blocks

**Exit criteria:**

- `oca install --yes` installs everything on a fresh Ubuntu 22.04 VM
- `oca install` adds a managed block to `~/.zshrc` and `~/.bashrc` with PATH + completions
- `oca completion zsh` and `oca completion bash` print working completion scripts
- `oca uninstall` removes all managed blocks without touching user-owned content
- Installer handles partial install recovery (if interrupted, re-running completes)

**Tasks (high-level):**

- tk-phase6-01: Implement prerequisite check (git, tmux version, OpenCode, vision binary, Temporal CLI)
- tk-phase6-02: Implement shell profile block management (add/remove idempotent)
- tk-phase6-03: Implement `oca install` command with all steps
- tk-phase6-04: Implement `oca completion <shell>` for bash/zsh/fish
- tk-phase6-05: Implement `oca uninstall` command
- tk-phase6-06: Integration test: fresh-VM install in Docker/chroot
- tk-phase6-07: Integration test: install → uninstall → install again is clean

---

## Phase 7: Migration + Doctor Expansion

**Goal:** `oca migrate from-open-chad` works on the maintainer's real open-chad state. Doctor expands to cover Advance state, Temporal health, and cross-component consistency. Migration produces a Temporal-ready stack.toml (leveraging Phase 5).

**Estimate:** 4-5 days

**Deliverables:**

- `internal/migrate/openchad.go` — reads open-chad state, emits stack.toml
- `cmd/oca/migrate.go` — `oca migrate from-open-chad`, `oca migrate init`
- `internal/health/advance.go` — ADV state checks
- `internal/health/cross.go` — cross-component consistency checks
- Expanded `oca doctor` output with all checks (including Temporal scope from Phase 5)

**Exit criteria:**

- `oca migrate from-open-chad` successfully reads the maintainer's open-chad state and produces a valid stack.toml
- The generated stack.toml, when applied, reproduces the user's current state (verified by diff)
- `oca doctor` reports on: Advance checkout, plugin build state, ADV state directory, Temporal health, cross-component consistency (e.g., agent model references a declared provider)
- Migration preserves user customizations (plugin paths, provider models, agent assignments)

**Tasks (high-level):**

- tk-phase7-01: Implement open-chad state reader (opencode.json + vision/servers.yaml + open-chad.json + config/opencode/)
- tk-phase7-02: Implement stack.toml emitter
- tk-phase7-03: Implement `oca migrate from-open-chad` command
- tk-phase7-04: Implement `oca migrate init` command (minimal starter)
- tk-phase7-05: Implement ADV state health checks
- tk-phase7-06: Implement cross-component consistency checks
- tk-phase7-07: Integration test: migration reproduces state
- tk-phase7-08: Manual verification: run migration on maintainer's real environment

---

## Phase 8: Extras + Polish

**Goal:** Discord integration (with new taglines), release pipeline, final README, polish pass.

**Estimate:** 3-5 days

- `lib/discord/` — ported and rewritten Discord Rich Presence integration
- `lib/discord/taglines.toml` — all new taglines (data-driven, no "chad" era jokes)
- `cmd/oca/discord.go` — `oca discord enable/disable/status`
- `.github/workflows/release.yml` — goreleaser release pipeline
- `.goreleaser.yaml` — cross-platform build config
- Final README rewrite with install instructions, command reference, troubleshooting
- `INSTALL.md` with detailed install guide
- `CHANGELOG.md` generated from git log

**Exit criteria:**

- `oca discord enable` works and shows presence in Discord
- All Discord taglines are rewritten (no "chad" references)
- `git tag v1.0.0 && git push origin v1.0.0` triggers release pipeline that produces cross-platform binaries
- Binaries are published to GitHub Releases with SHA256SUMS.txt
- README is complete and internally consistent
- Manual smoke test: fresh install from GitHub Release binary on Ubuntu 22.04

**Tasks (high-level):**

- tk-phase7-01: Port Discord Rich Presence integration from open-chad
- tk-phase7-02: Rewrite all taglines (data-driven TOML file)
- tk-phase7-03: Implement `oca discord enable/disable/status`
- tk-phase7-04: Set up goreleaser config
- tk-phase7-05: Write `.github/workflows/release.yml`
- tk-phase7-06: Test release pipeline with a pre-release tag (v1.0.0-rc1)
- tk-phase7-07: Write final README
- tk-phase7-08: Write INSTALL.md
- tk-phase7-09: Generate CHANGELOG.md
- tk-phase7-10: Final polish pass: inconsistencies, typos, dead links

---

## Cross-phase rules

### Clean cutover enforcement

Every phase MUST:

- Never write to `~/.config/opencode/`, `~/.config/vision/`, `~/.tmux.conf`, or shell rc files
- Honor `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_PLUGIN_CHECKOUT_ROOT`, `OCA_CACHE_DIR` environment overrides
- Run all tests against isolated test directories (not production state)

### ADV workflow

Every phase is developed as one or more ADV changes following the 7-gate workflow:

1. `/adv-proposal` creates the change with the phase goal + deliverables as the proposal
2. `/adv-discover` gathers context, analyzes current state, identifies objectives and knowledge gaps
3. `/adv-agree` presents objectives and constraints for user acceptance
4. `/adv-design` validates architecture decisions with mandatory `adv-researcher` validation
5. `/adv-present` presents concise design overview for user review before planning
6. `/adv-prep` expands the high-level tasks into a concrete task graph with TDD intent
7. `/adv-apply` implements tasks one at a time with red/green TDD evidence
8. `/adv-review` reviews the implementation across 12 dimensions
9. `/adv-accept` presents deliverable summary and acceptance criteria checklist to user
10. `/adv-harden` runs coverage/slop/doc checks
11. `/adv-validate` checks the change against any specs created during the phase
12. `/adv-archive` applies deltas and closes the change

### Phase dependencies

- Phase 1 blocks on Phase 0 (needs the wordmark + palette primitives)
- Phase 2 blocks on Phase 1 (needs config/render/health foundations)
- Phase 3 blocks on Phase 2 (extends the apply + target system)
- Phase 3.5 blocks on Phase 3 (extends the apply + render system with new config surfaces)
- Phase 4 blocks on Phase 3.5 (session/theme work requires full config coverage to be testable end-to-end)
- **Phase 5 (Temporal) blocks on Phase 2 only** (needs the Temporal inheritance hooks, generic subprocess runner, and health-check registry). Moved before installer/migration because Advance already depends on Temporal and every subsequent phase benefits from OCA managing it.
- Phase 6 (Installer) blocks on Phase 4 (uses session lifecycle + theme) and Phase 5 (includes Temporal prerequisite checks)
- Phase 7 (Migration) blocks on Phase 6 (migration produces a stack.toml, which needs full coverage to be useful)
- Phase 8 (Extras) blocks on Phase 7 (release is the last step)

### Out-of-phase work

Minor fixes, typos, doc updates, and CI tweaks can be committed outside of ADV changes if they are trivial. Anything that touches Go code, stack.toml schema, or render logic MUST go through an ADV change.

---

## Estimated timeline

| Phase                                        | Estimate    | Cumulative    |
| -------------------------------------------- | ----------- | ------------- |
| 0: Foundation + brand                        | 3-5 days    | 0.5-1 week    |
| 1: stack.toml + MCP apply                    | 1-2 weeks   | 1.5-3 weeks   |
| 2: Plugin + instruction mgmt                 | 1 week      | 2.5-4 weeks   |
| 3: Core opencode.json coverage               | 1 week      | 3.5-5 weeks   |
| 3.5: Skills + commands + formatters          | 3-4 days    | 4-5.5 weeks   |
| 4: Primary client UX + theme                 | 1-1.5 weeks | 5-7 weeks     |
| 5: Temporal enablement                       | 3-5 days    | 5.5-8 weeks   |
| 6: Installer + shell                         | 4-5 days    | 6-9 weeks     |
| 7: Migration + doctor                        | 4-5 days    | 7-10 weeks    |
| 8: Extras + polish                           | 3-5 days    | 7.5-11 weeks  |

**Total: 7.5-11 weeks of focused work.** Temporal enablement ships early (Phase 5) so installer, migration, and release all benefit. Longer if interleaved with other work.

The estimate intentionally allows for:

- Discovery and design time within each phase (ADV workflow includes `/adv-discover` + `/adv-design`)
- Code review iteration (ADV `/adv-review` and `/adv-harden` gates)
- Unforeseen complexity (Go template edge cases, Advance API drift, etc.)

If any phase blows its estimate by more than 2x, halt and re-plan. Do not push through.

---

## Post-v1: Temporal Dashboard (research complete, not yet scoped)

A web dashboard for ADV change lifecycle monitoring and control, consuming Temporal's workflow state. Research spike completed 2026-04-23; findings documented in [`../notes/2026-04-23-temporal-dashboard-research.md`](../notes/2026-04-23-temporal-dashboard-research.md).

**Prerequisite:** Phase 5 (Temporal Enablement) must ship first.

**Desired capabilities:**
- Change progress at a glance (gates, tasks, blockers)
- Multi-change orchestration view
- Agent activity and cost tracking (retries, time invested, doom-loop state)
- Temporal operational health (server, worker, namespace)
- Full control surface (approve gates, retry tasks, cancel changes, manage agenda)

**Key research findings:**
- Temporal Web has no plugin system — custom domain views require a separate frontend
- Browser → Temporal Server direct is not possible (no gRPC-Web, no CORS); requires a proxy
- Viable paths: ui-server sidecar (low effort, bundled with dev server), custom Node proxy, or OCA-embedded Go proxy (best UX, most effort)
- Advance exposes 5 change queries + 11 updates, 5 project queries + 4 updates — sufficient for a full control surface
- Search attributes enable filtered workflow listing across all changes

**Phase placement TBD** — depends on Phase 6.5 completion and user demand. Likely v1.1+ scope.
