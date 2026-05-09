# Cross-Repo Boundary: Advance vs OCA

**Date:** 2026-05-08 (revised 2026-05-09)
**Status:** Draft
**Repos:** Sharper-Flow/Opencode-Advance (OCA) + Sharper-Flow/Advance (ADV)

---

## Principle

**Advance must work standalone. OCA enhances.**

Advance is a complete, self-sufficient spec-driven workflow plugin. A user can install Advance into a stock OpenCode environment and have a fully functional system. OCA is the deluxe environment layer that wraps an Advance install with quality-of-life features: tmux integration, multi-project session orchestration, unified status surfaces, remote access (future), and config-as-code via `stack.toml`.

This matches the existing dependency direction:

> OpenCode Advance depends on Advance; Advance does not depend on OpenCode Advance.

— `AGENTS.md`

## Litmus Test

Two-step test. Default bias is toward Advance:

1. **Capability test**: If OCA did not exist, *could* Advance implement this within its scope as an OpenCode plugin?
   - Yes → **Advance owns it**
   - No → see test 2

2. **Plugin-feasibility test**: Is this something that *can't* be done with just an OpenCode plugin? (Requires external orchestration, runs before plugins load, spans sessions, integrates with the host environment, exposes a non-plugin surface.)
   - Yes → **OCA owns it**

Stated as one rule: **Advance owns everything achievable as a plugin. OCA owns only what plugins fundamentally can't do.**

### Examples

| Capability | Test 1 (could Advance do it as a plugin?) | Owner |
|------------|------------------------------------------|-------|
| Git mutation guard | Yes — plugin tool/hook intercepts bash | **Advance** |
| Session debt detection | Yes — plugin can query SQLite | **Advance** |
| ADV instruction footprint | Yes — plugin owns its own assets | **Advance** |
| Cross-project ADV CLI helper (`opencode-adv.sh`) | Yes — script ships with the plugin | **Advance** |
| TMUX session/window management | No — plugin can spawn tmux but can't own a multi-session topology, named socket, or persistence across OpenCode restarts | **OCA** |
| `stack.toml` → `opencode.json` rendering | No — `opencode.json` is read before plugins load | **OCA** |
| Status bar / dashboard surfaces | No — plugin can expose data but can't own a tmux status bar, HTML dashboard, or remote endpoint | **OCA** |
| Remote access (future) | No — plugin runs inside one OpenCode session; remote requires a daemon outside any session | **OCA** |
| Multi-project session orchestration | No — plugin runs in one session at a time | **OCA** |
| Surfacing Advance signals (worker health, session debt) in user-visible places | Plugin produces the signal; surfacing in tmux/doctor/dashboard is outside the plugin | **Advance produces, OCA surfaces** |

---

## Ownership Map

| Domain | Advance | OCA |
|--------|---------|-----|
| **Workflow** | Changes, gates, tasks, specs, validators, review, hardening | — |
| **Temporal** | Workflows, activities, search attributes, state machines, worker | Dev server supervision (`oca temporal start/stop`) |
| **Agents** | All ADV agents (adv, plan, build, adv-engineer, etc.), instruction content | Environment agents (build, explore, librarian, mechanic) |
| **Tools** | All `adv_*` MCP tools, `/adv-*` slash commands | `oca doctor`, `oca apply`, `oca session`, `oca adv-status` |
| **Plugins** | Plugin runtime (its own tools/hooks/events) | Plugin install + wiring into `opencode.json` |
| **Git** | Mutation guard policy + enforcement (its own workflows need it) | Worktree lifecycle helpers (`oca session` integrates) |
| **State** | Authoritative state for changes/tasks/gates/specs/worktrees | Reads ADV state for surfacing only |
| **Tmux** | Zero tmux code | Full tmux ownership (sessions, windows, status bar) |
| **Diagnostics** | `adv_status`, `adv_temporal_diagnose` (its own diagnostics) | `oca doctor` (extends ADV signals + adds environment checks) |
| **Config** | Writes its own assets (agents, commands, skills, instructions) into `~/.config/opencode/` | Renders `opencode.json` from `stack.toml`; manages MCP, providers, permissions |
| **Surfaces** | MCP tools, slash commands, file artifacts | Status bar, doctor, future dashboard, remote access |

---

## Overlap Re-classification

Five overlaps were catalogued in the original audit. Under the cleaner principle, the recommendations shift:

### O1. TMUX Operations — OCA owns end to end

**Current:** Advance has ~2,073 LOC of tmux infrastructure.

**Under principle:** Advance never needed tmux. OCA owns it.

**Action:** Advance deletes its tmux code and calls OCA CLI instead (`oca session ensure-window`, `oca session rename-window`). Tracked: OCA #23.

**Fallback:** When `oca` is not on PATH, Advance falls back to a thin direct `tmux new-window` (~30 LOC). Optional behavior, not required for Advance to function in non-tmux environments.

---

### O2. ADV State File Reads — Advance exposes a stable surface

**Current:** OCA's `adv_status.sh` (442 LOC shell + `jq`) parses Advance's internal state files (`change.json`, `snapshot.json`, `temporal.env`).

**Under principle:** Advance owns its state format. OCA must not couple to Advance internals; it reads through a stable surface.

**Action:**

1. Advance exposes a stable read surface — either:
   - CLI: `adv status --format json` (recommended; works from any context)
   - Or a documented JSON projection at a known path
2. OCA replaces `adv_status.sh` with `oca adv-status` Go subcommand that calls Advance's stable surface. Tracked: OCA #24.
3. The shell parser of internals is deleted entirely.

The `internal/advruntime/workspace_projection.go` Go reader of `snapshot.json` is similarly upgraded to call the stable surface or accept that it consumes a documented contract.

---

### O3. opencode.json writes — Section discipline, document the contract

**Current:** Both write `opencode.json` to different sections.

**Under principle:** Advance must be installable without OCA, so Advance must wire itself into `opencode.json` (declare plugin path, instructions, agents). OCA's value-add is the *declarative source of truth* (`stack.toml`). Both writers are correct.

**Action:** Document the section contract:

| Section | Owner |
|---------|-------|
| `.mcp` | OCA (renders from `stack.toml`) |
| `.plugin` | Both: OCA renders OCA-declared plugins; Advance adds itself |
| `.instructions` | Both: OCA renders OCA-declared instructions; Advance adds its own |
| `.agent` (provider variants, ADV agent prompts) | Advance |
| `.permission`, `.lsp`, `.formatter`, `.theme`, etc. | OCA (renders from `stack.toml`) |
| Unknown keys | Both preserve |

Tracked: OCA #26 retitled to "Document opencode.json section ownership contract." No code consolidation needed; the duplication is required by the standalone-Advance principle.

**Contract documented:** `docs/design/opencode-json-section-contract.md` — full section ownership matrix, merge semantics, write ordering, and discipline rules.

---

### O4. Session Debt Scan — Advance owns; OCA consumes

**Current:** Both repos independently scan OpenCode SQLite. OCA: 388 LOC. Advance: 237 LOC.

**Under principle:** Advance needs session debt detection for its own diagnostics (`adv_status`, agent context). OCA needs the *signal* for surfacing in `oca doctor` and `oca adv-status`, but should not duplicate the scan.

**Action (reverses original recommendation):**

- Advance keeps `plugin/src/utils/opencode-session-debt.ts` (its diagnostic logic)
- OCA deletes `internal/advruntime/session_debt.go`
- OCA reads the signal from Advance via `adv status --format json` (the same stable surface from O2)
- Threshold logic lives once, in Advance
- Tracked: OCA #25 retitled to "Read session debt from Advance instead of duplicating scan"

---

### O5. Worktree Registry — already clean

Advance owns worktree state. OCA reads `snapshot.json` for session enrichment.

**Under principle:** Already correct. Formalize via the same stable read surface (O2) when convenient. No urgency.

---

### O6. Search Attribute Knowledge — already clean

OCA's `internal/advruntime/search_attributes.go` mirrors Advance's `ADV_SEARCH_ATTRIBUTES` for dashboard rendering.

**Under principle:** Advance owns the canonical list. OCA's mirror is enhancement (powering the dashboard); duplication is acceptable for a 10-entry static list. Documented coupling.

**Action:** Add a comment in OCA pointing to the Advance source. No structural change.

---

## Refactor Priority

| Priority | Item | Effort | Impact |
|---------|------|--------|--------|
| 1 | O1: TMUX boundary (OCA #23) | 2-4 days cross-repo | ~1,500 LOC deletion in Advance, eliminates socket conflicts |
| 2 | O2: Stable Advance read surface (OCA #24 + new ADV issue) | 1-2 days each repo | Decouples OCA from Advance internals |
| 3 | O4: OCA stops scanning session debt (OCA #25 reversed) | 0.5 day | Removes 388 LOC duplicate |
| 4 | O3: Document section contract (OCA #26 reframed) | 0.5 day | Pure documentation |
| 5 | O5/O6 | None | Accept |

---

## Implications for Open Issues

### Issues that were OCA candidates but stay in Advance

Under the standalone-Advance principle, these stay in Advance because Advance needs them to function:

| # | Title | Why Advance |
|---|-------|-------------|
| ADV #102 | Git mutation guard blocks archive push | Guard is Advance's own workflow tooling |
| ADV #92 | Session debt framing in `adv_status` | Advance's own diagnostic text |
| ADV #91 | Classify blank rows orphan-vs-live | Advance's own session-debt logic |
| ADV #85 | Programmatic git mutation guard | Advance's own guard |
| ADV #72 | Scope ADV instruction load | Advance must control its own instruction footprint |
| ADV #71 | `opencode-adv.sh` CLI helper | Advance's own helper for cross-project ADV calls |

### Issues that move to OCA

| Source | Target | Why OCA |
|--------|--------|---------|
| ADV #74 | OCA #23 | Tmux integration is pure environment enhancement |
| ADV #70 | OCA #31 | Surfacing worker exhaustion across user-visible paths is OCA |

### Borderline candidates (defer)

| # | Title | Note |
|---|-------|------|
| ADV #96 | Cross-project session view | Multi-project orchestration leans OCA, but `adv_session_list` is also useful inside Advance. Re-evaluate after the stable read surface (O2) lands. |

---

## TMUX Refactor Plan (O1)

(unchanged from original audit)

### Phase 1: OCA adds CLI surface

- `oca session rename-window --target <session:window> --title <title>`
- `oca terminal detect`
- `oca session current`
- `oca adv-status` (covers O2 too)

### Phase 2: Advance migrates window creation

- Replace `openTerminal()` → `execFileSync("oca", ["session", "ensure-window", ...])`
- Delete `openMacOSTerminal`, `openLinuxTerminal`, `openWindowsTerminal`, `openWSLTerminal`
- Replace `detectTerminalType()` with `process.env.TMUX` check
- Remove `tmuxMutex` (OCA named socket handles serialization)

### Phase 3: Advance migrates tab title management

- Replace `setTitle()` tmux branch → `oca session rename-window`
- Keep `buildTabTitle()` logic in Advance
- Keep non-tmux fallback (`/dev/tty`, stdout)
- Keep `updateTerminalStatus()` decision logic; delegate execution

### Phase 4: Cleanup

- Delete ~1,500 LOC from Advance
- Shrink `events/terminal.ts` (remove tmux branches, keep TTY + title logic)
- Update `adv-worktree` skill to document delegation
- Update `AGENTS.md` to record the boundary

### Fallback

`oca` not on PATH → thin direct `tmux new-window` call (~30 LOC). Optional; Advance must function without tmux.

---

## Stable Read Surface (O2)

Advance to add (new ADV issue):

```bash
# Status bar text
adv status --format text

# Machine-readable
adv status --format json

# Specific queries
adv status --query active-change
adv status --query worktrees
adv status --query temporal-health
adv status --query session-debt
```

OCA's `oca adv-status` (#24) and `oca doctor` shell out to this. The shell parser of `change.json`/`snapshot.json` is retired.

---

## Success Criteria

- [ ] Advance has zero direct `tmux` command calls (all via OCA CLI; thin fallback acceptable)
- [ ] OCA has zero direct reads of Advance internal state files (`change.json`, `snapshot.json`, etc.)
- [ ] Advance exposes a stable read surface (`adv status --format json` or equivalent)
- [ ] Session debt scan exists in exactly one place (Advance)
- [ ] `opencode.json` section ownership is documented
- [ ] ~1,500 LOC deleted from Advance
- [ ] Both test suites pass
- [ ] Advance functions standalone (no `oca` binary required) — verified by CI

## Risks

| Risk | Mitigation |
|------|------------|
| Process spawn latency for tmux ops | ~50-100ms, acceptable for infrequent operations |
| `oca` not installed when Advance runs | Thin fallback to direct `tmux` call; Advance must work without OCA |
| Advance state schema changes break OCA | Stable read surface insulates OCA |
| Cross-repo coordination timing | Each phase is independent per repo; lands asynchronously |
