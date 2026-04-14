# Next Steps

This file is the fastest way to resume work in `~/dev/opencodeadvance` later.

## Current state

- Repository scaffold exists and is pushed to `origin/trunk`
- Brand, wordmark, palette, architecture, schema, and CLI docs are written
- The compact 3-line pagga wordmark is the canonical form everywhere
- CI verifies the scaffold builds and tests cleanly
- No real implementation work has started yet
- **Active ADV change:** `refreshOcaPlanningDocs` — planning doc refresh (complete through planning gate)

## Resume from here

Open OpenCode in this repo:

```bash
cd ~/dev/opencodeadvance
opencode
```

### If planning doc refresh is not yet archived

Check status and finish the `refreshOcaPlanningDocs` change first:

```
/adv-status
```

Then apply, review, and archive it before starting Phase 0.

### After planning doc refresh is archived — start Phase 0

1. `/adv-proposal Phase 0: Foundation + brand — go.mod, wordmark render, palette, boot splash`
2. `/adv-discover opencodeAdvancePhase0`
3. `/adv-agree opencodeAdvancePhase0`
4. `/adv-design opencodeAdvancePhase0`
5. `/adv-prep opencodeAdvancePhase0`
6. `/adv-apply opencodeAdvancePhase0`

## Recommended workflow

- Keep `refreshOcaPlanningDocs` archived before starting any implementation change
- Do implementation in separate per-phase changes following `docs/proposals/phases.md`
- Start with **Phase 0** as the first implementation change
- Archive each phase before starting the next one

## Immediate implementation target (Phase 0)

- initialize `go.mod` and add `cobra` dependency
- replace the scaffold CLI with a minimal `oca version` subcommand
- implement `lib/palette.sh` with all color constants
- implement `lib/wordmark.sh` with render function
- implement `lib/boot_splash.sh` (minimal — no animation yet, just render the wordmark)
- keep CI green

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
