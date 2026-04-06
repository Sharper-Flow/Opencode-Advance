# `stack.toml` Schema Reference

This document is the canonical reference for the `stack.toml` schema. A complete working example lives at [`stack.example.toml`](../../stack.example.toml).

## Top-level tables

| Table            | Required | Purpose                                              |
| ---------------- | -------- | ---------------------------------------------------- |
| `[meta]`           | yes      | Stack name, version, description                    |
| `[mcp]`            | yes      | MCP server declarations                             |
| `[plugins.*]`      | yes      | Plugin declarations with source, build, wiring     |
| `[instructions]`   | yes      | Ordered list of instruction files to load           |
| `[providers.*]`    | no       | Provider/model configurations                      |
| `[agents]`         | no       | Agent → model assignments                           |
| `[permissions]`    | no       | Permission rules (bash, external_directory)         |
| `[watcher]`        | no       | File watcher ignore globs                           |
| `[lsp.*]`          | no       | LSP server configurations                          |
| `[session]`        | no       | tmux session / UX settings                          |
| `[discord]`        | no       | Discord Rich Presence config                       |

Missing optional tables mean "use defaults" — OCA ships sane defaults for each.

---

## `[meta]`

```toml
[meta]
version     = "1.0.0"         # stack.toml schema version (not product version)
name        = "default"       # arbitrary identifier for this stack
description = "My stack"      # human-readable description
```

| Field         | Type   | Required | Default     | Notes                              |
| ------------- | ------ | -------- | ----------- | ---------------------------------- |
| `version`       | string | yes      | —           | Schema version. Currently `"1.0.0"`. |
| `name`          | string | no       | `"default"`   |                                    |
| `description`   | string | no       | `""`          |                                    |

---

## `[mcp.servers.<name>]`

Declares an MCP server that `oca` will register with Vision and wire into `opencode.json` `.mcp`.

```toml
[mcp.servers.context7]
port      = 6276
command   = "npx"
args      = ["-y", "@upstash/context7-mcp@latest"]
timeout   = 10000
autostart = true
source    = "https://github.com/upstash/context7"
enabled   = true
```

| Field         | Type     | Required | Default | Notes                                                                    |
| ------------- | -------- | -------- | ------- | ------------------------------------------------------------------------ |
| `port`          | integer  | yes      | —       | Port Vision will listen on for this server                              |
| `type`          | string   | no       | `"stdio"` | `"stdio"` (command+args), `"remote"` (url), or `"daemon"` (Vision itself)    |
| `command`       | string   | yes\*    | —       | Executable to spawn. Required for `type=stdio`.                            |
| `args`          | string[] | no       | `[]`      | Command arguments                                                        |
| `env`           | table    | no       | `{}`      | Environment variables passed to the server                               |
| `env_file`      | string   | no       | —       | Path to a `.env` file with sensitive values; not committed              |
| `url`           | string   | no       | —       | For `type=remote` only                                                     |
| `timeout`       | integer  | no       | `5000`    | Request timeout in milliseconds                                          |
| `autostart`     | boolean  | no       | `true`    | Whether Vision should auto-start this server                            |
| `required`      | boolean  | no       | `false`   | If true, `oca apply` aborts on failure to install/verify                   |
| `source`        | string   | no       | —       | Source URL for documentation / traceability                              |
| `enabled`       | boolean  | no       | `true`    | Set false to keep the declaration but disable                           |

### Required MCP servers

The `required = true` flag marks servers that the stack cannot function without. Currently:

- `vision` (the daemon itself)
- Any server marked `required` by the user

---

## `[plugins.<name>]`

Declares a plugin that OpenCode will load.

```toml
[plugins.advance]
source       = "https://github.com/Sharper-Flow/Advance.git"
ref          = "trunk"
checkout     = "~/dev/oc-plugins/advance"
subdir       = "plugin"
build        = ["pnpm install", "pnpm build"]
path         = "{checkout}/{subdir}"
sync         = "{checkout}/scripts/sync-global.sh --fix"
provides     = ["adv-commands", "adv-agents", "adv-skills", "adv-overlays"]
instructions = ["{checkout}/ADV_INSTRUCTIONS.md"]
```

| Field         | Type       | Required | Default          | Notes                                                         |
| ------------- | ---------- | -------- | ---------------- | ------------------------------------------------------------- |
| `source`        | string     | yes      | —                | `git+https://...`, `https://...git`, or `npm:<pkg>`              |
| `ref`           | string     | no       | `"trunk"`          | Branch, tag, or SHA                                           |
| `checkout`      | string     | yes\*    | —                | Local clone path. Required for git sources.                   |
| `subdir`        | string     | no       | `""`               | Subdirectory inside the checkout that contains the plugin     |
| `build`         | string[]   | no       | `[]`               | Shell commands to run after clone/update                      |
| `path`          | string     | yes      | —                | Final path that OpenCode loads (supports `{checkout}`, `{subdir}`) |
| `sync`          | string     | no       | —                | Optional sync script to run after build                       |
| `provides`      | string[]   | no       | `[]`               | Asset categories this plugin owns (OCA skips them)            |
| `instructions`  | string[]   | no       | `[]`               | Instruction files to append to the instructions list         |
| `enabled`       | boolean    | no       | `true`             |                                                               |

### Provides categories

When `provides` includes a category, `oca apply` will NOT render or copy assets in that category from its own `assets/` dir. The categories are:

| Category           | What OCA skips                                     |
| ------------------ | -------------------------------------------------- |
| `adv-commands`       | `assets/command/adv-*.md`                            |
| `adv-agents`         | `assets/agents/{plan,scout,refine,adv-*,tron}.md`    |
| `adv-skills`         | `assets/skills/adv-*/`                               |
| `adv-overlays`       | Overlay blocks in shared agent files                |
| `adv-instructions`   | `ADV_INSTRUCTIONS.md` in the instructions list       |

### npm-distributed plugins

```toml
[plugins.md-table-formatter]
source = "npm:@franlol/opencode-md-table-formatter@latest"
```

For `npm:` sources, `checkout`, `subdir`, `build`, and `path` are not required — OpenCode resolves npm packages itself. Only `source` is needed.

---

## `[instructions]`

Ordered list of instruction files to load. OCA resolves each path and writes it to `opencode.json` `.instructions` in the declared order.

```toml
[instructions]
order = [
  "identity.md",
  "rules.yaml",
  "shell_strategy.md",
  # ...
]
```

Paths may be:

- Bare filenames (resolved against `~/.config/opencode/instructions/`)
- Absolute paths
- `~/`-prefixed paths
- `{checkout}`-templated paths for plugin-provided instructions

Plugin-provided instructions (declared via `[plugins.*].instructions`) are appended to the user-declared order automatically. They always sort after user-owned instructions.

---

## `[providers.<name>]`

Provider configurations rendered into `opencode.json` `.provider`.

```toml
[providers.openai]
# top-level options apply to all models under this provider

[providers.openai.options]
reasoningEffort  = "medium"
reasoningSummary = "auto"
textVerbosity    = "medium"
store            = false
include          = ["reasoning.encrypted_content"]

[providers.openai.models."gpt-5.2"]
name    = "GPT 5.2 (OAuth)"
context = 272000
output  = 128000
inputs  = ["text", "image"]
outputs = ["text"]

[providers.openai.models."gpt-5.2".variants.high]
reasoningEffort  = "high"
reasoningSummary = "detailed"
textVerbosity    = "medium"
```

The structure mirrors what OpenCode expects in `opencode.json` `.provider`. OCA passes through the fields without interpreting model names.

---

## `[agents]`

Simple flat table mapping agent names to model IDs.

```toml
[agents]
build     = "anthropic/claude-opus-4-6"
plan      = "openai/gpt-5.2"
scout     = "zai-coding-plan/glm-5"
refine    = "anthropic/claude-opus-4-6"
librarian = "openrouter/anthropic/claude-haiku-4.5:nitro"
explore   = "zai-coding-plan/glm-5"
```

Rendered into `opencode.json` `.agent.<name>.model`.

---

## `[permissions]`

```toml
[permissions]
default = "allow"
doom_loop = "ask"

[permissions.external_directory]
"*"                                            = "ask"
"~/dev/**"                                     = "allow"
"~/scratch/**"                                 = "allow"
"~/.local/share/opencode/plugins/advance/**"   = "allow"

[permissions.bash]
"*"              = "allow"
"git push*"      = "ask"
"rm -rf *"       = "ask"
"sudo *"         = "ask"
"mkfs*"          = "deny"
```

Rendered into `opencode.json` `.permission`. OCA preserves the existing TOML-friendly structure and translates to OpenCode's expected shape.

---

## `[watcher]`

```toml
[watcher]
ignore = [
  "node_modules/**",
  "dist/**",
  "build/**",
  ".git/**",
  "**/coverage/**",
]
```

Rendered into `opencode.json` `.watcher.ignore`.

---

## `[lsp.<name>]`

```toml
[lsp.pyrefly]
command    = ["pyrefly", "lsp"]
extensions = [".py", ".pyi"]

[lsp.typescript]
command = ["node", "./node_modules/typescript/lib/tsserver.js", "--useInferredProjectPerProjectRoot"]
extensions = [".ts", ".tsx", ".svelte"]
```

Rendered into `opencode.json` `.lsp`.

---

## `[session]`

```toml
[session]
prefix               = "oca-"        # tmux session name prefix
reaper               = true          # kill unattached sessions after timeout
reaper_timeout_hours = 4
theme                = "obsidian"    # theme name under assets/themes/
boot_splash          = true          # show GBA wordmark on session create
boot_splash_timeout  = 1000          # ms before dropping into the session
```

---

## `[discord]`

```toml
[discord]
enabled  = false                                  # opt-in
mode     = "builtin"                              # "builtin" or "custom"
app_id   = ""                                     # only for mode="custom"
```

Tagline pool lives separately at `lib/discord/taglines.toml` (data-driven, replaceable).

---

## Variable interpolation

The following tokens are resolved at parse time:

| Token          | Resolves to                                  |
| -------------- | --------------------------------------------- |
| `$HOME`          | User home directory                          |
| `~/...`          | Same as `$HOME/...`                             |
| `{checkout}`     | Value of `checkout` field in the same plugin block |
| `{subdir}`       | Value of `subdir` field in the same plugin block   |
| `$XDG_CONFIG_HOME` | Resolved environment variable                |
| `$XDG_DATA_HOME`   | Resolved environment variable                |
| `${ENV_VAR}`     | Any environment variable                     |

Unknown tokens cause a validation error.

---

## Validation errors

OCA emits structured validation errors with field paths:

```
error: stack.toml validation failed

  [mcp.servers.context7].port: required field missing
  [plugins.advance].path: unresolved token {subdir} (no subdir declared)
  [providers.openai.models."gpt-5.2".variants.high].reasoningEffort: unknown value "ultra" (expected: none, low, medium, high, xhigh)

3 errors. aborting.
```

Field paths follow TOML-like notation with `.` separators and `[]` for map keys.
