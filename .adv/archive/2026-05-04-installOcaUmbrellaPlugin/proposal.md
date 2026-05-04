## Success Criteria

1. `package.json` `main` field changed from `"dist/index.js"` to `"src/index.ts"` — matches LBP pattern of other 4 local plugins, eliminates build dependency.
2. `opencode.json` plugin array contains 6 entries including OCA umbrella plugin absolute path.
3. After OpenCode restart, plugin loads cleanly (no error in session start).
4. Pane state: `~/.local/state/oca/panes/<socket>/<paneId>.json` created with valid `paneState` schema (`sessionID`, `directory`, `ts`) when opencode runs in a pane.
5. `oca pane restart-tui` reads state file and resumes via `opencode -s <session-id>` (not `--continue` fallback).
6. No regression in existing 5 plugins.
7. `stack.example.toml` updated to include OCA plugin entry.

## Out of Scope

- Migrating operator to stack.toml-driven config
- Schema bump for Pattern B (OCA #1)
- Watchdog activation defaults (OCA #2 hibernation)
- Publishing plugin to npm
- New OCA plugin features
- Removing `dist/` directory (leave in place, just not the active entrypoint)

## Constraints

- MUST NOT modify any of the 5 existing registered plugins
- MUST be reversible (revert package.json main + remove opencode.json entry)
- MUST handle JSONC comments in opencode.json correctly
- Install path MUST use absolute path to local checkout directory (matching existing pattern)

## Acceptance Criteria (operator-verifiable)

1. Change `plugins/oca/package.json` `main` from `"dist/index.js"` to `"src/index.ts"`
2. Add `/home/jrede/dev/opencodeadvance/plugins/oca` to `opencode.json` plugin array
3. Update `stack.example.toml` to include OCA plugin in `[plugins]` section
4. Post-state: plugin array has 6 entries, OCA plugin points to correct directory
5. Restart OpenCode, open session, verify plugin loads (no error)
6. Verify `~/.local/state/oca/panes/<socket>/<paneId>.json` created with valid JSON
7. `oca pane restart-tui` uses session-id resume (not `--continue`)
8. ADV workflow still works (no regression)

## Discovery Findings

### LBP Research (2026-05-04)

OpenCode plugin resolution chain (source: `packages/opencode/src/plugin/shared.ts`):
- `resolvePathPluginTarget()` — if directory + `package.json` found → returns `file://` URL to directory
- Server loading: uses `exports["./server"]` → falls back to `package.json` `main`
- TUI loading: looks for `index.{ts,tsx,js,mjs,cjs}` in directory root; never uses `main`
- Bun handles TS imports natively — no build step needed for local plugins

All 4 other local plugins use raw TS in `main`:
- ADV: `main: "src/index.ts"`
- claude-max: `main: "./src/index.ts"`
- morph-fast-apply: `main: "index.ts"`
- vision: `main: "src/index.ts"`

LBP: change OCA plugin `main` to `"src/index.ts"`, matching the ecosystem pattern and eliminating the build dependency.

### Verified State

- opencode.json: 5 plugins. OCA absent.
- `~/.local/state/oca/panes/` does not exist.
- No stack.toml exists.

### Edge Cases

- EC1: JSONC comment preservation — must use exact string edit, not JSON round-trip
- EC2: Existing sessions need restart to pick up new plugin
- EC3: Path must be absolute directory-level, matching other 4 entries
- EC4: `dist/` left in place but no longer active entrypoint — future `bun build` still works if needed

### Decisions

- DQ1 (agent-resolved, LBP): Change `main` to `"src/index.ts"` — matches all 4 other local plugins, eliminates build step
- DQ2 (user-resolved): Watchdog left OFF — deferred to hibernation change (OCA #2)
- DQ3 (user-resolved): Manual restart — document requirement, don't prompt
- DQ4 (user-resolved): Update stack.example.toml to include OCA plugin

### Related Changes

- `perPaneOpencodeTuiRestart` (archived) — built pane state reading in Go CLI
- `patternBSessionTopologyOne` (archived) — downstream consumer of pane state
- Future: `gracefulHibernation` — downstream consumer of watchdog