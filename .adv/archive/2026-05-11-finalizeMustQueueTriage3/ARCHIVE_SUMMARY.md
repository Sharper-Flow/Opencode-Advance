# Archive: Finalize MUST queue: triage 3 residual instruction files (M4), fix migration npm-spec TOML key bug + add round-trip tests (M5), add docs/test enforcement for banned skill terms (M6) — last cutover-blockers before open-chad replacement.

**Change ID:** finalizeMustQueueTriage3
**Archived:** 2026-05-11T22:04:19.582Z
**Created:** 2026-05-11T20:21:12.737Z

## Tasks Completed

- ✅ M5 RED: Add failing test reproducing npm-spec TOML key parse failure. In `internal/migrate/emit_test.go` (new file), construct a minimal `OpenChadState` with `Plugins: [{Name: "opencode-openai-codex-auth@latest", Checkout: "opencode-openai-codex-auth@latest"}]`. Call `EmitTOML(state)`, write to tempfile, attempt `cfg.Load()` — must fail with the `@` parse error currently observed. This locks the regression.
  > Task checkpoint completed
- ✅ M5 GREEN: Implement `classifyPluginEntry(entry string) (PluginState, error)` in `internal/migrate/openchad.go`. Detect path (contains `/` or `\`) vs npm-scoped (`@scope/...`) vs npm-unscoped (`name@spec`) vs plain. Emit `source = "npm:<full>"` for npm forms with `Name = <base-without-spec>` and `Checkout = ""`. Replace the read-loop at lines 244-259 with `classifyPluginEntry` call. Add unit tests for the full classification matrix from design.md.
  > Task checkpoint completed
- ✅ M5 GREEN: Add `sanitizePluginKey(name string) (string, error)` in `internal/migrate/emit.go` as defense-in-depth. Regex `^[A-Za-z0-9._-]+$`. Wire into the plugin emit loop (line 138 area) — on error, append a state warning and skip the plugin. Unit tests cover plain/dotted/underscore/error cases.
  > Task checkpoint completed
- ✅ M5 integration test: Add `TestFromOpenChadRoundTrip_NPMSpec` in `internal/migrate/openchad_test.go`. Canned opencode.json fixture with `"plugin": ["opencode-openai-codex-auth@latest", "/abs/path/morph", "@franlol/foo@latest"]`. Pipeline: `ReadOpenChadState → EmitTOMLToFile → cfg.Load()`. Assert: load succeeds; plugin `opencode-openai-codex-auth` has `source = "npm:opencode-openai-codex-auth@latest"` and `IsNPMSource()` true; plugin `morph` has empty source and non-empty `Checkout`.
  > Task checkpoint completed
- ✅ M6: Add `tests/skills_banned_terms_test.go` walking `assets/skills/**/*.md`. Three banned-term categories per design.md: `legacy-openchad` (openchad/open-chad/oc switch), `legacy-mcp-names` (kagi_search_fetch/firecrawl_scrape/context7_resolve_library_id), `deleted-adv-skills` (5 methodology names) with `README.md` allow-list for the inlined-and-deleted note. Test must pass against current content. Manually verify it catches drift by adding a temp `openchad` ref to a skill file, running test → expect fail → revert.
  > Task checkpoint completed
- ✅ M4 asset op: Copy `~/.config/opencode/instructions/global-verify-policy.md` verbatim into `assets/instructions/global-verify-policy.md`. Update `assets/instructions/README.md`: add HTML inventory markers `<!-- INVENTORY:START -->` / `<!-- INVENTORY:END -->` wrapping the canonical file list; insert `global-verify-policy.md` row with one-line description. Verify the list matches actual `assets/instructions/` contents.
  > Task checkpoint completed
- ✅ M4 parity test: Add `TestInstructionAssetsReadmeParity` to `internal/render/instructions_test.go` (or new file in same package). Parse `assets/instructions/README.md` between INVENTORY markers; list `assets/instructions/` excluding README.md; assert symmetric difference is empty. Fail message must show the diff explicitly.
  > Task checkpoint completed
- ✅ Final verification: run `go test ./... && go vet ./... && gofmt -d .` — all clean. Then rebuild oca binary and run live smoke test against the operator's open-chad state: `oca migrate from-open-chad --output /tmp/oca-cutover.toml && oca debug validate --config /tmp/oca-cutover.toml` — must produce parseable, valid TOML. Capture the output for archive evidence.
  > Task checkpoint completed

## Specs Modified


## Wisdom Accumulated

- **[failure]** Smoke-test against the operator's live open-chad state revealed that the M5 npm-spec fix unblocks PARSING but the migrator emits a stack with 18 SEMANTIC validation errors (pre-existing data-coverage gaps): all MCP servers have type="remote" (schema allows stdio|http|sse|daemon only); vision/gh_grep missing required port; 4 plugins missing source; 3 slot groups missing base_port+count; plus the path-plugin name collision where multiple `<name>/plugin` paths all collapse to `[plugins.plugin]`. None of these are caused by M5 — they are pre-existing migrator data-coverage gaps. M5 SUCCESS LEVEL: migration goes from "unparseable garbage" (cfg.Load parse error on @-in-key) to "parseable but requires manual triage of 18 items". Net improvement but NOT yet "safely replace open-chad". Cutover requires a follow-up change to address: (1) MCP server type translation (`remote` → `http` or `stdio` based on URL/command shape), (2) plugin source field population for path entries lacking discoverPlugins enrichment, (3) slot group field defaulting, (4) plugin name collision via parent-dir-aware naming. File as `migrateCoverageGaps` after this archive.
- **[pattern]** Inline TDD with the design "read-side classify + emit-side defensive sanitize" pattern produces a clean validation hierarchy: classifyPluginEntry enforces structural correctness at the boundary where untrusted input enters (P33 alignment); sanitizePluginKey is defense-in-depth catching anything that leaks through programmatic state construction (tests, custom callers). The combination gives a clear t1→t2→t3 progression: t1 RED locks the regression; t2 GREEN at read fixes the canonical flow; t3 GREEN at emit catches the leak path. Each step is committable independently and the test failure between t1 and t3 directly demonstrates which defensive layer is doing the work.
