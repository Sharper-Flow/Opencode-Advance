# Archive: Close the 4 pre-existing data-coverage gaps in `internal/migrate/` so `oca migrate from-open-chad` produces a stack that passes `oca debug validate` against the operator's live state — final cutover-blocker for v1.0 / safe open-chad replacement.

**Change ID:** close4PreExistingDataCoverage
**Archived:** 2026-05-11T23:40:22.304Z
**Created:** 2026-05-11T22:26:08.546Z

## Tasks Completed

- ✅ Gap 3 (slot groups): Inline TDD. Add RED test in `internal/migrate/openchad_test.go` (or `emit_test.go`) with a real Vision YAML fixture containing `base_port`/`count`/`group_port`/`template`. Expect cfg.Load to succeed AND each slot group to round-trip with correct base_port/count. GREEN: drop `MinSlots`/`MaxSlots` from `SlotGroupState` (lines 158-163) and the inline yaml struct (lines 430-436); add `BasePort int` / `Count int` with `yaml:"base_port"`/`yaml:"count"` tags; update emit (`emit.go:114-130`) to write `base_port = N` and `count = N`; skip empty `servers` array. Update any existing golden tests that asserted the old fields.
  > Task checkpoint completed
- ✅ Gap 1 (MCP type translation): Inline TDD. Add `translateMCPType(t, url string) (translated, warning string)` + `portFromURL(rawURL string) int` helpers in `internal/migrate/openchad.go`. Matrix test cases per design.md: `remote + .../mcp` → `http`; `remote + non-/mcp URL` → `sse`; `remote + no URL` → `""` with warning; `stdio|http|sse|daemon` → passthrough; unknown → unchanged with warning. Wire into `readOpenCodeJSON` MCP loop (after reading type/url). Add post-read fix-up loop in `ReadOpenChadState` to fill `mcs.Port` from URL when still 0 after Vision read. Integration test: fixture with operator-shape opencode.json (6 remote servers with mixed /mcp suffix) produces a state where each server has the correct translated type + port.
  > Task checkpoint completed
- ✅ Gap 2 (plugin source fallback): Inline TDD. Add `fallbackResolvePluginSources(state)` + `expandTilde(p string) string` helpers in `internal/migrate/openchad.go`. Logic per design.md: for plugins with empty Source and non-empty Checkout, expand tilde to absolute; if `<abs>/.git` exists → readGitRemoteURL → Source; else → `source = "local:<abs>"` and clear Checkout. Wire into `ReadOpenChadState` immediately after `discoverPlugins`. Unit tests: (a) git-existence path resolves to remote URL; (b) non-git path → local: source; (c) tilde expansion correctness for `~/foo`, `~`, `/abs` (no-op), `relative` (no-op).
  > Task checkpoint completed
- ✅ Gap 4 (plugin name from /plugin suffix): Inline TDD. Refine `classifyPluginEntry` path branch per design.md: when `filepath.Base(entry) == "plugin"`, use `filepath.Base(filepath.Dir(entry))` as Name instead. Defensive: skip if parent is "" or "." or "/". Unit tests: `/x/foo/plugin` → name=`foo`; `/x/foo` → name=`foo`; `/x/foo/bar.js` → name=`bar.js`; `/plugin` (root-level) → name unchanged. Integration test: fixture with two entries `/x/foo/plugin` and `/x/bar/plugin` produces 2 plugins named `foo` and `bar` (no `[plugins.plugin]` collision).
  > Task checkpoint completed
- ✅ Header comment for local: portability + verification sweep. Update `emit.go` header template to emit a `# Plugin sources marked local:<path> reference paths on this operator's machine and are not portable.` comment when any plugin has a `local:` source. Run `go test ./... && go vet ./... && gofmt -d .` — all clean. Update any golden tests that captured old behavior.
  > Task checkpoint completed
- ✅ Acceptance smoke test: Build fresh `oca` binary from current source. Run `oca migrate from-open-chad --output /tmp/oca-cutover-v2.toml` against the operator's actual `~/.config/opencode/opencode.json` + `~/.config/vision/servers.yaml`. Run `oca debug validate --config /tmp/oca-cutover-v2.toml` — must return exit 0 with `ok` output. Capture the count of plugins, MCP servers, slot groups in the output as evidence. If any validation errors remain, do NOT expand scope — diagnose, fix within the 4-gap surface, re-run.
  > Task checkpoint completed

## Specs Modified


## Wisdom Accumulated

- **[pattern]** Live smoke test surfaced two scope items not in the original 4-gap design that had to be solved within the change to reach the 0-errors acceptance gate: (1) URL field was never emitted by emit.go — pre-fix this masked itself because validation didn't run cleanly; once Gap 1 produced valid types the missing URL emit surfaced; fix: add `url = %q` emission. (2) Multi-source state conflict — opencode.json declares user intent (type+url for remote) while vision/servers.yaml declares Vision's local proxy impl (command+args for stdio). Both were copied to state, producing schema-invalid `cannot specify both command and url`. Fix: `normalizeTransportFields` post-read pass drops Command/Args for http/sse types and drops URL for stdio/daemon types. Also discovered OCA schema forces port ∈ [6275, 6325] for all MCP servers — external services like https://mcp.grep.app can't fit; emit-time skip-with-warning is the correct compromise (operator hand-adds post-migrate). These three sub-fixes belonged in scope because they were entirely inside the migrator surface and necessary to reach the acceptance criterion.
