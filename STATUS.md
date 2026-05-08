# Project Status

## Summary

OpenCode Advance has **completed Phases 0–8** on `trunk`. v1.0 release-candidate work is landed; tag publication pending.

- Implementation status: **All core phases archived and merged.** Operator dashboard, pane management, session watchdog, Temporal CLI detection shipped as out-of-phase additions.
- Repository status: **Full config rendering** (MCP, plugins, instructions, providers, permissions, watcher, LSP, skills, commands, formatters, OpenCode toggles, Temporal, MCP slot groups). **Doctor scopes** (mcp, plugins, skills, temporal, adv-assets, adv-plugin, cross, adv-runtime).
- Search attributes aligned with current Advance (10 attributes matching `plugin/src/temporal/search-attributes.ts`).
- Recommended next action: **v1.0 tag + release**, or resume post-v1 reliability queue.
- Latest dependency note: **Current Advance uses the Temporal signal-driven era.** OCA search attributes and workflow queries are aligned. Signal-cutover readiness gates in `docs/notes/2026-05-05-advance-signal-cutover-readiness.md` track future schema-v2 migration.

## What is done

- Repository created, Go module scaffolded, CI + release workflows
- Full design docs: brand, wordmark, palette, theme, architecture, stack schema, CLI surface
- Phase 0: brand (wordmark, palette, boot splash, Discord taglines)
- Phase 1: stack.toml parser + MCP apply + Vision rendering
- Phase 2: plugin lifecycle, instruction rendering, sync delegation, pin/update
- Phase 3: providers, permissions, watcher, LSP rendering
- Phase 3.5: skills, commands, formatters, OpenCode toggles + skills doctor
- Phase 4: session lifecycle, theme assets, tmux status bar, boot splash, `oca theme`
- Phase 5: Temporal enablement
- Phase 5.5: Vision slot group support
- Phase 6: installer, shell profile, completion
- Phase 6.5: Temporal dev-server supervision
- Phase 7: open-chad migration, doctor expansion
- Phase 8: extras, polish, Discord, Pattern B session topology
- Out-of-phase: operator dashboard, pane management, session watchdog, Temporal CLI detection
- OCA umbrella plugin (TypeScript, bun-built)
- Maintenance planner (merge candidates, rebuilds, cleanup)

## What is not done

- Release packaging workflow (`release.yml` ready, tag not published)
- Post-v1 OCA reliability queue (12 proposals staged, not yet filed)
- Post-v1 session-architecture work (7 proposals staged, not yet filed)

## Decision log snapshot

- New repo: `Sharper-Flow/Opencode-Advance`
- Default branch: `trunk`
- Language: Go 1.22+
- CLI style: cobra
- Config source of truth: `stack.toml`
- Advance remains separate and standalone
- OCA depends on Advance, not the other way around
- Development must use isolated config targets, never live `~/.config/opencode/`

## Resume guidance

Start with `NEXT_STEPS.md`, review `docs/notes/2026-05-05-advance-signal-cutover-readiness.md` for Advance signal-driven migration status, then either:

1. Publish v1.0 tag (`git tag v1.0.0 && git push origin v1.0.0`), OR
2. Begin filing the accepted OCA reliability queue per `docs/proposals/phases.md` § "Post-v1: OCA Reliability + Runtime Correctness Queue".
