# Next Steps

This file is the fastest way to resume work in `~/dev/opencodeadvance` later.

## Current state

- Repository scaffold exists and is pushed to `origin/trunk`
- Brand, wordmark, palette, architecture, schema, and CLI docs are written
- The compact 3-line pagga wordmark is the canonical form everywhere
- CI verifies the scaffold builds and tests cleanly
- No real implementation work has started yet
- No ADV change has been created in this repo yet

## Resume from here

Open OpenCode in this repo and start the ADV workflow:

```bash
cd ~/dev/opencodeadvance
opencode
```

Then run these in order:

1. `/adv-status`
2. `/adv-proposal OpenCode Advance v1.0 — clean rewrite of open-chad as a declarative Go-based configuration platform`
3. Use `docs/proposals/v1-implementation.md` as the proposal body
4. `/adv-proposal Phase 0: Foundation + brand — go.mod, wordmark render, palette, boot splash`
5. `/adv-research opencodeAdvancePhase0`
6. `/adv-prep opencodeAdvancePhase0`
7. `/adv-apply opencodeAdvancePhase0`

## Recommended workflow

- Keep `opencodeAdvanceV1` as the umbrella / tracking change
- Do implementation in separate per-phase changes
- Start with **Phase 0** as its own change
- Archive each phase before starting the next one

## Immediate implementation target

Phase 0 deliverables:

- add `cobra`
- replace the scaffold CLI with a minimal `oca version` flow
- implement `lib/palette.sh`
- implement `lib/wordmark.sh`
- implement `lib/boot_splash.sh` (minimal, no fancy animation required yet)
- keep CI green

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
