# Next Steps

This file is the fastest way to resume work in `~/dev/opencodeadvance` later.

## Current state

- Repository scaffold exists and is pushed to `origin/trunk`
- Brand, wordmark, palette, architecture, schema, and CLI docs are written
- The compact 3-line pagga wordmark is the canonical form everywhere
- Phases 0, 1, 2, 3, 3.5, 4, 5, and 5.5 are archived and merged
- `oca doctor --scope adv-assets` is shipped as out-of-phase hardening for plugin/OCA asset ownership drift
- CI and local verification cover Go tests, vet, builds, and race runs for shipped phases
- **Current ADV focus:** Phase 6 installer/shell profile/completion work

## Resume from here

Open OpenCode in this repo:

```bash
cd ~/dev/opencodeadvance
opencode
```

Then:

1. Run `/adv-status`
2. Confirm there are no new active changes to finish first
3. Continue Phase 6 work, or start the next implementation change from `docs/proposals/phases.md`

## Recommended workflow

- Treat archived phase changes as shipped references
- Do not reimplement `adv-assets` in Phase 7; build migration/ADV/cross-component doctor checks on top of existing scopes
- Keep implementation in separate per-phase changes following `docs/proposals/phases.md`
- Archive each phase before starting the next one

## Immediate implementation target (Phase 6)

- Finish installer/shell profile hardening and documentation
- Prepare Phase 7 migration + expanded doctor checks after Phase 6 is release-ready
- Keep shipped config rendering behavior stable while layering installer/migration work on top
- Keep all writes isolated from live user config

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
