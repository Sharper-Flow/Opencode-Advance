# Skills

This directory holds the skill definitions that OpenCode Advance owns — reusable methodology skills that are not part of Advance itself.

When `oca apply` runs, these skills are copied to `~/.config/opencode/skills/`.

## Inventory

| Skill             | Purpose                                                       |
| ----------------- | ------------------------------------------------------------- |
| `lgrep/`            | lgrep dual-engine code intelligence (semantic + symbols)      |
| `morph/`            | morph_edit usage guidance                                     |
| `prioritizer/`      | Context-aware tradeoff analysis                              |
| `worktree/`         | Git worktree workflow guidance                               |
| `mcp-selection/`    | MCP tool selection decision matrix                           |

## Not in this directory

The following skills are owned by the Advance plugin and synced via `advance/scripts/sync-global.sh`:

| Skill                        | Owned by |
| ---------------------------- | -------- |
| `adv-tron/`                    | Advance  |
| `adv-review-methodology/`      | Advance  |
| `adv-harden-methodology/`      | Advance  |
| `adv-slop-detection/`          | Advance  |

## Status

Empty — populated in Phase 5 by copying from the current open-chad bundled skills (which already contain these exact files).
