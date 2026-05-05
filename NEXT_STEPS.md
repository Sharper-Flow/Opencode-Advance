# Next Steps

This file is the fastest way to resume work in `~/dev/opencodeadvance` later.

## Quick answer: what's next on roadmap?

Three tracks, in priority order:

| Priority | Track | What | Blocked by |
|---|---|---|---|
| 1 | **Finish v1.0** | Phase 8 release-candidate validation + tag publication. Phase 8 work is RC-shipped; remaining is smoke test and `git tag v1.0.0`. See `docs/proposals/phases.md` § "Phase 8: Extras + Polish". | Nothing |
| 2 | **Post-v1 OCA reliability queue** | Reviewed/accepted 2026-05-03; extended 2026-05-04 with Advance self-update handoff. File and ship OCA MUST proposals in order: M3 OCA plugin install → M2 bare apply lifecycle parity → M4/M5/M6 instruction assets, starter/migration validity, skill refresh. Then SHOULD proposals: S1/S2 → S3/S4 → S5/S6/S7. See `docs/proposals/phases.md` § "Post-v1: OCA Reliability + Runtime Correctness Queue" and `docs/proposals/2026-05-03-oca-roadmap-queue.md`. | Fresh OpenCode restart + runtime canary proving selected ADV provider agent no longer resolves to `[ADV:PROVIDER_STUB_UNEXPANDED]`. |
| 3 | **Post-v1 session architecture** | 7-change split decision-locked 2026-05-03. Pattern B session topology + graceful hibernation + tmux-resurrect on the OCA side; idle worker reaper + peer-session topology + sync-global prompt-ref fix on the ADV side; OCA umbrella plugin install is now M3 in the OCA reliability queue. See `docs/proposals/phases.md` § "Post-v1: Session & Resource Architecture" + `docs/proposals/2026-05-03-session-and-resource-architecture.md` for the parent decision lock. | ADV prompt runtime canary + OCA reliability MUST queue. |

If asked "what's next" without further context: confirm v1.0 finalization (track 1) before starting post-v1 OCA reliability or session-architecture work.

## Current state

- Repository scaffold exists and is pushed to `origin/trunk`
- Brand, wordmark, palette, architecture, schema, and CLI docs are written
- The compact 3-line pagga wordmark is the canonical form everywhere
- Phases 0, 1, 2, 3, 3.5, 4, 5, 5.5, 6, 6.5, and 7 are archived and merged
- `oca doctor --scope adv-assets` is shipped as out-of-phase hardening for plugin/OCA asset ownership drift
- CI and local verification cover Go tests, vet, builds, and race runs for shipped phases
- **Current focus:** Phase 8 extras/polish
- **Dependency watch:** Current Advance repair/projection-era tooling remains live. A signal-driven workflow refactor is planned upstream but not landed; keep OCA runtime behavior compatible until the readiness gates in `docs/notes/2026-05-05-advance-signal-cutover-readiness.md` pass. See `docs/notes/2026-05-02-advance-plugin-impact-check.md` before current-era cleanup or release prep.
- **Post-v1 staged (2026-05-03; updated 2026-05-04):** Session & resource architecture work decision-locked. Seven change proposals drafted in `docs/proposals/2026-05-03-*.md`. OCA reliability/runtime correctness queue reviewed and accepted into the roadmap (M3 → M2 → M4/M5/M6, then S1/S2 → S3/S4 → S5/S6/S7). S7 is OCA issue `#9`, companion to Advance `#40`: deterministic Advance rebuild + fresh-session/worktree handoff. File proposals only after a fresh OpenCode restart and runtime canary prove a non-stub ADV provider agent.

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
- For current Advance builds, `repairTemporalMigrationDebt` has landed; run `adv_migrate_cleanup` dry-run against OCA before any approved cleanup with backup/commit
- Keep OCA doctor warnings bundle-aware for the current era: warn legacy in-repo ADV state (`.adv/changes`, `.adv/db`, `.adv/agenda*`, non-bundle `.adv/archive` residue) while preserving `.adv/specs/` and valid historical `.adv/archive/*/change.json` bundles
- Do not start the signal-driven OCA runtime refactor until `docs/notes/2026-05-05-advance-signal-cutover-readiness.md` gates pass
- Keep shipped config rendering behavior stable while layering release polish on top
- Keep all writes isolated from live user config

## Post-v1 OCA reliability queue (reviewed 2026-05-03)

OCA-owned MUST/SHOULD queue accepted into the roadmap. Proposal files are ready in `docs/proposals/`; create live ADV changes from a fresh, non-stub ADV session.

| Order | Priority | Change | Proposal |
|---|---|---|---|
| 1 | M3 | OCA umbrella plugin install | `docs/proposals/2026-05-03-oca-plugin-install.md` |
| 2 | M2 | Bare `oca apply` lifecycle parity | `docs/proposals/2026-05-03-oca-apply-lifecycle-parity.md` |
| 3 | M4 | OCA-owned instruction assets are real files | `docs/proposals/2026-05-03-oca-instruction-assets-real.md` |
| 4 | M5 | Starter and migration stack validity | `docs/proposals/2026-05-03-oca-starter-migration-validity.md` |
| 5 | M6 | OCA skill guidance refresh | `docs/proposals/2026-05-03-oca-skill-guidance-refresh.md` |
| 6 | S1 | OCA runtime doctor canaries | `docs/proposals/2026-05-03-oca-runtime-doctor-canaries.md` |
| 7 | S2 | Context budget audit + instruction loading diet | `docs/proposals/2026-05-03-context-budget-audit.md` |
| 8 | S3 | Permission-first agent configuration | `docs/proposals/2026-05-03-agent-permission-first-config.md` |
| 9 | S4 | MCP tool suite profiles and per-agent exposure | `docs/proposals/2026-05-03-mcp-tool-suite-profiles.md` |
| 10 | S5 | Legacy in-repo ADV state doctor warning | `docs/proposals/2026-05-03-legacy-adv-state-doctor-warning.md` |
| 11 | S6 | Resume hint reprint in outer terminal | `docs/proposals/2026-05-04-oca-resume-hint-outer-terminal.md` |
| 12 | S7 | Advance self-update rebuild/session handoff | `docs/proposals/2026-05-04-oca-advance-self-update-handoff.md` |

## Post-v1 staged session/resource work (2026-05-03)

Session & resource architecture work is decision-locked and ready to file as ADV changes after the ADV provider runtime canary passes and the OCA reliability MUST queue is filed/shipped. **7 change proposals** drafted across OCA + ADV repos:

| #     | Change                                                  | Repo            | Effort       |
| ----- | ------------------------------------------------------- | --------------- | ------------ |
| **6** | sync-global single-ref prompt fix (BLOCKER)             | ADV plugin      | 1–2 hours    |
| **0** | OCA umbrella plugin install (PREREQ for #1, #2)         | OCA             | 1–2 hours    |
| 1     | Pattern B + per-window status decode + smart `oca` entry | OCA             | 3–6 days     |
| 2     | Graceful opencode session hibernation                   | OCA             | 2–4 days     |
| 3     | tmux-resurrect (manual) + concurrency warning polish    | OCA             | ~1 day       |
| 4     | Idle Temporal worker reaper                             | ADV plugin      | 1–2 days     |
| 5     | Peer-session topology distinction                       | ADV plugin      | 1–2 hours    |

**Sequence:** ADV #6 first (blocker) → OCA reliability MUST queue (M3/OCA #0 → M2 → M4/M5/M6) → OCA #1 → OCA #2 (parallel #3) → ADV #4 → ADV #5.

**Resume sequence when ADV provider runtime canary passes:**

1. Fresh OpenCode session in `~/dev/oc-plugins/advance` (ADV repo)
2. Verify `opencode debug agent adv-gpt` (or selected provider agent) no longer resolves to `[ADV:PROVIDER_STUB_UNEXPANDED]`
3. File ADV #6 via `/adv-proposal` if not already archived, paste body from [`docs/proposals/2026-05-03-adv-sync-prompt-ref-fix.md`](docs/proposals/2026-05-03-adv-sync-prompt-ref-fix.md)
4. After #6 archives: re-run `bash ~/dev/oc-plugins/advance/scripts/sync-global.sh --fix` and restart OpenCode
5. Move to OCA repo and file OCA reliability MUST queue: M3/OCA #0, M2, then M4/M5/M6
6. Continue OCA session architecture: #1, then #2 and #3
7. Move to ADV repo and file #4, #5 (parallelizable)

**Companion staging:** [`docs/notes/2026-05-03-session-arch-companion-staging.md`](docs/notes/2026-05-03-session-arch-companion-staging.md) holds pre-drafted spec deltas + companion doc updates for each change. The `/adv-apply` phase of each change should pull from this file rather than redrafting from scratch.

**Parent decision lock:** [`docs/proposals/2026-05-03-session-and-resource-architecture.md`](docs/proposals/2026-05-03-session-and-resource-architecture.md) carries the full research trail (memory math, industry consensus, SIGSTOP rejection, Q1–Q13 decisions, and §10 discovery items per change).

## Constraints to keep in mind

- Do **not** write to live user config during development
- Always use isolated config dirs via `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_PLUGIN_CHECKOUT_ROOT`, and `OCA_CACHE_DIR`
- Advance is a required dependency, but OCA must not duplicate Advance-owned assets
- Do not manually delete legacy `.adv/` state; for current Advance builds use landed cleanup tooling, and always preserve `.adv/specs/` plus valid historical `.adv/archive/*/change.json` bundles
- Prefer per-phase ADV changes over one giant implementation change

## Key docs

- `STATUS.md` — project snapshot
- `docs/proposals/first-boot.md` — exact ADV startup flow
- `docs/proposals/v1-implementation.md` — umbrella proposal content
- `docs/proposals/phases.md` — implementation sequencing
- `docs/design/architecture.md` — system shape
- `docs/design/stack-toml-schema.md` — declarative config model
- `stack.example.toml` — complete reference example
