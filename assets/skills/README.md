# Skills

This directory holds the skill definitions that OpenCode Advance owns — reusable methodology skills that are not part of Advance itself.

When `oca apply --target skills` runs, these skills are copied to `~/.config/opencode/skills/`.

## Inventory

| Skill               | Purpose                                                       |
| ------------------- | ------------------------------------------------------------- |
| `lgrep/`              | lgrep dual-engine code intelligence (semantic + symbols)      |
| `morph/`              | morph_edit usage guidance                                     |
| `prioritizer/`        | Context-aware tradeoff analysis                              |
| `worktree/`           | Git worktree workflow guidance                               |
| `mcp-selection/`      | MCP tool selection decision matrix                           |
| `caveman/`            | Ultra-compressed communication mode                          |
| `caveman-commit/`     | Compressed commit message generator                          |
| `caveman-review/`     | Compressed code review comments                              |

## Not in this directory

The following skills are owned by the Advance plugin and synced via `advance/scripts/sync-global.sh`:

| Skill                          | Owned by |
| ------------------------------ | -------- |
| `adv-tron/`                      | Advance  |
| `adv-review-methodology/`        | Advance  |
| `adv-harden-methodology/`        | Advance  |
| `adv-slop-detection/`            | Advance  |
| `adv-discover-methodology/`      | Advance  |
| `adv-prep-methodology/`          | Advance  |
| `adv-apply-methodology/`         | Advance  |

OCA must never copy or overwrite skills from the Advance-owned list. The `[plugins.advance].provides = ["adv-skills"]` declaration causes `oca apply` to skip the `adv-*/` directories automatically.

## Status

Populated in Phase 3.5. Source files are the canonical OCA skill definitions already in use today at `~/.config/opencode/skills/`.
