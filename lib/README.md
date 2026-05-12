# lib/ — Shell-only integration helpers

This directory holds bash scripts that handle integration with tmux, the shell, and the terminal — work that is genuinely cleaner in bash than in Go.

## Inventory

| File                       | Purpose                                                        | Phase |
| -------------------------- | -------------------------------------------------------------- | ----- |
| `palette.sh`                 | Color constants (OCA_COLOR_OBSIDIAN, etc.) as env vars       | 0     |
| `wordmark.sh`                | Wordmark rendering functions (full / compact / short)        | 0     |
| `boot_splash.sh`             | Boot splash with wordmark reveal + version line              | 0 / 4 |
| `session_lifecycle.sh`       | tmux session creation / list primitives (shell counterpart to `internal/session/`) | 4     |
| `status_bar.sh`              | Tmux status bar renderer (row 0: session/git/ADV state/host/clock; row 1: window list/LLM gauges/date) | 4     |
| `obsidian.tmux.conf`         | *(moved to `assets/themes/` — referenced by `resolveTmuxConf()` at runtime)* | 4     |

## Why bash here?

- Tmux scripting is native to bash and awkward from Go
- Status bar rendering is triggered by tmux at 30-second intervals; shell is faster to spawn
- Terminal color output and boot splash timing are easier with bash's direct `\e[...]` control
- Maintainability: the set of tmux primitives is small and stable

## Why NOT bash for the rest?

The Go CLI (`cmd/oca/`) handles everything else — parsing, validation, rendering, plugin management, health checks, migration — because those benefit significantly from a real type system and stdlib libraries for TOML/JSON/HTTP.

## Status

Populated through Phases 0 and 4. Current files: `palette.sh`, `wordmark.sh`, `boot_splash.sh`, `session_lifecycle.sh`, `status_bar.sh`, `llm_gauge.sh`. ADV status data comes from `oca adv-status`.
