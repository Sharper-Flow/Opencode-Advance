# `stack.toml` Schema Reference

This document is the canonical reference for the `stack.toml` schema. A complete working example lives at [`stack.example.toml`](../../stack.example.toml).

> Status note: `[meta]`, `[mcp]`, `[plugins.*]`, and `[instructions]` are typed and actively rendered as of Phase 2. `[providers.*]`, `[permissions]`, `[watcher]`, and `[lsp.*]` are typed and actively rendered in Phase 3. `[skills]`, `[formatters.*]`, `[commands.*]`, and `[opencode]` are typed and actively rendered in Phase 3.5. `[temporal]` is typed but reserved for Phase 6.5 (accepted without rendering). `[agents]`, `[session]`, and `[discord]` remain deferred unless otherwise noted.

## Top-level tables

| Table            | Required | Purpose                                              |
| ---------------- | -------- | ---------------------------------------------------- |
| `[meta]`           | yes      | Stack name, version, description                    |
| `[mcp]`            | yes      | MCP server declarations                             |
| `[plugins.*]`      | active   | Plugin declarations with source, build, wiring (Phase 2) |
| `[instructions]`   | active   | Ordered list of instruction files to load (Phase 2) |
| `[providers.*]`    | active   | Provider/model configurations                       |
| `[agents]`         | deferred | Agent model mapping is intentionally not rendered in Phase 3; see OMP docs for preference routing |
| `[permissions]`    | active   | Permission rules (bash, external_directory)         |
| `[watcher]`        | active   | File watcher ignore globs                           |
| `[lsp.*]`          | active   | LSP server configurations                           |
| `[session]`        | no       | tmux session / UX settings                          |
| `[discord]`        | no       | Discord Rich Presence config                       |
| `[skills]`         | no       | OCA-owned skills to copy to `~/.config/opencode/skills/` |
| `[formatters.*]`   | no       | Custom code formatters rendered into `opencode.json` `.formatter` |
| `[commands.*]`     | no       | Custom slash commands rendered into `opencode.json` `.command`    |
| `[opencode]`       | no       | OpenCode-level toggles (default_agent, sharing, updates, compaction, provider lists) |

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
| `type`          | string   | no       | inferred | OCA transport hint: `"stdio"`, `"http"`, `"sse"`, or OCA-only `"daemon"` (Vision admin MCP itself). `transport` is accepted as a Vision-compatible alias. |
| `transport`     | string   | no       | inferred | Vision-native alias for `type`; values `"stdio"`, `"http"`, `"sse"`. OCA accepts either field. |
| `command`       | string   | yes\*    | —       | Executable to spawn. Required for stdio transport.                            |
| `args`          | string[] | no       | `[]`      | Command arguments                                                        |
| `env`           | table    | no       | `{}`      | Environment variables passed to the server                               |
| `env_file`      | string   | no       | —       | Path to a `.env` file with sensitive values; not committed. For new configs, prefer `{env:VAR_NAME}` values in `env`. |
| `url`           | string   | no       | —       | For `http` / `sse` transports. `http` URLs must end in `/mcp`.             |
| `timeout`       | integer  | no       | `5000`    | Request timeout in milliseconds                                          |
| `autostart`     | boolean  | no       | `true`    | Whether Vision should auto-start this server                            |
| `required`      | boolean  | no       | `false`   | Required server. Vision preserves the flag; `oca doctor` upgrades missing/failed required servers to `fail`. |
| `source`        | string   | no       | —       | Source URL for documentation / traceability                              |
| `description`   | string   | no       | —       | Human-readable summary preserved into `vision/servers.yaml`               |
| `enabled`       | boolean  | no       | `true`    | Set false to keep the declaration but disable                           |
| `restart_policy`| string   | no       | Vision default | Vision passthrough: `always`, `on-failure`, `never`                 |
| `max_restarts`  | integer  | no       | Vision default | Vision passthrough                                                 |
| `stateful`      | boolean  | no       | Vision default | Vision passthrough                                                  |
| `availability_profile` | string | no | Vision default | Vision passthrough; currently `networked` supported                |
| `session_timeout` | string | no | Vision default | Vision duration string passthrough                                        |
| `max_sessions`  | integer | no | Vision default | Vision passthrough                                                        |
| `session_ttl`   | string | no | Vision default | Vision duration string passthrough                                         |
| `health_check_interval` | string | no | Vision default | Vision duration string passthrough                                |
| `request_timeout` | string | no | Vision default | Vision duration string passthrough. OCA's integer `timeout` convenience field renders here. |
| `headers`       | table | no | `{}` | Vision passthrough for upstream HTTP/SSE servers                                   |
| `retry`         | table | no | — | Vision passthrough (`max_attempts`, `initial_delay`, `max_delay`, `retryable_errors`) |
| `circuit_breaker` | table | no | — | Vision passthrough (`failure_threshold`, `recovery_timeout`)               |
| `shared_read_only_tools` | string[] | no | `[]` | Vision passthrough                                            |
| `shared_result_cache_ttl` | string | no | — | Vision passthrough                                                     |
| `shared_result_cache_size` | integer | no | 0 | Vision passthrough                                                    |
| `max_in_flight_requests` | integer | no | 0 | Vision passthrough                                                     |

**Type-specific requirements:**
- inferred / `type=stdio` / `transport=stdio` → `command` required; `args` optional
- `type=http` / `transport=http` → `url` required and must end in `/mcp`
- `type=sse` / `transport=sse` → `url` required
- `type=daemon` → OCA-only special case for the Vision admin MCP itself. It renders an `opencode.json` `.mcp.vision` entry pointing at `http://localhost:<port>/mcp` but **does not** emit a `vision` entry into `vision/servers.yaml`.

### Native OpenCode token pass-through

OCA expands its own shell-style variables at parse time:

- `~/...`
- `$HOME`
- `$XDG_CONFIG_HOME`
- `$XDG_DATA_HOME`
- `${VAR}` / `${VAR:-default}`

OCA **does not** resolve native OpenCode tokens. These pass through unchanged:

- `{env:VAR_NAME}`
- `{file:/path/to/file}`

This keeps OCA secret-blind: it records references but never reads the underlying env vars or files.

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
provides     = ["adv-commands", "adv-agents", "adv-skills", "adv-overlays", "adv-instructions"]
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
| `temporal_bundle` | string   | no       | —                  | Reserved for Phase 6.5 Temporal inheritance bundle metadata   |
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
| `adv-temporal`       | Reserved Temporal-owned inheritance surfaces         |

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

When a plugin declares `provides = ["adv-instructions", ...]`, OCA still preserves the declared ownership boundary but omits those instruction entries from its own rendered `.instructions` array. The plugin's sync step is responsible for patching them afterward.

### Ordering contract

For plugin-managed assets, the apply order is strict:

1. render `opencode.json` / related OCA-owned files
2. commit atomic writes successfully
3. invoke plugin sync command (`[plugins.*].sync`) afterward

If rendering fails, sync MUST NOT run. If sync fails, rendered files remain on disk (no rollback of already-committed writes).

---

## `[temporal]`

Reserved for Phase 6.5 Temporal inheritance.

```toml
[temporal]
# reserved — implemented in a later phase (see docs/proposals/phases.md § Phase 6.5)
# enabled = true
```

Current behavior:

- parser accepts the section shape
- validation emits advisory messaging only
- `oca apply --target temporal` and `oca doctor --scope temporal` return a reserved-for-later error
- `adv-temporal` provides category is pre-allocated so ownership boundaries stay stable when Phase 6.5 lands

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

The structure mirrors what OpenCode expects in `opencode.json` `.provider`. OCA passes through the fields without interpreting model names. Unknown additional keys on providers and models are preserved and passed through to the rendered JSON.

---

## `[agents]`

`[agents]` remains accepted as deferred input but is **not rendered by OCA in Phase 3**.

- OCA keeps the section in `DeferredSections` for forward compatibility.
- Existing `.agent.*` config is left untouched by `oca apply`.
- Model preference routing for spawned agents should follow the `opencode-model-preferences` plugin/docs rather than this Phase 3 renderer.

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

Rendered into `opencode.json` `.permission`. OCA preserves the existing TOML-friendly structure and translates to OpenCode's expected shape. Unknown additional keys are preserved and passed through.

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

Rendered into `opencode.json` `.watcher.ignore`. Unknown sibling keys under `[watcher]` are preserved and passed through.

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

Rendered into `opencode.json` `.lsp`. Unknown additional keys per LSP server are preserved and passed through.

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

Design notes for Phase 4 session behavior:

- OCA sessions are same-host tmux sessions. Re-entry from another device means reconnecting to the same host, not syncing sessions across machines.
- Existing host access controls (local shell, SSH, Tailscale, OS account permissions) are the access boundary. No separate OCA session-auth config is planned here.
- Reaper settings must be interpreted with resume safety in mind: a temporarily detached session may still be expected to come back later.
- Multi-client / mobile-terminal behavior needs an explicit tmux window-size policy so a phone-sized client does not unintentionally degrade a desktop session. If that policy becomes user-tunable later, this section is where the schema should expose it.

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

## `[skills]`

Declares which OCA-owned skills should be copied from `assets/skills/` to `~/.config/opencode/skills/` during `oca apply`.

```toml
[skills]
order = [
  "lgrep",
  "mcp-selection",
  "morph",
  "prioritizer",
  "worktree",
  "caveman",
  "caveman-commit",
  "caveman-review",
]
```

| Field  | Type     | Required | Default | Notes                                                              |
| ------ | -------- | -------- | ------- | ------------------------------------------------------------------ |
| `order`  | string[] | no       | all     | Skills to copy in declared order. Omit to copy all OCA-owned skills. |

**Ownership rule:** Only OCA-owned skills may be listed here. Plugin-owned skills (for example ADV methodology skills under `adv-*/`) are managed by their plugin's sync script and MUST NOT appear in this list.

---

## `[formatters.<name>]`

Custom code formatters rendered into `opencode.json` `.formatter`. Each entry defines one formatter. Use `disabled = true` to disable a built-in formatter without defining a replacement.

```toml
[formatters.prettier]
command     = ["npx", "prettier", "--write", "$FILE"]
extensions  = [".ts", ".tsx", ".js", ".jsx", ".svelte", ".json", ".md"]

[formatters.prettier.environment]
NODE_OPTIONS = "--max-old-space-size=4096"

[formatters.ruff]
command    = ["ruff", "format", "$FILE"]
extensions = [".py"]

# Disable a built-in formatter:
[formatters.gofmt]
disabled = true
```

| Field       | Type     | Required | Default | Notes                                                             |
| ----------- | -------- | -------- | ------- | ----------------------------------------------------------------- |
| `command`     | string[] | yes*     | —       | Formatter command + args. `$FILE` is replaced with the file path. Required unless `disabled = true`. |
| `extensions`  | string[] | yes*     | —       | File extensions this formatter handles. Required unless `disabled = true`. |
| `disabled`    | boolean  | no       | `false`   | Set true to disable a built-in formatter. Omit `command`/`extensions` when set. |
| `environment` | table    | no       | `{}`      | Environment variables for the formatter process.                  |

Rendered into `opencode.json` `.formatter.<name>`.

---

## `[commands.<name>]`

Custom slash commands rendered into `opencode.json` `.command`. Each entry defines one command available in the OpenCode command palette.

```toml
[commands.review-pr]
description = "Review the current PR with conventional comments"
template    = "Review the pull request at $ARGUMENTS for correctness, security, and clarity."
agent       = "adv"
model       = "anthropic/claude-opus-4-6"

[commands.test-coverage]
description = "Run tests with coverage and report failures"
template    = "Run the full test suite with coverage. Show failures and suggest fixes."
agent       = "build"
```

| Field        | Type   | Required | Default | Notes                                                          |
| ------------ | ------ | -------- | ------- | -------------------------------------------------------------- |
| `description`  | string | yes      | —       | Shown in the OpenCode command picker                           |
| `template`     | string | yes      | —       | Prompt template. `$ARGUMENTS` is replaced with user-provided text. |
| `agent`        | string | no       | —       | Target agent name                                              |
| `model`        | string | no       | —       | Override model for this command. Falls back to agent default.  |

Rendered into `opencode.json` `.command.<name>`.

---

## `[opencode]`

Top-level OpenCode behavior toggles rendered directly into `opencode.json`. These map to native OpenCode config keys without transformation.

```toml
[opencode]
theme          = "obsidian"    # OpenCode UI theme name
default_agent  = "adv"        # default primary agent
share          = "disabled"   # "manual", "auto", or "disabled"
snapshot       = true         # file change snapshots (set false for large repos)
autoupdate     = false        # true, false, or "notify"

[opencode.compaction]
auto     = true
prune    = true
reserved = 10000              # token buffer to reserve during compaction

[opencode.disabled_providers]
list = ["ollama"]

[opencode.enabled_providers]
list = ["google", "openai", "openrouter"]
```

| Field                     | Type          | Required | Default    | Notes                                                            |
| ------------------------- | ------------- | -------- | ---------- | ---------------------------------------------------------------- |
| `theme`                     | string        | no       | —          | OpenCode UI theme name (controls terminal color scheme)          |
| `default_agent`             | string        | no       | —          | Default primary agent used for new sessions                      |
| `share`                     | string        | no       | `"manual"`   | `"manual"` (on-demand), `"auto"`, or `"disabled"`                  |
| `snapshot`                  | boolean       | no       | `true`       | Disable for large repos where file change tracking is slow       |
| `autoupdate`                | bool/string   | no       | `true`       | `true`, `false`, or `"notify"` (alert without auto-downloading)    |
| `compaction.auto`           | boolean       | no       | `true`       | Auto-compact context when approaching token limits               |
| `compaction.prune`          | boolean       | no       | `true`       | Prune old tool outputs during compaction                         |
| `compaction.reserved`       | integer       | no       | —          | Token buffer to keep free during compaction                      |
| `disabled_providers.list`   | string[]      | no       | `[]`         | Blocklist. Takes precedence over `enabled_providers`.            |
| `enabled_providers.list`    | string[]      | no       | `[]`         | Allowlist. If set, only listed providers are available.          |

Note: `disabled_providers` takes precedence over `enabled_providers`. If a provider appears in both, it is disabled.

Rendered directly into `opencode.json` at the top level.

---

## Variable interpolation

OCA supports two classes of token resolution:

### OCA-resolved tokens (resolved at parse time)

The following tokens are resolved by OCA before rendering to `opencode.json`:

| Token              | Resolves to                                          |
| ------------------ | ---------------------------------------------------- |
| `$HOME`              | User home directory                                  |
| `~/...`              | Same as `$HOME/...`                                    |
| `{checkout}`         | Value of `checkout` field in the same plugin block   |
| `{subdir}`           | Value of `subdir` field in the same plugin block     |
| `$XDG_CONFIG_HOME`   | Resolved environment variable                        |
| `$XDG_DATA_HOME`     | Resolved environment variable                        |
| `${ENV_VAR}`         | Any environment variable                             |

Unknown OCA tokens cause a validation error.

### Native OpenCode tokens (passed through to opencode.json)

OpenCode itself supports native variable substitution in config values. OCA **passes these tokens through unchanged** — it does not resolve them. OpenCode resolves them at runtime:

| Token                  | OpenCode behavior                                                |
| ---------------------- | ---------------------------------------------------------------- |
| `{env:VARIABLE_NAME}`    | Substituted with the value of the named environment variable     |
| `{file:path/to/file}`    | Substituted with the contents of the file at the given path      |

**Use `{env:...}` and `{file:...}` for secrets and dynamic values** instead of OCA-specific `env_file` references where possible. These tokens are natively understood by OpenCode and avoid the need for OCA to resolve or store sensitive values.

Example:
```toml
[mcp.servers.kagi]
port    = 6279
command = "uvx"
args    = ["--from", "kagimcp", "kagimcp"]

[mcp.servers.kagi.env]
KAGI_API_KEY = "{env:KAGI_API_KEY}"
```

The `env_file` field on MCP servers remains supported for cases where `.env` file loading is preferred over runtime environment variable injection.

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

---

## Security & trust boundaries

OCA's validation layer does **not** enforce a host or registry allowlist for
plugin sources. `stack.toml` is a user-authored file; if you declare a
plugin source, OCA trusts that you have vetted it.

This is intentional:

- Private forks and mirrors must be supported (`git.internal.example.com`,
  self-hosted Gitea, corporate GitLab, etc.).
- A hard-coded allowlist would block the long tail of legitimate plugin
  hosting without meaningfully deterring a hostile `stack.toml`, since an
  attacker who can modify `stack.toml` already owns the trust boundary.

Defense-in-depth against hostile *remotes* (as opposed to hostile config)
is layered elsewhere:

- `internal/plugin/git.go` prepends `-c protocol.file.allow=user -c
  protocol.ext.allow=false` to every git invocation to disable
  transport-confusion attack surface (CVE-2022-39253 class).
- `validateGitRef` rejects refs that start with `-`, contain `..`, or use
  characters outside `[A-Za-z0-9._/-]`, preventing option-injection
  through crafted branch names.
- `internal/plugin/prepare.go` uses `os.Lstat` to reject symlinked
  checkout paths and compares `remote.origin.url` to `source` on every
  apply to detect drift between stack.toml and on-disk state.

Treat the local `stack.toml` as the trust root. Treat everything else —
the remote, the network, the ref name — as untrusted.

### Secret redaction in sync output

When OCA invokes the Advance `scripts/sync-global.sh` (see
`internal/sync/advance.go`), stdout/stderr are passed through
`render.Redact` before display. This is a best-effort pattern match
against well-known token shapes (GitHub PATs, OpenAI keys, generic
`secret=...` formats). It is **not** a security boundary: scripts that
print secrets in exotic formats may leak them to the terminal. Treat the
redaction as a convenience, not a guarantee, and avoid logging
user-provided secrets in sync scripts.
