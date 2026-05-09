# Cross-Repo Boundary Audit: Advance vs OCA

**Date:** 2026-05-08
**Status:** Draft
**Repos:** Sharper-Flow/Opencode-Advance (OCA) + Sharper-Flow/Advance (ADV)

---

## Overview

OCA and Advance share two filesystem trees and one database. This document catalogs every overlap, collision risk, and misplaced ownership, then proposes refactoring into a clean boundary.

**Principle:** OCA owns the environment layer (tmux, config rendering, session lifecycle, health checks, diagnostics). Advance owns the workflow layer (changes, gates, tasks, specs, worktree state). When both need the same data, OCA is the infrastructure provider and Advance is the consumer.

---

## Overlap Map

### O1. TMUX Operations (HIGH — ~2,073 LOC misplaced in Advance)

**Current state:** Advance contains a full tmux infrastructure layer.

| File (Advance) | LOC | What it does |
|---|---|---|
| `plugin/src/tools/worktree/terminal.ts` | 1,115 | `openTmuxWindow()`, cross-platform terminal spawning, mutex, `detectTerminalType()` |
| `plugin/src/events/terminal.ts` | 924 | `rename-window`, TTY detection, tab title construction, terminal alerts |
| `plugin/src/utils/terminal-detect.ts` | 34 | `isTmux()` check |

**Collision risk:**
- Advance uses default tmux socket; OCA uses `-L oca` named socket → windows created by Advance are invisible to OCA session management
- Advance's `openTmuxWindow` creates windows on whatever session is active → ignores OCA's Pattern B per-project session topology
- Both implement mutex serialization independently

**OCA already has:**
- `internal/session/session.go` (757 LOC) — full session lifecycle
- `cmd/oca/session.go` — `oca session ensure-window --session X --name Y --dir Z`
- `Manager.EnsureWindow()`, `Manager.ListWindows()`, `Manager.killWindow()`

**Proposal:**
- OCA adds `oca session rename-window --target X --title Y`
- Advance replaces `openTmuxWindow()` → calls `oca session ensure-window`
- Advance replaces `rename-window` → calls `oca session rename-window`
- Advance deletes ~1,500 LOC of tmux infrastructure, keeps ~500 LOC of title construction logic (`buildTabTitle`, `generateProjectShortname`, `updateTerminalStatus` intent)
- Full plan in this document's § TMUX Refactor below

---

### O2. ADV State File Reads (HIGH — fragile coupling to internal format)

**Current state:** OCA reads ADV's internal state files directly from the filesystem.

| OCA file | What it reads | ADV source |
|---|---|---|
| `internal/advruntime/workspace_projection.go` | `$XDG_DATA_HOME/opencode/plugins/advance/{pid}/snapshot.json` | ADV Temporal ticker writes snapshot |
| `lib/adv_status.sh` (15 functions, 442 LOC) | `{pid}/changes/{id}/change.json`, `snapshot.json`, `temporal.env` | ADV Temporal workflows |
| `internal/advruntime/session_debt.go` (388 LOC) | `$XDG_DATA_HOME/opencode/opencode.db` (SQLite) | OpenCode core |

**Collision risk:**
- `snapshot.json` format is ADV-internal. If ADV changes schema (signal cutover), OCA breaks silently (returns empty).
- `change.json` path and structure is Temporal adapter persistence format. Not a public API.
- `adv_status.sh` reads 3 different ADV-internal file formats with `jq` from shell — no contract, no versioning.

**Advance also reads the same data:**
- `plugin/src/utils/opencode-session-debt.ts` (237 LOC) — same SQLite scan for blank assistant rows
- `plugin/src/storage/store-disk.ts` — reads/writes the same `$XDG_DATA_HOME/opencode/plugins/advance/` tree

**Proposal:**
- OCA should stop reading ADV internal state files directly
- Option A: ADV exposes a thin CLI or HTTP endpoint (`adv status --format json`) that OCA calls instead of parsing files
- Option B: OCA calls `oca doctor --scope adv-runtime --format json` (already exists) which uses `advruntime` package to read state through Go APIs
- `adv_status.sh` should be rewritten to call `oca adv-status` (new Go subcommand) instead of parsing ADV files with `jq`
- Session debt scan should be deduplicated — one authoritative scanner, the other calls it

---

### O3. OpenCode Config Tree Writes (MEDIUM — same target, different scopes)

**Current state:** Both repos write to `~/.config/opencode/`.

| OCA writes (`oca apply`) | Advance writes (`sync-global.sh --fix`) |
|---|---|
| `opencode.json` — MCP, plugins, providers, permissions, watcher, LSP, toggles, skills, commands, formatters | `opencode.json` — agent config (provider variants, prompts), plugin entry, instruction entries |
| `skills/{lgrep,morph,prioritizer,...}/` — OCA-owned skills | `skills/adv-*/` — ADV-owned skills |
| `instructions/{identity,rules,...}.md` — OCA-owned instructions | Overlay blocks injected into shared agent files |
| Rendered from `stack.toml` (declarative) | Rendered from `.opencode/` tree in plugin source |

**Collision risk:**
- Both write `opencode.json` — OCA merges specific sections (`.mcp`, `.plugin`, `.instructions`); Advance patches the same file for `.plugin` and agent config
- If OCA runs `oca apply` after `sync-global.sh --fix`, OCA's JSON merge preserves user-owned keys but could reorder/reformat
- If Advance runs `sync-global.sh --fix` after `oca apply`, it could overwrite agent config that OCA also manages

**Current mitigation:**
- OCA uses section-scoped JSON merge (only touches `.mcp`, `.plugin`, `.instructions`, declared keys)
- Advance's `sync-global.sh` uses `jq` patches that only touch ADV-specific keys
- Both preserve unknown keys
- OCA's `adv-assets` doctor scope detects ownership drift

**Proposal:**
- The current separation is **mostly correct** — OCA owns config structure, Advance owns agent/command/skill content
- Remaining risk: both write `.plugin` array entries. OCA adds the plugin path; Advance verifies it exists. This is currently non-conflicting but fragile.
- Long-term: OCA should be the sole writer of `opencode.json`. Advance should write agent/command/skill files only and delegate JSON config patches to OCA. This is tracked as M2 (apply lifecycle parity) and M3 (plugin install) in the OCA reliability queue.

---

### O4. Session Debt Scan (MEDIUM — 625 LOC duplicated)

**Current state:** Both repos independently scan the same OpenCode SQLite database for stale blank assistant messages.

| Repo | File | LOC | What it does |
|---|---|---|---|
| OCA | `internal/advruntime/session_debt.go` | 388 | SQL query, classification, threshold logic |
| ADV | `plugin/src/utils/opencode-session-debt.ts` | 237 | Same SQL query, same classification, same thresholds |

**Collision risk:** No write collision (both read-only). But the duplication means threshold changes must be made in two places.

**Proposal:**
- OCA owns the session debt scan (runs outside OpenCode agent context, in `oca doctor` and `adv_status.sh`)
- Advance should call OCA's scan or share the threshold constant
- Short-term: document the duplication and ensure thresholds match
- Long-term: Advance calls `oca doctor --scope adv-runtime --format json` for session debt data

---

### O5. Worktree Registry (LOW — correct read-only coupling)

**Current state:** OCA reads ADV's worktree registry (via `snapshot.json`) for session enrichment.

**Collision risk:** Minimal — OCA is read-only, ADV is authoritative.

**Proposal:** Keep as-is but formalize the contract. When signal cutover lands, `snapshot.json` may go away and OCA will need a different data source. Tracked in `docs/notes/2026-05-05-advance-signal-cutover-readiness.md`.

---

### O6. Search Attribute Knowledge (LOW — already aligned)

**Current state:** OCA's `internal/advruntime/search_attributes.go` maintains its own copy of ADV's search attribute list. Both must match.

**Current mitigation:** OCA was updated to match Advance's `ADV_SEARCH_ATTRIBUTES` constant in this session. `TestSearchAttribute*` tests cover it.

**Proposal:** Accept the duplication — it's 10 key-value pairs. Add a comment in OCA pointing to the Advance source. No structural change needed.

---

## Refactoring Priority

| Priority | Overlap | Effort | Impact |
|---|---|---|---|
| **1** | O1: TMUX boundary | 2-4 days (cross-repo) | Eliminates socket conflicts, session topology issues, ~1,500 LOC deletion |
| **2** | O2: State file reads → API boundary | 1-2 days (OCA) | Makes OCA resilient to ADV schema changes |
| **3** | O4: Session debt dedup | 0.5 day | Small but eliminates threshold drift |
| **4** | O3: Config tree writes | Tracked (M2/M3) | Already in OCA reliability queue |
| **5** | O5/O6 | No action | Accept current coupling |

---

## § TMUX Refactor (O1 — Detailed Plan)

### Phase 1: OCA adds CLI surface

| Task | Description |
|---|---|
| Add `oca session rename-window` | `--target <session:window> --title <title>`. Wraps `tmux -L oca rename-window`. |
| Add `oca terminal detect` | Returns terminal type. For ADV to query context. |
| Add `oca session current` | Returns current session name, window name, pane ID. |
| Add `oca adv-status` | Go subcommand replacing `adv_status.sh`. Reads ADV state through `advruntime` package, outputs status bar text or JSON. |

### Phase 2: Advance migrates worktree window creation

| Task | Description |
|---|---|
| Replace `openTerminal()` with CLI call | `openTmuxWindow` → `execFileSync("oca", ["session", "ensure-window", ...])` |
| Remove cross-platform terminal code | Delete `openMacOSTerminal`, `openLinuxTerminal`, `openWindowsTerminal`, `openWSLTerminal` |
| Remove `detectTerminalType()` | Replace with `process.env.TMUX` check |
| Remove `tmuxMutex` | OCA named socket handles serialization |

### Phase 3: Advance migrates tab title management

| Task | Description |
|---|---|
| Replace `setTitle()` tmux branch | Call `oca session rename-window` or emit event to OCA plugin |
| Keep `buildTabTitle()` logic | Title construction stays in ADV |
| Keep non-tmux fallback | `/dev/tty` and `stdout` write stays in ADV |
| Keep `updateTerminalStatus()` intent | Decision logic stays; execution delegated |

### Phase 4: Cleanup

| Task | Description |
|---|---|
| Delete ~1,500 LOC from Advance | `terminal.ts` shrinks from 1,115 → ~200 LOC |
| Shrink `events/terminal.ts` | Remove tmux-specific branches, keep TTY + title logic |
| Update `adv-worktree` skill | Document delegation to OCA |
| Update AGENTS.md | Record the tmux ownership boundary |

### Fallback

If `oca` binary is not available when Advance tries to open a window, fall back to direct `tmux new-window` call (keep thin 30-line fallback path).

---

## § State File Reads (O2 — Detailed Plan)

### Current `adv_status.sh` reads (all should move to Go)

```
changes/{id}/change.json  →  active change status
snapshot.json             →  workspace state, worktree registry
temporal.env              →  Temporal server address
```

### Proposal: `oca adv-status` subcommand

```bash
# Status bar text (replaces adv_status.sh output)
oca adv-status --format text

# Machine-readable (replaces direct file reads)
oca adv-status --format json

# Specific queries
oca adv-status --query active-change
oca adv-status --query worktrees
oca adv-status --query temporal-health
```

Implementation: thin CLI wrapper around `advruntime` package functions. Shell status bar calls `oca adv-status` instead of `jq` parsing.

### Session debt dedup (O4)

- OCA owns `internal/advruntime/session_debt.go` — authoritative scanner
- Advance imports threshold constant from shared location, or calls `oca doctor --scope adv-runtime --format json` for debt data
- Delete `plugin/src/utils/opencode-session-debt.ts` or reduce it to a thin wrapper that calls OCA

---

## Success Criteria

- [ ] Advance zero direct `tmux` command calls (all via OCA CLI)
- [ ] `adv_status.sh` replaced by `oca adv-status` Go subcommand
- [ ] OCA reads zero ADV internal state files directly from shell (`jq`)
- [ ] Go reads of `snapshot.json` documented as coupling contract with signal-cutover watch
- [ ] Session debt scanned by one authoritative implementation
- [ ] ~1,500 LOC deleted from Advance
- [ ] Both test suites pass

## Risks

| Risk | Mitigation |
|---|---|
| Process spawn latency for tmux ops | ~50-100ms, acceptable for infrequent operations |
| OCA not installed when ADV opens window | Thin fallback to direct `tmux` call |
| `snapshot.json` schema changes before API is built | Document coupling; `workspace_projection.go` already degrades gracefully |
| Cross-repo coordination timing | Each phase is independent per repo; can land asynchronously |
