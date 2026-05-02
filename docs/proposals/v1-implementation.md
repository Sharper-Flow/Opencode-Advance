# v1.0 Implementation Proposal

**Change ID (suggested):** `opencodeAdvanceV1`
**Scope:** Full v1.0 implementation of OpenCode Advance — a declarative, Go-based, professional rewrite of the `open-chad` environment platform.

This document is the source material for the first ADV change in this repository. When OpenCode is opened in this repo and ADV is initialized, this proposal will be loaded by `/adv-proposal` to create the formal change state.

## Progress snapshot

- Phase 0 (`phase0FoundationBrand`) is complete, archived, and merged to `trunk`
- Phase 1 (`phase1StackTomlParserMcpApply`) is complete, archived, and merged to `trunk`
- Phase 2 (`phase2PluginInstruction`) is complete, archived, and merged to `trunk`
- Phase 3 (`phase3CoreOpencodeJsonCoverage`) is complete, archived, and merged to `trunk`
- Phase 3.5 (`addPhase35ConfigCoverageSkills`) is complete, archived, and merged to `trunk`
- Phase 4 (`phase4FoundationSessionTheme` + `completePhase4SessionLifecycle`) is complete, archived, and merged to `trunk`
- Phase 5 (`phase5TemporalEnablement`) is complete, archived, and merged to `trunk`
- Phase 5.5 (`addVisionSlotGroupSupport`) is complete, archived, and merged to `trunk`
- Out-of-phase hardening (`ocadoctorassetdrift`) is complete, archived, and merged to `trunk`; `oca doctor --scope adv-assets` now covers plugin/OCA asset ownership drift
- Phase 6 (installer + shell profile) is complete, archived, and merged to `trunk`
- Phase 6.5 (`phase65TemporalDevServer`) is complete, archived, and merged to `trunk`
- Phase 7 (`phase7MigrationOpenChadDoctor`) is complete, archived, and merged to `trunk`
- Phase 8 (`phase8ExtrasPolishDiscord`) is implemented as the v1.0 release-candidate change
- Out-of-phase: operator dashboard (`oca dashboard`), pane management (`oca pane`), session watchdog (`oca watchdog`), Temporal CLI detection (`internal/temporal/detect.go`)
- This document remains the umbrella roadmap and historical context for the v1.0 effort

---

## Problem statement

### Current state

The user currently runs two entangled projects that together provide their OpenCode environment and workflow:

1. **`open-chad`** (`JRedeker/open-chad`) — a bash-based meta-installer built in early 2025. It provisions a tmux-driven environment (sessions, status bar, boot animation, LLM quota gauges), installs a small set of plugins, and copies canned files into `~/.config/opencode/`. Visual identity is playful — synthwave colors, per-session randomized borders, NvChad-inspired agent palette, "chad"-era branding throughout.

2. **`Advance`** (`Sharper-Flow/Advance`) — a TypeScript OpenCode plugin for spec-driven development. It provides slash commands (`/adv-*`), a dedicated ADV orchestrator agent, consolidated specialist agents (including plan/build, `adv-researcher`, and `tron`), skills, and workflow state management. Runs standalone. Has its own sync script (`scripts/sync-global.sh`) that copies its assets and injects overlay blocks into shared agents in `~/.config/opencode/`.

Both projects write to `~/.config/opencode/`. The overlap is 19 files (4 agents, 15 commands, plus skills). The agent files are divergent supersets — Advance versions add ADV tool grants and overlay markers that the open-chad versions lack. Whichever project's sync script runs last wins.

### Pain points

1. **File ownership conflict.** The two-writers-one-directory problem creates silent drift. If the user runs Advance's `sync-global.sh`, then later runs `openchad update`, the Advance-specific agent tool grants are wiped. The reverse is also true in the inverse direction.

2. **Incomplete MCP management.** open-chad installs 4 MCP servers (context7, grep-app, lgrep, firecrawl). The user's live stack evolved beyond that baseline: Vision, context7, svelte-mcp, kagi, firecrawl, lgrep, sentry, and Playwright slot groups are core; other servers are project/on-demand. Anything outside open-chad's 4-server set was hand-wired into `opencode.json` / `vision/servers.yaml` with no single source of truth.

3. **No provider/agent/permission management.** open-chad does not manage providers (google/openai/openrouter model configs), agent-to-model assignments, bash/directory permission rules, or LSP configs. All of these live in a manually-edited `~/.config/opencode/opencode.json`.

4. **No reproducibility.** Plugin checkouts pull `latest` from `trunk` on every `openchad update`. There is no way to pin a known-good version of the Advance plugin, morph-fast-apply, or any other plugin. A reproduction of "my environment on April 6th" is impossible.

5. **No declarative source of truth.** The user's complete environment is spread across:
   - `~/.config/opencode/opencode.json` (hand-edited)
   - `~/.config/vision/servers.yaml` (managed by open-chad, partially)
   - `~/dev/open-chad/config/opencode/` (bundled templates)
   - `~/dev/oc-plugins/advance/.opencode/` (Advance's overlay/command sources)
   - shell profile rc files (paths, completions)
   - `~/.tmux.conf` (theme source, session hooks)
   - ad-hoc configurations stored nowhere

6. **Brand tone mismatch.** The "chad" heritage was appropriate for the project's earlier playful spirit but no longer matches the user's intended audience or professional positioning. Synthwave edges, rotating Discord taglines, NvChad-inspired agent palette, and the "chad" wordmark all need to be replaced.

7. **No vertical integration.** Advance tracks rich workflow state (active changes, gates, tasks, wisdom) but the environment layer (tmux status bar, doctor checks) knows nothing about it. `openchad doctor` does not verify that the Advance plugin is built or that ADV state is readable. The tmux status bar does not show the active change or gate progress.

8. **Hand-rolled shell scripting at scale.** open-chad is ~50 bash files, including 828-line `wizard.sh` and ~60-line Node.js heredocs embedded in shell scripts for JSON manipulation. This is at the limit of what bash should be doing. Adding new features means deeper bash, not cleaner code.

### What the user wants

- **A professional brand.** New name ("OpenCode Advance"), new wordmark (GBA-stylized "Advance" with frosted indigo accent), new palette (obsidian/slate/graphite), new theme, new boot animation, new Discord taglines. No "chad" heritage.

- **A declarative stack.** One `stack.toml` file that owns every slice of the OpenCode configuration the user cares about. `oca apply` renders it. `oca doctor` verifies it. `oca diff` shows drift. `oca pin` captures reproducibility. Full coverage: MCP servers, plugins, instructions, providers, permissions, watcher, LSP, primary client/session UX, theme, plus the remaining OCA-owned config surfaces in later phases.

- **Tight MCP integration.** Core MCP servers managed through one tool. No more hand-editing `opencode.json`. No more drift between `vision/servers.yaml` and what OpenCode actually knows about.

- **Advance as a clean dependency.** The Advance plugin is declared in `stack.toml`, cloned by OCA, built by OCA, and wired by OCA — but its asset sync is delegated to its own `sync-global.sh` script. OCA never duplicates Advance-owned files.

- **Clean cutover.** Development happens in isolated test config directories. open-chad stays the daily driver until v1.0 is ready. At release time: one-shot migration, clean uninstall of open-chad, clean install of OpenCode Advance.

- **Vertical integration with Advance.** The current status UI shows active ADV changes and gate progress. `oca doctor` verifies ADV plugin state. The boot splash knows about the workflow context.

---

## Success criteria

The v1.0 release is ready when all of the following are true:

### Brand & identity

- [ ] Repository is `Sharper-Flow/Opencode-Advance`
- [ ] No references to "chad", "open-chad", or "openchad" anywhere in the repository
- [ ] Wordmark ("OpenCode" over "ADVANCE") renders correctly in terminal with indigo coloring
- [ ] Color palette (Obsidian/Slate/Graphite/Ivory/Indigo) is implemented in `lib/palette.sh` and used consistently
- [ ] New Obsidian theme assets exist for the current primary client and tmux status bar (`assets/themes/obsidian.tmux.conf`)
- [ ] Boot splash renders wordmark with frosted indigo effect, opt-out via `OCA_BOOT_SPLASH=0`
- [ ] All Discord Rich Presence taglines are rewritten (no "chad"-era jokes)
- [ ] README presents the new brand with wordmark, palette, and professional tone

### CLI core (`oca`)

- [ ] `oca install` performs first-time setup end-to-end
- [ ] `oca apply` renders `stack.toml` into all target files idempotently
- [ ] `oca apply --dry-run` shows the plan without writing
- [ ] `oca diff` shows drift between stack.toml and actual state on disk
- [ ] `oca doctor` runs 30+ health checks in parallel, reports results with Obsidian-themed output
- [ ] `oca pin` captures current plugin git SHAs into stack.toml
- [ ] `oca update` pulls latest (respecting pins), rebuilds, re-applies
- [ ] `oca add mcp <name>` and `oca add plugin <name>` interactively add declarations
- [ ] `oca remove mcp <name>` and `oca remove plugin <name>` remove declarations
- [ ] `oca migrate from-open-chad` reads current state and emits a valid stack.toml
- [ ] `oca version` prints wordmark + version + commit SHA
- [ ] All commands exit with documented exit codes
- [ ] All commands support `--dry-run` where applicable

### Declarative coverage

`stack.toml` and `oca apply` manage all of:

**Curated v1 baseline (not an exact mirror of the maintainer's live environment):**

- [ ] MCP servers: all relevant servers can be declared with explicit `autostart` / `enabled` semantics; the curated v1 example includes the current core server set (vision, context7, svelte-mcp, kagi, firecrawl, lgrep, sentry) and Playwright slot groups (`playwright-headless`, `playwright-headed`, `playwright-auth`) declared via `[mcp.slot_groups.*]`; additional on-demand/project servers (sonarqube, arxiv-mcp, time, figma-mcp, xray, pokeedge data/sync ops, etc.) remain modelable in `stack.toml` or hand-managed where they depend on project-local paths
- [ ] Plugins: advance, morph-fast-apply, vision-opencode, openai-codex-auth, md-table-formatter, anthropic-auth — mix of git checkout and npm sources
- [ ] Skills: 8 OCA-owned skills (lgrep, mcp-selection, morph, prioritizer, worktree, caveman, caveman-commit, caveman-review) copied from `assets/skills/` to `~/.config/opencode/skills/`; ADV methodology skills remain plugin-owned
- [ ] Instructions: identity, rules, shell_strategy, test_resource_guardrails, lbp, temp_directory, mcp-tools, lgrep-tools, morph-tools, worktree-guide, caveman, ADV_INSTRUCTIONS (auto-wired via plugin)
- [ ] Providers: google (Gemini 2.5-flash, 2.5-pro, 3-flash-preview, 3-pro-preview), openai (GPT-5.2 with reasoning variants), openrouter (Claude Haiku 4.5 Nitro) — curated baseline, not all live models
- [ ] Agent model assignment rendering is explicitly deferred past v1.0; OCA docs remain aligned with current OpenCode + Advance ownership boundaries and do not reintroduce obsolete `scout` / `refine` assumptions
- [ ] Permissions: default, doom_loop, external_directory, bash
- [ ] Watcher: ignore globs
- [ ] LSP: pyrefly, pyright, typescript, typescript-language-server
- [ ] Formatters: custom formatter config rendered into `opencode.json` `.formatter`
- [ ] Custom commands: project-specific slash commands rendered into `opencode.json` `.command`
- [ ] OpenCode toggles: default_agent, share, snapshot, autoupdate, compaction, disabled_providers, enabled_providers
- [ ] Primary client/session UX: prefix, reaper, theme, boot_splash
- [ ] Discord: enabled, mode

### Advance integration

- [ ] Advance plugin is declared as a dependency in `stack.toml` with `provides = ["adv-commands", "adv-agents", "adv-skills", "adv-overlays"]`
- [ ] `oca apply` clones/updates the Advance checkout, runs `pnpm install && pnpm build`, wires the plugin path into `opencode.json`, and invokes `advance/scripts/sync-global.sh --fix` to let Advance sync its own assets
- [ ] OCA does NOT copy or duplicate any file that Advance owns
- [ ] `oca doctor` includes checks for: Advance checkout present, plugin built (`dist/index.js` exists), ADV state directory readable (`~/.local/share/opencode/plugins/advance/{project-id}/`)
- [ ] current status UI shows active ADV change + gate progress when an ADV change is active in the current session's project

### Primary client/session lifecycle

- [ ] `oca session new` creates a tmux session (`oca-<epoch>-<pid>`), renders obsidian theme, shows boot splash, and launches the configured primary client
- [ ] `oca session list` lists active sessions with window count and memory
- [ ] `oca session attach` / `switch` / `killall` / `restart` work, with `attach` supporting same-host re-entry from another terminal/device and `switch` remaining the in-tmux retarget flow
- [ ] Stale session reaper cleans up unattached sessions after 4 hours (configurable) without deleting sessions a user would reasonably expect to resume later
- [ ] Per-session cache in `$OCA_CACHE_DIR/<session-id>/`
- [ ] Cross-device support is defined as same-host tmux re-entry via existing access paths (local shell, SSH, Tailscale), not multi-host sync or custom transport

### Clean cutover

- [ ] All development, testing, and CI runs in isolated config directories (never modifies `~/.config/opencode/`, `~/.config/vision/`, or `~/.tmux.conf`)
- [ ] Environment overrides (`OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_PLUGIN_CHECKOUT_ROOT`, `OCA_CACHE_DIR`) are honored throughout
- [ ] `oca migrate from-open-chad` generates a valid stack.toml from the user's current open-chad state
- [ ] Migration is reversible (backups of all pre-migration config are retained)
- [ ] Release-day cutover script is documented and tested

### Quality gates

- [ ] `go test ./...` passes (>= 200 tests; coverage TBD)
- [ ] `go vet ./...` passes
- [ ] `gofmt -d .` is clean
- [ ] Integration tests run the full `oca install → apply → doctor` cycle against an isolated test environment
- [ ] CI runs on every PR and push to trunk (`.github/workflows/ci.yml`)
- [ ] Release pipeline builds cross-platform binaries via goreleaser on version tags (`.github/workflows/release.yml`)
- [ ] README, AGENTS.md, and docs/ are complete and internally consistent
- [ ] `oca doctor` returns clean on a reference test stack

### Non-goals for v1.0

Explicitly out of scope for the first release:

- Windows native support (WSL2 is supported as a special case of Linux)
- macOS support (can come in v1.1+)
- Multi-client picker/configuration abstraction
- Plugin marketplace / discovery
- Non-OpenCode IDE integration
- Multi-user / team configuration sharing
- Custom secrets management layer — OpenCode natively supports `{env:VARIABLE_NAME}` and `{file:path}` variable substitution which should be preferred over OCA-specific abstractions; `env_file` remains supported for `.env` file loading; password manager integration deferred to v1.1
- Automatic plugin version updates (user must run `oca update` explicitly)

---

## Constraints

### Must honor

1. **Clean cutover during development.** No modifications to the user's live `~/.config/opencode/`, `~/.config/vision/`, `~/.tmux.conf`, or shell rc files while v1.0 is under construction. All dev/test runs use isolated config directories via `OCA_*_DIR` environment overrides.

2. **Advance is untouched.** The Advance repo at `~/dev/oc-plugins/advance` is not modified by this work. OpenCode Advance consumes it as a dependency.

3. **File ownership boundary.** OCA owns only the non-ADV slice of the environment. Any file Advance declares via `provides` is off-limits for OCA.

4. **Reproducibility.** Plugin references in stack.toml support both tracking refs (e.g., `ref = "trunk"`) and pinned SHAs (e.g., `ref = "abc123..."`). Users can `oca pin` to lock all plugins at known-good versions.

5. **Linux-first.** Ubuntu/Debian is the primary target. Other distros (Arch, Fedora) should work but are not CI-verified for v1.0.

6. **Non-interactive where possible.** Every command supports `--yes` or non-interactive equivalents for CI/unattended use.

7. **Atomic writes.** All file writes use temp-file + rename to prevent half-written state on interruption.

8. **Backups.** Every write to a managed file creates a `.bak.<epoch>` backup with rotation (default: keep 5).

### Must not

1. **Must not duplicate Advance-owned files.** Skipping `adv-*` assets when Advance is declared as a plugin is non-negotiable.

2. **Must not auto-update plugins silently.** `oca apply` should not pull new commits from tracking refs unless explicitly asked via `oca update`.

3. **Must not embed Node.js heredocs.** All JSON/TOML manipulation in Go, not shell.

4. **Must not require root.** The installer may prompt for sudo to install system packages, but must not require root for the core OCA functionality.

5. **Must not break Advance.** If Advance's `sync-global.sh` API changes during development, OCA must adapt — but must never modify Advance's source to work around OCA.

6. **Must not ship partial v1.0.** Full vision or nothing. No intermediate releases.

---

## Approach

### Language & tools

- **Go 1.22+** for the CLI core
- **cobra** (`github.com/spf13/cobra`) for command tree + help generation
- **pelletier/go-toml/v2** (`github.com/pelletier/go-toml/v2`) for TOML parsing
- **stdlib `text/template`** for rendering (no external template engine)
- **stdlib `encoding/json`** for JSON manipulation
- **goreleaser** for release builds (cross-platform binaries to GitHub Releases)
- **Bash** for the current tmux-first theme, boot splash, and status bar (tmux scripting is cleanest in bash)
- **Go stdlib `testing`** + golden files for unit tests
- **Shell test scripts** for tmux/shell integration tests

### Phases

See [`phases.md`](phases.md) for the full phase sequencing. Rough shape:

1. **Phase 0: Foundation + brand** — completed baseline: branded CLI, shared brand runtime, shell helpers, verification wiring
2. **Phase 1: stack.toml + MCP apply** — next: parser, schema, MCP rendering, Vision integration
3. **Phase 2: Plugin + instruction management** — plugin lifecycle, ADV delegation, instructions rendering
4. **Phase 3: Core opencode.json coverage** — providers, permissions, LSP, watcher, diff command
5. **Phase 3.5: Skills + commands + formatters + toggles** — OCA-owned skills, custom commands, formatter config, OpenCode-level toggles
6. **Phase 4: Primary client UX + theme** — current tmux-first lifecycle, obsidian theme, new status bar, boot splash
7. **Phase 5: Temporal enablement** — `[temporal]` config, apply rendering, health checks, status bar integration
8. **Phase 6: Installer + shell** — `oca install`, shell profile wiring, completions
9. **Phase 7: Migration + doctor** — `oca migrate from-open-chad`, expanded doctor, ADV state integration
10. **Phase 8: Extras + polish** — Discord (new taglines), release pipeline, final README

Estimated total: 6.5-8.5 weeks of focused work.

### Development methodology

- Every phase is one or more ADV changes in this repository
- Each change follows the full 7-gate workflow: proposal → discovery → design → planning → execution → acceptance → release
- TDD is mandatory for all Go code (test before implementation)
- Phase dependencies are strict: Phase N cannot start until Phase N-1 archives successfully
- Integration tests run the full oca install → apply → doctor cycle against isolated test environments

### Testing strategy

| Layer              | Test type                                                                          |
| ------------------ | ---------------------------------------------------------------------------------- |
| TOML parser        | Go unit tests + golden files for valid/invalid inputs                              |
| Schema validation  | Go unit tests covering every validation rule                                       |
| Template rendering | Go unit tests with golden JSON outputs                                             |
| Plugin lifecycle   | Go integration tests with a local mock git remote                                  |
| MCP health checks  | Go integration tests with a mock HTTP server                                       |
| ADV state reading  | Go integration tests with a mock ADV state directory                               |
| Shell integration  | Bash tests (`bats`-style or plain assertions) for tmux/boot splash/status bar      |
| End-to-end install | Bash test running `oca install --yes` in a containerized / chroot-like environment |
| Migration          | Bash test with a canned open-chad state snapshot as input                          |

### Observability

- Structured logging via `log/slog` (Go stdlib)
- `OCA_LOG_LEVEL=debug` for verbose traces
- All subprocess invocations (git, pnpm, vision, sync-global.sh) have their output captured and included in error messages
- `oca doctor` emits JSON output when `--output json` is set (for scripting)

---

## Risks

| Risk                                                            | Likelihood | Impact | Mitigation                                                                                                |
| --------------------------------------------------------------- | ---------- | ------ | --------------------------------------------------------------------------------------------------------- |
| OpenCode's opencode.json schema changes during development      | Medium     | High   | Use schema_version field in stack.toml, pin to a specific OpenCode schema version, test against latest OC |
| Advance's sync-global.sh contract changes                       | Low        | Medium | Document the delegation contract; coordinate with Advance maintainer (self); test cross-repo changes      |
| Vision daemon YAML format changes                               | Low        | Low    | Vision is self-owned, can coordinate changes                                                              |
| Go template complexity for nested opencode.json structures      | Medium     | Medium | Write golden-file tests for every template; keep templates flat where possible                            |
| Migration from open-chad loses user customizations              | Medium     | High   | Conservative migration: emit to file for user review, don't auto-apply; preserve `.bak` files             |
| 6.5-8.5 week estimate is too optimistic                         | High       | Medium | Phase boundaries allow re-planning between phases; deferring polish to v1.1 if needed                     |
| New obsidian theme looks worse than open-chad's synthwave theme | Low        | Low    | Preview early in Phase 4; get user feedback before committing                                             |
| Clean cutover fails (open-chad residue left behind)             | Medium     | Medium | Migration script includes `open-chad uninstall` verification; doctor reports open-chad residue            |
| Dependency on Vision daemon availability during install         | High       | Low    | Vision install is required; document as prerequisite; non-fatal degradation if Vision missing             |

---

## Open questions (to resolve during per-phase research)

### Resolved during planning

1. **cobra vs urfave/cli.** ✅ Resolved: cobra. Standard, well-maintained, widely adopted.

2. **TOML parser choice.** ✅ Resolved: `pelletier/go-toml/v2`. De facto standard.

3. **Secrets handling.** ✅ Resolved: prefer OpenCode native `{env:VARIABLE_NAME}` and `{file:path}` variable substitution in rendered `opencode.json` values. OCA passes these tokens through unchanged — it does not resolve them itself. `env_file` remains supported for cases where `.env` file loading is preferred. Password manager integration deferred to v1.1.

4. **Provider allow/deny lists.** ✅ Resolved: use `disabled_providers` and `enabled_providers` in `[opencode]` table; `disabled_providers` takes precedence over `enabled_providers`.

### Open (to resolve during discovery for each relevant phase)

1. **Exact OpenCode theme JSON schema.** What keys does OpenCode expect in `themes/obsidian.json`? Verify against OpenCode's theme documentation before implementing in Phase 4.

2. **Cross-platform boot splash.** The ASCII wordmark uses Unicode box-drawing characters. Verify rendering on WSL2 Windows Terminal, mosh, SSH, and tmux across multiple terminal emulators. Needs verification in Phase 0.

3. **Go binary size.** Expect ~15-25 MB static binary. Acceptable for CLI tool but worth measuring.

4. **`oca add mcp <name>` interactive UX.** How interactive should the add wizard be? Full prompt-by-prompt, or a one-shot `--port 6280 --command npx --args "..."` flag style? Decide in Phase 1.

5. **On-demand MCP health checks.** Should `oca doctor` check on-demand servers at all, or only autostart ones? Decide in Phase 1.
