# Next Steps

This file is the fastest way to resume work in `~/dev/opencodeadvance` later.

## Current state

- Repository scaffold exists and is pushed to `origin/trunk`
- Brand, wordmark, palette, architecture, schema, and CLI docs are written
- The compact 3-line pagga wordmark is the canonical form everywhere
- Phase 0 foundation work now exists in the current implementation branch
- CI and local verification now cover Go + shell checks for the Phase 0 baseline
- **Current ADV focus:** finish `phase0FoundationBrand` if it is still active; otherwise start the next phase change

## Resume from here

Open OpenCode in this repo:

```bash
cd ~/dev/opencodeadvance
opencode
```

### If `phase0FoundationBrand` is still active

Resume the current implementation change first:

```
/adv-status
```

Then continue from the first incomplete gate:

- if release is still pending, continue harden / archive for `phase0FoundationBrand`
- if `phase0FoundationBrand` is already archived, move to the next phase change instead of reopening Phase 0

### After `phase0FoundationBrand` is archived

Start the next implementation change from the roadmap (likely Phase 1 config loading/validation), not another fresh Phase 0 scaffold change.

## Recommended workflow

- Finish and archive `phase0FoundationBrand` before starting the next implementation change
- Do implementation in separate per-phase changes following `docs/proposals/phases.md`
- Use `phase0FoundationBrand` as the reference implementation for later phases
- Archive each phase before starting the next one

## Immediate implementation target (Phase 0)

- finish remaining release / archive steps for `phase0FoundationBrand` if it is still active
- preserve the new minimal Cobra CLI and branded version output
- preserve shell helper behavior and CI wiring added in Phase 0
- do not expand Phase 0 into parser/render/apply scope

## Constraints to keep in mind

- Do **not** write to live user config during development
- Always use isolated config dirs via `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_PLUGIN_CHECKOUT_ROOT`, and `OCA_CACHE_DIR`
- Advance is a required dependency, but OCA must not duplicate Advance-owned assets
- Prefer per-phase ADV changes over one giant implementation change

## Key docs

- `STATUS.md` — project snapshot
- `docs/proposals/first-boot.md` — exact ADV startup flow
- `docs/proposals/v1-implementation.md` — umbrella proposal content (refreshed)
- `docs/proposals/phases.md` — implementation sequencing (refreshed, includes Phase 3.5)
- `docs/design/architecture.md` — system shape
- `docs/design/stack-toml-schema.md` — declarative config model (refreshed)
- `stack.example.toml` — complete reference example (refreshed)
