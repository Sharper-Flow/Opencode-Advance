## Problem Statement

### Current state

The OCA umbrella plugin (`plugins/oca/`, package `@sharperflow/oca-plugin`) provides session-pane state tracking and the watchdog primitive that downstream changes (Pattern B, hibernation) build on. Plugin source exists at `~/dev/opencodeadvance/plugins/oca/` but is NOT registered in the operator's `opencode.json` plugin array.

Verified empirical state (2026-05-04):
- `opencode.json` plugin array has 5 entries: ADV, claude-max, morph-fast-apply, vision, codex-auth. OCA plugin is absent.
- `~/.local/state/oca/panes/` does not exist → no pane state being written.
- `plugins/oca/dist/index.js` exists but is stale (Apr 30) vs source (May 4) → needs rebuild.
- No `stack.toml` exists → operator runs hand-edited config.

### Impact

- OCA #1 (Pattern B): pane-state schema bump in `cmd/oca/pane.go` depends on plugin writing state. Without it, `oca pane restart-tui` falls back to `opencode --continue`.
- OCA #2 (hibernation): keep-alive opt-out persistence depends on per-pane state. No plugin = no state to read/write.

Both changes implicitly assume the plugin is loaded. This prerequisite makes it explicit.

### Resolution

Option A (recommended): manual opencode.json edit adding the absolute path to `plugins/oca/`, matching the pattern of the other 5 local-checkout plugins. Stack.toml-driven config is out of scope (deferred to migration story).