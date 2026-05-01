# `oca` CLI Surface

This document separates the **implemented CLI** from the broader **planned v1.0 command surface**.

Phase 1 shipped `oca version`, `oca apply --target mcp`, `oca doctor --scope mcp`, and `oca debug`. Phase 2 extended the surface with plugin/instructions targets on `apply`, added new `oca pin` and `oca update` commands, and extended `oca doctor` with a `plugins` scope and a `--network` flag. Phases 3–3.5 extended apply/diff/doctor. Phase 4 shipped session lifecycle. Phase 5 shipped temporal config. Phase 5.5 shipped slot groups. Phase 6 (in progress) ships `oca install`, `oca uninstall`, and `oca completion`.

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

## Shipped in Phase 2

Phase 2 adds plugin and instruction management. It extends existing commands and ships two new ones.

### `oca apply --target plugins | --target instructions | --target temporal`

`oca apply` now accepts additional targets alongside `--target mcp`:

| Target | Behavior |
| --- | --- |
| `plugins` | Runs `internal/plugin.Prepare` for every enabled git-source plugin (clone-or-update + build) before rendering. |
| `instructions` | Renders the `instructions` flat array into `opencode.json` via the `MergeArray` primitive. |
| `temporal` | Reserved for Phase 6.5. Currently exits with a "reserved" error + doc pointer. |

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

### `oca doctor --scope plugins` + `--network`

`oca doctor` adds a `plugins` scope and a `--network` flag:

| Flag | Purpose |
| --- | --- |
| `--scope plugins` | run plugin health checks (checkout presence, git clean state, build artifacts) |
| `--scope mcp` | existing MCP health surface |
| `--scope temporal` | reserved, returns "reserved for Phase 6.5" |
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

## Shipped in Phase 6 (foundation)

Phase 6 adds the installer, uninstaller, and shell completion. These commands are shipped but Phase 6 is not yet fully complete.

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

## Planned later-phase commands

The following are still design targets, not shipped:

- `oca migrate ...`
- `oca add ...`
- `oca remove ...`
- `oca clean`
- `oca session attach`
- `oca session switch`
- `oca session killall`
- `oca session restart`
- `oca theme ...`

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
