# Design — finalizeMustQueueTriage3

**Validator pass v2 (2026-05-11):** adv-researcher returned CAUTION on v1 design for promoting `@spec` into `ref` field. Verified against `stack.example.toml:157-166` (`source = "npm:opencode-openai-codex-auth@latest"`), `internal/config/types.go:236-237` (`ref` is git-only; `source` carries `npm:` prefix), `internal/config/parse_plugin_test.go:29` (canonical npm test). v2 corrects to canonical OCA npm pattern + adopts validator recommendation 4 (read-side fix). v1 superseded.

## Surface

Three independent slices touching disjoint code:

| Slice | Files touched | Code added | Tests added |
|---|---|---|---|
| M5 | `internal/migrate/openchad.go` (read), `internal/migrate/emit.go` (defensive sanitize), `internal/migrate/openchad_test.go`, `internal/migrate/emit_test.go` (new), fixtures | `classifyPluginEntry()`, `sanitizePluginKey()` (defensive) | npm-classification matrix + round-trip from-open-chad |
| M4 | `assets/instructions/global-verify-policy.md` (new copy), `assets/instructions/README.md` (update), `internal/render/instructions_test.go` (parity) | None (asset-only) | README↔dir parity test |
| M6 | `tests/skills_banned_terms_test.go` (new) | None | Banned terms + banned MCP names + banned deleted-skill claims scan |

## M5 — Migration npm-spec handling

### Canonical OCA plugin patterns (the contract we must honor)

From `stack.example.toml` and `internal/config/types.go`:

| Plugin type | Canonical TOML form | Discriminator |
|---|---|---|
| Git plugin | `source = "https://github.com/.../foo.git"` + `ref = "trunk"` (or SHA) + `checkout = "<path>"` | `IsGitSource()` |
| NPM plugin | `source = "npm:pkg@version"` (full spec inside source); no `ref`, no `checkout` | `IsNPMSource()` |
| Local plugin | `source = "local:<absolute path>"`; no `ref`, no `checkout` | `IsLocalSource()` |

`ref` field is **git-only** (`types.go:237` comment: "branch/tag/SHA; default trunk"). Putting an npm version spec into `ref` would produce a semantically invalid stack that downstream code would try to use as a git ref.

### Bug root

`internal/migrate/openchad.go:243-260` reads opencode.json's `"plugin"` array as path strings:

```go
if strings.HasPrefix(path, "@") {
    continue          // skips scoped npm (e.g. "@franlol/foo")
}
name := filepath.Base(path)
state.Plugins = append(state.Plugins, PluginState{
    Name:     name,
    Checkout: path,   // ← treats npm spec as filesystem path
})
```

`opencode-openai-codex-auth@latest` doesn't start with `@`, so it falls through. `filepath.Base` returns the literal string. The entry is added as `{Name: "opencode-openai-codex-auth@latest", Checkout: "opencode-openai-codex-auth@latest"}`. At emit, `[plugins.opencode-openai-codex-auth@latest]` is invalid TOML.

### Fix — read-side classification (validator rec 4)

New helper in `openchad.go`:

```go
// classifyPluginEntry inspects an opencode.json plugin entry string and returns
// the PluginState fields appropriate for the source kind.
//
// Recognized forms:
//   - "@scope/pkg" or "@scope/pkg@spec"     → npm scoped (source="npm:<full>", Name="pkg")
//   - "name@spec" (no path separator)       → npm unscoped (source="npm:<full>", Name=<name-without-spec>)
//   - "<path with separators>"              → local checkout (source="", Checkout=<path>, Name=basename)
//
// Returns error if name remains TOML-unsafe after classification.
func classifyPluginEntry(entry string) (PluginState, error)
```

Call site replaces lines 244-259:

```go
for _, pRaw := range pluginsRaw {
    path, ok := pRaw.(string)
    if !ok {
        continue
    }
    ps, err := classifyPluginEntry(path)
    if err != nil {
        state.Warnings = append(state.Warnings,
            fmt.Sprintf("plugin entry %q: %v; skipped", path, err))
        continue
    }
    state.Plugins = append(state.Plugins, ps)
}
```

### Defensive sanitization at emit

`internal/migrate/emit.go:138` adds a single-line guard in case any future read-side source forgets to classify:

```go
for _, p := range state.Plugins {
    cleanKey, err := sanitizePluginKey(p.Name)
    if err != nil {
        state.Warnings = append(state.Warnings,
            fmt.Sprintf("plugin %q has TOML-unsafe name; skipped", p.Name))
        continue
    }
    parts = append(parts, fmt.Sprintf("[plugins.%s]", cleanKey))
    // existing field emission unchanged
}

// sanitizePluginKey returns name unchanged if it matches [A-Za-z0-9._-]+; errors otherwise.
func sanitizePluginKey(name string) (string, error)
```

This is **defense-in-depth**, not the primary fix. Read-side classification is the canonical fix.

### Test matrix — classifyPluginEntry

| Input | Name | Source | Checkout | Error |
|---|---|---|---|---|
| `morph-fast-apply` | `morph-fast-apply` | `""` | `morph-fast-apply` | nil |
| `/abs/path/to/plugin` | `plugin` | `""` | `/abs/path/to/plugin` | nil |
| `opencode-openai-codex-auth@latest` | `opencode-openai-codex-auth` | `npm:opencode-openai-codex-auth@latest` | `""` | nil |
| `some-pkg@1.2.3` | `some-pkg` | `npm:some-pkg@1.2.3` | `""` | nil |
| `some-pkg@^1.0.0` | `some-pkg` | `npm:some-pkg@^1.0.0` | `""` | nil |
| `@franlol/foo@latest` | `foo` | `npm:@franlol/foo@latest` | `""` | nil |
| `@franlol/foo` | `foo` | `npm:@franlol/foo` | `""` | nil |
| `bad name with spaces` | — | — | — | error |
| `bad[bracket]name` | — | — | — | error |

### Test matrix — sanitizePluginKey (defensive)

| Input | Output | Error |
|---|---|---|
| `morph-fast-apply` | `morph-fast-apply` | nil |
| `some.dotted.name` | `some.dotted.name` | nil |
| `some_underscore` | `some_underscore` | nil |
| `pkg@latest` | — | error (caller should have classified) |
| `bad space` | — | error |

### Round-trip integration test

Feed canned opencode.json fixture containing the live `opencode-openai-codex-auth@latest` entry. Run `ReadOpenChadState` → `EmitTOML` → write file → `cfg.Load()`. Assert:

1. Output parses without error.
2. Loaded stack contains plugin entry `opencode-openai-codex-auth` with `source = "npm:opencode-openai-codex-auth@latest"`.
3. `IsNPMSource()` returns true on the loaded plugin.

### Rejected alternatives

1. **Sanitize at emit only (v1 design)** — leaves unsanitized name in `PluginState`; emit becomes domain-aware instead of pure formatter. v1 also promoted spec to wrong field (`ref` not `source`).
2. **Quoted TOML keys (`["plugins"."pkg@latest"]`)** — `pelletier/go-toml/v2` parses fine, but the `@spec` would leak into Go map keys, file paths, display names. Stack schema's `IsNPMSource()` would not trigger since source field would still be empty.
3. **Fail-fast on any TOML-unsafe name** — hostile UX for a one-time migration. Partial output with warnings is the right call.

## M4 — Instruction asset triage operations (unchanged from v1)

### File operations

1. Copy `~/.config/opencode/instructions/global-verify-policy.md` → `assets/instructions/global-verify-policy.md` (verbatim, single line).
2. Update `assets/instructions/README.md` inventory: add `global-verify-policy.md` row with one-line description.
3. No action for `criteria-prioritizer.md` and `post_install_verification.md` — they stay out of `assets/instructions/`. Next `oca apply --target instructions` against the live env will leave them in place; `oca apply` does not delete unmanaged files.

### Parity test design

`internal/render/instructions_test.go` (extend existing):

```go
func TestInstructionAssetsReadmeParity(t *testing.T) {
    // 1. Read assets/instructions/README.md
    // 2. Parse markdown between <!-- INVENTORY:START --> and <!-- INVENTORY:END -->
    // 3. ls assets/instructions/ (excluding README.md itself)
    // 4. Assert symmetric difference is empty; if not, fail with diff
}
```

## M6 — Banned-term test (unchanged from v1, validator confirmed)

### Test location

`tests/skills_banned_terms_test.go` (new).

### Test design

```go
func TestSkillsHaveNoBannedTerms(t *testing.T) {
    cases := []struct {
        category string
        terms    []string
        allow    []string // file paths relative to assets/skills/
    }{
        {"legacy-openchad", []string{"openchad", "open-chad", "oc switch"}, nil},
        {"legacy-mcp-names", []string{"kagi_search_fetch", "firecrawl_scrape", "context7_resolve_library_id"}, nil},
        {"deleted-adv-skills", []string{
            "adv-review-methodology", "adv-apply-methodology",
            "adv-harden-methodology", "adv-discover-methodology", "adv-prep-methodology",
        }, []string{"README.md"}},
    }
    // walk assets/skills/**/*.md; scan; fail with category+file+line on any hit not in allow
}
```

Validator note (recommendation 5): verified banned MCP name list against `instructions/mcp-tools.md` § Tool Name Discovery — `kagi_search_fetch` (without double-prefix) is the legacy form; `kagi_kagi_search_fetch` is current. List in test matches.

## Risks (design-level)

| Risk | Mitigation |
|---|---|
| `classifyPluginEntry` misclassifies a path containing `@` (e.g., `/some/path@with@at`) | Path detection precedes npm detection: presence of any `/` or `\` makes it a local path, not npm |
| `sanitizePluginKey` regex too permissive or too restrictive | Tight regex `^[A-Za-z0-9._-]+$` + explicit error path; defense-in-depth only |
| Parity test fragile to README formatting | Use HTML-comment markers to delimit inventory section |
| Banned-term test false positive | Per-category allow-list (README.md only for inlined-and-deleted note) |
| M5 fix changes existing test fixtures | Run full `go test ./...` after each step; update fixtures if existing tests asserted current bug-output |

## Dependencies

None across slices. M4, M5, M6 can execute in any order. Recommended order in planning: **M5 → M6 → M4** (real bug first; test infrastructure second; lightest asset work last).