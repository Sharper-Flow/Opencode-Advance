# Templates

Go `text/template` files that `oca apply` renders into user config files.

Note: Phase 1–3 rendering is **programmatic** (no templates). Templates are used where content is user-visible text or config that benefits from a block-wrapping pattern.

## Inventory

| Template                     | Renders into                                     | Phase |
| ---------------------------- | ------------------------------------------------ | ----- |
| `tmux.conf.block.gotmpl`       | Managed block wrapping for tmux conf injection     | 4     |

Planned additions:

| `shell_profile.block.gotmpl`   | Block injected into `~/.zshrc` / `~/.bashrc`         | 6     |

## Status

Populated starting in Phase 4 foundation (`tmux.conf.block.gotmpl`). Phase 1–3 rendering is entirely programmatic via `internal/render/` — no Go templates are used for `opencode.json` or `vision/servers.yaml` generation.
