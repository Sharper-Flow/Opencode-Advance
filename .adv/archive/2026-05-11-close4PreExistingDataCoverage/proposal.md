## Cross-Project Origin

This change was created as a follow-up from **opencode-advance**.

| Field | Value |
|-------|-------|
| Source project | opencode-advance |
| Source path | `/home/jrede/dev/opencodeadvance` |

> **Note:** The originating project should be consulted for context on why this change is needed.


## Success Criteria

### Gap 1 — MCP type translation

- [ ] `internal/migrate/openchad.go` translates `type` values at read time: `"remote" + url present` → `"http"`; `"remote" + no url` → `"http"` with warning; existing `"stdio" | "http" | "sse" | "daemon"` pass through unchanged.
- [ ] Servers with `type = "remote"` that have only `url` and no Vision-side port pick up the port from the `url` (parse port from `http://host:NNN/...`) or warn and emit `port = 0` if unparseable.
- [ ] Unknown type values emit a warning and pass through unchanged (caller surfaces validation error).
- [ ] Unit tests cover the translation matrix.

### Gap 2 — Plugin source field for path entries

- [ ] Path-style plugin entries that don't match `<OcPluginsDir>/<name>/plugin` are still emitted with a valid `source` field. Detection order:
  1. If `<path>/.git` exists, attempt `readGitRemoteURL(path)` and set `source = <url>`.
  2. Otherwise set `source = "local:<absolute-path>"`.
- [ ] `discoverPlugins` continues to enrich entries it matches (no regression on the canonical `<OcPluginsDir>/<name>/plugin` convention).
- [ ] Unit tests cover all three plugin-source resolution paths (git URL discovery, local fallback, oc-plugins canonical).

### Gap 3 — Slot group schema alignment

- [ ] `readVisionServers` struct tags match the real Vision YAML: `base_port` (int), `count` (int), `group_port` (int), `template` (string). Drop unused `min_slots` / `max_slots`.
- [ ] `SlotGroupState` struct gains `BasePort`, `Count` fields; emit writes `base_port = N`, `count = N`.
- [ ] Dead `MinSlots` / `MaxSlots` fields removed from `SlotGroupState` (opportunistic cleanup; verified zero callers).
- [ ] Unit test: round-trip Vision YAML fixture → state → emit → cfg.Load → validate clean.

### Gap 4 — Path-plugin name derivation

- [ ] `classifyPluginEntry` detects the `<X>/<name>/plugin` convention and uses `<name>` as the plugin Name instead of the literal `plugin` basename. Plain paths without the `/plugin` suffix keep their existing `filepath.Base` behavior.
- [ ] Unit tests cover: `/path/foo/plugin` → name=`foo`; `/path/foo` → name=`foo`; `/path/foo/bar.js` → name=`bar.js` (no special handling for non-`/plugin` files).
- [ ] Integration test: fixture with two entries `/x/foo/plugin` and `/x/bar/plugin` produces 2 distinct plugins named `foo` and `bar` (no collision).

### Global — cutover smoke test

- [ ] `go test ./... && go vet ./... && gofmt -d .` clean.
- [ ] **Acceptance gate**: live smoke test `oca migrate from-open-chad --output /tmp/x.toml && oca debug validate --config /tmp/x.toml` against the operator's actual open-chad state returns **0 errors** (warnings OK).
- [ ] Plugin list in /tmp/x.toml contains every distinct entry from `~/.config/opencode/opencode.json` `plugin` array with no duplicate `[plugins.plugin]` collapses.

## Approach

Order (lowest-coupling first, smoke test last):

1. **Gap 3** (slot group schema): smallest surface, no behavioral change for non-slot-group paths. Lock with fixture-based RED.
2. **Gap 1** (MCP type translation): contained to `readOpenCodeJSON`. Translation table is tight.
3. **Gap 2** (plugin source fallback): touches `classifyPluginEntry` integration plus an enrichment fallback in the reader.
4. **Gap 4** (path name derivation): refines `classifyPluginEntry` for the `/plugin` suffix convention. Touches the same code as Gap 2 — sequence Gap 2 before Gap 4 to avoid merge churn.
5. **Live smoke test**: rebuild binary, run against operator state, assert 0 errors.

Each gap follows inline TDD: RED test asserting the specific validation error reproduces against a canned fixture, GREEN fix, integration verification.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `remote → http` translation wrong for some servers | Medium | Medium | Warning-with-passthrough on unknown types; unit-test the full matrix; operator can correct individual entries post-migrate |
| `local:` source form not supported by stack apply | Low | High | Verified against `stack.example.toml:166` — supported |
| Path name derivation changes existing fixture assertions | Medium | Low | Update tests as part of the green step; document name semantics in the helper |
| Vision YAML field names change again | Low | Low | Test asserts against canonical field names from current Vision; out-of-band schema changes are a separate concern |
| Slot group `servers` field empty in output | Low | Low | Out-of-scope per proposal; emit-skip when empty rather than emit `servers = []` |
| Smoke test still fails on residual edge case | Medium | High | Acceptance is gated on `0 errors`; iterate within scope if needed; do not expand to apply-layer changes |