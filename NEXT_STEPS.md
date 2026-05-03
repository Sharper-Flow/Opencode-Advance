# Next Steps

This file is the fastest way to resume work in `~/dev/opencodeadvance` later.

## Current state

- Repository scaffold exists and is pushed to `origin/trunk`
- Brand, wordmark, palette, architecture, schema, and CLI docs are written
- The compact 3-line pagga wordmark is the canonical form everywhere
- Phases 0, 1, 2, 3, 3.5, 4, 5, 5.5, 6, 6.5, and 7 are archived and merged
- `oca doctor --scope adv-assets` is shipped as out-of-phase hardening for plugin/OCA asset ownership drift
- CI and local verification cover Go tests, vet, builds, and race runs for shipped phases
- **Current focus:** Phase 8 extras/polish
- **Dependency watch:** Advance has in-progress Temporal migration repair work. See `docs/notes/2026-05-02-advance-plugin-impact-check.md` before cleanup or release prep.
- **Post-v1 staged (2026-05-03):** Session & resource architecture work decision-locked. Seven change proposals drafted in `docs/proposals/2026-05-03-*.md`. Awaiting ADV blocker change #6 (`syncGlobalPromptRefSingleFile`) to ship before any of the 7 can be filed via `/adv-proposal`. Workaround in place locally so the next fresh OpenCode session has a working ADV orchestrator agent.

## Resume from here

Open OpenCode in this repo:

```bash
cd ~/dev/opencodeadvance
opencode
```

Then:

1. Run `/adv-status`
2. Confirm there are no new active changes to finish first
3. Review `docs/notes/2026-05-02-advance-plugin-impact-check.md` for Advance plugin changes that may affect OCA
4. Start Phase 8 extras/polish from `docs/proposals/phases.md`

## Recommended workflow

- Treat archived phase changes as shipped references
- Do not reimplement `adv-assets` in Phase 7; build migration/ADV/cross-component doctor checks on top of existing scopes
- Keep implementation in separate per-phase changes following `docs/proposals/phases.md`
- Archive each phase before starting the next one

## Immediate implementation target (Phase 8)

- Add release packaging and remaining polish from `docs/proposals/phases.md`
- Re-check Advance `repairTemporalMigrationDebt` before release prep; if landed, run `adv_migrate_cleanup` dry-run against OCA and consider cleanup with backup/commit
- Consider an OCA doctor warning for legacy in-repo ADV state (`.adv/changes`, `.adv/archive`, `.adv/db`, `.adv/agenda*`) while preserving `.adv/specs/`
- Keep shipped config rendering behavior stable while layering release polish on top
- Keep all writes isolated from live user config

## Post-v1 staged work (2026-05-03)

Session & resource architecture work is decision-locked and ready to file as ADV changes once the blocker ships. **7 change proposals** drafted across OCA + ADV repos:

| #     | Change                                                  | Repo            | Effort       |
| ----- | ------------------------------------------------------- | --------------- | ------------ |
| **6** | sync-global single-ref prompt fix (BLOCKER)             | ADV plugin      | 1–2 hours    |
| **0** | OCA umbrella plugin install (PREREQ for #1, #2)         | OCA             | 1–2 hours    |
| 1     | Pattern B + per-window status decode + smart `oca` entry | OCA             | 3–6 days     |
| 2     | Graceful opencode session hibernation                   | OCA             | 2–4 days     |
| 3     | tmux-resurrect (manual) + concurrency warning polish    | OCA             | ~1 day       |
| 4     | Idle Temporal worker reaper                             | ADV plugin      | 1–2 days     |
| 5     | Peer-session topology distinction                       | ADV plugin      | 1–2 hours    |

**Sequence:** ADV #6 first (blocker) → OCA #0 (prereq) → OCA #1 → OCA #2 (parallel #3) → ADV #4 → ADV #5.

**Resume sequence when ADV #6 ships:**

1. Fresh OpenCode session in `~/dev/oc-plugins/advance` (ADV repo)
2. Switch to `adv-claude` agent (workaround C in place — agent works post-restart)
3. File ADV #6 via `/adv-proposal`, paste body from [`docs/proposals/2026-05-03-adv-sync-prompt-ref-fix.md`](docs/proposals/2026-05-03-adv-sync-prompt-ref-fix.md)
4. After #6 archives: re-run `bash ~/dev/oc-plugins/advance/scripts/sync-global.sh --fix` (will produce single-ref prompt form natively, workaround C becomes redundant)
5. Move to OCA repo and file #0, then #1, #2, #3 in sequence
6. Move to ADV repo and file #4, #5 (parallelizable)

**Companion staging:** [`docs/notes/2026-05-03-session-arch-companion-staging.md`](docs/notes/2026-05-03-session-arch-companion-staging.md) holds pre-drafted spec deltas + companion doc updates for each change. The `/adv-apply` phase of each change should pull from this file rather than redrafting from scratch.

**Parent decision lock:** [`docs/proposals/2026-05-03-session-and-resource-architecture.md`](docs/proposals/2026-05-03-session-and-resource-architecture.md) carries the full research trail (memory math, industry consensus, SIGSTOP rejection, Q1–Q13 decisions, and §10 discovery items per change).

## Constraints to keep in mind

- Do **not** write to live user config during development
- Always use isolated config dirs via `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_PLUGIN_CHECKOUT_ROOT`, and `OCA_CACHE_DIR`
- Advance is a required dependency, but OCA must not duplicate Advance-owned assets
- Do not manually delete legacy `.adv/` state; use Advance cleanup tooling after it lands, and always preserve `.adv/specs/`
- Prefer per-phase ADV changes over one giant implementation change

## Key docs

- `STATUS.md` — project snapshot
- `docs/proposals/first-boot.md` — exact ADV startup flow
- `docs/proposals/v1-implementation.md` — umbrella proposal content
- `docs/proposals/phases.md` — implementation sequencing
- `docs/design/architecture.md` — system shape
- `docs/design/stack-toml-schema.md` — declarative config model
- `stack.example.toml` — complete reference example
