# lib/ — Shell-only integration helpers

This directory holds bash scripts that handle integration with tmux, the shell, and the terminal — work that is genuinely cleaner in bash than in Go.

## Inventory (target state at v1.0)

| File                       | Purpose                                                        | Phase |
| -------------------------- | -------------------------------------------------------------- | ----- |
| `palette.sh`                 | Color constants (OCA_COLOR_OBSIDIAN, etc.) as env vars       | 0     |
| `wordmark.sh`                | Wordmark rendering functions (full / compact / short)        | 0     |
| `boot_splash.sh`             | Boot splash animation (wordmark reveal with indigo pulse)    | 0 / 4 |
| `status_bar.sh`              | Tmux status bar renderer (replaces status_left/right)        | 4     |
| `session_lifecycle.sh`       | tmux session creation / teardown / reaper hooks             | 4     |
| `obsidian.tmux.conf`         | Live tmux theme file (referenced by templates)               | 4     |
| `discord/setup.sh`           | Discord Rich Presence wizard (enable/disable/status)         | 7     |
| `discord/update.sh`          | Rate-limited Discord presence updater                        | 7     |
| `discord/taglines.toml`      | Data-driven tagline pool (new, non-"chad" era)             | 7     |

## Why bash here?

- Tmux scripting is native to bash and awkward from Go
- Status bar rendering is triggered by tmux at 30-second intervals; shell is faster to spawn
- Terminal color output and boot splash timing are easier with bash's direct `\e[...]` control
- Maintainability: the set of tmux primitives is small and stable

## Why NOT bash for the rest?

The Go CLI (`cmd/oca/`) handles everything else — parsing, validation, rendering, plugin management, health checks, migration — because those benefit significantly from a real type system and stdlib libraries for TOML/JSON/HTTP.

## Status

Empty — populated in Phases 0 and 4.
