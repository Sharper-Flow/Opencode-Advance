# `oca` CLI Surface

This document separates the **implemented CLI** from the broader **planned v1.0 command surface**.

Phase 1 shipped `oca version`, `oca apply --target mcp`, `oca doctor --scope mcp`, and `oca debug`. Phase 2 extended the surface with plugin/instructions targets on `apply`, added new `oca pin` and `oca update` commands, and extended `oca doctor` with a `plugins` scope and a `--network` flag. Phases 3–3.5 extended apply/diff/doctor. Phase 4 shipped session lifecycle. Phase 5 shipped temporal config. Phase 5.5 shipped slot groups. Phase 6 shipped `oca install`, `oca uninstall`, and `oca completion`. Phase 6.5 shipped `oca temporal` dev-server supervision subcommands. Phase 7 shipped `adv-plugin` and `cross` doctor scopes. The worktree occupancy visibility change added `oca occupancy`. The standalone maintenance change adds `oca maintain` for offline verified merges, plugin rebuilds, and safe cleanup when no OpenCode/ADV runtime is active.

## Shipped in Phase 1

Shared shipped flags:

- `--config <path>` — path to `stack.toml` (default: `./stack.toml` or `$XDG_CONFIG_HOME/opencode-advance/stack.toml`)
- `--output <text|json>` — output format
- `--verbose`, `-v` — enable debug logging
- `--quiet`, `-q` — suppress non-error output

Notes:

- `yaml` output is **not** implemented
- `--no-color` is **not** a CLI flag; terminal color fallback is controlled by environment/runtime behavior

## `oca version`

Print the branded version banner.

## `oca apply --target mcp`

Render the MCP slice from `stack.toml` into:

- `opencode.json` `.mcp`
- `vision/servers.yaml`

Flags:

| Flag | Purpose |
| --- | --- |
| `--target mcp` | required in Phase 1 |
| `--dry-run` | print the computed plan without writing |

Behavior:

- applies only the MCP target in Phase 1
- rejects unknown or missing targets with a clear error
- honors `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, and `OCA_CACHE_DIR`
- supports `--output text|json`

## `oca doctor --scope mcp`

Verify Vision compatibility and declared MCP server state.

Flags:

| Flag | Purpose |
| --- | --- |
| `--scope mcp` | only supported scope in Phase 1 |
| `--timeout <duration>` | per-request timeout |

Behavior:

- probes Vision `GET /version`
- requires `api.v1_servers = true`
- probes Vision `GET /v1/servers`
- reports pass / warn / fail per declared server
- still reports local `env_file` and command-path advisories when Vision is down

## `oca debug`

### `oca debug plan`

Emit the Phase 1 render plan as JSON.

### `oca debug validate`

Run config load/resolve/validate and print the result without writing files.

## `oca maintain`

Plan and optionally execute offline OCA/ADV maintenance. This command is designed for use after closing OpenCode sessions so host-loaded plugin code, Temporal workers, and ADV state are not mutated under a live runtime.

Flags:

| Flag | Purpose |
| --- | --- |
| `--dry-run` | Show the maintenance plan without mutation (default) |
| `--execute` | Execute safe actions after hard active-session gates pass |
| `--include-merge` | Include verified completed ADV change branches |
| `--include-rebuild` | Include plugin rebuild actions for missing/stale build markers |
| `--include-cleanup` | Include conservative merged/clean/process-free worktree cleanup |
| `--project <path>` | Project root to inspect |

Behavior:

- hard-refuses `--execute` when OpenCode/OCA/ADV runtime processes are active
- consumes Advance `scripts/maintenance/inspect.mjs` output and plans merges only for archived changes with `release` gate `done`
- uses `git merge --ff-only` for verified merge actions
- rebuilds plugins through declared `build` commands and writes `dist/oca-build.json` with source git SHA/build metadata
- removes only merged, clean, process-free, session-free worktrees
- calls Advance `scripts/maintenance/reconcile-worktree.mjs` when available; otherwise it does not directly mutate ADV state
- Temporal recovery remains report-only in this surface unless a future standalone Advance script provides an explicit safe executor

## Shipped in Phase 2

Phase 2 adds plugin and instruction management. It extends existing commands and ships two new ones.

### `oca apply --target plugins | --target instructions | --target temporal`

`oca apply` now accepts additional targets alongside `--target mcp`:

| Target | Behavior |
| --- | --- |
| `plugins` | Runs `internal/plugin.Prepare` for every enabled git-source plugin (clone-or-update + build) before rendering. After rendering, invokes each plugin's `sync` command; sync failures produce a structured diagnostic (plugin name, exit code/class, redacted output excerpt, actionable `oca doctor` suggestion). |
| `instructions` | Renders the `instructions` flat array into `opencode.json` via the `MergeArray` primitive. |
| `temporal` | Renders `$OCA_CACHE_DIR/temporal.env` for Advance's Temporal client settings. |

Targets may be combined (`--target mcp --target plugins --target instructions`). Ordering is enforced structurally at the call site: render → sync.

### `oca pin [plugin...]`

Captures the current HEAD SHA of each git-source plugin and writes it into `stack.toml` as the plugin's `ref`.

Flags:

| Flag | Purpose |
| --- | --- |
| `--config <path>` | path to `stack.toml` |

Behavior:

- acquires the same apply lock used by `oca apply` so concurrent invocations cannot corrupt `stack.toml`
- writes atomically (temp file + rename) so a crash mid-write cannot leave a truncated `stack.toml`
- skips npm-source plugins (no SHA concept)
- skips already-pinned plugins where `ref` matches the captured SHA (idempotent)
- when arguments are supplied, scopes to named plugins; otherwise iterates all plugins in declared order
- exit codes: `0` success, `2` capture error (git failure, missing checkout), `3` write error (lock or I/O)

### `oca update [plugin...]`

Fetches, checks out, builds, and (if configured) syncs git-source plugins.

`oca update --check [plugin...]` is the read-only update-awareness mode. It runs `git ls-remote` for enabled git-source plugins, writes `$OCA_CACHE_DIR/drift_cache.json`, and reports `up_to_date`, `update_available`, or `pinned` without `git fetch`, checkout mutation, build, apply, or sync. Available updates are a successful check result, not an error exit.

Flags:

| Flag | Purpose |
| --- | --- |
| `--config <path>` | path to `stack.toml` |
| `--force` | update even when `ref` is pinned to a SHA |
| `--output <text\|json>` | reporting format |

Behavior:

- always runs `git fetch` on floating refs (branches, tags) per judgment call `jc-update1`
- runs `build` commands with `CI=true`, `DEBIAN_FRONTEND=noninteractive`, `npm_config_yes=true` merged into the environment
- honors `--frozen-lockfile` semantics (caller's responsibility in the build command string)
- per-plugin status reported; one plugin's failure does not halt the rest
- sync failures produce a structured diagnostic: plugin name, exit code/class, truncated redacted output excerpt, and an actionable `oca doctor --scope plugins` suggestion

### `oca doctor --scope plugins` + `--network`

`oca doctor` adds a `plugins` scope and a `--network` flag:

| Flag | Purpose |
| --- | --- |
| `--scope plugins` | run plugin health checks (checkout presence, git clean state, build artifacts) |
| `--scope mcp` | existing MCP health surface |
| `--scope temporal` | Temporal health checks |
| `--scope adv-assets` | audit plugin-provided asset ownership boundaries (ORPHANED, DUPLICATE-OWNER, STALE) |
| `--network` | additionally probe plugin remotes with `git ls-remote` (local-only by default per judgment call `jc-doctor1`) |

### New env overrides

| Variable | Default | Purpose |
| --- | --- | --- |
| `OCA_PLUGIN_CHECKOUT_ROOT` | `~/dev/oc-plugins/` | base dir for `{checkout}` token expansion |

## Shipped in Phase 3

Phase 3 adds core `opencode.json` render targets and drift detection.

### `oca apply` core targets + all-target mode

`oca apply` now supports:

| Target | Behavior |
| --- | --- |
| `providers` | Renders `[providers.*]` into `opencode.json` `.provider` with `limit.*` / `modalities.*` shape translation. |
| `permissions` | Renders `[permissions]` into `opencode.json` `.permission`, translating `default` to `"*"`. |
| `watcher` | Renders `[watcher].ignore` into `opencode.json` `.watcher.ignore`, preserving user-added ignore entries. |
| `lsp` | Renders `[lsp.*]` into `opencode.json` `.lsp`. |

When `--target` is omitted, `oca apply` composes all supported targets in dependency order and applies them under one lock using the existing apply engine.

### `oca diff`

Read-only drift detection against the rendered target set.

Flags:

| Flag | Purpose |
| --- | --- |
| `--target <name>` | optional single-target filter (`mcp`, `plugins`, `instructions`, `providers`, `permissions`, `watcher`, `lsp`, `skills`, `commands`, `formatters`, `toggles`) |
| `--output <text\|json>` | report format |

Behavior:

- exit `0` when all selected targets are in sync
- exit `1` when drift is present
- exit `2` for invalid config or invalid target
- exit `3` for runtime planning failures

### `oca doctor --scope adv-assets`

Read-only asset ownership audit across deployed OpenCode config directories.

Findings:

| Class | Meaning | Status |
| --- | --- | --- |
| `ORPHANED` | Non-`adv-*` file exists inside a plugin-owned asset category directory | warn |
| `DUPLICATE-OWNER` | Two or more plugins declare the same `provides` category | fail |
| `STALE` | `adv-{provider}.md` exists but `{provider}` is not configured in `[providers]` | warn |

The scope uses `stack.toml` plugin `provides` declarations as the ownership source of truth and never deletes files.

### `oca doctor --scope adv-plugin`

Advance plugin checkout and state health checks.

Behavior:

- verifies the Advance plugin is declared in `stack.toml`
- checks the checkout directory exists
- validates the build artifact (`dist/index.js`) exists and is non-empty (0-byte artifacts fail)
- checks the ADV state directory is readable (`$XDG_DATA_HOME/opencode/plugins/advance`)
- all checks are local; no plugin code execution or network requests

### `oca doctor --scope cross`

Cross-component consistency checks between `stack.toml` declarations and actual filesystem state.

Behavior:

- detects source drift by comparing `remote.origin.url` in plugin checkouts against `stack.toml` `source` (trailing slashes, `.git` suffixes, and protocol variants are normalized)
- detects MCP server port collisions across all declared `[mcp.servers.*]` entries
- verifies instruction file paths resolve to existing files
- all checks are read-only

## Phase 3.5 additions

### `oca apply` (no target)

When `--target` is omitted, `oca apply` composes all shipped targets in dependency order under one lock. As of Phase 3.5, that includes:

- `mcp`
- `plugins`
- `instructions`
- `providers`
- `permissions`
- `watcher`
- `lsp`
- `skills`
- `commands`
- `formatters`
- `toggles`

### `oca apply --target skills`

Copies OCA-owned skill directories from `assets/skills/` to `~/.config/opencode/skills/`. Respects `[skills].order` and excludes `adv-*` reserved namespace.

### `oca apply --target commands`

Renders `[commands.*]` into `opencode.json` `.command` section.

### `oca apply --target formatters`

Renders `[formatters.*]` into `opencode.json` `.formatter` section.

### `oca apply --target toggles`

Renders `[opencode]` toggles into `opencode.json` (theme, default_agent, share, snapshot, autoupdate, compaction, disabled_providers, enabled_providers).

### `oca doctor --scope skills`

Checks OCA-owned skill deployment health: declared skills present, reserved namespace enforcement, extra skill detection.

## Shipped in Phase 4 (foundation)

Phase 4 foundation adds the session command group with `new` and `list` subcommands, plus theme/tmux assets.

### `oca session new [--name <name>] [--no-splash]`

Create a new detached OCA tmux session with Obsidian theming.

Flags:

| Flag | Purpose |
| --- | --- |
| `--name <name>` | explicit session name (default: auto-generated `oca-<slug>-<n>`) |
| `--no-splash` | skip boot splash on session creation |

Behavior:

- creates a detached tmux session on the dedicated OCA socket (`OCA_TMUX_SOCKET`, default `"oca"`)
- auto-generates name from repo slug basename + next sequential number if `--name` not provided
- resolves the Obsidian tmux conf from `assets/themes/obsidian.tmux.conf`
- triggers `lib/boot_splash.sh` via `tmux send-keys` unless `--no-splash`
- boot splash version line shows `v<version> · <working_dir>` via `OCA_SPLASH_VERSION` + `OCA_SPLASH_DIR`
- supports `--output text|json`
- exit codes: `0` success, `1` session creation failure (tmux error, invalid dir)

### `oca session list`

List all OCA-managed tmux sessions on the current socket.

Aliases: `ls`

Behavior:

- enumerates sessions filtered by `oca-*` prefix on the dedicated socket
- text output: `<name>\t<attached|detached>` per line
- JSON output: array of `{name, attached}` objects
- empty output: `"no OCA sessions"`
- supports `--output text|json`

### Exact-name session commands

`oca session attach <name>`, `switch <name>`, `kill <name>`, and
`restart <name>` are exact-name operations on the dedicated OCA tmux socket.
They intentionally preserve custom names created with `oca session new --name`;
the socket (`OCA_TMUX_SOCKET`, default `"oca"`) is the containment boundary.

Bulk/safety operations are stricter:

- `oca session list` only shows sessions with the generated `oca-` prefix.
- `oca session killall` only destroys sessions returned by `list`.
- `oca session reap` only reaps stale, detached, `oca-` prefixed sessions.

Implementation caveat: `internal/session.GetSessionByName` delegates through
`List()`, so it is also `oca-` prefix scoped. Exact-name commands can still
target custom names directly, but custom-name restarts should pass an explicit
working directory rather than relying on path inference.

### `oca session ensure-window --session <name> --name <window> --cwd <path>`

Create or reuse a named window in a Pattern B project tmux session with cwd set to a worktree path.

Flags:

| Flag | Purpose |
| --- | --- |
| `--session <name>` | Project session name (default: inferred from git root) |
| `--name <window>` | Window name (default: cwd basename) |
| `--cwd <path>` | Window working directory (default: current directory) |

Behavior:

- Creates a new tmux window with the given `--cwd` if `--name` does not exist in the session
- Reuses an existing window with matching `--name` if it exists (idempotent)
- If an existing window has a stale cwd (directory deleted or inaccessible), replaces the window content
- Rejects missing `--cwd` with exit code 1
- Available in OCA v1.0+ (Pattern B session topology, Phase 4 extension)

### `oca session reconcile`

Scan the ADV worktree layout and create missing project-session windows. No filesystem watcher is required.

Flags:

| Flag | Purpose |
| --- | --- |
| `--session <name>` | Project session name (default: inferred from git root) |
| `--dry-run` | Print actions without executing them |

Behavior:

- Scans `$XDG_DATA_HOME/opencode/worktree/*/change/*` for ADV change worktrees
- Ensures one tmux window per change worktree in the project session
- Idempotent: skips already-existing windows
- Available in OCA v1.0+ (Pattern B session topology, Phase 4 extension)

### Smart no-arg root command (`oca` / `oc`)

Running `oca` or `oc` with no arguments in a git repository:

- **Project mode** (default, `[session].mode = "project"`): resolves git root, creates/reuses a project tmux session, and attaches
- **Outside git**: prints `oca session list` hint instead of creating a session
- **Per-invocation mode** (`[session].mode = "per-invocation"`): delegates to `oca session new` (Pattern A escape valve)
- Explicit subcommands (`oca session new`, `oca apply`, etc.) are unaffected by session mode

## Shipped in Phase 6 (foundation)

Phase 6 adds the installer, uninstaller, and shell completion. These commands are shipped and Phase 6 is complete.

### `oca install [--yes]`

Run end-to-end first-time setup: prerequisite checks, `oca apply`, shell profile managed-block injection.

Flags:

| Flag | Purpose |
| --- | --- |
| `--yes` | skip interactive confirmations |
| `--config <path>` | path to `stack.toml` |

Behavior:

- runs `internal/install.PrereqChecker` to verify git, tmux, OpenCode, vision, and (optionally) Temporal CLI
- runs `oca apply` with all targets
- injects a managed block into `~/.zshrc` / `~/.bashrc` with PATH and completion wiring via `internal/install.ShellProfile`
- idempotent: re-running completes any interrupted steps
- exit codes: `0` success, `1` failure, `2` prereq missing

### `oca uninstall`

Remove all OCA-managed blocks from shell profiles.

Behavior:

- scans `~/.zshrc` and `~/.bashrc` for the OCA managed block delimiters
- removes the block without touching user-owned content
- reports what was removed
- exit codes: `0` success, `1` failure

### `oca completion <shell>`

Generate shell completion scripts.

Arguments: `bash`, `zsh`, or `fish`

Behavior:

- prints a completion script to stdout suitable for `source` or direct file write
- integrated into `oca install` shell profile wiring

## Shipped in Phase 6.5

Phase 6.5 adds Temporal dev-server supervision subcommands.

### `oca temporal status`

Report local Temporal dev-server state for agents and scripts.

Behavior:

- supports `--output text|json`
- reports configured/enabled state, address, namespace, PID, managed/running/reachable/healthy booleans, state enum, log path, and DB path
- treats reachable servers without OCA PID metadata as `unmanaged`
- does not mutate runtime state

### `oca temporal start`

Start the OCA-managed local Temporal dev server with `temporal server start-dev`.

Behavior:

- uses `$OCA_CACHE_DIR/temporal/temporal.db` as persistent storage by default
- writes OCA-owned PID metadata under `$OCA_CACHE_DIR/temporal/temporal.pid.json`
- appends combined stdout/stderr to `$OCA_CACHE_DIR/temporal/temporal.log`
- refuses non-loopback dev-server starts and refuses to overwrite unmanaged reachable servers
- idempotently returns current state when the OCA-managed process is already running

### `oca temporal stop`

Stop only the OCA-managed Temporal dev-server process group.

Behavior:

- sends SIGTERM, waits for shutdown, then escalates to SIGKILL if needed
- removes stale PID metadata when the recorded process is gone
- refuses to kill unmanaged reachable servers; it never kills by port lookup alone

### `oca temporal restart`

Compose `stop` then `start` while preserving `$OCA_CACHE_DIR/temporal/temporal.db`.

### `oca temporal logs`

Print bounded recent log output and exit by default.

Flags:

| Flag | Purpose |
| --- | --- |
| `--lines <n>` | Number of recent lines to print (default: `200`) |
| `--follow` | Explicitly stream appended log output until interrupted |

Behavior:

- no pager
- no prompt
- no default long-running stream
- missing log file prints empty output and exits successfully

## Shipped out-of-phase

### `oca dashboard [--bind <addr>] [--port <n>] [--no-open]`

Launch a read-only web dashboard showing cross-project ADV changes, tmux sessions, and Temporal/worker health.

Flags:

| Flag | Purpose |
| --- | --- |
| `--bind <addr>` | bind address (default: `127.0.0.1`) |
| `--port <n>` | listen port (default: `8080`) |
| `--no-open` | skip auto-opening browser |

Behavior:

- serves a Go-embedded web UI via `//go:embed`
- polls Temporal for cross-project ADV change state (2–5s interval)
- watches tmux sessions via control-mode client for real-time session state
- pushes state updates to browser via SSE
- loopback-only by default; bind address is configurable for future v2.0 readiness

### `oca pane`

Per-pane state operations for OCA tmux sessions. Reads and writes JSON state in `$XDG_STATE_HOME/oca/panes/` for session ↔ directory ↔ watchdog correlation.

### `oca occupancy`

Show which OCA panes occupy which worktrees. The command reads local pane state from `$XDG_STATE_HOME/oca/panes/`, reconciles with live tmux pane metadata when available, groups occupants by project/worktree, and warns when multiple active sessions share one worktree.

Flags:

| Flag | Purpose |
| --- | --- |
| `--all` | include stale pane-state records |
| `--json` | emit machine-readable groups, warnings, malformed count, and stale count |
| `--socket <name>` | filter pane-state records by tmux socket |
| `--path <dir>` | filter occupants by directory/worktree path prefix |
| `--status` | emit compact current-pane status for tmux status bar use |
| `--pane <id>` | target pane for `--status` mode |

Privacy/behavior:

- local CLI output may include local worktree paths and branch names;
- the compact status path is fail-closed and prints `?` on missing state/errors;
- no chat/message content is read;
- visibility only — it never locks or blocks multiple sessions.

### `oca watchdog`

Session watchdog that monitors pane activity, tracks bump counts, last-activity timestamps, and idle-state transitions for OCA-managed tmux sessions.

### `oca doctor --scope adv-runtime`

ADV runtime health check. Inspects Temporal search attributes, workflow queues, session debt, and worktree census.

Flags:

| Flag | Purpose |
| --- | --- |
| `--scope adv-runtime` | required |

Behavior:

- Search attributes: verifies required ADV attrs (AdvProjectId, AdvChangeId, etc.) are present and correct
- Workflow queues: lists Running ADV workflows, checks poller counts, flags stale queues
- Session debt: scans OpenCode SQLite DB for stale blank assistant messages
- Worktree census: counts active/stale OCA worktrees and orphan ADV state roots
- Degrades gracefully when Temporal or DB is unavailable — warnings, not hard fails

### `oca session doctor`

Diagnose and optionally clean up stale OpenCode session debt.

Flags:

| Flag | Purpose |
| --- | --- |
| `--db <path>` | OpenCode SQLite database path (default: `$XDG_DATA_HOME/opencode/opencode.db`) |
| `--threshold <duration>` | Stale message age threshold (default: `5m`) |
| `--apply` | Actually delete repairable rows (requires `--backup-dir`) |
| `--backup-dir <dir>` | Backup directory (required with `--apply`) |

Behavior:

- Dry-run by default — scans and reports without deleting
- Repairable = assistant messages with no parts, no finishReason, older than threshold
- `--apply` refuses without `--backup-dir` for safety
- Creates backup manifest JSON alongside copied DB files

### `oca adv recover --dry-run`

Render an ordered ADV recovery plan from the current runtime state.

Flags:

| Flag | Purpose |
| --- | --- |
| `--dry-run` | Required in v1 — shows plan without executing |
| `--project <id>` | Filter plan by project ID |
| `--change <id>` | Filter plan by change ID |

Behavior:

- Loads runtime report from all scanners
- Synthesizes ordered recovery steps: search attrs → queues → session debt → worktrees
- All steps are dry-run only in v1
- Refuses to run without `--dry-run` flag

## Planned later-phase commands

The following are still design targets, not shipped:

- `oca add ...`
- `oca remove ...`
- `oca clean`

### Planned `oca session` behavior (Phase 4 target)

- tmux-backed session lifecycle stays **same-host only**. OCA does not design multi-host session sync.
- `oca session new` creates a detached session on the dedicated OCA tmux socket, then applies normal OCA boot/theme behavior.
- `oca session list` enumerates OCA-managed sessions on that socket only.
- `oca session attach` is the outside-tmux re-entry path: reconnect from any terminal that can reach the same host, including occasional SSH/Tailscale/phone-terminal usage.
- `oca session switch` is the inside-tmux retarget flow and should stay distinct from `attach`.
- `oca session restart` and `killall` must operate only on OCA-managed sessions and preserve clear recovery semantics when failures occur.
- Session re-entry relies on existing host access controls (local shell, SSH, Tailscale, OS account), not OCA-managed auth or transport.
- Reaper behavior must assume temporary disconnect/resume is normal and must not delete sessions a user would reasonably expect to resume.

Implementation sequencing for those commands lives in [`../proposals/phases.md`](../proposals/phases.md).
