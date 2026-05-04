# Phase Sequencing

OpenCode Advance v1.0 is developed in sequential phases. Each phase is one or more ADV changes. Phase dependencies are strict: Phase N cannot start until Phase N-1 has been archived.

---

## Now / Next (2026-05-04)

**Where we are:**

- Phases 0–8 shipped. v1.0 release-candidate work landed; tag publication pending.
- Operator Dashboard v1.0 shipped (`oca dashboard`, ~1,900 LOC).
- **In flight (other repo):** ADV plugin change `syncglobalpromptrefsinglefile` — fixes multi-`{file:...}` prompt-ref expansion failure that breaks all four ADV provider variants. Filed cross-project from this repo 2026-05-04. BLOCKER for clean `/adv-proposal` runs.

**Immediate next (in order, after ADV #6 archives):**

1. M3 — `installOcaUmbrellaPlugin` (1–2 h)
2. M2 — `applyLifecycleParity`
3. M4 — `instructionAssetsReal` (refinements pre-baked 2026-05-04: 4-file triage required)
4. M5 — `starterMigrationValidity`
5. M6 — `skillGuidanceRefresh` (refinements pre-baked 2026-05-04: worktree skill out of scope)

All five proposal drafts pre-flight-verified 2026-05-04 — see [`../notes/2026-05-04-m-queue-preflight-verification.md`](../notes/2026-05-04-m-queue-preflight-verification.md). Filing-ready.

**Then (Session & Resource Architecture):** OCA #1 Pattern B (in delivery) → OCA #2 hibernation → OCA #3 tmux-resurrect polish → ADV #4 idle-worker reaper → ADV #5 peer-session topology distinction.

**Backlog (SHOULD):** S1 runtime doctor canaries → S2 context budget audit → S3 permission-first agent config → S4 MCP suite profiles → S5 legacy-state doctor warning → S6 resume-hint outer terminal.

**Hygiene:** OpenCode session debt cleared 2026-05-04 (3 stale blank rows deleted via `bun ~/dev/oc-plugins/advance/scripts/opencode-session-doctor.ts --apply`).

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
- All declared MCP server variants from the curated stack are represented in tests
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

**Status:** COMPLETE — delivered in archived change `phase5TemporalEnablement` and merged to `trunk`.

**Goal:** Wire OCA to manage the `[temporal]` section in `stack.toml` and render it into the environment that Advance consumes. Scope is intentionally narrow: config expansion, apply rendering, health checks, and status bar integration. Dev-server supervision, `oca temporal` subcommands, and `temporal_bundle` wiring are deferred to Phase 6.5 or declined.

**Why this phase ships before installer/migration:** Advance already depends on Temporal. Every subsequent phase benefits from OCA managing it — the installer should install Temporal awareness, migration should produce a Temporal-ready stack, and v1.0 should ship with Temporal tooling. Waiting until after migration means shipping without Temporal support and retrofitting it later.

**Estimate:** 2-3 days (narrow scope)

**Deliverables (4 pillars):**

1. **Config expansion** — `internal/config/types.go` expanded with `TemporalSection`, `TemporalDevServer`, `TemporalEnvVar` structs; TOML parsing and validation for all `[temporal]` fields
2. **Apply rendering** — `internal/render/temporal.go` renders `[temporal]` config into `$OCA_CACHE_DIR/temporal.env` with `ADV_TEMPORAL_*` values using atomic write (reuses Phase 1 `WriteAtomic`); `cmd/oca/apply.go` wires `--target temporal`
3. **Doctor health checks** — `internal/health/temporal.go` registers reachability and namespace checks via the Phase 2 health-check registry; `cmd/oca/doctor.go` wires `--scope temporal`
4. **Status bar integration** — `lib/adv_status.sh` reads `[temporal]` config from `stack.toml` and shows Temporal reachability state in the tmux status bar when `[temporal].enabled = true`

**What was NOT delivered (explicitly out of scope):**

- Dev-server supervision (`temporal server start-dev` PID management, log capture, start/stop/restart) — deferred to Phase 6.5
- `oca temporal {status,start,stop,restart,logs}` subcommands — deferred to Phase 6.5
- `temporal_bundle` wiring into Advance plugin boot env — declined; Advance manages its own Temporal client bundle
- Production Temporal cluster management — out of scope for v1.0
- Temporal Cloud integration — generic `address` + `allow_remote` only

**Exit criteria:**

- [x] `stack.toml` parses and validates the full `[temporal]` section (enabled, address, namespace, allow_remote, dev_server, env_vars)
- [x] `oca apply --target temporal` with `[temporal].enabled = false` (default) is a no-op that reports "temporal disabled"
- [x] `oca apply --target temporal` with `[temporal].enabled = true` writes `$OCA_CACHE_DIR/temporal.env` atomically with all declared `ADV_TEMPORAL_*` values
- [x] `oca doctor --scope temporal` returns pass when address is reachable and namespace exists; fails with actionable remediation otherwise
- [x] Status bar shows Temporal state (enabled/disabled, reachability icon) when `[temporal].enabled = true`
- [x] `stack.example.toml` includes a fully documented `[temporal]` example block
- [x] All temporal touchpoints use isolated test config directories (never modify production state)

**Historical implementation reference:**

- Archived ADV change: `phase5TemporalEnablement`
- Merge commit on `trunk`: (to be recorded at archive time)
- Next recommended phase: **Phase 6: Installer + Shell Profile**

### Retrospective

Phase 5 shipped a narrow 4-pillar Temporal enablement. Scope was deliberately cut from the original proposal (which included dev-server supervision and `oca temporal` commands) to avoid blocking the installer phase. The config expansion and apply rendering were straightforward reuses of Phase 1/2 primitives. Health checks followed the existing registry pattern. Status bar integration required minimal changes to the existing `adv_status.sh` parser. The key lesson: Temporal infrastructure has a natural seam between "render config and check health" (Phase 5) and "supervise long-running processes" (Phase 6.5). Shipping the first seam unblocked all downstream phases without committing to process supervision that would require significant new shell/Go plumbing.

---

## Phase 5.5: Vision Slot Group Support

**Status:** Complete — delivered in archived change `addVisionSlotGroupSupport` and merged to `trunk`.

**Goal:** Add first-class `[mcp.slot_groups.*]` support to `stack.toml` so Playwright pools (and future slot-group use cases) can be declared declaratively and rendered by `oca apply` into both `vision/servers.yaml` and `opencode.json`.

**Estimate:** 2-3 days

**Why a .5 phase:** Slot groups are a narrow, self-contained feature that blocks Phase 6 (installer) because the installer must produce a complete `stack.toml` including Playwright slot groups. It does not warrant a full numbered phase, but it has enough surface (parser, validation, render, health, docs) to need explicit scoping.

**Deliverables:**

- `internal/config/types.go` — `SlotGroup` struct + `MCPSection.SlotGroups` field
- `internal/config/validate.go` — `validateMCPSlotGroups` with full collision detection (servers ↔ groups ↔ slots, template-name vs declared keys)
- `internal/render/vision_yaml.go` — emit top-level `slot_groups:` map in `vision/servers.yaml`
- `internal/render/opencode_json.go` — `RenderSlotGroupFragment` emitting single `.mcp.<group>` remote entry at `group_port`
- `internal/health/mcp.go` — tiered slot group health probe (`/v1/slots/{group}` → TCP fallback → warn on missing API)
- `stack.example.toml` — replace Playwright comments with three real slot group declarations
- `docs/design/stack-toml-schema.md` — document `[mcp.slot_groups.<name>]` section
- Golden-file tests updated to include slot groups in the full-stack fixture

**Exit criteria:**

- [x] `stack.toml` parses and validates `[mcp.slot_groups.<name>]` with all fields
- [x] `oca apply --target mcp` renders slot groups into `vision/servers.yaml` under `slot_groups:`
- [x] `oca apply --target mcp` emits a single `.mcp.<group-name>` entry per group at `group_port`
- [x] Port collision detection covers servers, group ports, and synthesized slot ports across groups
- [x] `oca doctor --scope mcp` reports slot group health (pass/warn) via Vision's slots API or TCP fallback
- [x] `stack.example.toml` includes working Playwright slot group examples that pass validation
- [x] All tests pass with no regressions in existing MCP server behavior

**Dependency notes:**

- Blocks Phase 6 (installer): installer must emit a complete stack including Playwright slot groups
- Blocks Phase 7 (migration): migration must convert existing hand-managed slot groups into declarative config
- Depends on Phase 1 (MCP apply foundations) and Phase 5 (Temporal) — the actual blockers are Phase 1 render/health primitives, not Temporal

---

## Phase 6: Installer + Shell Profile

**Status:** Complete — delivered in commits `172fa2c..bbbb99d` and merged to `trunk`.

**Goal:** `oca install` performs end-to-end first-time setup. Shell profile wiring (PATH, completions) is idempotent and reversible. Installer includes Temporal prerequisite checks and optional dev-server setup (leveraging Phase 5).

**Estimate:** 4-5 days

**Shipped deliverables:**

- `internal/install/shellprofile.go` — managed-block injection into `~/.zshrc` and `~/.bashrc` with idempotent add/remove/diff/edit-detection
- `internal/install/prereq.go` — prerequisite checker for git, tmux (≥3.4), OpenCode, vision, and optional Temporal CLI
- `internal/install/flow.go` — `Install` and `Uninstall` flow orchestration (prereqs → apply → profile injection or removal)
- `cmd/oca/install.go` — `oca install [--yes]` with end-to-end setup
- `cmd/oca/uninstall.go` — `oca uninstall` removes managed blocks without touching user content
- `cmd/oca/completion.go` — `oca completion <bash|zsh|fish>` shell completion generation
- `templates/shell_profile.block.gotmpl` — managed block template with PATH + completion wiring
- Integration tests for install → uninstall → install round-trips
- Unit tests for shell profile block operations (WriteBlock, RemoveBlock, DiffBlock, IsEdited)

**Exit criteria:**

- [x] `oca install --yes` runs end-to-end: prerequisite checks, `oca apply` all targets, shell profile block injection
- [x] `oca install` adds a managed block to `~/.zshrc` and `~/.bashrc` with PATH + completions
- [x] `oca completion zsh` and `oca completion bash` print working completion scripts
- [x] `oca uninstall` removes all managed blocks without touching user-owned content
- [x] Re-running `oca install` after interruption completes remaining steps (idempotent)

**Historical implementation reference:**

- Commit range on `trunk`: `172fa2c..bbbb99d` (5 commits)
- Next recommended phase: **Phase 6.5** or **Phase 7**

### Retrospective

Phase 6 shipped the installer, uninstaller, shell completion, and shell profile managed-block system. The shell profile package follows a sentinel-delimited block pattern (OCA-specific delimiters wrapping PATH export + completion source) that supports idempotent add, clean removal, drift detection (`DiffBlock`), and user-edit detection (`IsEdited`). The prerequisite checker validates tool versions (e.g., tmux ≥3.4) and gracefully handles missing optional dependencies (Temporal CLI). Flow orchestration composes prereqs → apply → profile injection as discrete steps so an interrupted install can be resumed. The managed block intentionally excludes tmux conf injection — that remains the session lifecycle's responsibility.

---

## Phase 6.5: Temporal Dev-Server Supervision

**Status:** Shipped — `phase65TemporalDevServer`.

**Goal:** `oca temporal` subcommands for dev-server lifecycle management: start, stop, restart, status, logs. Manages the Temporal dev-server process that Advance depends on for durable workflow state.

**Estimate:** 2-3 days

**Why a .5 phase:** Narrow process-supervision scope that extends Phase 5's config rendering. Advance already uses Temporal as its primary state backend; OCA needs to manage the dev-server process lifecycle for local development. Does not warrant a full numbered phase.

**Delivered:**

- `cmd/oca/temporal.go` — `oca temporal {status,start,stop,restart,logs}`
- `internal/temporal/supervise.go` — dev-server PID management, log capture, health polling
- `$OCA_CACHE_DIR/temporal/` runtime metadata: PID JSON, log file, and persistent `temporal.db`
- Status bar remains compatible through the existing Phase 5 `temporal.env` reachability probe

**Exit criteria:**

- `oca temporal start` launches `temporal server start-dev` in the background and writes a PID file
- `oca temporal stop` sends SIGTERM and waits for clean shutdown
- `oca temporal status` reports server health (running/stopped, namespace reachable)
- `oca temporal logs` prints bounded recent log output by default; `--follow` streams explicitly
- Dev-server state is visible in the tmux status bar when `[temporal].enabled = true`
- All commands use isolated test config directories

---

## Phase 7: Migration + Doctor Expansion

**Status:** Shipped — `phase7MigrationOpenChadDoctor`.

**Goal:** `oca migrate from-open-chad` works on the maintainer's real open-chad state. Doctor expands to cover Advance state, Temporal health, and cross-component consistency. Migration produces a Temporal-ready stack.toml (leveraging Phase 5).

**Estimate:** 4-5 days

**Delivered:**

- `internal/migrate/openchad.go` — reads open-chad state, emits stack.toml
- `cmd/oca/migrate.go` — `oca migrate from-open-chad`, `oca migrate init`
- `internal/health/advance.go` — ADV state checks
- `internal/health/cross.go` — cross-component consistency checks
- Expanded `oca doctor` output with migration-era checks, building on shipped `mcp`, `plugins`, `skills`, `temporal`, and `adv-assets` scopes

**Also delivered out-of-phase:**

- `oca doctor --scope adv-assets` shipped in archived change `ocadoctorassetdrift` and merged to `trunk`
- The scope audits plugin-provided asset ownership boundaries (`ORPHANED`, `DUPLICATE-OWNER`, `STALE`) and is read-only
- Phase 7 should not reimplement asset-ownership drift; it should add migration/ADV/cross-component checks only

**Exit criteria:** ✓ Complete

- `oca migrate from-open-chad` successfully reads the maintainer's open-chad state and produces a valid stack.toml
- The generated stack.toml, when applied, reproduces the user's current state (verified by diff)
- `oca doctor` reports on: Advance checkout, plugin build state, ADV state directory, Temporal health, cross-component consistency (e.g., agent model references a declared provider), and asset-ownership drift via the shipped `adv-assets` scope
- Migration preserves user customizations (plugin paths, provider models, agent assignments)

**Tasks (completed):**

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

**Status:** Release-candidate implementation — `phase8ExtrasPolishDiscord`.

**Goal:** Discord integration (with new taglines), release pipeline, final README, polish pass.

**Estimate:** 3-5 days

- `cmd/oca/discord.go` + `internal/discord/` — Go-native Discord Rich Presence integration
- `assets/discord/taglines.toml` — all new taglines (data-driven, no "chad" era jokes)
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

- tk-phase8-01: Port Discord Rich Presence integration from open-chad
- tk-phase8-02: Rewrite all taglines (data-driven TOML file)
- tk-phase8-03: Implement `oca discord enable/disable/status`
- tk-phase8-04: Set up goreleaser config
- tk-phase8-05: Write `.github/workflows/release.yml`
- tk-phase8-06: Test release pipeline with a pre-release tag (v1.0.0-rc1)
- tk-phase8-07: Write final README
- tk-phase8-08: Write INSTALL.md
- tk-phase8-09: Generate CHANGELOG.md
- tk-phase8-10: Final polish pass: inconsistencies, typos, dead links

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
- **Phase 5.5 (Slot Groups) blocks on Phase 1** (extends MCP render/health/validation). Narrow feature that blocks the installer because the example stack must include Playwright slot groups.
- Phase 6 (Installer) blocks on Phase 4 (uses session lifecycle + theme), Phase 5 (includes Temporal prerequisite checks), and Phase 5.5 (installer must emit complete stack with slot groups)
- Phase 6.5 (Temporal supervision) blocks on Phase 5 (config rendering) and Phase 6 (installer prerequisite checks)
- Phase 7 (Migration) blocks on Phase 6 (migration produces a stack.toml, which needs full coverage to be useful)
- Phase 8 (Extras) blocks on Phase 7 (release is the last step)

### Out-of-phase shipped work

The following capabilities were shipped outside of numbered phase ADV changes:

- **Operator dashboard** (`oca dashboard`) — read-only web dashboard showing cross-project ADV changes, tmux sessions, and Temporal/worker health, updated via SSE. `internal/dashboard/` (~1,900 LOC) with Datastar frontend, Temporal polling, tmux control-mode session watcher. Shipped as a standalone change. Evolution plans in [`../notes/2026-05-01-dashboard-architecture-research.md`](../notes/2026-05-01-dashboard-architecture-research.md).
- **Pane management** (`oca pane`) — per-pane state operations for OCA tmux sessions. Reads/writes per-pane JSON state (`$XDG_STATE_HOME/oca/panes/`) for session ↔ directory ↔ watchdog correlation.
- **Session watchdog** (`oca watchdog`) — monitors pane activity and tracks bump counts, last-activity timestamps, and idle-state transitions for OCA-managed tmux sessions.
- **Temporal CLI detection** (`internal/temporal/detect.go`) — host-level detection for `temporal` CLI and `node` binary presence, used by the installer prerequisite checker and health system.
- **`oca doctor --scope adv-assets`** — plugin-provided asset ownership drift audit. Shipped in archived change `ocadoctorassetdrift`.

### Unphased planned commands

The following commands are documented as planned but have no phase assignment yet:

- `oca add mcp <name>` / `oca add plugin <name>` — interactive declaration addition (listed in v1-implementation.md success criteria)
- `oca remove mcp <name>` / `oca remove plugin <name>` — interactive declaration removal
- `oca clean` — cleanup stale backups, orphaned cache entries, and expired state files
- Agent rendering — rendering agent model assignments from `stack.toml` into `opencode.json`. Explicitly deferred past v1.0 per v1-implementation.md

---

## Estimated timeline

| Phase                                        | Estimate    | Status      |
| -------------------------------------------- | ----------- | ----------- |
| 0: Foundation + brand                        | 3-5 days    | ✓ Complete  |
| 1: stack.toml + MCP apply                    | 1-2 weeks   | ✓ Complete  |
| 2: Plugin + instruction mgmt                 | 1 week      | ✓ Complete  |
| 3: Core opencode.json coverage               | 1 week      | ✓ Complete  |
| 3.5: Skills + commands + formatters          | 3-4 days    | ✓ Complete  |
| 4: Primary client UX + theme                 | 1-1.5 weeks | ✓ Complete  |
| 5: Temporal enablement                       | 3-5 days    | ✓ Complete  |
| 5.5: Vision slot group support               | 2-3 days    | ✓ Complete  |
| 6: Installer + shell                         | 4-5 days    | ✓ Complete  |
| 6.5: Temporal dev-server supervision          | 2-3 days    | ✓ Shipped   |
| 7: Migration + doctor                        | 4-5 days    | ✓ Complete  |
| 8: Extras + polish                           | 3-5 days    | RC shipped   |

**Total shipped: Phases 0–8 (11.5–15.5 weeks).** Remaining before v1.0: release-candidate smoke validation and tag publication.

---

## Post-v1: Operator Dashboard

**Status:** v1.0 shipped — `oca dashboard` is live with read-only unified table, SSE real-time updates, tmux session watcher, and Temporal health polling.

The dashboard is a Go-embedded web UI (`internal/dashboard/`, ~1,900 LOC) using Datastar for reactive rendering, SSE for server-push state updates, and tmux control-mode for event-driven session tracking. Launched via `oca dashboard [--bind <addr>] [--port <n>] [--no-open]`.

**Architecture research:**

- Direction locked and documented in [`../notes/2026-05-01-dashboard-architecture-research.md`](../notes/2026-05-01-dashboard-architecture-research.md)
- Frontend: Datastar + server-rendered HTML + Tailwind (no SPA, no JS framework)
- Binary delivery: single Go binary via `//go:embed`
- Real-time: server polls Temporal (gRPC, 2–5s) → SSE push to browser
- Temporal access: `client.ListWorkflow` + existing search attributes
- tmux state: control-mode client (`tmux -Loca -C`) parsing `%`-notifications
- Cross-host: out of scope for v1.x

### Evolution roadmap (from research note)

| Phase | Capability | Cost |
|---|---|---|
| v1.0 (shipped) | Read-only unified table; loopback only | — |
| v1.1 | Per-row controls (gate approve, retry, cancel) + local "switch session" (tmux switch-client) | 1-2 weeks |
| v1.2 | Local "open in new terminal window" (spawn OS terminal) + auth scaffolding (token-based) | 1 week |
| v2.0 | Web terminal (xterm.js + PTY broker) + Tailscale-friendly bind + cross-host federation | 3-4 weeks |

### Session resume / web terminal (v2.0 target)

The long-term goal: clicking a session row in the dashboard opens or resumes that session — including from a browser on another device. Three approaches documented:

- **A.** tmux switch-client (local only, trivial)
- **B.** Spawn OS terminal + tmux attach (local only, low cost)
- **C.** Web terminal (xterm.js + WebSocket PTY broker) — the real answer for cross-device

Architecture implications for v2.0: WebSocket transport alongside SSE, auth model required, network exposure beyond loopback, xterm.js bundle (~200 KB gzip). None of these break v1.0 architecture — they coexist as additional routes on the same Go HTTP server.

Full details, open questions, and reference implementations in the [research note](../notes/2026-05-01-dashboard-architecture-research.md).

---

## Post-v1: OCA Reliability + Runtime Correctness Queue

**Status:** Reviewed and accepted 2026-05-03. **Pre-flight verified 2026-05-04** — all five MUST proposals filing-ready; M4 and M6 drafts updated in place to bake in pre-flight refinements. Filing waits on ADV #6 (`syncglobalpromptrefsinglefile`) archive + a fresh non-stub provider session.

**Goal:** address OCA-owned correctness gaps surfaced by the layer-strategy follow-up before broad post-v1 session-architecture work. These changes make `oca apply`, OCA-owned assets, generated stacks, skill guidance, runtime doctor checks, and per-agent tool exposure trustworthy enough to support the larger Pattern B / hibernation track.

**Pre-flight findings (2026-05-04):**

- M3, M2, M5 — claims clean as drafted; file as-is.
- M4 — live `~/.config/opencode/instructions/` has 14 files, README inventory lists 10. Proposal expanded to require triage of `caveman.md`, `criteria-prioritizer.md`, `global-verify-policy.md`, `post_install_verification.md` and a README-vs-asset-dir parity gate.
- M6 — `worktree/SKILL.md` already clean (claim was stale); proposal scope narrowed to `assets/skills/README.md` deleted-skill cleanup + `mcp-selection/SKILL.md` schema-current MCP function names. Banned-term docs check requirement added.

**Handoff queue:** [`./2026-05-03-oca-roadmap-queue.md`](./2026-05-03-oca-roadmap-queue.md)
**Pre-flight notes:** [`../notes/2026-05-04-m-queue-preflight-verification.md`](../notes/2026-05-04-m-queue-preflight-verification.md)

### Accepted roadmap entries

| Order | Priority | Change | Proposal file | Roadmap placement |
|---|---|---|---|---|
| 1 | **M3** | OCA umbrella plugin install | [`./2026-05-03-oca-plugin-install.md`](./2026-05-03-oca-plugin-install.md) | Prerequisite for Pattern B and hibernation; same work as Session Architecture change #0. |
| 2 | **M2** | OCA apply lifecycle parity | [`./2026-05-03-oca-apply-lifecycle-parity.md`](./2026-05-03-oca-apply-lifecycle-parity.md) | Must land before treating bare `oca apply` as the source-of-truth apply command. |
| 3 | **M4** | OCA-owned instruction assets are real files | [`./2026-05-03-oca-instruction-assets-real.md`](./2026-05-03-oca-instruction-assets-real.md) | Source-of-truth asset fix; pairs with M5 and M6. |
| 4 | **M5** | Starter and migration stack validity | [`./2026-05-03-oca-starter-migration-validity.md`](./2026-05-03-oca-starter-migration-validity.md) | Ensures generated starter/migration stacks validate and apply. |
| 5 | **M6** | OCA skill guidance refresh | [`./2026-05-03-oca-skill-guidance-refresh.md`](./2026-05-03-oca-skill-guidance-refresh.md) | Removes stale OpenChad/old-ADV/tool-name guidance from OCA-owned skills. |
| 6 | **S1** | OCA runtime doctor canaries | [`./2026-05-03-oca-runtime-doctor-canaries.md`](./2026-05-03-oca-runtime-doctor-canaries.md) | Adds runtime checks for failures static config checks miss. |
| 7 | **S2** | Context budget audit + instruction loading diet | [`./2026-05-03-context-budget-audit.md`](./2026-05-03-context-budget-audit.md) | Measures prompt/tool-schema load before trimming or profile changes. |
| 8 | **S3** | Permission-first agent configuration | [`./2026-05-03-agent-permission-first-config.md`](./2026-05-03-agent-permission-first-config.md) | Aligns OCA agent config with current OpenCode permission guidance. |
| 9 | **S4** | MCP tool suite profiles and per-agent exposure | [`./2026-05-03-mcp-tool-suite-profiles.md`](./2026-05-03-mcp-tool-suite-profiles.md) | Makes MCP tool exposure explicit per agent/profile. |
| 10 | **S5** | Legacy in-repo ADV state doctor warning | [`./2026-05-03-legacy-adv-state-doctor-warning.md`](./2026-05-03-legacy-adv-state-doctor-warning.md) | Warns about legacy mutable `.adv` state while preserving `.adv/specs`. |

### Sequencing rule

Run the MUST queue in order **M3 → M2 → M4/M5/M6** before starting broad OCA session-architecture implementation beyond the plugin prerequisite. Then run SHOULD items as capacity allows: **S1/S2 → S3/S4 → S5**. S1 is especially valuable immediately because runtime prompt-resolution failures can pass static config checks.

ADV caveat: the proposals are drafted but not created as live ADV changes in this review pass. Start them with `/adv-proposal` only after a fresh OpenCode session verifies that the selected ADV provider agent no longer resolves to `[ADV:PROVIDER_STUB_UNEXPANDED]`.

---

## Post-v1: Session & Resource Architecture

**Status:** Decision-locked 2026-05-03. **Seven change proposals drafted** (one added 2026-05-03 post-reconnaissance). **ADV #6 filed in ADV-plugin repo 2026-05-04** as `syncglobalpromptrefsinglefile` (cross-project from OCA, source draft preserved as provenance). OCA #0 = M3 in the reliability queue above. Broad implementation of OCA #1–3 + ADV #4–5 starts after the ADV provider prompt blocker is fixed and the OCA reliability MUST queue above has shipped.

**Goal:** adapt OCA's session topology and resource-sharing model for the operator's actual workload — 8+ concurrent agents per project, 4–5 active projects, on a 42 GB WSL2 ceiling. Replace per-invocation tmux sessions (Pattern A) with session-per-project (Pattern B), surface per-window ADV status markers in the tmux status bar, and add graceful opencode hibernation as the dominant RAM lever (~16 GB savings at 50% idle of 40 agents).

**Decision lock + research trail:**

- Parent doc (decision-lock + lever analysis + research citations): [`./2026-05-03-session-and-resource-architecture.md`](./2026-05-03-session-and-resource-architecture.md)
- Resolutions for 13 questions (10 from parent §6, 3 from architectural critique) recorded in parent §9
- Discovery-phase items per change inlined into each child proposal

### Seven-change split

Work splits into **7 ADV changes across 2 repositories** (OCA + ADV plugin), plus the existing in-flight Operator Dashboard track. The split grew from 5 to 7 during 2026-05-03 reconnaissance: ADV #6 surfaced as a blocker (multi-`{file:...}` prompt-ref expansion failure in OpenCode 1.14.33), and OCA #0 surfaced as a prerequisite (the OCA umbrella plugin is not registered in the operator's opencode.json today).

| #     | Change                                                  | Repo            | Effort       | Proposal file                                                                                                  |
| ----- | ------------------------------------------------------- | --------------- | ------------ | -------------------------------------------------------------------------------------------------------------- |
| **6** | **sync-global.sh single-ref prompt fix (BLOCKER)**      | **ADV plugin**  | 1–2 hours    | [`./2026-05-03-adv-sync-prompt-ref-fix.md`](./2026-05-03-adv-sync-prompt-ref-fix.md)                           |
| **0** | **OCA umbrella plugin install (PREREQ for #1, #2)**     | **OCA**         | 1–2 hours    | [`./2026-05-03-oca-plugin-install.md`](./2026-05-03-oca-plugin-install.md)                                     |
| 1     | Pattern B + per-window status decode + smart `oca` entry | OCA             | 3–6 days     | [`./2026-05-03-pattern-b-session-topology.md`](./2026-05-03-pattern-b-session-topology.md)                     |
| 2     | Graceful opencode session hibernation                   | OCA             | 2–4 days     | [`./2026-05-03-graceful-hibernation.md`](./2026-05-03-graceful-hibernation.md)                                 |
| 3     | tmux-resurrect (manual) + concurrency warning polish    | OCA             | ~1 day       | [`./2026-05-03-tmux-resurrect-warning-polish.md`](./2026-05-03-tmux-resurrect-warning-polish.md)               |
| 4     | Idle Temporal worker reaper                             | ADV plugin      | 1–2 days     | [`./2026-05-03-adv-idle-worker-reaper.md`](./2026-05-03-adv-idle-worker-reaper.md)                             |
| 5     | Peer-session topology distinction (rescoped from coordinated marker reframe) | ADV plugin | 1–2 hours | [`./2026-05-03-adv-coordinated-session-marker.md`](./2026-05-03-adv-coordinated-session-marker.md) |

**Reconnaissance findings 2026-05-03 (post-decision-lock):**

- ADV #5 was originally scoped as "replace `[ADV:WARN] Concurrent OpenCode sessions detected` with `[ADV:COORDINATED]` info marker." Reconnaissance discovered the marker reframe is **already shipped** — ADV plugin emits `[ADV:PEER_SESSIONS] N peer session(s) active` at info level (not warn) since a prior 2026-04-era change. ADV #5 was rescoped to add only the **topology distinction** (worktree-aware coordinated-vs-conflict classification), reducing effort from 0.5–1 day to 1–2 hours.
- OCA umbrella plugin (`plugins/oca/`) source exists in OCA repo but is **not registered** in operator's `opencode.json` plugin array (only ADV, claude-max, morph-fast-apply, vision, codex-auth are registered). `~/.local/state/oca/panes/` doesn't exist → no pane state being written. Created OCA #0 to address the gap before #1 + #2 work begins.

ADV-repo proposals (#4, #5, #6) are drafted in this OCA repo for split coherence; each carries an explicit "Filing path" header instructing the operator to move/redraft the proposal in the ADV repo before invoking `/adv-proposal` from inside `~/dev/oc-plugins/advance`.

### Sequencing

| Order | Change                                          | Reason                                                                                                          |
| ----- | ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| 1     | **ADV #6** — sync-global single-ref prompt fix   | BLOCKER. OpenCode 1.14.33 does not expand multi-`{file:...}` refs in `agent.X.prompt`; ADV provider variants run in degraded persona until fixed. Must ship before any other change can run `/adv-proposal` cleanly. **Status (2026-05-04):** filed in ADV-plugin repo as `syncglobalpromptrefsinglefile` (cross-project from OCA, source draft preserved at `docs/proposals/2026-05-03-adv-sync-prompt-ref-fix.md`). Operator workaround D in effect until archive. |
| 2     | **OCA reliability MUST queue** — M3 → M2 → M4/M5/M6 | PREREQ before broad OCA session architecture. M3 is the umbrella plugin install / Session Architecture #0; M2/M4/M5/M6 make apply/assets/generated stacks/skills trustworthy before larger changes. **Pre-flight verified 2026-05-04:** see [`../notes/2026-05-04-m-queue-preflight-verification.md`](../notes/2026-05-04-m-queue-preflight-verification.md) — M3/M2/M5 ready as drafted; M4 needs README expansion (4 live files missing); M6 should drop the worktree-skill claim (already fixed) and focus on README + mcp-selection. |
| 3     | OCA #1 — Pattern B                              | Highest UX value at 8-agent scale; unblocks #2's status surface and the rest of the OCA work.                  |
| 4     | OCA #2 — Hibernation                            | Dominant RAM lever (~16 GB headroom). Depends on #1's status bar to surface 💤 marker.                          |
| 5     | OCA #3 — tmux-resurrect + warning polish        | Small composable; can run in parallel with #2.                                                                  |
| 6     | ADV #4 — Idle worker reaper                     | ADV-repo work; parallelizable with OCA. Lands once worker-singleton handover is verified at 8-agent scale.      |
| 7     | ADV #5 — Peer-session topology distinction      | Quality-of-life enhancement on existing peer-session detection; lands any time after Pattern B is in operator hands. |

### Provider prompt workaround note

This section historically documented a manual `adv-claude` concatenated-prompt workaround. Treat that workaround as ephemeral: any `sync-global.sh --fix` run can replace it. The current source of truth is the runtime canary: `opencode debug agent adv-gpt` (or the selected provider agent) must not resolve to `[ADV:PROVIDER_STUB_UNEXPANDED]` before filing new ADV proposals.

### Composition with Operator Dashboard track

Pattern B (#1) materially improves dashboard rendering — one row per project instead of per-invocation slug clutter. Hibernation (#2) surfaces "5 agents active, 3 hibernated" per row; resume-from-row buttons land at dashboard v1.2. Web terminal (v2.0) gives cross-device session resume that composes with hibernation to extend hibernation across devices. The two tracks are independent: dashboard work continues per its existing v1.1 → v2.0 plan whether session-architecture work has shipped or not.

---

## Post-v1: Context Budget & Instruction Loading Diet

**Status:** Accepted as OCA roadmap SHOULD item S2 in the OCA Reliability + Runtime Correctness Queue. Drafting via `/adv-proposal` deferred until ADV provider prompt refs are healthy.

**Goal:** measure and reduce baseline context load without weakening strict tool, shell, MCP, lgrep, morph, security, or ADV safety instructions. OCA should make prompt/tool-schema cost visible before trimming anything.

| Change | Repo | Effort | Proposal file |
| ------ | ---- | ------ | ------------- |
| Context Budget Audit + Instruction Loading Diet | OCA, possible ADV follow-up | 1–3 days | [`./2026-05-03-context-budget-audit.md`](./2026-05-03-context-budget-audit.md) |

Planned outcomes:

- Add `oca doctor --scope context` to report estimated baseline context cost.
- Break down context contributors by instructions, agent prompts, plugin instruction refs, commands, skills, and exposed tool schemas.
- Compare agent profiles (`adv`, `build`, `explore`, `librarian`, `mechanic`, `general`) by tool count and estimated schema load.
- Surface top prompt contributors and actionable recommendations.
- Keep strict tool/safety rules detailed; reduce load through agent-specific exposure and on-demand methodology loading.
- Investigate whether non-ADV sessions can avoid loading full ADV workflow detail while ADV sessions retain required ADV law.
