# Environment-level Instructions

This directory holds the canonical instruction files that OpenCode Advance owns — the non-ADV subset of instruction content.

When `oca apply` runs, these files are copied to `~/.config/opencode/instructions/`.

## Inventory

| File                           | Purpose                                                 |
| ------------------------------ | ------------------------------------------------------- |
| `identity.md`                    | OpenCode identity + configuration paths reference      |
| `rules.yaml`                     | Priority-ranked agent behavior rules (P01 through P26)  |
| `shell_strategy.md`              | Non-interactive shell command policy                   |
| `test_resource_guardrails.md`    | Test execution concurrency and resource limits         |
| `lbp.md`                         | Long-term best practice stance                         |
| `temp_directory.md`              | Dedicated cache/temp directory policy                  |
| `mcp-tools.md`                   | MCP tool selection guide                               |
| `lgrep-tools.md`                 | lgrep code exploration policy                          |
| `morph-tools.md`                 | morph_edit vs edit/write policy                        |
| `worktree-guide.md`              | Git worktree usage guide                               |

## Not in this directory

The following instruction is owned by the Advance plugin. It is referenced in stack.toml via `[plugins.advance].instructions` and auto-appended to the instructions list:

| File                  | Owned by |
| --------------------- | -------- |
| `ADV_INSTRUCTIONS.md`   | Advance  |

## Status

Populated. Source files are the canonical OCA instruction definitions synced from the current live config.
