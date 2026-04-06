# `oca` CLI Surface

The complete command reference for the `oca` binary. All commands share:

- `--config <path>`: path to `stack.toml` (default: `./stack.toml` or `$XDG_CONFIG_HOME/opencode-advance/stack.toml`)
- `--output <text|json|yaml>`: output format (default: `text`)
- `--no-color`: disable ANSI color in output
- `--verbose` / `-v`: verbose logging
- `--quiet` / `-q`: suppress non-error output

Exit codes:

| Code | Meaning                                    |
| ---- | ------------------------------------------ |
| `0`    | Success                                    |
| `1`    | User error (bad arguments, missing file)   |
| `2`    | Validation error (stack.toml invalid)      |
| `3`    | Runtime error (network, IO, subprocess)    |
| `130`  | Interrupted (Ctrl+C)                       |

---

## Lifecycle commands

### `oca install`

First-time setup. Performs the full sequence:

1. Validate `stack.toml`
2. Install system prerequisites (prompts for sudo if needed)
3. Clone and build all declared plugins
4. Install/verify MCP server dependencies (Vision, etc.)
5. Render `opencode.json`, `vision/servers.yaml`, tmux config block
6. Copy static assets to `~/.config/opencode/`
7. Delegate ADV asset sync to Advance
8. Wire shell profile (zsh/bash completions, PATH)
9. Verify with `oca doctor`

Flags:

| Flag          | Purpose                                         |
| ------------- | ----------------------------------------------- |
| `--yes`         | Non-interactive mode (for CI / unattended)     |
| `--dry-run`     | Show what would happen without making changes |
| `--skip-shell`  | Don't wire shell profile                       |
| `--skip-tmux`   | Don't inject tmux config block                |

### `oca apply`

Re-render `stack.toml` into config files. Idempotent. Safe to re-run.

Flags:

| Flag       | Purpose                                |
| ---------- | -------------------------------------- |
| `--dry-run`  | Print the plan without applying       |
| `--force`    | Re-render even if no changes detected |
| `--target <name>` | Apply only a specific target (mcp, plugins, instructions, providers, agents, permissions, lsp, watcher, theme, tmux) |

### `oca diff`

Show drift between `stack.toml` and actual state on disk.

```
oca diff
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Drift report
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  ~ opencode.json
    mcp.svelte-mcp.port  →  stack.toml: 6278, actual: 6290  [MODIFIED]
    plugins[3]            →  stack.toml: missing, actual: "/some/path"  [EXTRA]

  + vision/servers.yaml
    playwright            →  stack.toml: declared, actual: missing  [MISSING]

  2 modified, 1 extra, 1 missing. run `oca apply` to reconcile.
```

### `oca doctor`

Verify every declared component is live and healthy.

Checks:

- Vision daemon running
- All MCP servers respond on their declared ports
- All plugins are cloned, built, and have the expected entry points
- All instruction files exist and are readable
- All providers have at least one model declared
- All agent model assignments reference known providers
- Advance plugin state is readable
- tmux theme is sourced in `~/.tmux.conf`

Flags:

| Flag              | Purpose                                        |
| ----------------- | ---------------------------------------------- |
| `--scope <scope>`   | Limit to one scope: `mcp`, `plugins`, `adv`, `shell` |
| `--timeout <sec>`   | Per-check timeout (default 5s)                 |
| `--parallel <n>`    | Max concurrent checks (default 8)              |

### `oca update`

Pull latest from all plugin sources, rebuild, re-apply. Respects `ref` pins in `stack.toml`.

### `oca pin`

Capture current plugin git SHAs into `stack.toml`. After `oca pin`, running `oca update` will only pull to the pinned SHAs, not tracking branch heads.

### `oca uninstall`

Remove managed shell profile blocks, tmux config block, and session helpers. Does NOT remove plugin checkouts or user data (specs, changes, wisdom, agenda).

---

## Stack mutation commands

### `oca add mcp <name>`

Interactive wizard to add a new MCP server declaration to `stack.toml` and re-apply.

```
oca add mcp sentry
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ? Port: 6289
  ? Command: npx
  ? Args: -y, @sentry/mcp-server@latest
  ? Timeout (ms): 15000
  ? Autostart: yes
  ? Environment file: ~/.config/opencode-advance/secrets/sentry.env
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Updated stack.toml
  Running oca apply...
  ✓ sentry registered with Vision
  ✓ opencode.json updated
```

### `oca add plugin <name>`

Same pattern for plugins.

### `oca remove mcp <name>`

Removes the `[mcp.servers.<name>]` block from `stack.toml` and re-applies.

### `oca remove plugin <name>`

Removes the `[plugins.<name>]` block from `stack.toml` and re-applies. Leaves the checkout directory on disk; use `oca clean` to remove it.

### `oca clean`

Removes orphaned plugin checkouts (directories in `$OCA_PLUGIN_CHECKOUT_ROOT` that don't correspond to any declared plugin).

---

## Migration commands

### `oca migrate from-open-chad`

Read the current state of an `open-chad` installation and emit a `stack.toml` reflecting it.

```
oca migrate from-open-chad --output ~/stack.toml.draft
```

Reads:

- `~/.config/opencode/opencode.json`
- `~/.config/vision/servers.yaml`
- `~/.config/opencode/open-chad.json`
- `~/dev/open-chad/config/opencode/` (for reference)

Writes to stdout or `--output <path>`. Does NOT modify any live config.

### `oca migrate init`

Create a minimal starter `stack.toml` in the current directory.

---

## Session commands

### `oca session new [<project-path>]`

Create a new tmux session. Replaces `openchad`/`oc` session creation.

```bash
oca session new                    # launch in current directory
oca session new ~/dev/my-project   # launch in specific directory
oca session new --no-splash        # skip boot splash animation
```

### `oca session list`

List active `oca-*` tmux sessions with window count and memory use. Replaces `oc-list`.

### `oca session attach`

Attach to a running session. Prompts if multiple active.

### `oca session switch`

Pick a session to switch to from the current session. Fuzzy-searchable.

### `oca session killall`

Kill all `oca-*` sessions. Prompts for confirmation unless `--yes`.

### `oca session restart`

Restart OpenCode in the current tmux pane (reloads config changes without creating a new session).

---

## Theme commands

### `oca theme list`

List available themes under `assets/themes/`.

### `oca theme set <name>`

Switch to a different theme. Updates `stack.toml` `session.theme` and re-applies.

### `oca theme preview <name>`

Print theme colors as swatches in the terminal.

---

## Discord commands

### `oca discord enable`

Enable Discord Rich Presence with the built-in app ID.

### `oca discord enable --custom <app-id>`

Enable with a custom Discord app ID.

### `oca discord status`

Show current Discord presence state and bridge status (WSL2).

### `oca discord disable`

Disable Discord presence.

---

## Utility commands

### `oca version`

Print the wordmark, version, commit SHA, build date, and Go version.

```
  OpenCode ADVANCE
  v1.0.0  ·  commit d4f5e6a  ·  built 2026-04-06  ·  go1.22.1
  ~/dev/opencodeadvance
```

### `oca info`

Print resolved configuration paths:

```
  stack.toml:               /home/jrede/stack.toml
  opencode config:          /home/jrede/.config/opencode
  vision config:            /home/jrede/.config/vision
  plugin checkout root:     /home/jrede/dev/oc-plugins
  cache dir:                /run/user/1000/opencode-advance
  session log:              /run/user/1000/opencode-advance/sessions.log
```

### `oca completion <shell>`

Print shell completion script for `bash`, `zsh`, or `fish`. Used by the installer to wire completions.

### `oca help [<command>]`

Print help. Cobra-generated.

---

## Hidden / developer commands

### `oca debug plan`

Print the internal render plan as JSON. Useful for debugging template issues.

### `oca debug validate`

Run only the validation pass on `stack.toml` without rendering anything.

### `oca debug checks`

List all available doctor checks without running them.

---

## Environment variables

| Variable                    | Default                                     | Purpose                                        |
| --------------------------- | ------------------------------------------- | ---------------------------------------------- |
| `OCA_CONFIG`                  | `./stack.toml` or XDG path                    | Path to stack.toml                             |
| `OCA_OPENCODE_CONFIG_DIR`     | `~/.config/opencode`                          | Target OpenCode config dir                    |
| `OCA_VISION_CONFIG_DIR`       | `~/.config/vision`                            | Target Vision config dir                      |
| `OCA_PLUGIN_CHECKOUT_ROOT`    | `~/dev/oc-plugins`                            | Where to clone plugins                        |
| `OCA_CACHE_DIR`               | `$XDG_RUNTIME_DIR/opencode-advance`           | Runtime cache dir                             |
| `OCA_LOG_LEVEL`               | `info`                                        | `debug`, `info`, `warn`, `error`                  |
| `OCA_NO_COLOR`                | unset                                       | Disable ANSI color in output                  |
| `OCA_BOOT_SPLASH`             | `1`                                           | Set `0` to disable boot splash                  |

All environment overrides are respected for development, testing, and isolation.
