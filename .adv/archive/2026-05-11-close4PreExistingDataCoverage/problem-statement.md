## Problem Statement

After `finalizeMustQueueTriage3` (M5) fixed the npm-spec TOML key bug, `oca migrate from-open-chad` against the operator's live state now PARSES cleanly but produces a stack with **18 semantic validation errors** that block `oca apply`. The errors fall into 4 distinct data-coverage gaps in `internal/migrate/openchad.go` (read side) and `internal/migrate/emit.go` (emit side). All are pre-existing bugs predating M5.

Source of evidence: live smoke test 2026-05-11 captured in wisdom `ws-qAXOaG`:

```
oca migrate from-open-chad --output /tmp/oca-cutover.toml
oca debug validate --config /tmp/oca-cutover.toml
→ stack.toml validation failed (18 error(s))
```

### Gap 1 — MCP `type` translation missing (8 errors)

The migrator reads `mcp.<server>.type` verbatim from `opencode.json`. Vision and OpenCode both use `type = "remote"` for HTTP MCP servers. OCA's schema (`internal/config/validate.go:38-41`) allows only `stdio | http | sse | daemon`. The migrator never maps `remote → http`.

**Errors:** `[mcp.servers.{context7,firecrawl,gh_grep,kagi,lgrep,vision}.type]: unknown type "remote"` (6 servers, 6 errors) + `[mcp.servers.{gh_grep,vision}.port]: required field missing` (2 servers missing port because they're URL-only remotes with no Vision-side port entry — same gap, downstream effect).

**Root cause:** `internal/migrate/openchad.go:320-321`:

```go
if t, ok := srv["type"].(string); ok {
    mcs.Type = t  // verbatim — no translation
}
```

### Gap 2 — Plugin `source` field unpopulated for unenriched path entries (4 errors)

Path-style plugin entries (e.g., `/home/jrede/dev/opencodeadvance/plugins/oca`) get `Source=""` from `classifyPluginEntry`. `discoverPlugins` enriches `Source` only when the path matches the `<OcPluginsDir>/<name>/plugin` pattern via `pluginCheckoutMatches`. Plugin paths that don't follow that pattern (or live outside `OcPluginsDir`) keep `Source=""`. OCA schema requires `source` for every plugin.

**Errors:** `[plugins.{oca,claude-max,anthropic-auth-max-override,opencode-plugin}.source]: required field missing` (4 plugins).

**Root cause:** The reader has no fallback for "I see a path-style plugin but it doesn't match the oc-plugins discovery pattern." Real cutover requires: any path entry should either get a `source = "local:<path>"` form OR be discovered as git via direct `.git` read at the literal path.

### Gap 3 — Slot group schema drift (6 errors)

The migrator's `readVisionServers` reads slot groups with fields `{Servers, GroupPort, Template, MinSlots, MaxSlots}` — but the actual `~/.config/vision/servers.yaml` uses `{base_port, count, group_port, template}`. The migrator's YAML field tags `yaml:"min_slots"` and `yaml:"max_slots"` don't exist in the real Vision YAML, so they're always zero. `base_port` and `count` aren't read at all. The emit then writes `min_slots/max_slots` but OCA's validator (`validate.go:362,372`) requires `base_port` and `count`.

**Errors:** `[mcp.slot_groups.playwright-{auth,headed,headless}.{base_port,count}]: required field missing` (3 groups × 2 fields = 6 errors).

**Root cause:** stale `readVisionServers` struct definition predates Vision's slot-group finalization. Fields in the live YAML are `base_port: N`, `count: N` (verified by `grep -A8 'playwright-headless:' ~/.config/vision/servers.yaml`).

### Gap 4 — Path-plugin name collision (0 errors today; future bug)

`filepath.Base("/home/jrede/dev/oc-plugins/<name>/plugin")` returns `plugin` for any path ending in `/plugin`. The operator's live `opencode.json` has multiple such entries (`advance/plugin`, possibly others). All collapse to `[plugins.plugin]` — first writer wins, subsequent duplicates would conflict with TOML duplicate-key parsing IF emitted (validator currently catches via missing source first; deduplication is silent).

**Visible in output:** `/tmp/oca-cutover.toml` contains a single `[plugins.plugin]` table that points to Advance — the Advance entry won; other `/plugin`-tailed paths got silently dropped/overwritten. This is data loss masquerading as a validation pass.

**Root cause:** `classifyPluginEntry` uses `filepath.Base(entry)` for the Name. For `/path/<name>/plugin`, it should detect the trailing `/plugin` convention and use the parent dir name (matching `pluginCheckoutMatches` semantics at the read site).

## Evidence

- Live smoke test output captured 2026-05-11 (wisdom `ws-qAXOaG`)
- Schema enum verified at `internal/config/validate.go:38-41` (stdio/http/sse/daemon)
- Slot group required fields verified at `internal/config/validate.go:362,372` (base_port, count)
- Vision YAML field names verified by direct grep of `~/.config/vision/servers.yaml`
- Migrator read sites located: `openchad.go:320-321` (mcp type), `openchad.go:430-436` (slot group struct tags), `classifyPluginEntry` (plugin name derivation)
- Plugin definition file: `internal/config/types.go:236` (`Source` is required field via Validate)
- `local:` source form is canonical: `stack.example.toml:166` `source = "local:~/dev/opencodeadvance/plugins/oca"`

## Why this is the right scope

These 4 gaps are the entire remaining cutover-blocker between "migration parses" and "migration validates". They share:

1. **Same goal**: `oca migrate from-open-chad && oca debug validate` returns 0 errors against the operator's live state.
2. **Same surface**: all 4 fixes touch `internal/migrate/` (openchad.go + emit.go) plus tests; no schema changes, no validate.go changes, no apply changes.
3. **Same TDD pattern**: each gap can be locked with a RED test reproducing the validation error against a canned fixture, then GREEN'd with a read-side or emit-side fix.
4. **Same release window**: shipping one without the others leaves cutover still blocked.

After this archives, the cutover sequence is unblocked:

```
oca migrate from-open-chad → /tmp/stack.toml  (parses + validates)
review the generated stack.toml             (operator one-time review)
oca apply --dry-run                          (preview render plan)
oca apply                                    (cutover step)
oca uninstall openchad                       (remove legacy)
v1.0.0 tag → release.yml                     (binary release)
```

## Out of scope

- Schema changes to `internal/config/` (no new `type` enum values; no new slot group fields).
- Behavior of `oca apply` (downstream consumer of the migrated stack).
- Migration of slot group `servers` list semantics (the field was empty in v1 design; keep deferring).
- Handling NPM plugins emitted with no Vision/upstream pin (`source = "npm:pkg"` without `@version`).
- `migrate init` (covered by M5; already validates clean).
- v1.0.0 tag publication (separate post-archive step).