# Agreement

## Objectives

1. Register the OCA umbrella plugin (`plugins/oca/`) in the operator's `opencode.json` so pane state tracking and watchdog primitives activate.
2. Align plugin entrypoint with LBP pattern (raw TS, no build step) matching all 4 other local plugins.
3. Update `stack.example.toml` so future `oca apply` / `oca migrate init` includes the OCA plugin.

## Acceptance Criteria

1. Change `plugins/oca/package.json` `main` from `"dist/index.js"` to `"src/index.ts"`
2. Add `/home/jrede/dev/opencodeadvance/plugins/oca` to `opencode.json` plugin array
3. Update `stack.example.toml` to include OCA plugin in `[plugins]` section
4. Plugin array has 6 entries after edit
5. After OpenCode restart, plugin loads cleanly
6. `~/.local/state/oca/panes/<socket>/<paneId>.json` created with valid JSON when opencode runs in a tmux pane
7. `oca pane restart-tui` uses session-id resume (not `--continue` fallback)
8. All 5 existing plugins still load; ADV workflow functional

## Constraints

- MUST NOT modify any of the 5 existing registered plugins
- MUST be reversible (revert package.json + remove opencode.json entry)
- MUST preserve JSONC comments in opencode.json
- MUST use absolute directory path (matching existing pattern)
- Watchdog MUST remain OFF (deferred to OCA #2)

## Avoidances

- Stack.toml-driven config migration (separate change)
- Publishing plugin to npm (premature)
- Schema bump for Pattern B (OCA #1 scope)
- Watchdog activation defaults (OCA #2 scope)
- Removing `dist/` directory (leave in place)

## Decisions

### User Decisions
- **Watchdog activation:** Leave OFF — deferred to hibernation change (OCA #2). User chose minimal scope.
- **Restart timing:** Manual restart, document requirement. Don't prompt for immediate restart.
- **stack.example.toml:** Update to include OCA plugin. User chose full reference coverage.

### Agent Decisions (LBP)
- **Plugin entrypoint:** Changed from `dist/index.js` to `src/index.ts` — matches all 4 other local plugins, eliminates build dependency. Verified via OpenCode source (`packages/opencode/src/plugin/shared.ts`): Bun resolves TS natively, server loading uses `package.json` `main` field.
- **Path format:** Directory-level absolute path `/home/jrede/dev/opencodeadvance/plugins/oca` — confirmed via `resolvePathPluginTarget()`: directory + `package.json` → returns `file://` URL to directory.

## Deferred Questions

None — all questions resolved.

## Sign-Off

Approved by user at AC checkpoint (Phase 4.5.1).