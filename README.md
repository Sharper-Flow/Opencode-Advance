<div align="center">

```
░█▀█░█▀█░█▀▀░█▀█░█▀▀░█▀█░█▀▄░█▀▀   ░▟█▙░█▀▄░█░█░▟█▙░█▀█░█▀▀░█▀▀
░█░█░█▀▀░█▀▀░█░█░█░░░█░█░█░█░█▀▀   ░█▀█░█░█░▀▄▀░█▀█░█░█░█░░░█▀▀
░▀▀▀░▀░░░▀▀▀░▀░▀░▀▀▀░▀▀▀░▀▀░░▀▀▀   ░█░█░▀▀░░░▀░░█░█░▀░▀░▀▀▀░▀▀▀
```

<sub>Rendered in terminal: `OpenCode` in ivory (`#E8E6E3`), `ADVANCE` in frosted indigo (`#6C7AB8`). The stylized `▟█▙` apex on each `A` references the flat-topped, angular-cut letterform of the Game Boy Advance wordmark. The `░` stipple characters create anti-aliased frosted-glass edges.</sub>

**A declarative, reproducible OpenCode environment and workflow platform.**

*Status: v0 — under active development. First stable release will be v1.0.*

</div>

---

## What OpenCode Advance is

OpenCode Advance (`oca`) is the configuration, installation, and session management layer for [OpenCode](https://opencode.ai) — paired tightly with the [Advance](https://github.com/Sharper-Flow/Advance) spec-driven workflow plugin.

One source of truth (`stack.toml`). One command to apply it (`oca apply`). One command to verify it (`oca doctor`). The MCP servers, plugins, instructions, providers, agent model assignments, permissions, LSP configs, and themes that make up an opinionated AI-assisted development environment — all declared in a single file, rendered into the places OpenCode expects them, and versioned for reproducibility.

## Why it exists

The previous iteration of this project (`open-chad`) was a bash-based installer and tmux environment. It worked, but it had structural limits:

- It managed a fraction of the real MCP/plugin stack, leaving the rest to hand-edit
- It pulled plugin dependencies at `latest` with no reproducibility
- It duplicated files that the Advance plugin also owned, creating sync conflicts
- Its visual identity was playful in a way that did not match the intended audience

OpenCode Advance is a clean rewrite in Go. The stack is declarative. The Advance plugin is a declared dependency, never duplicated. The visual identity is professional, with a signature wordmark inspired by the Game Boy Advance and an obsidian/indigo palette.

## Core concepts

| Concept          | What it means                                                                                                                       |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| **`stack.toml`**     | Single source of truth. Declares MCP servers, plugins, instructions, providers, agents, permissions, LSP, watcher, and session UX. |
| **`oca apply`**      | Idempotent rendering of `stack.toml` into `~/.config/opencode/opencode.json`, `~/.config/vision/servers.yaml`, and related config files. |
| **`oca doctor`**     | Live verification of every declared component: MCP ports responding, plugins built, instructions resolvable, agent models reachable. |
| **`oca diff`**       | Drift detection between `stack.toml` and the rendered state on disk.                                                                  |
| **`oca pin`**        | Capture current plugin git SHAs into `stack.toml` for reproducible builds.                                                            |
| **`oca session`**    | Tmux session lifecycle — create, list, attach, switch, restart, killall.                                                            |
| **Obsidian theme**   | Obsidian/slate/graphite base palette with a signature frosted indigo accent (`#6C7AB8` → `#8B9FE0`).                                    |
| **Advance plugin**   | Declared dependency. `oca` clones, builds, and wires it — and delegates ADV asset sync to Advance's own `sync-global.sh`.               |

## Status

This repository is a scaffold. The v1.0 implementation has not started yet. Everything here is design, proposal, and planning. See:

- [`docs/proposals/v1-implementation.md`](docs/proposals/v1-implementation.md) — the full v1.0 implementation proposal
- [`docs/proposals/phases.md`](docs/proposals/phases.md) — phase sequencing and dependencies
- [`docs/design/brand.md`](docs/design/brand.md) — brand identity, wordmark, palette
- [`docs/design/architecture.md`](docs/design/architecture.md) — system architecture overview
- [`docs/design/stack-toml-schema.md`](docs/design/stack-toml-schema.md) — `stack.toml` schema reference
- [`docs/design/cli-surface.md`](docs/design/cli-surface.md) — `oca` command reference
- [`stack.example.toml`](stack.example.toml) — complete example `stack.toml` grounded in real usage
- [`AGENTS.md`](AGENTS.md) — internal reference for developers and AI agents

## Relationship to Advance

OpenCode Advance and Advance are paired but separate:

| Project                | Repo                                                                                 | Role                                                                        |
| ---------------------- | ------------------------------------------------------------------------------------ | --------------------------------------------------------------------------- |
| **Advance**                | [Sharper-Flow/Advance](https://github.com/Sharper-Flow/Advance)                            | Spec-driven workflow plugin — specs, changes, tasks, gates, TDD evidence. Standalone. |
| **OpenCode Advance**       | [Sharper-Flow/Opencode-Advance](https://github.com/Sharper-Flow/Opencode-Advance) (this)   | Environment, installer, configuration, session management. Requires Advance.  |

You can run Advance without OpenCode Advance. You cannot run OpenCode Advance without Advance — it is a required dependency, declared in `stack.toml` and installed by `oca install`.

## Requirements

- Linux (Ubuntu / Debian primary target; other distros best-effort)
- `git`
- `tmux` 3.2+
- OpenCode CLI
- Internet access for initial plugin/MCP server installation
- `vision` daemon binary on PATH (installed separately, required for MCP server lifecycle)

Detailed requirements and optional dependencies will be documented in `INSTALL.md` as v1.0 approaches.

## Development

This repo is driven through the Advance spec-driven workflow. To contribute:

1. Clone the repo
2. Make sure Advance is installed and working in your OpenCode environment
3. Open OpenCode in this repo directory
4. Run `/adv-status` — this initializes the ADV state for this project
5. Run `/adv-proposal` to start a new change, or pick up the existing v1 implementation proposal

See [`docs/proposals/first-boot.md`](docs/proposals/first-boot.md) for the exact steps to initialize the v1.0 implementation work.

## License

MIT — see [`LICENSE`](LICENSE).
