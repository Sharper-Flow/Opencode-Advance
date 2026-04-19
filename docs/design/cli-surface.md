# `oca` CLI Surface

This document separates the **shipped Phase 1 CLI** from the broader **planned v1.0 command surface**.

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

## Planned later-phase commands

The following are still design targets, not shipped commands in Phase 1:

- `oca install`
- `oca diff`
- `oca update`
- `oca pin`
- `oca uninstall`
- `oca migrate ...`
- `oca add ...`
- `oca remove ...`
- `oca clean`
- `oca session ...`
- `oca theme ...`

Implementation sequencing for those commands lives in [`../proposals/phases.md`](../proposals/phases.md).
