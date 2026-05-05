# Archive: Prepare OCA for signal-driven Advance workflows

**Change ID:** prepareOcaSignalDrivenAdvance
**Archived:** 2026-05-05T15:13:10.509Z
**Created:** 2026-05-04T23:02:15.292Z

## Tasks Completed

- ✅ [DOCS] Add Advance signal cutover readiness checklist. Create an OCA-owned checklist note under `docs/notes/` documenting exact upstream gates before runtime refactor can proceed: Advance commit/tag landed, signal search attrs registered, projection schema v2 path/shape documented, deleted tools status known, durable trinity/archive transition known, migration story tested, verification target available. TDD intent: not_applicable (docs). Acceptance: agreement AC2, AC4.
  > Added `docs/notes/2026-05-05-advance-signal-cutover-readiness.md` with upstream readiness gates. Linked readiness gates from active docs and clarified current-era cleanup/projection guidance where needed.
- ✅ [DOCS] Clarify active OCA guidance for current Advance vs pending signal cutover. Update `AGENTS.md`, `STATUS.md`, `NEXT_STEPS.md`, and `docs/design/architecture.md` only where active guidance could imply signal architecture is landed or future-deleted tools are timeless guidance. Preserve current runtime compatibility language; do not change code behavior. TDD intent: not_applicable (docs). Acceptance: agreement AC1, AC3, AC5.
  > Clarified active OCA guidance in AGENTS, STATUS, NEXT_STEPS, and architecture docs so current Advance repair/projection behavior remains valid current-era guidance and the signal-driven refactor is explicitly pending upstream readiness gates.
- ✅ [VERIFY] Verify docs-only waiting-state route. Run text scans proving active docs do not claim signal-driven Advance is landed and do not present deleted-tool guidance without current-era context. Run `go test ./...`, `go vet ./...`, and `gofmt -d .`; run relevant shell/docs tests if docs touched. TDD intent: separate_verification. Acceptance: agreement AC5, AC6.
  > Verified docs-only waiting-state route. Docs scan passed; `go test ./...` passed; `go vet ./...` passed. `gofmt -d .` ran and reported pre-existing formatting diffs in untouched `internal/advruntime` files, left unchanged to preserve no-runtime-code-change scope.

## Specs Modified


## Wisdom Accumulated

- **[gotcha]** When upstream architecture is designed but not landed, keep OCA runtime docs explicitly split between current-era guidance and pending cutover gates; do not update specs or runtime code until live Advance contracts match the planned model.
