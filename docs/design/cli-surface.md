# `oca` CLI Surface

This document separates the **shipped Phase 1 + Phase 2 CLI** from the broader **planned v1.0 command surface**.

Phase 1 shipped `oca version`, `oca apply --target mcp`, `oca doctor --scope mcp`, and `oca debug`. Phase 2 extended the surface with plugin/instructions targets on `apply`, added new `oca pin` and `oca update` commands, and extended `oca doctor` with a `plugins` scope and a `--network` flag.

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

Not implemented in Phase 1:

- `--parallel`
- scopes such as `plugins`, `adv`, or `shell`

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

## Planned later-phase commands

The following are still design targets, not shipped:

- `oca install`
- `oca diff`
- `oca uninstall`
- `oca migrate ...`
- `oca add ...`
- `oca remove ...`
- `oca clean`
- `oca session ...`
- `oca theme ...`

Implementation sequencing for those commands lives in [`../proposals/phases.md`](../proposals/phases.md).
