# Architecture Overview

This document is the canonical high-level architecture reference for the code that is **actually implemented today**.

Current implemented scope on this branch is **Phase 0 through Phase 7 complete, plus ADV runtime hardening (work-in-progress)**:

- `stack.toml` parsing for `[meta]`, `[mcp]`, `[plugins]`, `[instructions]`, `[providers]`, `[permissions]`, `[watcher]`, `[lsp]`, `[skills]`, `[commands]`, `[formatters]`, `[opencode]`, `[temporal]`, plus deferred future sections
- `oca apply --target mcp|plugins|instructions|providers|permissions|watcher|lsp|skills|commands|formatters|toggles|temporal`
- `oca apply` with no `--target` for composed all-target apply
- `oca diff`
- `oca doctor --scope mcp`, `oca doctor --scope plugins`, `oca doctor --scope skills`, `oca doctor --scope temporal`, `oca doctor --scope adv-assets`, `oca doctor --scope adv-plugin`, and `oca doctor --scope cross`
- `oca debug plan` and `oca debug validate`
- `oca pin` and `oca update`
- `oca session new/list/attach/switch/kill/killall/restart/reap` (Phase 4)
- `oca theme list/apply` (Phase 4)
- `oca install [--yes]` and `oca uninstall` (Phase 6, shipped)
- `oca completion <shell>` (Phase 6, shipped)
- `oca temporal status/start/stop/restart/logs` for local Temporal dev-server supervision (Phase 6.5, shipped)
- `oca migrate from-open-chad` and `oca migrate init` (Phase 7, shipped)
- `oca dashboard` (out-of-phase, shipped)
- `oca pane` and `oca watchdog` (out-of-phase, shipped)
- MCP slot group rendering for `[mcp.slot_groups.*]` (Phase 5.5)
- Temporal config rendering, health checks, and local dev-server supervision (Phase 5 + Phase 6.5)
- plugin lifecycle: git clone/pull, build, pin, sync-global.sh delegation
- Obsidian theme assets (JSON + tmux conf) and managed-block tmux template
- Shell profile managed-block injection via `internal/install` (Phase 6)

Future phases are tracked in [`../proposals/phases.md`](../proposals/phases.md).

## Core principle: single source of truth

The user owns one file: `stack.toml`. OCA renders declarative slices from that file into `opencode.json` and companion config files.

```text
stack.toml
  └─► oca apply [--target ...]
        ├─► opencode.json (.mcp/.plugin/.instructions/.provider/.permission/.watcher/.lsp/.command/.formatter/.compaction merges)
        ├─► vision/servers.yaml (authoritative full write)
        └─► skills/ (OCA-owned skill directories copied from assets/skills/)
```

OCA renders skills, commands, formatters, OpenCode toggles, session settings, and Discord configuration. Phase 4 foundation added session lifecycle (`internal/session/`, `cmd/oca/session.go`) and theme assets (`assets/themes/`, `templates/tmux.conf.block.gotmpl`). Phase 8 promotes Discord from deferred config to typed runtime support through `cmd/oca/discord.go` and `internal/discord/`. OCA still does **not** render agents through `oca apply`; agents remain deferred.

## Subsystems

### 1. Configuration (`internal/config/`)

Config is typed where implementation exists and deferred where ownership is intentionally postponed.

- Parses TOML with `github.com/BurntSushi/toml`
- Decodes typed sections: `Meta`, `MCP`, `Plugins`, `Instructions`, `Temporal`, `Providers`, `Permissions`, `Watcher`, `LSP`, `Skills`, `Formatters`, `Commands`, `OpenCode`, `Session`, `Discord`
- Preserves deferred future sections in `Stack.DeferredSections` (`agents`)
- Resolves shell-style paths and env variables in-place
- Emits aggregated field-path validation errors
- Collects non-fatal `env_file` warnings

Current core types:

```go
type Stack struct {
    Meta             Meta
    MCP              MCPSection
    Plugins          PluginsSection
    Instructions     InstructionsSection
    Temporal         *TemporalSection
    Providers        ProvidersSection
    Permissions      PermissionsSection
    Watcher          WatcherSection
    LSP              LSPSection
    Skills           SkillsSection
    Formatters       FormattersSection
    Commands         CommandsSection
    OpenCode         OpenCodeSection
    DeferredSections map[string]any
    Warnings         []Warning
}
```

Current pipeline:

```text
ParseFile -> Resolve -> Validate -> collect env_file warnings
```

Still deferred:

- project-local override merge
- agent/session/theme/toggle structures
- migration import logic

### 2. Rendering (`internal/render/`)

Rendering is **programmatic**, not template-based.

- `RenderMCPFragment` builds one OpenCode `.mcp` entry
- `MergeMCP` overwrites declared MCP keys and preserves user-added keys
- `MergeArray` merges flat arrays like `.plugin` and `.instructions`
- `MergeObject` merges nested object slices like `.provider`, `.permission`, and `.lsp`
- `MergeWatcherIgnore` merges `.watcher.ignore` while preserving user-added entries and sibling watcher keys
- `RenderVisionServers` writes authoritative `servers.yaml`
- `PlanMCP`, `PlanPlugins`, `PlanInstructions`, `PlanProviders`, `PlanPermissions`, `PlanWatcher`, `PlanLSP` compute deterministic target operations
- `PlanSkills` copies OCA-owned skill directories from `assets/skills/` to target
- `PlanCommands`, `PlanFormatters`, `PlanToggles` render typed config into `opencode.json` sections
- `ComposeApplyPlan` chains composed apply/diff through a running in-memory `opencode.json`
- `Apply` executes the plan with dry-run support, lock acquisition, atomic writes, and backup rotation

Current write guarantees:

- `opencode.json` mode: `0644`
- `vision/servers.yaml` mode: `0600`
- same-directory temp-file + rename atomicity
- `.bak.<UnixNano>` backup rotation
- orphan `*.bak.*.tmp` cleanup before write
- host-local `flock` on `$OCA_CACHE_DIR/apply.lock`

Phase 1 special rule:

- `type = "daemon"` is an OCA-only directive
- it still renders an OpenCode remote MCP URL
- it is **not** written into `vision/servers.yaml`

### 3. Health (`internal/health/`)

Phase 1 health started with MCP-only. Phase 3.5 added skills health.

- `CheckMCP` probes Vision `GET /version`
- requires `api.v1_servers = true`
- then probes `GET /v1/servers`
- classifies declared servers as pass / warn / fail
- treats required-but-not-running servers as failures
- still reports local `env_file` and command-path advisory checks when Vision is down or incompatible
- `CheckSkills` verifies OCA-owned skill deployment health:
  - declared skills have source asset directories
  - reserved `adv-*` namespace rejected
  - declared skills present in target dir
  - extra OCA-owned skill dirs in target produce warnings
  - Advance-owned `adv-*` dirs in target are ignored
- `CheckAdvAssets` audits plugin-provided asset ownership boundaries:
  - ownership is derived from `stack.Plugins[*].Provides` at runtime
  - duplicate owners for the same `provides` category fail
  - non-`adv-*` files in plugin-owned category dirs warn as ORPHANED
  - stale `adv-{provider}.md` instruction files warn when `{provider}` is absent from `[providers]`
  - the check is read-only and never deletes deployed files
- `CheckADVPlugin` verifies the Advance plugin checkout and state directory:
  - plugin declared in `stack.toml`
  - checkout directory exists
  - build artifact (`dist/index.js`) exists and is non-empty
  - ADV state directory (`$XDG_DATA_HOME/opencode/plugins/advance`) is readable
  - all checks are local; no plugin code execution or network requests
- `CheckCross` verifies cross-component consistency:
  - plugin checkout source drift: compares `remote.origin.url` against `stack.toml` `source`, normalizing trailing slashes, `.git` suffixes, and protocol variants (`https://`, `http://`, `ssh://`, `git@`)
  - MCP server port collisions across all `[mcp.servers.*]` entries
  - instruction file path resolution (all declared paths must exist)
  - agent provider reference check (placeholder; deferred until OCA owns agent rendering)

Current defaults:

- Vision admin URL: `http://127.0.0.1:6275`
- per-request timeout: caller-controlled, defaulted by CLI
- execution model: sequential, not worker-pooled

### 3.5. ADV Runtime (`internal/advruntime/`)

ADV runtime diagnostics layer. Read-only by default; never silently mutates ADV workflow state.

- `Report` data model aggregates findings from all scanners
- `WorkflowClassifier` lists Running ADV workflows via Temporal visibility, groups by task queue, detects stale queues (no pollers + stale age)
- `SearchAttributeChecker` verifies required ADV search attributes via `ListSearchAttributes`
- `SessionDebtScanner` classifies repairable blank assistant messages in OpenCode SQLite DB
- `WorktreeCensus` scans OCA worktree roots and ADV project state roots
- `RecoveryPlanSynthesizer` emits ordered dry-run recovery steps from a report
- `ClientProvider` manages lazy/idempotent Temporal client lifecycle
- Narrow interfaces (`WorkflowService`, `OperatorService`) for testability without live Temporal
- All thresholds tunable via `Config` with conservative defaults

### 4. Session (`internal/session/`) — Phase 4 foundation + Pattern B (v1.0)

Phase 4 foundation. Manages OCA tmux sessions on a dedicated socket.

- `Manager` struct with socket + tmux binary path
- `NewManager(socket)` validates tmux binary on PATH
- `Create(ctx, name, workingDir, tmuxConfPath)` runs `tmux -L <socket> [-f <conf>] new-session -d -s <name> -c <dir>` via `subprocess.Run`
- `List(ctx)` parses `tmux -L <socket> list-sessions -F '#{session_name}\t#{session_attached}'`, filters by `oca-` prefix
- `NextSessionName(ctx, repoSlug)` scans existing sessions, finds next sequential `oca-<slug>-<n>`
- `Attach`, `SwitchClient`, `Kill`, `KillAll`, `Restart`, `GetSessionByName`, `SetGlobalEnv` for full session lifecycle
- Pre-validates working dir exists before tmux call
- All external commands through `internal/subprocess`
- `OCA_TMUX_SOCKET` env override (default: `"oca"`)

**Pattern B session topology (v1.0):**

- `ProjectRoot(cwd string) string` — resolves git root from any nested subdirectory
- `ProjectSessionName(projectRoot string) string` — derives a slug-based session name (e.g., `opencode-advance`) without `oca-` prefix or numeric suffix
- `GetOrCreateProjectSession(ctx, projectRoot, tmuxConfPath)` — creates the project session with a `trunk` window on first call, then reuses
- `EnsureWindow(ctx, sessionName, windowName, cwd)` — idempotent: creates missing window or reuses existing, replaces stale-cwd windows
- `ListWindows(ctx, sessionName) []Window` — parses `tmux list-windows` output
- Smart root command (`oca` / `oc` with no args): project mode resolves git root → project session; per-invocation mode (`[session].mode = "per-invocation"`) delegates to Pattern A
- `EnsureWindow` and `Reconcile` commands extend the session manager for worktree-aware workflows
- ADV worktree hook integration: `advWorktreeCreate` calls `ocaEnsureWindow(sessionName, changeId, worktreePath)` after worktree creation/reuse — non-fatal, best-effort

**Pane state v2 (`cmd/oca/pane.go`):**

- v2 fields: `schemaVersion`, `changeID`, `role`, `projectId`, `worktreeBranch`, `worktreePath`, `startedAt`
- `derivePaneContext(state)` pure function extracts change context from branch/path (no persisted derived fields)
- v1/v2 read compatibility; v2 write preserves all fields

**Worktree-first trunk protection (ADV):**

- `plugin/src/utils/system-block.ts` — trunk guard fires when `activeChange.id` exists and `isWorktree === false`
- Emits `[ADV:TRUNK_GUARD]` banner with routing instructions and emergency override documentation
- Emergency overrides (merge/push/deploy) require audit evidence documentation

**One-writer-per-worktree lease (ADV):**

- `plugin/src/utils/worktree-lease.ts` — lease state keyed by `(projectID, canonicalWorktreePath)`
- Records: `pid`, `sessionId`, `acquiredAt`, `heartbeatAt`
- Functions: `acquireLease`, `refreshHeartbeat`, `reclaimStaleLease`, `releaseLease`, `checkLease`
- Heartbeat-stale + dead-PID triggers automatic reclaim with `previousLease` audit trail

### 4.5. Install (`internal/install/`)

Phase 6 (shipped). End-to-end first-time setup and teardown.

- `PrereqChecker` verifies git, tmux (≥3.4), OpenCode, vision, and optional Temporal CLI presence
- `ShellProfile` manages idempotent managed-block injection into `~/.zshrc` and `~/.bashrc`:
  - adds PATH and shell completion wiring between OCA delimiters
  - removes only the managed block without touching user-owned content
  - handles multiple profile files and detects shell type
- `Install` and `Uninstall` flow orchestration: runs prereqs → apply → profile injection (or removal)
- `templates/shell_profile.block.gotmpl` for the managed block content
- Integration tests cover install → uninstall → install round-trips

### 5. CLI (`cmd/oca/`)

Current shipped commands:

- `oca version`
- `oca apply [--target ...]` (targets: mcp, plugins, instructions, providers, permissions, watcher, lsp, skills, commands, formatters, toggles, temporal)
- `oca diff`
- `oca doctor --scope mcp|plugins|skills|temporal|adv-assets|adv-plugin|adv-runtime|cross`
- `oca debug plan`
- `oca debug validate`
- `oca pin` and `oca update`
- `oca session new/list/attach/switch/kill/killall/restart/reap`
- `oca session ensure-window --session <name> --name <window> --cwd <path>` (Pattern B)
- `oca session reconcile [--dry-run]` (Pattern B)
- `oca session list` (aliases: `ls`)
- `oca theme list/apply`
- `oca install [--yes]` and `oca uninstall`
- `oca completion <shell>`
- `oca dashboard [--bind <addr>] [--port <n>] [--no-open]`
- `oca pane` and `oca watchdog`
- `oca temporal status/start/stop/restart/logs` (Phase 6.5, shipped)
- `oca session doctor [--db] [--threshold] [--apply --backup-dir]` (ADV runtime hardening)
- `oca adv recover --dry-run [--project] [--change]` (ADV runtime hardening)

Shared shipped flags:

- `--config`
- `--output text|json`
- `--verbose` / `-v`
- `--quiet` / `-q`

Not implemented yet:

- migration commands beyond scaffolding
- `oca theme` extended subcommands beyond list/apply
- `oca add` / `oca remove` interactive flows

## Data flow: `oca apply --target mcp`

```text
1. Load stack.toml                (config.Load)
2. Build render plan             (render.PlanMCP)
3. --dry-run? print plan, exit
4. Acquire apply lock            (render.AcquireApplyLock)
5. Merge opencode.json .mcp      (render.MergeMCP)
6. Render vision/servers.yaml    (render.RenderVisionServers)
7. Write targets atomically      (render.WriteAtomic)
8. Report per-target result      (CLI JSON/text output)
```

## Data flow: `oca doctor --scope mcp`

```text
1. Load stack.toml               (config.Load)
2. Probe Vision /version         (health.fetchVersion)
3. Probe Vision /v1/servers      (health.fetchServers)
4. Classify server states        (health.CheckMCP)
5. Add local advisory checks     (env_file, command-path)
6. Render report                 (CLI JSON/text output)
```

## Clean cutover policy during development

OCA must not touch live user config during development. Phase 1 + Phase 2 code explicitly honors:

- `OCA_OPENCODE_CONFIG_DIR`
- `OCA_VISION_CONFIG_DIR`
- `OCA_CACHE_DIR`
- `OCA_PLUGIN_CHECKOUT_ROOT` (Phase 2 — base dir for `{checkout}` token expansion)

Default production paths remain:

- `~/.config/opencode/opencode.json`
- `~/.config/vision/servers.yaml`
- `~/dev/oc-plugins/<plugin>/` (plugin checkouts)

Tests and local dev runs use temporary override directories instead.

## Shell auto-refresh + update awareness

Mutating OCA flows write `~/.config/oca/env.sh` atomically, then touch `$OCA_CACHE_DIR/env.stamp`. The managed shell block checks `env.stamp` at prompt time and sources only the OCA-owned env file; it does not re-source `.bashrc`, `.zshrc`, or other user-owned code.

Plugin update awareness is read-only. `oca update --check` uses `git ls-remote` with git transport hardening and `GIT_TERMINAL_PROMPT=0`, then persists results to `$OCA_CACHE_DIR/drift_cache.json`. The shell drift surfacer reads that cache in passive mode and only runs `oca update --check --quiet` automatically when `[update_probe].default = "warn"`.

The local plugin doctor remains the owner for checkout/build/ref health. The update probe adds remote comparison and shares status vocabulary instead of duplicating doctor semantics.

## Phase 2 subsystems (shipped)

Phase 2 added three internal packages and one render primitive. All external commands route through `internal/subprocess`; no `exec.Command` calls live outside that package.

### `internal/subprocess`

Generic command runner: `Run(ctx, Cmd) (Result, error)`. Implements the canonical Go 2026 pattern — `exec.CommandContext` with `Cmd.Cancel` (SIGTERM) and `Cmd.WaitDelay` (SIGTERM→SIGKILL grace). Classifies exits as `ExitSuccess | ExitNonZero | ExitTimeout | ExitSignal`. Merges a caller-provided env map onto the inherited process environment. Single `bytes.Buffer` for combined stdout+stderr (the kernel serializes writes at the OS level). Zero knowledge of `git`, `npm`, or `sh` — callers supply argv and timeout.

Design references: K1, AC17, AC18, AC19.

### `internal/plugin`

Thin domain wrappers over `internal/subprocess`:

- `git.go` — `Clone`, `Fetch`, `Checkout`, `RevParseHEAD`, `StatusClean`, `RemoteOriginURL`. Every invocation is prefixed with `-c protocol.file.allow=user -c protocol.ext.allow=false` to defuse transport-confusion attack surface (CVE-2022-39253 class). Ref names pass through `validateGitRef` (regex `^[A-Za-z0-9._/-]+$`, rejects empty / leading `-` / `..`).
- `build.go` — `RunBuild` executes each declared build step via `sh -c` with `CI=true`, `DEBIAN_FRONTEND=noninteractive`, `npm_config_yes=true` merged into the environment. Failures report step index and captured output for remediation.
- `npm.go` — literal-version handler for npm-source plugins. No clone, no build.
- `prepare.go` — clone-or-update orchestrator. Uses `os.Lstat` to refuse symlinked checkout paths, compares `remote.origin.url` against stack.toml `source` to detect drift, aborts on dirty worktrees.
- `pin.go` — `CapturePin` (reads HEAD SHA) + `WritePin` (atomic temp-file + rename write of the `ref` field). The caller (`cmd/oca/pin.go`) holds the apply lock so concurrent `oca apply` / `oca pin` cannot corrupt `stack.toml`.

Design references: K1, K2, jc-pnpm1, jc-update1.

### `internal/sync`

`InvokeAdvance(ctx, plugin)` runs the plugin's configured `sync` shell command from its checkout directory and returns combined stdout/stderr filtered through `render.Redact`. The redaction is best-effort pattern matching (GitHub PATs, OpenAI keys, generic `key=value` forms); it is NOT a security boundary and is documented as such.

### `internal/render` — `MergeArray`

New primitive for flat-array sections (`instructions` ordering). Preserves user-declared paths, dedups against declared + plugin-provided entries via `canonicalPath` (which refuses to normalize traversal above the reference point), and prunes stale worktree paths whose base dir is gone. Companion: `.bak.<epoch>` rotation with per-target `SuppressBackup` (see wisdom `ws-GcnRm8`).

### Health registry

Pluggable health checks with a `ResetForTesting()` contract so within-package tests remain serial but don't leak across test binaries.

## Future phases

The following remain planned, not shipped:

- agent rendering
- `oca add` / `oca remove` interactive flows
- status bar richness (metrics, LLM fuel gauges, ADV state) — status bar exists but metrics integration is ongoing

Advance's Temporal dependency is **current**. Advance runs two durable workflows (`changeWorkflow`, `projectWorkflow`) via Temporal with file-backed fallback. OCA's Phase 6.5 (shipped) manages the Temporal infra (CLI detection, dev-server supervision, env-file rendering, health checks) that Advance needs to function.

See [`../proposals/phases.md`](../proposals/phases.md) for sequencing.
