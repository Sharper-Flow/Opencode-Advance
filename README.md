<div align="center">

```
░█▀█░█▀█░█▀▀░█▀█░█▀▀░█▀█░█▀▄░█▀▀   ░▟█▙░█▀▄░█░█░▟█▙░█▀█░█▀▀░█▀▀
░█░█░█▀▀░█▀▀░█░█░█░░░█░█░█░█░█▀▀   ░█▀█░█░█░▀▄▀░█▀█░█░█░█░░░█▀▀
░▀▀▀░▀░░░▀▀▀░▀░▀░▀▀▀░▀▀▀░▀▀░░▀▀▀   ░█░█░▀▀░░░▀░░█░█░▀░▀░▀▀▀░▀▀▀
```

<sub>Rendered in terminal: <code>OpenCode</code> in ivory (<code>#E8E6E3</code>), <code>ADVANCE</code> in frosted indigo (<code>#6C7AB8</code>). The stylized <code>▟█▙</code> apex on each <code>A</code> references the flat-topped, angular-cut letterform of the Game Boy Advance wordmark. The <code>░</code> stipple characters create anti-aliased frosted-glass edges.</sub>

**A declarative, reproducible OpenCode environment and workflow platform.**

_Status: Phases 1, 2, 3, and 3.5 are implemented on `trunk`. Current next milestone: Phase 4. First stable release remains v1.0._

</div>

---

## What OpenCode Advance is

OpenCode Advance (`oca`) is the configuration, installation, and client/session UX layer for [OpenCode](https://opencode.ai), paired with the [Advance](https://github.com/Sharper-Flow/Advance) spec-driven workflow plugin.

The goal is simple:

- one source of truth: `stack.toml`
- one command to apply it: `oca apply`
- one command to verify it: `oca doctor`

The MCP servers, plugins, instructions, providers, permissions, LSP config, watcher config, theme, and primary client behavior that make up an opinionated OpenCode environment are declared in a single file, rendered to the places OpenCode expects, and tracked for reproducibility.

## Why it exists

The previous iteration of this project (`open-chad`) worked, but had structural limits:

- it only managed part of the real MCP/plugin stack
- it relied heavily on bash for logic that wants strong typing and tests
- it pulled dependencies at `latest` with weak reproducibility
- it duplicated files that the Advance plugin also owned
- its branding and tone no longer matched the intended audience

OpenCode Advance is the clean rewrite:

- **Go-first** for parsing, validation, rendering, health checks, and orchestration
- **declarative** via `stack.toml`
- **reproducible** via pinning and explicit dependency metadata
- **cleanly layered** with Advance as a required dependency, not a tangled peer
- **professional in tone** with an obsidian / slate / graphite palette and frosted indigo accent

## Relationship to Advance

OpenCode Advance and Advance are paired, but separate.

| Project              | Repo                                                                              | Role                                                                    |
| -------------------- | --------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| **Advance**          | [Sharper-Flow/Advance](https://github.com/Sharper-Flow/Advance)                   | Spec-driven workflow plugin: specs, changes, tasks, gates, TDD evidence |
| **OpenCode Advance** | [Sharper-Flow/Opencode-Advance](https://github.com/Sharper-Flow/Opencode-Advance) | Environment, installer, configuration, client/session UX, migration     |

You can run Advance without OpenCode Advance.

You cannot run OpenCode Advance without Advance — Advance is a required dependency declared in `stack.toml`, cloned and wired by `oca`, with ADV-owned assets synced by Advance's own `sync-global.sh`.

## Current status

This repository has **Phase 1 + Phase 2 + Phase 3 + Phase 3.5 implementations** in place.

### Done

#### Phase 1 — foundation

- repo scaffolded
- `go.mod` initialized
- CI scaffold added
- design docs written
- implementation proposal written
- phase plan written
- ADV first-boot guide written
- compact canonical wordmark finalized
- shared runtime brand assets and Go renderer added
- minimal Cobra CLI added (`oca`, `oca version`)
- `stack.toml` parse / resolve / validate implemented for `[meta]` + `[mcp]`
- `oca apply --target mcp` shipped with atomic writes, backups, and locking
- `oca doctor --scope mcp` shipped with Vision `/version` + `/v1/servers` checks
- `oca debug plan` and `oca debug validate` shipped
- MCP render / health / concurrency / JSON-shape test coverage added
- shell brand helpers added (`lib/palette.sh`, `lib/wordmark.sh`, `lib/boot_splash.sh`)
- broader Go + shell verification wiring added

#### Phase 2 — plugin + instruction management

- typed `[plugins.*]`, `[instructions]`, and reserved `[temporal]` sections in `stack.toml` with 6-category `provides` enum
- generic `internal/subprocess` runner with timeout / signal / env-merge semantics
- `internal/plugin` package: git clone/fetch/checkout/status with protocol hardening and ref allowlist, build-step runner, npm literal handler, clone-or-update orchestrator with symlink rejection and remote-URL drift detection
- `internal/render.MergeArray` primitive + `.bak.<epoch>` backup rotation with per-target suppression
- `internal/sync.InvokeAdvance` for post-apply plugin sync with best-effort output redaction
- pluggable health-check registry with `ResetForTesting()` contract
- `oca apply --target plugins --target instructions --target temporal` extensions
- `oca pin [plugin...]` — atomic stack.toml SHA capture under the apply lock
- `oca update [plugin...]` — fetch + checkout + build + sync (with `--force` on pinned refs)
- `oca doctor --scope plugins` with local-only default and `--network` opt-in for remote probes
- integration tests covering plugin apply end-to-end + backup rotation policy
- new `.adv/specs/plugin-apply/` capability spec with 13 rq-* requirements
- `OCA_PLUGIN_CHECKOUT_ROOT` env override documented in `AGENTS.md`
- trust-boundary + secret-redaction docs in `docs/design/stack-toml-schema.md`

#### Phase 3 — core `opencode.json` coverage

- typed `[providers.*]`, `[permissions]`, `[watcher]`, and `[lsp.*]` sections in `stack.toml`
- validation for providers, permissions, watcher, and LSP config
- render modules for `.provider`, `.permission`, `.watcher.ignore`, and `.lsp`
- composed `oca apply` with no `--target`
- `oca diff` with target filtering and text/json output
- NoRollback protection for composed apply
- integration tests for apply-all, diff, and render merge behavior
- docs/spec/example refresh removing agents from shipped Phase 3 scope

#### Phase 3.5 — remaining config coverage

- typed `[skills]`, `[formatters.*]`, `[commands.*]`, and `[opencode]` sections in `stack.toml`
- validation for skills order, reserved `adv-*` names, commands, formatters, and OpenCode toggles
- render modules for `.command`, `.formatter`, top-level OpenCode toggles, and OCA-owned skills asset copy
- `oca apply --target skills|commands|formatters|toggles`
- `oca doctor --scope skills`
- canonical `assets/skills/` inventory populated with OCA-owned skills
- integration tests for composed apply and per-target parity
- Phase 3.5 spec + docs refresh shipped

### Not done yet

- installer / migration / client-session lifecycle logic
- release packaging / distribution workflow
- `oca install`, `oca migrate`, and interactive `oca add` / `oca remove` flows
- Phase 4 tmux/session/theme UX work

### Resume here

- [`NEXT_STEPS.md`](NEXT_STEPS.md) — fastest path to continue from this repo later
- [`STATUS.md`](STATUS.md) — current snapshot of what is done vs. not done
- [`docs/proposals/first-boot.md`](docs/proposals/first-boot.md) — exact ADV startup flow in this repo

## Planned v1.0 scope

At v1.0, OpenCode Advance is intended to provide:

- declarative `stack.toml` parsing and validation **(shipped Phase 1)**
- MCP server rendering into both OpenCode and Vision config **(shipped Phase 1)**
- plugin clone / build / pin / update workflows **(shipped Phase 2)**
- instruction, provider, permission, watcher, LSP, skill, command, formatter, and OpenCode-toggle rendering (instructions shipped Phase 2; providers/permissions/watcher/LSP shipped Phase 3; skills/commands/formatters/toggles shipped Phase 3.5; agents intentionally deferred)
- clean ownership boundaries between OCA-owned and Advance-owned assets
- migration from existing `open-chad` state into `stack.toml`
- primary client/session lifecycle, theme, boot splash, and shell integration
- doctor / diff / debug flows to verify and explain rendered state

## Planned command surface

These commands describe the intended v1.0 UX. Some are already shipped; others remain design targets.

| Command                      | Purpose                                                        |
| ---------------------------- | -------------------------------------------------------------- |
| `oca apply`                  | Render declared stack config into target files                 |
| `oca doctor`                 | Verify rendered environment health                             |
| `oca diff`                   | Show drift between `stack.toml` and rendered state             |
| `oca pin`                    | Capture current plugin refs / SHAs for reproducibility         |
| `oca update`                 | Update dependencies and re-apply                               |
| `oca migrate from-open-chad` | Import current open-chad-managed environment into `stack.toml` |
| `oca session`                | Primary client/session lifecycle management                    |
| `oca debug`                  | Explain plans, validation, and rendered output                 |

## Development model

This repo is built through the Advance spec-driven workflow.

Recommended flow:

1. run `/adv-status` first and finish any already-active implementation change
2. use `phase0FoundationBrand` as the archived reference baseline for future work
3. start the next phase change from `docs/proposals/phases.md` (currently Phase 4: primary client UX + theme)
4. archive each phase before starting the next one

In other words:

- **umbrella change** = long-range tracking and proposal context when needed
- **phase changes** = actual implementation work

## Safe development policy

During development, OpenCode Advance must **not** modify your live OpenCode setup.

Always use isolated directories through environment overrides:

- `OCA_OPENCODE_CONFIG_DIR`
- `OCA_VISION_CONFIG_DIR`
- `OCA_PLUGIN_CHECKOUT_ROOT`
- `OCA_CACHE_DIR`

The intent is that all development and testing happen in a disposable sandbox until v1.0 is ready.

## Repository layout

```text
cmd/oca/                 Go CLI entry point
internal/config/         stack.toml parser + validation
internal/render/         programmatic MCP rendering + merge logic
internal/health/         MCP doctor checks
internal/migrate/        open-chad import path
assets/                  agent / instruction / skill / theme assets
templates/               reserved for later phases (Phase 1 render is programmatic)
lib/                     shell/client UX helpers (palette, wordmark, tmux, splash)
tests/                   integration and shell-level tests
docs/design/             architecture, schema, brand, CLI, theme
docs/proposals/          v1 proposal, phase plan, first-boot guide
.adv/                    ADV specs / changes / archive directories
```

## Requirements

Current intended target environment:

- Linux (Ubuntu / Debian primary target; others best-effort)
- `git`
- `tmux` 3.2+ (current planned primary session runtime)
- OpenCode CLI
- internet access for initial plugin and MCP setup
- `vision` daemon available on `PATH`
- Go 1.22+ for local development

Detailed installation guidance will land closer to v1.0 in `INSTALL.md`.

## Documentation map

### Start here

- [`NEXT_STEPS.md`](NEXT_STEPS.md)
- [`STATUS.md`](STATUS.md)
- [`docs/proposals/first-boot.md`](docs/proposals/first-boot.md)

### Design

- [`docs/design/brand.md`](docs/design/brand.md)
- [`docs/design/wordmark.md`](docs/design/wordmark.md)
- [`docs/design/palette.md`](docs/design/palette.md)
- [`docs/design/theme.md`](docs/design/theme.md)
- [`docs/design/architecture.md`](docs/design/architecture.md)
- [`docs/design/stack-toml-schema.md`](docs/design/stack-toml-schema.md)
- [`docs/design/cli-surface.md`](docs/design/cli-surface.md)

### Planning

- [`docs/proposals/v1-implementation.md`](docs/proposals/v1-implementation.md)
- [`docs/proposals/phases.md`](docs/proposals/phases.md)

### Developer reference

- [`AGENTS.md`](AGENTS.md)
- [`stack.example.toml`](stack.example.toml)

## Quick start for future resume

When you come back later and want to start or continue implementation:

```bash
cd ~/dev/opencodeadvance
opencode
```

Then:

1. `/adv-status` — check for any active changes to complete first
2. Use `phase0FoundationBrand` as the shipped Phase 0 reference point
3. Start the next phase change from `docs/proposals/phases.md` (currently Phase 4)
4. Use `NEXT_STEPS.md` for the exact resume sequence and current state

## Contributing

While implementation is still early, the most meaningful contributions are:

- design clarification
- proposal refinement
- schema review
- roadmap decomposition
- testability review

Once implementation starts, keep commits atomic and use conventional commit messages:

- `feat:`
- `fix:`
- `docs:`
- `test:`
- `refactor:`
- `chore:`

## License

MIT — see [`LICENSE`](LICENSE).
