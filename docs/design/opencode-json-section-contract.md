# opencode.json Section Ownership Contract

**Status:** Active
**Last updated:** 2026-05-09
**Owners:** OCA (`oca apply`), Advance (`sync-global.sh`)

---

## Principle

Both OCA and Advance write to `~/.config/opencode/opencode.json`. This is **required**: Advance must work standalone (without OCA), so it wires itself into `opencode.json`. OCA's value is the declarative source of truth (`stack.toml`) and environment-level orchestration.

Conflict is avoided through **section discipline**: each top-level key has exactly one primary writer. Merge semantics preserve unknown keys, so both writers coexist without stomping.

---

## Section Ownership Matrix

| Section | Primary Writer | Secondary Writer | Merge Strategy | Notes |
|---------|---------------|-----------------|----------------|-------|
| `.mcp` | **OCA** | — | `MergeObject` (key-level) | OCA renders declared servers + slot groups. Advance never writes MCP entries. |
| `.plugin` | **OCA** | Advance | `MergeArray` (entry-level) | OCA renders declared plugins. Advance adds itself via `sync-global.sh`. Both entries preserved. |
| `.instructions` | **OCA** | Advance | `MergeArray` (entry-level) | OCA renders base order + declared. Advance appends `ADV_INSTRUCTIONS.md` (suppressed when plugin provides adv-instructions). |
| `.provider` | **OCA** | — | `MergeObject` (key-level) | OCA translates `stack.toml` providers with default→`"*"` expansion. |
| `.permission` | **OCA** | — | `MergeObject` (key-level) | OCA translates `stack.toml` permissions. |
| `.watcher` | **OCA** | — | `MergeWatcherIgnore` | Only `.watcher.ignore` managed; other watcher keys preserved. |
| `.lsp` | **OCA** | — | `MergeObject` (key-level) | OCA translates `stack.toml` LSP entries. |
| `.formatter` | **OCA** | — | `MergeObject` (key-level) | OCA translates `stack.toml` formatters. |
| `.command` | **OCA** | — | `MergeObject` (key-level) | OCA translates `stack.toml` commands. |
| `.theme` | **OCA** | — | `MergeObject` (top-level key) | OCA renders theme name. |
| `.agent` | **Advance** | — | Direct filesystem write | OCA does not manage agent files. `sync-global.sh` writes ADV agents to `~/.config/opencode/agents/`. |
| `.skills` | **OCA** | Advance | Filesystem copy (not JSON) | OCA copies `assets/skills/` → `~/.config/opencode/skills/`. Advance syncs ADV skills via `sync-global.sh`. Same coexistence pattern as `.plugin`. |
| `.mcp_servers` | — | — | — | Legacy key. Neither writer produces it. Preserved if present. |

---

## Merge Semantics

All JSON merges use OCA's `internal/render/merge.go` functions:

| Function | Used By | Behavior |
|----------|---------|----------|
| `MergeObject` | `.mcp`, `.provider`, `.permission`, `.lsp`, `.formatter`, `.command`, `.theme` | Declared keys overwrite existing; unknown keys preserved |
| `MergeArray` | `.plugin`, `.instructions` | Declared entries prepended; user-added entries preserved; stale worktree paths pruned |
| `MergeWatcherIgnore` | `.watcher.ignore` | Pattern-list merge |

Advance's `sync-global.sh` operates at the **filesystem level** (copying agent/command/skill files) and does not parse or merge `opencode.json` directly. It appends entries to `.instructions` and `.plugin` arrays via its own mechanism.

---

## Write Ordering

1. `oca apply` runs first — renders all declared sections from `stack.toml`
2. `sync-global.sh` runs after — adds Advance-specific entries without removing OCA entries
3. Result: both writers' entries coexist in the merged file

---

## Discipline Rules

1. **OCA must never delete entries it didn't create.** Merge semantics preserve unknown keys.
2. **Advance must never delete entries it didn't create.** `sync-global.sh` appends or replaces only its own assets.
3. **Neither writer owns the full file.** `opencode.json` is a shared document with section-level ownership.
4. **New sections require coordination.** Before adding a new managed section, the proposing writer documents it here and verifies the other writer's merge logic handles unknown keys gracefully.
5. **Agent/skill/command files are filesystem-owned**, not JSON-owned. Both writers copy to `~/.config/opencode/{agents,skills,commands}/` with non-overlapping file sets.

---

## Cross-Links

- OCA merge logic: `internal/render/plan.go` (`composeTargetOps`)
- OCA merge functions: `internal/render/merge.go`
- Advance sync: `advance/scripts/sync-global.sh`
- Boundary audit: `docs/proposals/2026-05-08-cross-repo-boundary-audit.md` § O3
