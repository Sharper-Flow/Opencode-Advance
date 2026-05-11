## Problem Statement

The MUST queue from the 2026-05-03 OCA reliability handoff (`docs/proposals/2026-05-03-oca-roadmap-queue.md`) defined four pre-cutover correctness blockers (M2, M4, M5, M6) that must ship before OpenCode Advance can safely replace open-chad as the operator's daily driver.

Re-audit on 2026-05-11 against current trunk found:

- **M2 (`applyLifecycleParity`)** — already shipped 2026-05-04 as `fixBareOcaApplyLifecycleParity`. The 2026-05-03 proposal is now marked SUPERSEDED.
- **M3 (`installOcaUmbrellaPlugin`)** — shipped, archived 2026-05-11 in this session.
- **M4 (`instructionAssetsReal`)** — partially shipped via groundwork commit `8a747bd` which copied 11 instruction files into `assets/instructions/`. Three of the four pre-flight-flagged live files remain un-triaged: `criteria-prioritizer.md`, `global-verify-policy.md`, `post_install_verification.md`. README↔asset-dir parity check is missing.
- **M5 (`starterMigrationValidity`)** — partially shipped. `oca migrate init` now emits a valid stack that passes `cfg.Load()`. But `oca migrate from-open-chad` against live state produces an invalid TOML stack — npm version specs like `opencode-openai-codex-auth@latest` leak directly into TOML table keys (`[plugins.opencode-openai-codex-auth@latest]`), producing `expected '.' or ']' to end table name, but got '@'` at parse time. This is a real cutover blocker — the operator cannot migrate from their live open-chad state without hand-editing the output.
- **M6 (`skillGuidanceRefresh`)** — content fixed (no stale `openchad` refs, `mcp-selection` uses `kagi_kagi_search_fetch`/`firecrawl_firecrawl_scrape` schema-current names, README "Not in this directory" updated with inlined-and-deleted note). But no docs/test enforcement layer exists to catch future regressions.

Net residual: **one hard bug (M5 npm-spec key sanitization) and two polish items (M4 triage + parity check, M6 enforcement tests)**. Consolidating into a single change for one 7-gate pass — same scale of work, same M-queue cleanup surface.

### Evidence

- `2026-05-03-oca-apply-lifecycle-parity.md` lines 1-22 (now marked SUPERSEDED)
- `cmd/oca/apply.go:170` (`pluginpkg.Prepare`), `:195` (`invokeAdvance`), `:85-91` (`defaultApplyTargets` includes temporal)
- `cmd/oca/apply_test.go:137,225,261` (three bare-apply regression tests)
- `assets/instructions/` — 12 entries (11 instruction files + README)
- `~/.config/opencode/instructions/` — 14 entries; 3 not yet copied/triaged
- `internal/migrate/init.go:50` (path field added to advance plugin)
- Reproduced 2026-05-11: `oca migrate from-open-chad --output /tmp/oca-migrate-test.toml && oca debug validate --config /tmp/oca-migrate-test.toml` → `parse: line 150 (last key "plugins.opencode-plugin"): expected '.' or ']' to end table name, but got '@' instead`
- `assets/skills/README.md:34` (inlined-and-deleted note present)
- `assets/skills/mcp-selection/SKILL.md:30,57` (schema-current names confirmed)

## Why this is the right scope

These three residuals share:

1. **Same M-queue cleanup surface** — all originated from the same 2026-05-03 handoff, all are pre-cutover safety nets.
2. **Same effort scale** — each is ~half a day; combined ~1-1.5 days.
3. **Same release window** — none should ship without the others; cutover proceeds only when all three are clean.
4. **Same test pattern** — all three benefit from automated parity/banned-term tests that protect future regressions.

Splitting into three changes would triple the gate overhead for the same delivered value. Consolidating respects "scope validity once prep gate passes" — trust the planned scope, do not pre-split based on count.

## Out of scope

- Performing the live cutover itself (separate post-v1 procedure).
- Refactoring `internal/migrate/` beyond the npm-spec key sanitization fix.
- Adding new instruction files beyond the triage decisions on the three flagged candidates.
- Rewriting ADV-owned skills or changing skill loading behavior.
- Tagging v1.0.0 or running the release pipeline (separate post-v1 step).