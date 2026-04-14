# Next Steps

This file is the fastest way to resume work in `~/dev/opencodeadvance` later.

## Current state

- Repository scaffold exists and is pushed to `origin/trunk`
- Brand, wordmark, palette, architecture, schema, and CLI docs are written
- The compact 3-line pagga wordmark is the canonical form everywhere
- Phase 0 foundation work is archived and merged to `trunk`
- CI and local verification cover the Phase 0 baseline (`go test`, shell tests, `go vet`, `go build`)
- **Current ADV focus:** start Phase 1 (`stack.toml` + MCP apply)

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
4. Begin with Phase 1 discovery/design/planning, not another Phase 0 change

## Recommended workflow

- Treat `phase0FoundationBrand` as the shipped reference baseline for later phases
- Start the next implementation change for **Phase 1: stack.toml + MCP apply**
- Keep implementation in separate per-phase changes following `docs/proposals/phases.md`
- Archive each phase before starting the next one

## Immediate implementation target (Phase 1)

- define `stack.toml` parser + validation scope in `internal/config/`
- render MCP declarations into isolated OpenCode + Vision config targets
- add early `oca apply --target mcp` and `oca doctor --scope mcp` command surface
- keep all writes isolated from live user config
- do not pull Phase 2 plugin/session/theme work forward

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
