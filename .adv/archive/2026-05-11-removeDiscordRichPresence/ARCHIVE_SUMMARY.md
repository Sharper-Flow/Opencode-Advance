# Archive: Remove Discord Rich Presence integration

**Change ID:** removeDiscordRichPresence
**Archived:** 2026-05-11T17:26:14.885Z
**Created:** 2026-05-11T15:57:56.977Z

## Tasks Completed

- ✅ Test surgery: loosen Discord assertions in mixed-concern test files.
  > Task checkpoint completed
- ✅ Delete all pure-Discord files: CLI, package, assets, shell test.
  > Task checkpoint completed
- ✅ Remove `DiscordSection` type and `Stack.Discord` field; drop typed dispatch in parser.
  > Task checkpoint completed
- ✅ Migration code cleanup: remove orphaned `DiscordState` from open-chad importer.
  > Task checkpoint completed
- ✅ Status bar hook removal.
  > Task checkpoint completed
- ✅ Remove `[discord]` block from `stack.example.toml`.
  > Task checkpoint completed
- ✅ Documentation surface edits (non-phase-plan docs).
  > Task checkpoint completed
- ✅ Phase 8 rename: `phase8ExtrasPolishDiscord` → `phase8ExtrasPolish`; strike Discord deliverables.
  > Task checkpoint completed
- ✅ Dependency cleanup: `go mod tidy`.
  > Task checkpoint completed
- ✅ Final verification sweep across all AC1–AC8.
  > Task checkpoint completed

## Specs Modified


## Wisdom Accumulated

- **[pattern]** Backward-compat-by-deferred-section pattern: When removing a typed `stack.toml` section, retain its entry in `internal/config/parse.go knownSections` while deleting the typed dispatch + struct. The section falls through to the existing `DeferredSections` machinery; `Validate()` tolerates it because it's in `knownSections`. Zero new code paths, zero hard-fail on legacy configs. Same pattern used by `agents` (typed never, deferred always). Worth noting: this leaves a permanent backward-compat marker in `knownSections`. After a deprecation window, the marker can be removed and unknown-section validation will then surface stale configs explicitly.
- **[convention]** When acceptance criteria say "zero hits in `rg -i <feature>`", the spirit is "zero hits in active user-facing surface". Intentional residual references are valid when: (1) they prove backward-compat (test fixtures, knownSections tolerance markers), or (2) they record accurate history (CHANGELOG entries documenting add+remove cycles). The literal-zero reading produces bad outcomes (rewriting history, deleting backward-compat evidence). Document intentional residuals in the task verification notes when present.
- **[gotcha]** When removing a Go package, `go mod tidy` cleans up the direct dep AND any transitive deps that were only reachable through it. In this change, deleting `internal/discord/` (which imported `github.com/hugolgst/rich-go`) cleanly removed both `rich-go` AND `gopkg.in/natefinch/npipe.v2` (rich-go's only consumer of npipe). No manual go.sum surgery needed. Verify with `grep -E "<old-dep>|<transitive>" go.mod go.sum` returning zero.
