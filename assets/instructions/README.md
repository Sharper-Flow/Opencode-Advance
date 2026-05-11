# Environment-level Instructions

This directory holds the canonical instruction files that OpenCode Advance owns — the non-ADV subset of instruction content.

When `oca apply` runs, these files are copied to `~/.config/opencode/instructions/`.

## Inventory

The list between the `INVENTORY` markers is consumed by the parity test in
`internal/render/instructions_test.go::TestInstructionAssetsReadmeParity`,
which fails if README rows drift from the actual contents of this directory.
Edit the markers and rows together; do not move the markers.

<!-- INVENTORY:START -->
| File                           | Purpose                                                 |
| ------------------------------ | ------------------------------------------------------- |
| `caveman.md`                     | Caveman compressed-communication mode                  |
| `global-verify-policy.md`        | `/check`-style verification gate before completion     |
| `identity.md`                    | OpenCode identity + configuration paths reference      |
| `lbp.md`                         | Long-term best practice stance                         |
| `lgrep-tools.md`                 | lgrep code exploration policy                          |
| `mcp-tools.md`                   | MCP tool selection guide                               |
| `morph-tools.md`                 | morph_edit vs edit/write policy                        |
| `rules.yaml`                     | Priority-ranked agent behavior rules (P01 through P26)  |
| `shell_strategy.md`              | Non-interactive shell command policy                   |
| `temp_directory.md`              | Dedicated cache/temp directory policy                  |
| `test_resource_guardrails.md`    | Test execution concurrency and resource limits         |
| `worktree-guide.md`              | Git worktree usage guide                               |
<!-- INVENTORY:END -->

## Not in this directory

The following instruction is owned by the Advance plugin. It is referenced in stack.toml via `[plugins.advance].instructions` and auto-appended to the instructions list:

| File                  | Owned by |
| --------------------- | -------- |
| `ADV_INSTRUCTIONS.md`   | Advance  |

## Status

Populated. Source files are the canonical OCA instruction definitions synced from the current live config.
