# Changelog

All notable OpenCode Advance changes are summarized here.

## Unreleased

### Removed

- Removed Discord Rich Presence integration in full: `oca discord` CLI subcommand, `internal/discord/` package, `assets/discord/taglines.toml`, `[discord]` typed config section, `lib/status_bar.sh` tagline-rotation hook, and the `github.com/hugolgst/rich-go` dependency. Legacy `stack.toml` files containing a `[discord]` block continue to parse cleanly via the existing deferred-sections path. Phase 8 renamed `phase8ExtrasPolishDiscord` → `phase8ExtrasPolish`; release-pipeline and README/polish deliverables preserved.

## v1.0.0-rc1

Release candidate for the first public OpenCode Advance release.

### Phase 0 — Foundation + Brand

- Established Go CLI scaffold and module layout
- Added OpenCode Advance wordmark, Obsidian palette, shell brand helpers, and boot splash
- Added baseline CI, tests, and design documentation

### Phase 1 — stack.toml + MCP Apply

- Added `stack.toml` parser and schema validation
- Implemented MCP/Vision rendering and dry-run apply flow
- Added golden tests for config rendering and target paths

### Phase 2 — Plugin + Instruction Management

- Added plugin checkout/build/apply flow
- Added instruction asset management
- Delegated Advance-owned asset sync to Advance's own sync script

### Phase 3 — Core opencode.json Coverage

- Added providers, permissions, watcher, LSP, and diff support
- Expanded `oca apply`, `oca diff`, and validation coverage
- Added typed config structs and passthrough handling for forward compatibility

### Phase 3.5 — Skills + Commands + Formatters

- Added skills, commands, formatters, and OpenCode toggle rendering
- Added OCA-owned reusable skills and instruction routing
- Expanded stack example coverage

### Phase 4 — Session UX + Theme

- Added tmux-first session lifecycle: new, list, attach, switch, kill, killall, restart, reap
- Added Obsidian tmux theme, theme commands, status bar, boot splash integration, and LLM gauges
- Added pane state and watchdog-adjacent session primitives

### Phase 5 — Temporal Enablement

- Added Temporal config rendering, doctor checks, and status bar integration
- Added prerequisite detection for Temporal and Node runtimes
- Prepared OCA to manage Advance's durable workflow runtime

### Phase 5.5 — Vision Slot Groups

- Added Vision slot group schema, rendering, and health validation
- Added OpenCode MCP entries for virtual slot group listeners
- Updated stack examples for Playwright-style slot pools

### Phase 6 — Installer + Shell

- Added install and uninstall flows
- Added shell profile block management and completions
- Added isolated config-dir safeguards for tests and development

### Phase 6.5 — Temporal Dev-Server Supervision

- Added `oca temporal start`, `stop`, `restart`, `status`, and `logs`
- Added OCA-owned PID metadata, log capture, persistent dev-server DB path, and reachability checks
- Integrated Temporal lifecycle into local development workflow

### Phase 7 — Migration + Doctor Expansion

- Added `oca migrate init` and `oca migrate from-open-chad`
- Added open-chad state import for OpenCode, Vision, plugins, and Discord settings
- Expanded doctor checks for Advance, Temporal, cross-component consistency, and asset ownership drift

### Phase 8 — Extras + Polish

- Added Discord Rich Presence command surface and Go-native Rich Presence integration
- Added professional tagline pool and status/update runtime files under OCA cache
- Added GoReleaser config, GitHub release workflow, SHA256 checksums, and version injection
- Rewrote README for the v1 release candidate
- Added INSTALL.md and this changelog
- Completed stale-reference and command-surface polish pass
