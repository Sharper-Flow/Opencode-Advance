# Next Steps

This file is the fastest way to resume work in `~/dev/opencodeadvance` later.

## Current state

- Repository scaffold exists and is pushed to `origin/trunk`
- Brand, wordmark, palette, architecture, schema, and CLI docs are written
- The compact 3-line pagga wordmark is the canonical form everywhere
- Phase 0 foundation work is archived and merged to `trunk`
- Phase 1 parser + MCP apply/doctor/debug work is archived and merged
- Phase 2 plugin + instruction management is archived and merged
- Phase 3 providers/permissions/watcher/LSP rendering, composed apply, and `oca diff` are archived and merged
- Phase 3.5 skills/commands/formatters/OpenCode toggles rendering and `oca doctor --scope skills` are archived and merged
- CI and local verification cover Go tests, vet, builds, and race runs for shipped phases
- **Current ADV focus:** Phase 4 foundation in progress (`phase4FoundationSessionTheme` — 8/11 tasks done)

## Resume from here

Open OpenCode in this repo:

```bash
cd ~/dev/opencodeadvance
opencode
```

Then:

1. Run `/adv-status`
2. Confirm there are no new active changes to finish first
3. Start the next implementation change from the roadmap in `docs/proposals/phases.md`
4. Complete the active Phase 4 foundation change if not yet archived

## Recommended workflow

- Treat `phase0FoundationBrand` as the shipped reference baseline for later phases
- Treat `phase1StackTomlParserMcpApply`, `phase2PluginInstruction`, and `phase3CoreOpencodeJsonCoverage` as archived shipped references
- Start the next implementation change for **Phase 4 richness** (after foundation is archived)
- Keep implementation in separate per-phase changes following `docs/proposals/phases.md`
- Archive each phase before starting the next one

## Immediate implementation target (Phase 4)

- **Foundation (in progress):** complete integration tests, docs update, and spec for `phase4FoundationSessionTheme`
- **Richness (next):** status bar metrics, LLM fuel gauges, session attach/switch/killall/restart, boot splash animation, `oca theme` commands
- keep shipped config rendering behavior stable while layering UX/runtime work on top
- keep all writes isolated from live user config

## Constraints to keep in mind

- Do **not** write to live user config during development
- Always use isolated config dirs via `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_PLUGIN_CHECKOUT_ROOT`, and `OCA_CACHE_DIR`
- Advance is a required dependency, but OCA must not duplicate Advance-owned assets
- Prefer per-phase ADV changes over one giant implementation change

## Key docs

- `STATUS.md` — project snapshot
- `docs/proposals/first-boot.md` — exact ADV startup flow
- `docs/proposals/v1-implementation.md` — umbrella proposal content
- `docs/proposals/phases.md` — implementation sequencing
- `docs/design/architecture.md` — system shape
- `docs/design/stack-toml-schema.md` — declarative config model
- `stack.example.toml` — complete reference example
