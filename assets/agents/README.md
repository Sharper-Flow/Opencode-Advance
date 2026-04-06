# Environment-level Agents

This directory holds the canonical agent definitions that OpenCode Advance owns — the non-ADV subset of the agent fleet.

When `oca apply` runs, these files are copied to `~/.config/opencode/agents/`.

## Inventory

| File         | Purpose                                                        | Owned by     |
| ------------ | -------------------------------------------------------------- | ------------ |
| `build.md`     | Build/CI agent — tests, linters, type checkers                | OCA          |
| `explore.md`   | Codebase navigation agent — find usages, file structure       | OCA          |
| `librarian.md` | Documentation agent — Context7, grep.app, Kagi lookups        | OCA          |
| `general.md`   | General-purpose multi-step implementation agent               | OCA          |
| `mechanic.md`  | System/infrastructure agent — MCP, env, toolchain issues      | OCA          |

## Not in this directory

The following agents are owned by the Advance plugin and synced via `advance/scripts/sync-global.sh`. They MUST NOT be duplicated here:

| File               | Owned by |
| ------------------ | -------- |
| `plan.md`            | Advance  |
| `scout.md`           | Advance  |
| `refine.md`          | Advance  |
| `orca.md`            | Advance  |
| `adv-researcher.md`  | Advance  |
| `tron.md`            | Advance  |

If a file appears in both this directory and Advance's `.opencode/agents/`, the build pipeline will fail the duplicate-owner check in Phase 2.

## Status

Empty — populated in Phase 0 or Phase 5 (whichever brings the files over from the current open-chad bundled agents, with appropriate tool allowlists for the OCA environment).
