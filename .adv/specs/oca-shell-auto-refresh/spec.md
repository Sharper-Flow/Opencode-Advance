# oca-shell-auto-refresh

OCA maintains shell-visible environment state through an OCA-owned env file and stamp.

## Requirements

- `rq-shellAutoRefresh01` — Mutating OCA flows MUST write `~/.config/oca/env.sh` atomically before touching `$OCA_CACHE_DIR/env.stamp`.
- `rq-shellAutoRefresh02` — The managed shell block MUST source the OCA-owned env file and MUST NOT re-source user-owned rc files.
- `rq-shellAutoRefresh03` — Non-interactive shells MUST produce no auto-refresh output or network activity.
- `rq-shellAutoRefresh04` — Prompt-time steady state MUST check the `env.stamp` freshness marker before sourcing env state.
- `rq-shellAutoRefresh05` — `auto_refresh_notice` MUST support `off`, `once`, and `every`, defaulting to `off`.
- `rq-shellAutoRefresh06` — Development and tests MUST honor `OCA_OPENCODE_CONFIG_DIR` plus `OCA_CACHE_DIR` so shell state lands outside production config.
