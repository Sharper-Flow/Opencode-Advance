# Design — close4PreExistingDataCoverage

**Validator pass v2 (2026-05-11):** adv-researcher returned CAUTION on v1 Gap 1 design — blanket `remote → http` would fail validation on URLs not ending `/mcp` (e.g., operator's `gh_grep` at `https://mcp.grep.app`) because OCA's `inferTransport` (`validate.go:474-491`) classifies these as `sse`. Verified by inspection: validator at line 206 rejects empty `Type`, so simply clearing it is not viable; explicit URL-suffix-aware mapping is the correct fix. v1 superseded. Gaps 2–4 validated as safe in v1.

## Surface

Four disjoint slices in `internal/migrate/` (read + emit) plus tests:

| Slice | Files touched | Code added | Tests |
|---|---|---|---|
| Gap 3 (slot groups) | `openchad.go` (struct + yaml tags), `emit.go` (slot group block) | ~30 lines (MinSlots/MaxSlots → BasePort/Count) | YAML round-trip fixture |
| Gap 1 (MCP type) | `openchad.go` (`readOpenCodeJSON`) | ~50 lines (`translateMCPType` URL-suffix-aware + URL port parser) | Translation matrix incl. /mcp vs non-/mcp URLs |
| Gap 2 (plugin source) | `openchad.go` (`fallbackResolvePluginSources`) | ~20 lines | git-existence + local-fallback unit tests |
| Gap 4 (plugin name) | `openchad.go` (`classifyPluginEntry`) | ~10 lines (`/plugin` suffix check) | name-derivation unit tests |
| Smoke test | live operator state | n/a | rebuild oca + run + assert 0 errors |

## Gap 3 — Slot group schema alignment (UNCHANGED from v1)

`SlotGroupState`: drop `MinSlots`/`MaxSlots`; add `BasePort`/`Count`. Discovery verified 6 references all internal — hard delete safe.

`readVisionServers` inline struct: change yaml tags to `base_port`/`count`.

`emit.go:114-130`: write `base_port = N`, `count = N`. Skip empty `servers` array.

## Gap 1 — MCP type translation (REVISED)

### Validator concern

v1 design used `remote → http` for all types. But OCA's validator distinguishes `http` (URL must end `/mcp`) from `sse` (URL only). Operator's `gh_grep` at `https://mcp.grep.app` (no `/mcp` suffix) would fail with `http transport url must end with /mcp` (`validate.go:261`).

### Helper (v2)

```go
// translateMCPType maps legacy/external type names to OCA's schema enum,
// mirroring inferTransport's URL-suffix logic so the emitted stack passes
// validation directly. Returns the translated type and a warning string if
// translation was applied (caller appends to state.Warnings).
//
// Mappings:
//   "remote" + URL ending /mcp        → "http"  (with warning)
//   "remote" + URL not ending /mcp    → "sse"   (with warning)
//   "remote" + no URL                 → ""      (with warning; will fail validation)
//   "stdio"|"http"|"sse"|"daemon"     → passthrough (no warning)
//   ""                                → ""      (passthrough; caller may infer later)
//   anything else                     → unchanged with warning
func translateMCPType(t, url string) (translated string, warning string)
```

### URL-shape inference rationale

The schema's `inferTransport` (`validate.go:474-491`) already encodes this exact logic:

```go
if s.URL != "" {
    if strings.HasSuffix(s.URL, "/mcp") {
        return "http"
    }
    return "sse"
}
```

Mirroring it in the migrator means the emitted stack passes validation directly, without relying on inference at load time (which doesn't help here because `validTypes` rejects empty type at line 206 before inference runs).

### Wire-up

In `readOpenCodeJSON` MCP loop, after reading `type` and `url`:

```go
typeRaw, _ := srv["type"].(string)
urlRaw, _ := srv["url"].(string)

translated, warn := translateMCPType(typeRaw, urlRaw)
mcs.Type = translated
mcs.URL = urlRaw
if warn != "" {
    state.Warnings = append(state.Warnings, fmt.Sprintf("mcp.%s: %s", name, warn))
}
```

### URL port extraction

For URLs containing an explicit port (`http://host:NNN/...`), populate `mcs.Port` if Vision YAML doesn't provide one. URL-only servers without an explicit port (e.g., `https://mcp.grep.app`) leave `port=0` — and that's fine: per validator's recommendation 3, `sse` transport only requires URL (`validate.go:267-273`), not port.

```go
func portFromURL(rawURL string) int {
    u, err := url.Parse(rawURL)
    if err != nil || u.Port() == "" {
        return 0
    }
    port, err := strconv.Atoi(u.Port())
    if err != nil {
        return 0
    }
    return port
}
```

Post-read fix-up: after `readOpenCodeJSON` + `readVisionServers` complete, walk `state.MCPServers` and fill `Port` from URL where still 0.

### Expected matrix against operator state

| Server | URL | URL ends /mcp? | New type | Port from URL |
|---|---|---|---|---|
| vision | `http://localhost:6275/mcp` | Yes | `http` | 6275 |
| context7 | `http://localhost:6276/mcp` | Yes | `http` | 6276 |
| kagi | `http://localhost:6279/mcp` | Yes | `http` | 6279 |
| firecrawl | `http://localhost:6281/mcp` | Yes | `http` | 6281 |
| lgrep | `http://localhost:6285/mcp` | Yes | `http` | 6285 |
| gh_grep | `https://mcp.grep.app` | No | `sse` | 0 (URL-only) |

All 6 servers validate cleanly: 5 as `http` with `/mcp` URL + explicit port; 1 as `sse` with URL only.

### Rejected alternatives

1. **v1 blanket `remote → http`**: validator catches via `validate.go:261` for non-/mcp URLs. Wrong.
2. **Clear `type=""` and let inferTransport handle it**: `validate.go:206` rejects empty type BEFORE inference runs at line 240. Wrong.
3. **Inline translation in `readOpenCodeJSON`**: extracted helper makes the test matrix obvious and isolates future type-translation needs (e.g., if a new Vision type appears).

## Gap 2 — Plugin source fallback (UNCHANGED from v1; validator confirmed safe)

`fallbackResolvePluginSources` runs after `discoverPlugins`. For plugins without `Source`:

1. If `<expanded-checkout>/.git` exists → `readGitRemoteURL` → set Source.
2. Else → `source = "local:<absolute-expanded-path>"`, clear `Checkout`.

Validator confirmed: `prepare.go:29` returns early for `IsLocalSource()`; `validate.go:524` skips checkout/path validation for local sources. Clearing `Checkout` is safe.

Tilde expansion: `~/...` → absolute via `os.UserHomeDir()`. Validator notes this is operator-specific (not portable) — acceptable for per-operator migration tool. Add a comment header to the emitted stack documenting this.

## Gap 4 — Path-plugin name derivation (UNCHANGED from v1; validator confirmed safe)

`classifyPluginEntry` path branch: when basename is exactly `plugin`, use parent dir name. Aligns with `pluginCheckoutMatches` (`openchad.go:525`) which already encodes the convention.

## Cross-gap interaction (verified)

Ordering: `readOpenCodeJSON` → `classifyPluginEntry` (Gap 4 applies) → `readVisionServers` → `discoverPlugins` → `fallbackResolvePluginSources` (Gap 2 applies). Type translation (Gap 1) and URL port extraction happen inside `readOpenCodeJSON` and a post-read fix-up. No race conditions; each gap operates on a disjoint field of `OpenChadState`.

## Sequencing

Per agreement: Gap 3 → Gap 1 → Gap 2 → Gap 4 → smoke test. Each independently committable. Smoke test gates acceptance.

## Risks (design-level, v2)

| Risk | Mitigation |
|---|---|
| URL `/mcp` heuristic misclassifies a real http server with non-standard path | Operator can hand-correct `type` post-migrate; warning surfaces the inferred mapping |
| Operator's `https://mcp.grep.app` is actually http not sse | Validator catches if wrong; warning makes the decision visible; cost of correction is low |
| Tilde expansion fails for operator paths using `~user` | Out of scope; only handle `~/...` (verified operator state uses tilde, not `~user`) |
| Slot group struct change breaks emit golden test | Update golden expectations in green step |
| Smoke test reveals new gap | Iterate within scope; do not expand to apply-layer |

## Documentation note for emit

Add a header comment to the migrated stack.toml:

```
# Plugin sources marked `local:<absolute-path>` reference paths on this
# operator's machine and are not portable across hosts.
```

Emitted in `emit.go`'s header template only when any local: source is present.