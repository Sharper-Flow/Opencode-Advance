# Environment-level Agents

This directory documents the canonical agent definitions that OpenCode Advance owns — the non-ADV subset of the agent fleet.

When `oca apply` eventually manages agent assets, any agent markdown in this directory will be copied to `~/.config/opencode/agents/`. The directory is currently empty; the inventory below documents the intended OCA-owned subset.

## Inventory

| File         | Purpose                                                        | Owned by            |
| ------------ | -------------------------------------------------------------- | ------------------- |
| `build.md`     | Build/CI agent — tests, linters, type checkers                | OCA + ADV overlay   |
| `explore.md`   | Codebase navigation agent — find usages, file structure       | OCA                 |
| `librarian.md` | Documentation agent — Context7, grep.app, Kagi lookups        | OCA                 |
| `general.md`   | General-purpose multi-step implementation agent               | OCA + ADV overlay   |
| `mechanic.md`  | System/infrastructure agent — MCP, env, toolchain issues      | OCA                 |

## Not in this directory

The following agents are owned by the Advance plugin and synced via `advance/scripts/sync-global.sh`. They MUST NOT be duplicated here:

| File               | Owned by |
| ------------------ | -------- |
| `adv.md`             | Advance (synced global)  |
| `plan.md`            | Advance (synced global)  |
| `adv-researcher.md`  | Advance (bundled global) |
| `adv-engineer.md`    | Advance (bundled global) |
| `adv-tron.md`        | Advance (bundled global) |
| `adv-atc.md`         | Advance (bundled global) |

If a file appears in both this directory and Advance's `.opencode/agents/`, the build pipeline will fail the duplicate-owner check in Phase 2.

`build.md` remains OCA-owned here, but Advance patches its overlay block during sync.

## Status

Currently empty. Agent asset population is still deferred to a later implementation phase; this README documents the intended ownership boundary and duplicate-owner rules in the meantime.
