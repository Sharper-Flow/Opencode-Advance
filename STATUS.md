# Project Status

## Summary

OpenCode Advance currently has **completed Phases 0, 1, 2, 3, 3.5, 4, 5, 5.5, 6, 6.5, and 7** on `trunk`. **Phase 8 extras/polish is next**.

- Implementation status: **Phase 7 migration + doctor expansion and Phase 6.5 Temporal supervision are archived, merged, and pushed**
- Repository status: **config rendering ships for MCP, plugins, instructions, providers, permissions, watcher, LSP, skills, commands, formatters, OpenCode toggles, Temporal, and MCP slot groups; doctor scopes ship for mcp, plugins, skills, temporal, adv-assets, adv-plugin, and cross**
- Recommended next action: **start Phase 8 extras/polish**
- Latest dependency note: **Advance Temporal migration repair tooling has landed; OCA should use bundle-aware diagnostics and `adv_migrate_cleanup` for legacy `.adv/` residue, preserving `.adv/specs/` and valid `.adv/archive/*/change.json` bundles**. See `docs/notes/2026-05-02-advance-plugin-impact-check.md`.
- **Post-v1 staged (2026-05-03):** OCA reliability/runtime correctness queue reviewed and accepted into the roadmap (10 OCA proposal drafts: M3 → M2 → M4/M5/M6, then S1/S2 → S3/S4 → S5). Session & resource architecture remains decision-locked with **7 change proposals drafted** across OCA + ADV repos; broad OCA session work starts after the OCA reliability MUST queue. Live ADV proposal creation is deferred until a fresh OpenCode restart and runtime canary prove a non-stub ADV provider agent. See [`docs/proposals/2026-05-03-oca-roadmap-queue.md`](docs/proposals/2026-05-03-oca-roadmap-queue.md), [`docs/proposals/2026-05-03-session-and-resource-architecture.md`](docs/proposals/2026-05-03-session-and-resource-architecture.md), and [`docs/proposals/phases.md`](docs/proposals/phases.md).

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
- Phase 4 primary client UX/theme/session lifecycle shipped (session lifecycle, theme assets, tmux status bar, boot splash, `oca theme`, attach/switch/killall/restart)
- Phase 5 Temporal enablement shipped
- Phase 5.5 Vision slot group support shipped
- `oca doctor --scope adv-assets` shipped as out-of-phase hardening for plugin/OCA asset ownership drift
- Phase 6 installer/shell profile/completion shipped
- Phase 6.5 Temporal dev-server supervision shipped (`oca temporal status/start/stop/restart/logs`)
- Phase 7 migration + doctor expansion shipped (`oca migrate from-open-chad`, `oca migrate init`, `adv-plugin`, and `cross` doctor scopes)

## What is not done

- No release packaging workflow yet
- Phase 8 extras/polish not done
- OCA has not yet executed Advance's landed `adv_migrate_cleanup`; dry-run found legacy `.adv/changes` and archive residue to revisit with bundle-aware diagnostics before any approved cleanup
- **Post-v1 OCA reliability queue staged but not yet filed via `/adv-proposal`** — 10 OCA-owned proposal drafts accepted into the roadmap and sequenced before broad session architecture.
- **Post-v1 session-architecture work staged but not yet filed via `/adv-proposal`** — 7 changes drafted (`docs/proposals/2026-05-03-*.md`) waiting on ADV provider runtime prompt health before drafting can resume from a fresh OpenCode session with working ADV orchestrator persona

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

If you are returning to this repo later, start with `NEXT_STEPS.md`, verify there is no newer active change, review the latest Advance impact note, and either:

1. Start Phase 8 extras/polish from `docs/proposals/phases.md` (v1.0 release path), OR
2. If a fresh OpenCode session verifies the selected ADV provider agent no longer resolves to `[ADV:PROVIDER_STUB_UNEXPANDED]`: begin filing the accepted OCA reliability queue per `docs/proposals/phases.md` § "Post-v1: OCA Reliability + Runtime Correctness Queue", then the 7 post-v1 session-architecture proposals per § "Post-v1: Session & Resource Architecture".
