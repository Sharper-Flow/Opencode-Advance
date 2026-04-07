# Project Status

## Summary

OpenCode Advance is currently at the **scaffold + design** stage.

- Implementation status: **not started**
- Repository status: **ready to resume**
- Recommended next action: create the initial ADV changes and begin **Phase 0**

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

## What is not done

- No ADV umbrella change yet
- No Phase 0 change yet
- No real CLI implementation beyond scaffold placeholder
- No rendering / apply / doctor logic yet
- No shell integration scripts yet
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

If you are returning to this repo later, start with `NEXT_STEPS.md`.
