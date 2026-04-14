# Architecture Overview

OpenCode Advance is a Go CLI + static asset bundle + shell/client integration layer that turns a single declarative file (`stack.toml`) into a fully configured OpenCode environment and primary client UX.

This document is the canonical high-level architecture reference. Implementation details live in package-level docs and the source code itself.

## Core principle: single source of truth

The user owns one file: `stack.toml`. Everything else is rendered from it or delegated to declared dependencies.

```
                    ┌─────────────────┐
                    │   stack.toml    │ ← user-owned source of truth
                    └────────┬────────┘
                             │
                             ▼
                   ┌──────────────────┐
                   │   oca apply      │
                   └────────┬─────────┘
                            │
          ┌─────────────────┼─────────────────┐
          ▼                 ▼                 ▼
┌──────────────────┐ ┌────────────┐ ┌──────────────────┐
│  opencode.json   │ │ vision/    │ │ ~/.tmux.conf     │
│  (mcp, plugins,  │ │ servers.   │ │ (managed block,  │
│  instructions,   │ │ yaml       │ │ current client/  │
│  providers,      │ └────────────┘ │ session path)    │
│  agents, perms,  │                └──────────────────┘
│  lsp, watcher)   │
└──────────────────┘
          │
          │   delegates ADV asset sync to:
          ▼
┌────────────────────────────────────────┐
│ advance/scripts/sync-global.sh --fix   │
│ (syncs adv-*.md commands, ADV agents,  │
│  ADV skills, ADV instructions)         │
└────────────────────────────────────────┘
```

## Subsystems

### 1. Configuration (`internal/config/`)

- Parses `stack.toml` using `github.com/pelletier/go-toml/v2`
- Validates against a strict schema (see [`stack-toml-schema.md`](stack-toml-schema.md))
- Resolves variables (`{checkout}`, `{subdir}`, `$HOME`, etc.)
- Merges project-local overrides (`.opencode-advance/stack.toml` in project roots)
- Emits structured errors with field paths for invalid configs

Key types:

```go
type Stack struct {
    Meta         Meta                  `toml:"meta"`
    MCP          MCP                   `toml:"mcp"`
    Plugins      map[string]Plugin     `toml:"plugins"`
    Instructions Instructions          `toml:"instructions"`
    Providers    map[string]Provider   `toml:"providers"`
    Agents       map[string]string     `toml:"agents"`
    Permissions  Permissions           `toml:"permissions"`
    Watcher      Watcher               `toml:"watcher"`
    LSP          map[string]LSPServer  `toml:"lsp"`
    Session      Session               `toml:"session"`
    Discord      Discord               `toml:"discord"`
}
```

### 2. Rendering (`internal/render/`)

- Takes a validated `Stack` and produces the final config files
- Uses Go `text/template` with safe escaping for JSON targets
- Templates live in `templates/` as `.gotmpl` files
- Merges generated output into existing `opencode.json` using an idempotent merge algorithm:
  - Array elements: append if missing (deduped by identity)
  - Object fields: OCA-managed keys overwrite; non-managed keys preserved
  - Never overwrites user-added MCP servers, plugins, or instructions not declared in stack.toml (unless they conflict with an explicit declaration)
  - Writes via temp-file + `os.Rename` for atomicity
  - Creates `.bak.<epoch>` backups with configurable rotation
- Delegates ADV asset sync by invoking `<advance>/scripts/sync-global.sh --fix` as a subprocess after rendering opencode.json

Key operations:

```go
func Render(stack *Stack, target Target) (*Plan, error)
func Apply(plan *Plan, opts ApplyOptions) error
```

### 3. Health (`internal/health/`)

- Implements the checks invoked by `oca doctor`
- Each check returns a `Check` result: `{Name, Status, Message, Hint, Elapsed}`
- Check types:
  - **Filesystem**: path exists, is readable, has correct permissions
  - **Binary**: executable exists on PATH, reports expected version
  - **HTTP**: endpoint responds with expected status within timeout
  - **JSON field**: `opencode.json` has required field with expected value
  - **Plugin built**: plugin checkout contains `dist/index.js` or declared entry point
  - **Git repo**: plugin checkout is a git repo on expected ref/SHA
  - **ADV state**: `~/.local/share/opencode/plugins/advance/{project-id}/` exists and is readable
- All checks run in parallel with a shared bounded worker pool (default 8 concurrent)
- Total timeout per `oca doctor` run: 30 seconds

### 4. Migration (`internal/migrate/`)

- Implements `oca migrate from-open-chad`
- Reads the user's current `~/.config/opencode/opencode.json`, `~/.config/vision/servers.yaml`, `~/dev/open-chad/config/`, and `~/.config/opencode/open-chad.json`
- Produces a `stack.toml` that reflects the current state
- Does NOT modify any live config during migration — emits the file to stdout or a specified output path for user review
- Once the user reviews and places `stack.toml`, they run `oca install` for the actual cutover
- Preserves user-specific values (plugin paths, provider model lists, agent model assignments)

### 5. Primary client/session lifecycle (`lib/` + `cmd/oca/session*.go`)

v1 planning is currently tmux-first, but this subsystem should be treated as the client/session layer rather than a permanent commitment to one frontend.

- Tmux session creation, listing, attach, switch, killall, restart
- Session name convention: `oca-<epoch>-<pid>`
- Per-session cache directory: `$OCA_CACHE_DIR/<session-id>/`
- Stale session reaper: kills unattached `oca-*` sessions older than `session.reaper_timeout_hours` (default 4)
- Safe CWD resolution: fallback chain if the pane's current path is invalid or deleted
- Integration with Advance state: current tmux status surfaces read `~/.local/share/opencode/plugins/advance/{project-id}/` to show active change + gate progress in the status UI

The current tmux-level theme, boot splash, and status bar renderers are bash because tmux scripting is genuinely cleaner in bash than in Go. These live in `lib/`.

### 6. CLI entry (`cmd/oca/`)

- `main.go` — `cobra`-based command tree
- One file per top-level command group: `apply.go`, `doctor.go`, `diff.go`, `install.go`, `migrate.go`, `session.go`, `theme.go`, `discord.go`, etc.
- Every command accepts `--config` (path to stack.toml, default `./stack.toml` or `$XDG_CONFIG_HOME/opencode-advance/stack.toml`)
- Every command accepts `--dry-run` where applicable
- Structured output via `--output json|yaml|text` for scriptability
- Exit codes: `0` success, `1` user error, `2` validation error, `3` runtime error

## Data flow: `oca apply`

```
 1. Load stack.toml              (config.Load)
 2. Validate schema              (config.Validate)
 3. Resolve variables             (config.Resolve)
 4. Merge project overrides      (config.MergeLocal)
 5. Build render plan            (render.Plan)
 6. --dry-run? print plan, exit
 7. Backup existing target files (render.Backup)
 8. Render opencode.json         (render.OpencodeJSON)
 9. Render vision/servers.yaml   (render.VisionServers)
10. Render tmux/client session block(s) (render.TmuxBlock)
11. Clone/build declared plugins (render.ResolvePlugins)
12. Delegate to Advance sync     (render.DelegateAdvanceSync)
13. Copy static assets           (render.CopyAssets)
14. Write atomically              (render.Commit)
15. Report summary                (cli.Summary)
```

Each step is a pure function over validated inputs. Failures abort the whole apply with a rollback to backups.

## Data flow: `oca doctor`

```
1. Load stack.toml               (config.Load — non-fatal on missing)
2. Load actual state             (health.LoadActual)
3. Build check list              (health.BuildChecks)
4. Run checks in parallel        (health.RunAll)
5. Aggregate results              (health.Aggregate)
6. Render report                  (cli.RenderDoctor)
7. Exit with status               (0 if all pass, 1 if any warnings, 2 if any errors)
```

## Plugin lifecycle

OpenCode Advance manages plugin checkouts declaratively. Each `[plugins.<name>]` block in stack.toml specifies:

- `source`: git URL or `npm:<package>` for npm-distributed plugins
- `ref`: branch name, tag, or SHA (pin for reproducibility)
- `checkout`: local path to clone to
- `subdir`: subdirectory inside the checkout that contains the plugin
- `build`: shell commands to run for build (array of strings)
- `path`: final path to the plugin entry point (what OpenCode loads)
- `sync`: optional sync script to run after build (e.g., Advance's sync-global.sh)
- `provides`: optional declaration of what assets this plugin owns (OCA will not duplicate)

Plugin operations:

| Action      | Command                                                                                         | Behavior                                            |
| ----------- | ----------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| Install new | `oca apply`                                                                                     | Clones, builds, wires                               |
| Update      | `oca update`                                                                                    | Fetches latest (respects pin), rebuilds, re-wires   |
| Pin         | `oca pin`                                                                                       | Writes current SHAs for all plugins into stack.toml |
| Remove      | `oca apply` after removing from stack.toml — leaves checkout intact, removes from opencode.json |                                                     |

## Advance as a declared dependency

The Advance plugin is treated exactly like any other plugin in `stack.toml`, with one additional capability: it declares `provides` to signal ownership of certain asset categories, and `sync` to delegate its own asset sync.

```toml
[plugins.advance]
source    = "https://github.com/Sharper-Flow/Advance.git"
ref       = "trunk"
checkout  = "~/dev/oc-plugins/advance"
subdir    = "plugin"
build     = ["pnpm install", "pnpm build"]
path      = "{checkout}/{subdir}"
sync      = "{checkout}/scripts/sync-global.sh --fix"
provides  = ["adv-commands", "adv-agents", "adv-skills", "adv-overlays"]
instructions = ["{checkout}/ADV_INSTRUCTIONS.md"]
```

When `oca apply` runs:

1. It clones/updates the Advance checkout
2. It builds the plugin (`pnpm install && pnpm build`)
3. It wires the plugin path into `opencode.json` `.plugin` array
4. It appends `ADV_INSTRUCTIONS.md` to the instructions list
5. It invokes the `sync` script to let Advance manage its own assets (commands, agents, skills, overlays)
6. When rendering assets from `assets/agents/`, `assets/skills/`, etc., it SKIPS anything that Advance `provides`

This is the key mechanism that eliminates the open-chad sync conflict.

## Clean cutover policy during development

Until v1.0 is released and the migration is executed, OpenCode Advance MUST NOT touch the user's live configuration. Tests and dev runs use isolated target directories via environment overrides:

- `OCA_OPENCODE_CONFIG_DIR=$HOME/.opencode-advance-dev`
- `OCA_VISION_CONFIG_DIR=$HOME/.vision-dev`
- `OCA_PLUGIN_CHECKOUT_ROOT=/tmp/oca-test-plugins`

Release day:

1. User runs `oca migrate from-open-chad > ~/stack.toml.draft`
2. User reviews the generated stack.toml, makes adjustments
3. User places the finalized `stack.toml`
4. User runs `openchad uninstall` to remove open-chad's managed blocks
5. User runs `oca install` to apply the new stack
6. User runs `oca doctor` to verify everything is live

The cutover window is minimal and the migration is reversible (backups of all pre-migration config are retained).
