## Problem

Discord Rich Presence is wired through OCA across multiple layers:

- **Go runtime** — `cmd/oca/discord.go` (CLI subcommand: enable/disable/status/update), `internal/discord/` package (presence manager, taglines, rich-go IPC), `cmd/oca/root.go` command registration
- **Config schema** — typed `[discord]` section in `internal/config/types.go` + `parse.go`, documented in `docs/design/stack-toml-schema.md`, example block in `stack.example.toml`
- **Status bar** — `lib/status_bar.sh` invokes `_oca_discord_update` to push rotating taglines through Discord IPC and surface them in tmux
- **Asset** — `assets/discord/taglines.toml`
- **Tests** — `cmd/oca/discord_test.go`, `tests/phase3_types_test.go`, `tests/deferred_sections_test.go`, `tests/polish_docs_test.go`, `tests/readme_release_test.go`, `tests/shell/discord_status_hook_test.sh`
- **Docs** — README CLI table + stack.toml example, AGENTS.md tree, STATUS.md phase notes, `lib/README.md`, `docs/design/architecture.md`, `docs/proposals/v1-implementation.md`, `docs/proposals/phases.md`
- **Phase plan** — Phase 8 in `docs/proposals/phases.md` is named `phase8ExtrasPolishDiscord` and lists Discord as a primary deliverable

The integration adds a non-essential third-party IPC dependency (rich-go), couples the tmux status bar to a feature most users don't enable, and creates maintenance surface in CLI + config types for a presence feature that does not align with the project's current direction.

## Why now

User has decided Discord Rich Presence is not aligned with OCA's direction and should be removed before further surface accretes around it. The longer it sits in the tree, the more docs, tests, and CLI muscle-memory grow around it.

## User decisions (captured pre-proposal)

1. **Taglines** — drop entirely (no replacement rotator, no preserved data asset)
2. **Phase 8** — rename `phase8ExtrasPolishDiscord` → `phase8ExtrasPolish`; strike Discord-specific deliverables; keep release pipeline + final README + polish work
3. **Migration** — existing `stack.toml` files with `[discord]` blocks must parse cleanly (silently ignored as unknown section, optionally with a one-line deprecation warning)

## Success Criteria

- `oca discord` CLI subcommand and `cmd/oca/discord.go` + `discord_test.go` removed
- `internal/discord/` package deleted in full (presence.go, taglines.go, tests)
- `assets/discord/` directory deleted
- `rich-go` dependency removed from `go.mod` via `go mod tidy`
- `[discord]` typed config section removed from `internal/config/types.go` + `parse.go`
- Parsing a `stack.toml` containing `[discord]` succeeds without error (treated as unknown section)
- Status bar `_oca_discord_update` hook removed from `lib/status_bar.sh`; no tagline rotation invoked from status bar
- Phase 8 renamed to `phase8ExtrasPolish` in `docs/proposals/phases.md`; Discord deliverables struck; remaining deliverables (release pipeline, final README, polish pass) preserved
- README, STATUS.md, AGENTS.md, `lib/README.md`, `docs/design/architecture.md`, `docs/design/stack-toml-schema.md`, `docs/proposals/v1-implementation.md` updated to remove Discord references
- All Discord-mentioning tests deleted or rewritten to no longer depend on Discord surface
- `go test ./...`, `gofmt -d .`, `go vet ./...` all clean
- `rg -i discord` returns zero hits in source tree (or only intentional historical-changelog mentions)

## Acceptance Criteria

- AC1: `oca` binary builds cleanly with no Discord-related identifiers in the build graph
- AC2: `oca --help` does not show a `discord` subcommand
- AC3: A `stack.toml` containing a legacy `[discord]` table parses and `oca apply --dry-run` succeeds
- AC4: tmux status bar renders without referencing Discord wrapper paths
- AC5: Full test suite passes; no test references Discord package or commands
- AC6: Phase 8 doc rename and deliverable trim is reflected in `docs/proposals/phases.md` and `docs/proposals/v1-implementation.md`

## Out of Scope

- Replacement tagline/presence/rotator system — none planned
- Brand tone or boot-splash changes — untouched
- Other status-bar content (ADV change, gate, LLM gauges) — untouched
- Adding a deprecation warning emitter is permitted but not required — silent tolerance of `[discord]` is the floor

## Constraints

- P32 (trunk-is-prod): all work on a per-change worktree, not trunk
- Existing config files must not break — backward-compatible parse is required
- One atomic change; no half-removal leftover