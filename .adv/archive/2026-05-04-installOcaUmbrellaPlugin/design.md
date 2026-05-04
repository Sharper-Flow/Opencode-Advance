# Design: installOcaUmbrellaPlugin

## Implementation Strategy

3 file edits, no new code. No architecture decisions — everything is constrained by agreement and LBP research.

### Edit 1: `plugins/oca/package.json`

Change `"main": "dist/index.js"` → `"main": "src/index.ts"`

**Why:** Matches LBP pattern of all 4 other local plugins. Bun resolves TS natively. Eliminates build dependency. Verified via OpenCode source (`packages/opencode/src/plugin/shared.ts`): `resolvePathPluginTarget()` finds directory + `package.json` → `resolvePluginEntrypoint()` reads `main` field → server loading uses it directly.

**Risk:** `dist/` becomes unused but harmless. Future `bun build` still works if needed.

### Edit 2: `~/.config/opencode/opencode.json`

Add entry to `plugin` array:
```json
"/home/jrede/dev/opencodeadvance/plugins/oca"
```

**Placement:** After the 5th entry, before closing bracket. Exact string edit to preserve JSONC comments.

**Why:** Matches pattern of other 4 local-checkout plugins (absolute directory paths). OpenCode's `resolvePathPluginTarget()` handles directory + `package.json` resolution.

**JSONC safety:** Use exact string edit (`edit` tool with `oldString`/`newString`), not JSON round-trip.

### Edit 3: `stack.example.toml`

Add local-only plugin entry in `[plugins.*]` section:
```toml
[plugins.oca]
path = "~/dev/opencodeadvance/plugins/oca"
```

**Why:** OCA plugin lives in the same repo — no git source, no build, no sync. Minimal declaration so `oca apply` and `oca migrate init` include it. `path` is the only required field for local-only plugins.

**Placement:** After `[plugins.anthropic-auth]` and before `[temporal]`.

### Files Touched

| File | Change |
|------|--------|
| `plugins/oca/package.json` | `main` field update |
| `~/.config/opencode/opencode.json` | plugin array addition |
| `stack.example.toml` | new `[plugins.oca]` section |

### Verification (operator-manual, post-edit)

1. Restart OpenCode
2. Open session in tmux pane
3. Verify `~/.local/state/oca/panes/<socket>/<paneId>.json` exists
4. Verify `oca pane restart-tui` uses session-id

### No Spec Deltas

No existing specs cover plugin installation. No new capability spec needed — this activates existing code.

### Design Validator Assessment

Self-assessment: VALIDATED — 3 mechanical edits, zero architectural decisions, LBP confirmed via source code verification. No conflict, no compromise risk, single viable direction.