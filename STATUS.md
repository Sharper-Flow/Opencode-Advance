# Project Status

## Summary

OpenCode Advance currently has a **completed Phase 0 foundation baseline** and is ready for **Phase 1: stack.toml + MCP apply**.

- Implementation status: **Phase 0 archived and merged to `trunk`**
- Repository status: **ready for next phase planning**
- Recommended next action: start the Phase 1 change for config parsing, MCP rendering, and isolated apply/doctor foundations

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
- Shell brand helpers added (`lib/palette.sh`, `lib/wordmark.sh`, `lib/boot_splash.sh`)
- Broader Phase 0 verification added (Go + shell + CI wiring)
- `phase0FoundationBrand` archived and merged to `trunk`

## What is not done

- No ADV umbrella change yet
- No `stack.toml` parser or schema validation implementation yet
- No render / apply / doctor command implementation yet
- No installer / migration / session lifecycle implementation yet

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

If you are returning to this repo later, start with `NEXT_STEPS.md`, verify there are no new active changes, and begin the Phase 1 change.
