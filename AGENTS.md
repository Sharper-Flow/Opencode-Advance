# AGENTS.md — Developer & Agent Reference

Internal reference for AI agents and human developers working on OpenCode Advance.
For user-facing documentation, see [`README.md`](README.md).

---

## Project Overview

**OpenCode Advance** (`oca`) is the environment and configuration platform for [OpenCode](https://opencode.ai). It replaces the earlier `open-chad` project with a Go-based, declarative rewrite. It pairs with the [Advance](https://github.com/Sharper-Flow/Advance) spec-driven workflow plugin as a required dependency.

- **Repo**: `https://github.com/Sharper-Flow/Opencode-Advance.git`
- **Branch**: `trunk` (default), remote `origin`
- **Language**: Go 1.22+ (CLI core), Bash (tmux/shell integration only), Markdown (design docs, proposals)
- **Test framework**: Go stdlib `testing` + golden files; bash test scripts for shell integration

---

## Relationship to Advance

OpenCode Advance depends on Advance; Advance does not depend on OpenCode Advance.

- OpenCode Advance clones, builds, and wires the Advance plugin as part of `oca install` / `oca apply`
- OpenCode Advance delegates all Advance-owned asset sync to `advance/scripts/sync-global.sh --fix`
- OpenCode Advance does **not** duplicate any files that Advance owns: `adv-*.md` commands, ADV agents (`adv`, `plan`, `adv-researcher`, `tron`), ADV skills (`adv-*`), ADV overlays, or ADV instructions
- OpenCode Advance owns the non-ADV slice of the environment: environment-level agents (`build`, `explore`, `librarian`, `general`, `mechanic`), instructions (rules.yaml, identity, shell_strategy, etc.), MCP server lifecycle, plugin management, providers, session/tmux UX

This clean boundary is the central reason OpenCode Advance exists as a separate project. Each file in `~/.config/opencode/` has exactly one owner.

---

## Architecture

```
opencodeadvance/
├── stack.toml                          # THE source of truth (user creates; oca reads)
├── stack.example.toml                  # Complete example reference
│
├── cmd/oca/                            # CLI entry point (Go)
│   └── main.go
│
├── internal/                           # Internal Go packages
│   ├── config/                         # stack.toml parser + schema validation
│   ├── render/                         # Programmatic rendering + JSON merge logic
│   ├── health/                         # MCP/plugin/skills/temporal health checks
│   ├── subprocess/                     # Generic command runner with timeout/signal/exit classification
│   ├── plugin/                         # Git clone/pull, build, pin, npm handler
│   ├── sync/                           # Advance sync-global.sh invocation
│   ├── session/                        # Tmux session lifecycle (create/list/attach/switch/kill/restart/reap)
│   ├── install/                        # Prerequisite checks, shell profile management, install/uninstall orchestration
│   ├── temporal/                       # Temporal dev-server supervision (start/stop/restart/logs/status)
│   └── migrate/                        # open-chad → opencode-advance importer
│
├── assets/                             # Static files, copied as-is to ~/.config/opencode/
│   ├── agents/                         # build.md, explore.md, librarian.md, general.md, mechanic.md
│   ├── instructions/                   # identity.md, rules.yaml, shell_strategy.md, etc.
│   ├── skills/                         # lgrep, mcp-selection, morph, worktree, prioritizer, caveman, caveman-commit, caveman-review
│   └── themes/
│       ├── obsidian.json               # OpenCode UI theme (obsidian palette, indigo accent)
│       ├── obsidian.tmux.conf          # Tmux 2-row status bar theme
│       └── obsidian-light.tmux.conf    # Optional light variant (future)
│
├── templates/                          # Go text/template files rendered by oca apply
│   ├── opencode.json.gotmpl            # Merged into ~/.config/opencode/opencode.json
│   ├── vision-servers.yaml.gotmpl      # Rendered to ~/.config/vision/servers.yaml
│   ├── tmux.conf.block.gotmpl          # Block injected into ~/.tmux.conf
│   └── shell_profile.block.gotmpl      # Block injected into ~/.zshrc / ~/.bashrc
│
├── bin/                                # Shell-level entry points (post-install)
│   ├── oca                             # Thin wrapper calling the Go binary
│   ├── oc                              # Short alias (retains muscle memory)
│   ├── cds                             # Scratch dir launcher
│   └── ocashell.sh                     # Completion bootstrap
│
├── lib/                                # Shell-only helpers (tmux theming, boot splash)
│   ├── palette.sh                      # Truecolor/256/mono palette + NO_COLOR handling
│   ├── wordmark.sh                     # Compact 3-line wordmark renderer
│   ├── boot_splash.sh                  # Boot splash with wordmark reveal + version line
│   ├── session_lifecycle.sh            # tmux session creation/list primitives
│   └── discord/                        # Discord Rich Presence integration
│
├── tests/                              # Test suites
│   ├── config_test.go                  # stack.toml parse/validate
│   ├── render_test.go                  # Template rendering golden files
│   ├── apply_test.go                   # End-to-end apply with mock home
│   └── shell/                          # Bash tests for shell integration
│
├── docs/
│   ├── design/                         # Architecture, brand, schema, CLI reference
│   │   ├── architecture.md
│   │   ├── brand.md
│   │   ├── wordmark.md
│   │   ├── palette.md
│   │   ├── theme.md
│   │   ├── stack-toml-schema.md
│   │   └── cli-surface.md
│   ├── proposals/                      # Implementation planning docs
│   │   ├── v1-implementation.md        # Full v1.0 proposal content (for first ADV change)
│   │   ├── phases.md                   # Phase sequencing
│   │   └── first-boot.md               # How to initialize ADV in this repo
│   └── specs/                          # Generated spec docs (populated by ADV)
│
├── .adv/                               # ADV in-repo state (specs only)
│   └── specs/                          # Capability specs (written by ADV, git-tracked)
│                                       # Mutable state (changes, archive, wisdom, agenda) lives in
│                                       # $XDG_DATA_HOME/opencode/plugins/advance/{project-id}/
│                                       # managed by Temporal workflows with file-backed fallback
│
└── .github/
    └── workflows/                      # CI (populated in Phase 1)
```

---

## Development Model

This project is developed through the Advance spec-driven workflow. Every non-trivial change is an ADV change following the 7-gate lifecycle:

1. **Proposal** — problem statement, success criteria, constraints, discovery agenda
2. **Discovery** — context analysis, objectives, agreement (`/adv-discover` + `/adv-agree`)
3. **Design** — architecture decisions with mandatory validator pass (`/adv-design` + `/adv-present`)
4. **Planning** — task graph synthesized from validated design (`/adv-prep`)
5. **Execution** — TDD-driven implementation (`/adv-apply`)
6. **Acceptance** — code review + user sign-off (`/adv-review` + `/adv-accept`)
7. **Release** — quality verification, spec deltas applied, git finalized (`/adv-harden` + `/adv-archive`)

The v1.0 initial implementation is scoped in [`docs/proposals/v1-implementation.md`](docs/proposals/v1-implementation.md) and phased in [`docs/proposals/phases.md`](docs/proposals/phases.md).

### First ADV change

See [`docs/proposals/first-boot.md`](docs/proposals/first-boot.md) for the exact sequence to initialize ADV state in this repo and create the first change from the scaffolded proposal content.

---

## Clean cutover policy

During development, OpenCode Advance MUST NOT modify the user's live OpenCode configuration. All development and testing happens in isolated config directories:

- `~/.config/opencode-advance-dev/` (instead of `~/.config/opencode/`)
- `~/.config/vision-dev/` (instead of `~/.config/vision/`)
- Isolated tmux socket if needed

The `oca` CLI accepts environment overrides for all target paths so tests and dev runs never touch production state:

| Override                  | Purpose                                                         |
| ------------------------- | --------------------------------------------------------------- |
| `OCA_OPENCODE_CONFIG_DIR`    | Target OpenCode config dir (default: `~/.config/opencode`)        |
| `OCA_VISION_CONFIG_DIR`      | Target Vision config dir (default: `~/.config/vision`)            |
| `OCA_PLUGIN_CHECKOUT_ROOT`   | Where plugins are cloned (default: `~/dev/oc-plugins/`)           |
| `OCA_CACHE_DIR`              | Cache/runtime dir (default: `$XDG_RUNTIME_DIR/opencode-advance`)  |

At v1.0 release time, the user runs `oca migrate from-open-chad` which performs a one-shot read of the current `open-chad`-managed state, emits a `stack.toml` reflecting it, and then `oca install` applies it to production.

---

## What goes where (file ownership matrix)

| File / dir                                  | Owner         | Notes                                              |
| ------------------------------------------- | ------------- | -------------------------------------------------- |
| `stack.toml`                                  | **user**          | Source of truth. User edits. `oca` reads.            |
| `~/.config/opencode/opencode.json`            | **oca**           | Rendered from stack.toml + plugin-provided fragments |
| `~/.config/opencode/agents/build.md`          | **oca + overlay** | Base: oca. Advance injects ADV overlay block       |
| `~/.config/opencode/agents/explore.md`        | **oca**           | Environment-level agent                            |
| `~/.config/opencode/agents/librarian.md`      | **oca**           | Environment-level agent                            |
| `~/.config/opencode/agents/general.md`        | **oca + overlay** | Base: oca. Advance injects ADV overlay block       |
| `~/.config/opencode/agents/mechanic.md`       | **oca**           | Environment-level agent                            |
| `~/.config/opencode/agents/adv.md`            | **Advance**       | ADV orchestrator agent (via sync-global.sh)        |
| `~/.config/opencode/agents/plan.md`           | **Advance + overlay** | ADV agent with overlay block                       |
| `~/.config/opencode/agents/adv-researcher.md` | **Advance**       | ADV research + validation agent (bundled global)   |
| `~/.config/opencode/agents/adv-engineer.md`   | **Advance**       | ADV delegated code-writing executor (bundled global)|
| `~/.config/opencode/agents/tron.md`           | **Advance**       | ADV agent (repo-local, `.opencode/agents/`)        |
| `~/.config/opencode/command/adv-*.md`         | **Advance**       | ADV slash commands                                 |
| `~/.config/opencode/command/oca-*.md`         | **oca**           | OpenCode Advance slash commands (if any)          |
| `~/.config/opencode/skills/adv-*/`            | **Advance**       | ADV methodology skills                             |
| `~/.config/opencode/skills/lgrep/`            | **oca**           | Code exploration tool selection skill               |
| `~/.config/opencode/skills/morph/`            | **oca**           | Edit tool selection skill                           |
| `~/.config/opencode/skills/prioritizer/`      | **oca**           | Tradeoff analysis methodology skill                 |
| `~/.config/opencode/skills/worktree/`         | **oca**           | Git worktree workflow skill                         |
| `~/.config/opencode/skills/mcp-selection/`    | **oca**           | MCP tool selection decision matrix skill            |
| `~/.config/opencode/skills/caveman/`          | **oca**           | Compressed communication mode skill                 |
| `~/.config/opencode/skills/caveman-commit/`   | **oca**           | Compressed commit message skill                     |
| `~/.config/opencode/skills/caveman-review/`   | **oca**           | Compressed code review comments skill               |
| `~/.config/opencode/instructions/identity.md` | **oca**           | Environment-level instruction                      |
| `~/.config/opencode/instructions/rules.yaml`  | **oca**           | Environment-level instruction                      |
| `~/.config/opencode/instructions/shell_strategy.md` | **oca**     | Environment-level instruction                      |
| `~/.config/opencode/instructions/test_resource_guardrails.md` | **oca** | Environment-level instruction                      |
| `~/.config/opencode/instructions/lbp.md`      | **oca**           | Environment-level instruction                      |
| `~/.config/opencode/instructions/temp_directory.md` | **oca**     | Environment-level instruction                      |
| `~/.config/opencode/instructions/mcp-tools.md` | **oca**          | Environment-level instruction                      |
| `~/.config/opencode/instructions/lgrep-tools.md` | **oca**        | Environment-level instruction                      |
| `~/.config/opencode/instructions/morph-tools.md` | **oca**        | Environment-level instruction                      |
| `~/.config/opencode/instructions/worktree-guide.md` | **oca**     | Environment-level instruction                      |
| `~/.config/opencode/instructions/caveman.md`  | **oca**           | Environment-level instruction                      |
| `~/.config/opencode/instructions/ADV_*.md`    | **Advance**       | Advance instruction (path referenced)              |
| `~/.config/vision/servers.yaml`               | **oca**           | Rendered from stack.toml                           |
| `~/.tmux.conf` (OCA block only)               | **oca**           | Managed block, rest is user-owned                  |
| `~/.zshrc` / `~/.bashrc` (OCA block only)     | **oca**           | Managed block, rest is user-owned                  |
| `plugins/oca/` (source)                       | **oca**           | OCA umbrella plugin source (TypeScript, bun-built) |
| `~/.config/opencode/plugins/oca/index.js`     | **oca**           | Installed OCA plugin artifact                      |
| `$XDG_STATE_HOME/oca/panes/`                  | **oca**           | Per-pane session state (plugin write, CLI read)    |

Any file not in the "oca" or "Advance" column is user-owned and MUST NOT be touched by `oca apply`.

Recent Advance changes consolidated shared agents: `scout -> plan` and `refine -> build`. OCA docs should not describe `scout.md` or `refine.md` as current shipped Advance assets.

### Advance Temporal migration

Advance now uses **Temporal as its primary state backend**. Key implications for OCA:

- State storage moved from JSON files to Temporal durable workflows (`changeWorkflow`, `projectWorkflow`)
- File-backed JSON is the Temporal adapter's internal persistence layer (not a runtime fallback); `ADV_DISABLE_TEMPORAL=1` is a test/dev escape hatch only
- OCA's Phase 5 managed the Temporal infrastructure (CLI, dev server, env vars) that Advance depends on (completeTemporalOnlyMigration and retireLegacyStorageBackend branches pending upstream)
- New `adv-engineer` agent (bundled global) for delegated code-writing execution
- `adv-researcher` promoted from repo-scoped to bundled global
- Worker model: in-process (Node hosts) or out-of-process child (Bun hosts via `ADV_NODE_PATH`)
- Continue-as-new prevents unbounded workflow history (configurable thresholds)

---

## Commit style

- Conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `style:`
- Scope when relevant: `feat(config): add stack.toml schema validation`
- Atomic commits — one logical change per commit
- Never commit secrets or user-specific paths

---

## CI/Release

`.github/workflows/`:

- `ci.yml` — shipped: on every PR and push to trunk runs `go test ./...`, `go vet ./...`, `gofmt -d .`
- `release.yml` — Phase 8 (not yet shipped): on version tags (`v*`) goreleaser will build cross-platform binaries and publish to GitHub Releases
