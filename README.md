<div align="center">

```
░█▀█░█▀█░█▀▀░█▀█░█▀▀░█▀█░█▀▄░█▀▀   ░▟█▙░█▀▄░█░█░▟█▙░█▀█░█▀▀░█▀▀
░█░█░█▀▀░█▀▀░█░█░█░░░█░█░█░█░█▀▀   ░█▀█░█░█░▀▄▀░█▀█░█░█░█░░░█▀▀
░▀▀▀░▀░░░▀▀▀░▀░▀░▀▀▀░▀▀▀░▀▀░░▀▀▀   ░█░█░▀▀░░░▀░░█░█░▀░▀░▀▀▀░▀▀▀
```

**Declarative OpenCode environment management, paired with Advance.**

_Status: v1.0 release candidate._

</div>

---

## What OpenCode Advance is

OpenCode Advance (`oca`) is a Go CLI for building a reproducible OpenCode setup from one source of truth: `stack.toml`.

It owns the environment layer:

- MCP and Vision server configuration
- plugin checkout, build, pin, and apply flow
- OpenCode provider, permission, watcher, LSP, skill, command, formatter, and theme config
- tmux-first session lifecycle and Obsidian theme
- Temporal dev-server supervision for Advance
- migration from the older shell-based setup
- health checks, drift checks, dashboard, watchdog, panes, and Discord Rich Presence

Advance owns the spec-driven workflow layer. OCA installs and wires Advance, but does not duplicate Advance-owned agents, commands, skills, or state.

## Install quickstart

Download a release binary from GitHub Releases, then:

```bash
chmod +x oca
./oca install
./oca apply --dry-run
./oca apply
./oca doctor
```

Development checkout:

```bash
git clone https://github.com/Sharper-Flow/Opencode-Advance.git
cd Opencode-Advance
make build
./bin/oca version
```

Use environment overrides during development and tests:

```bash
export OCA_OPENCODE_CONFIG_DIR="$PWD/.dev/opencode"
export OCA_VISION_CONFIG_DIR="$PWD/.dev/vision"
export OCA_PLUGIN_CHECKOUT_ROOT="$PWD/.dev/plugins"
export OCA_CACHE_DIR="$PWD/.dev/cache"
```

## Command reference

| Command | Purpose |
|---|---|
| `oca version` | Print branded version output |
| `oca install` | Install prerequisites, shell hooks, and managed config blocks |
| `oca uninstall` | Remove OCA-managed shell/config blocks |
| `oca apply` | Render `stack.toml` to target config files |
| `oca apply --dry-run` | Show planned writes without changing files |
| `oca diff` | Show drift between desired and actual state |
| `oca doctor` | Run health checks across config, plugins, MCP, Temporal, and Advance |
| `oca maintain` | Offline maintenance plan for verified ADV merges, plugin rebuilds, and safe worktree cleanup |
| `oca pin` | Capture pinned plugin revisions |
| `oca update` | Update managed plugin checkouts safely |
| `oca migrate init` | Create a starter `stack.toml` |
| `oca migrate from-open-chad` | Import older environment state into `stack.toml` |
| `oca session new` | Start an OCA-managed tmux/OpenCode session |
| `oca session list` | List OCA-managed sessions |
| `oca session attach` | Attach to a session |
| `oca session switch` | Switch tmux client to a session |
| `oca session kill` / `killall` | Stop managed sessions |
| `oca session restart` | Restart a session while preserving context |
| `oca session reap` | Clean stale unattached sessions |
| `oca theme list` / `set` / `preview` | Manage bundled themes |
| `oca pane` | Inspect and manage per-pane OpenCode state |
| `oca watchdog` | Detect stuck panes and support recovery |
| `oca temporal start` | Start OCA-managed Temporal dev server |
| `oca temporal stop` | Stop OCA-managed Temporal dev server |
| `oca temporal status` | Show Temporal health and ownership state |
| `oca temporal logs` | Print Temporal logs |
| `oca dashboard` | Open the local operator dashboard |
| `oca discord enable` | Enable Discord Rich Presence updates |
| `oca discord disable` | Disable Discord Rich Presence updates |
| `oca discord status` | Show Discord Rich Presence runtime state |

## stack.toml overview

`stack.toml` declares the environment OCA should render and verify.

```toml
[meta]
version = "1.0.0"
name = "OpenCode Advance"

[mcp.servers.context7]
type = "http"
url = "http://localhost:6277/mcp"

[plugins.advance]
repo = "https://github.com/Sharper-Flow/Advance.git"
ref = "trunk"

[opencode]
theme = "obsidian"
default_agent = "adv"

[session]
theme = "obsidian"
boot_splash = true

[temporal]
enabled = true

[discord]
enabled = false
mode = "builtin"
```

See `stack.example.toml` for a complete reference.

## Safety model

OCA writes only OCA-owned files and managed blocks. User-owned files remain user-owned.

During development, never point OCA at live config. Use:

- `OCA_OPENCODE_CONFIG_DIR`
- `OCA_VISION_CONFIG_DIR`
- `OCA_PLUGIN_CHECKOUT_ROOT`
- `OCA_CACHE_DIR`

Tests and CI use isolated directories.

## Advance relationship

OpenCode Advance depends on Advance. Advance does not depend on OpenCode Advance.

OCA:

- clones/builds the Advance plugin
- calls Advance's own global sync script
- renders OpenCode/Vision/tmux environment config
- verifies Advance checkout, plugin build, Temporal state, and asset ownership

Advance:

- owns `/adv-*` commands
- owns ADV agents and skills
- owns workflow state and specs
- owns spec-driven development gates

## Release

Version tags (`v*`) trigger `.github/workflows/release.yml`, which runs GoReleaser and publishes cross-platform binaries plus `SHA256SUMS.txt`.

Local release build check:

```bash
make build VERSION=1.0.0-rc1
./bin/oca version
```

## Documentation

- `INSTALL.md` — install and first-run guide
- `CHANGELOG.md` — release history
- `docs/design/` — architecture, schema, theme, CLI surface
- `docs/proposals/` — historical implementation roadmap
- `AGENTS.md` — developer and agent reference
