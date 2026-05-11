## Current State

Discord Rich Presence surface mapped (post-discovery):

### Code
- `cmd/oca/discord.go` — CLI: enable/disable/status/update subcommands
- `cmd/oca/discord_test.go` — 3 test funcs covering CLI behavior
- `cmd/oca/root.go:104` — `cmd.AddCommand(newDiscordCmd(state))`
- `internal/discord/presence.go` + `presence_test.go` — `PresenceManager`, `RichGoClient`, `RPCClient` iface
- `internal/discord/taglines.go` + `taglines_test.go` — tagline loader with `DefaultTagline` fallback
- `assets/discord/taglines.toml` — tagline data

### Config schema
- `internal/config/types.go:5,47-48,951-` — `DiscordSection` type + `Stack.Discord` field
- `internal/config/parse.go:36,243-248` — `"discord": true` in typed-section map + dispatch case
- `stack.example.toml:406-408+` — `[discord]` example block

### Status bar
- `lib/status_bar.sh:68-70,208` — `_oca_discord_update()` reads `$_oca_cache_dir/discord/discord-update.sh` and invokes it as part of status assembly

### Tests
- `cmd/oca/discord_test.go` — 3 funcs
- `tests/phase3_types_test.go:357,374-375` — asserts `Discord` field parses
- `tests/deferred_sections_test.go:19,24-31` — discord in typed-section list + 3 assertions
- `tests/polish_docs_test.go:10-42,61-72` — Discord regression checks + phase8 identifier check
- `tests/readme_release_test.go:22` — `"oca discord enable"` in expected README command list
- `tests/shell/discord_status_hook_test.sh` — shell test for status bar hook

### Docs
- `README.md:29,94-96,126` — CLI table + stack.toml example
- `STATUS.md:17,28` — Phase 0 + Phase 8 mentions
- `AGENTS.md:81,87` — repo tree entries
- `lib/README.md:29` — Discord runtime location
- `docs/design/architecture.md:42,51` — typed section list + render description
- `docs/design/stack-toml-schema.md:5,21,525-534` — schema entry + section heading
- `docs/proposals/phases.md:497-525` — Phase 8 deliverables (renamed)
- `docs/proposals/v1-implementation.md:22,59,67,93,131,247` — phase reference + brand narrative + checklist items

### Dependencies
- `go.mod:7` — `github.com/hugolgst/rich-go v0.0.0-20240715122152-74618cc1ace2`
- `go.sum:42-43` — sum entries
- Likely indirect-dep cleanup: `gopkg.in/natefinch/npipe.v2` (rich-go's only path to it)

## Open Questions Resolved

| # | Question | Resolution |
|---|---|---|
| Q1 | Is `tests/polish_docs_test.go` purely Discord-centric? | **Mixed.** Keep `TestDocsNoStaleAgentNames`. Delete `TestPolishDocsRemoveDeferredDiscordReferences`. Update `TestPhase8RoadmapDocsShowReleaseCandidate` to assert new identifier `phase8ExtrasPolish` instead of `phase8ExtrasPolishDiscord`. |
| Q2 | External consumers of `phase8ExtrasPolishDiscord` identifier? | **None.** 3 in-repo hits only (test, phases.md, v1-implementation.md). Safe rename. |
| Q3 | `rich-go` usage beyond `internal/discord/`? | **None.** Contained entirely within `internal/discord/presence.go`. `go mod tidy` cleans direct + transitive. |

## Objectives

1. **Remove Discord runtime entirely** — Go package, CLI subcommand, assets, dependency
2. **Remove typed config schema** — `DiscordSection` type and parse dispatch
3. **Tolerate legacy `[discord]` in user `stack.toml`** — silent pass as unknown section, no parse error
4. **Remove tmux status-bar tagline hook** — no rotating taglines
5. **Update tests** — delete Discord-only tests; modify mixed-concern tests (`polish_docs_test.go`, `phase3_types_test.go`, `deferred_sections_test.go`, `readme_release_test.go`)
6. **Update docs** — strike Discord references from README, STATUS, AGENTS, lib/README, architecture, schema, proposal docs
7. **Rename Phase 8** — `phase8ExtrasPolishDiscord` → `phase8ExtrasPolish`; strike Discord deliverables; preserve release-pipeline + final-README + polish work items
8. **Verify cleanly** — `go mod tidy`, `gofmt`, `go vet`, `go test ./...`, shell tests, zero `rg -i discord` hits in source tree

## Acceptance Criteria

| ID | Criterion |
|---|---|
| AC1 | `oca` binary builds with no Discord identifiers in build graph (`go build ./...` clean) |
| AC2 | `oca --help` does not list `discord` subcommand |
| AC3 | `stack.toml` containing legacy `[discord]` table parses without error; `oca apply --dry-run` succeeds (silent tolerance) |
| AC4 | `lib/status_bar.sh` contains zero references to Discord; tmux status bar renders successfully without Discord wrapper paths |
| AC5 | `go test ./...` passes; no test depends on Discord package, CLI, or schema |
| AC6 | `docs/proposals/phases.md` and `docs/proposals/v1-implementation.md` use `phase8ExtrasPolish` (no `…Discord` suffix); Discord-specific deliverables struck; release-pipeline + final-README + polish items preserved |
| AC7 | `rg -i discord` returns zero hits in tracked source files (excluding ADV state, archive bundles, this change's archive metadata) |
| AC8 | `go mod tidy` removes `rich-go` from `go.mod` + `go.sum`; transitive cleanup applied |

## Out of Scope

- Replacement tagline/presence/rotator system
- Brand tone changes (palette, wordmark, boot splash)
- Other status-bar segments (ADV change, gate, LLM gauges)
- Re-numbering Phase 8 or restructuring its surrounding phases
- Cleanup of `$XDG_RUNTIME_DIR/.../discord/` cache files left by prior `oca discord enable` runs on user machines (cosmetic; user can delete manually)

## Risks Updated Post-Discovery

| ID | Risk | Status |
|---|---|---|
| R1 | `polish_docs_test.go` mixed-concern | **Resolved.** Strategy: surgical edit, not delete |
| R2 | `rich-go` transitive leftovers | **Resolved.** Confined to discord package; `go mod tidy` handles |
| R3 | Status bar layout breakage | **Open.** Verify `lib/status_bar.sh` still assembles cleanly with hook removed; visual smoke OK if no shell tests cover layout |
| R4 | External `phase8ExtrasPolishDiscord` consumers | **Resolved.** Zero external refs |