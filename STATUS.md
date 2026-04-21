# Project Status

## Summary

OpenCode Advance currently has **completed Phases 0, 1, 2, 3, and 3.5** on `trunk`.

- Implementation status: **Phase 3.5 archived and merged**
- Repository status: **config rendering now ships for MCP, plugins, instructions, providers, permissions, watcher, LSP, skills, commands, formatters, and OpenCode toggles, plus `oca diff`, composed apply, and `oca doctor --scope skills`**
- Recommended next action: start **Phase 4** from `docs/proposals/phases.md`

## What is done

- Repository created and pushed
- Go module scaffolded
- CI scaffold added
- README and AGENTS reference written
- Full design docs written:
  - brand
  - wordmark
  - palette
  - theme
  - architecture
  - stack schema
  - CLI surface
- Planning docs written:
  - v1 implementation proposal
  - phase sequencing
  - first-boot / ADV initialization guide
- Wordmark unified to the compact 3-line pagga form
- Shared runtime brand assets + Go renderer added
- Minimal Cobra CLI added (`oca`, `oca version`)
- `stack.toml` parser, resolver, validator, and deferred-section handling added
- `oca apply --target mcp` added with atomic writes, backups, and flock locking
- `oca doctor --scope mcp` added with Vision capability negotiation and per-server checks
- `oca debug plan` and `oca debug validate` added
- Phase 1 design/spec docs updated to match shipped code
- Shell brand helpers added (`lib/palette.sh`, `lib/wordmark.sh`, `lib/boot_splash.sh`)
- Broader Phase 0 verification added (Go + shell + CI wiring)
- `phase0FoundationBrand` archived and merged to `trunk`
- Phase 2 plugin lifecycle, instruction rendering, sync delegation, pin/update, and plugin doctor shipped
- Phase 3 providers/permissions/watcher/LSP rendering shipped
- Phase 3.5 skills/commands/formatters/OpenCode toggles rendering + skills doctor shipped
- `oca diff` shipped
- composed no-target `oca apply` shipped
- agents explicitly kept out of shipped Phase 3 scope

## What is not done

- No installer / migration / session lifecycle implementation yet
- No release packaging workflow yet
- No Phase 4 primary client/session/theme implementation yet

## Decision log snapshot

- New repo: `Sharper-Flow/Opencode-Advance`
- Default branch: `trunk`
- Language: Go 1.22+
- CLI style: cobra
- Config source of truth: `stack.toml`
- Advance remains separate and standalone
- OCA depends on Advance, not the other way around
- Canonical wordmark: compact 3-line pagga form with stylized `A`
- Development must use isolated config targets, never live `~/.config/opencode/`

## Resume guidance

If you are returning to this repo later, start with `NEXT_STEPS.md`, verify there is no newer active change, and begin the Phase 4 change.
