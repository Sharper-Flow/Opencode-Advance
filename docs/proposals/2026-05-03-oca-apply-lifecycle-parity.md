# OCA Apply Lifecycle Parity

**Status:** SUPERSEDED — shipped 2026-05-04 as `fixBareOcaApplyLifecycleParity` (archive: `.adv/archive/2026-05-04-fixBareOcaApplyLifecycleParity/`). All four success criteria (plugin prepare, plugin sync, temporal apply, read-only dry-run) addressed with regression tests in `cmd/oca/apply_test.go` (`TestApplyCommand_BareApplyRunsPluginSync`, `TestApplyCommand_BareApplyRendersTemporal`, `TestApplyCommand_BareApplyDryRunReportsLifecycleReadOnly`). The `render.ComposeApplyPlan` fast-path no longer exists in `cmd/oca/apply.go`. This proposal file is preserved for historical context only.

**Original status:** Proposal — staged  
**Date:** 2026-05-03  
**Target repo:** `~/dev/opencodeadvance`  
**Resume from:** `~/dev/opencodeadvance`  
**Suggested change ID:** `applyLifecycleParity`  
**Priority:** ~~MUST~~ — superseded

---

## Problem Statement

Bare `oca apply` does not execute the same lifecycle as targeted applies.

Current code path in `cmd/oca/apply.go`:

- no-target apply composes render operations with `render.ComposeApplyPlan(...)`
  and writes them directly.
- it bypasses git plugin prepare/build (`pluginpkg.Prepare`).
- it bypasses plugin sync (`Advance/scripts/sync-global.sh --fix`).
- it bypasses `applyTemporal` even though `temporal` is accepted as a target.

This conflicts with user-facing docs and `stack.example.toml`, which present
`oca apply` as the normal command to materialize the stack.

---

## Success Criteria

- [ ] Bare `oca apply` prepares/builds enabled git plugins before rendering
      plugin entries.
- [ ] Bare `oca apply` invokes configured plugin sync commands after plugin
      render succeeds.
- [ ] Bare `oca apply` applies Temporal env rendering when `[temporal]` is
      enabled.
- [ ] Dry-run remains read-only and reports the full planned lifecycle.
- [ ] Existing target-specific behavior remains unchanged.
- [ ] Tests prove `oca apply` and sequential target apply have lifecycle parity.

---

## Out of Scope

- Changing stack schema.
- Changing plugin source pinning/update semantics.
- Running live OpenCode restart automatically.

---

## Implementation Sketch

1. Replace no-target direct `ComposeApplyPlan` fast path with an orchestration
   path that calls the same target functions in dependency order.
2. Keep the in-memory merge behavior to avoid stale-Before stomps; do not
   regress `ComposeApplyPlan` where it is still useful for `diff`/dry-run.
3. Ensure plugin prepare runs before `.plugin` render.
4. Ensure plugin sync runs after `.plugin` write and only for successful writes.
5. Include `temporal` in default apply ordering.

---

## Acceptance Criteria

1. `oca apply --dry-run` shows all configured targets, including plugins and
   temporal when enabled.
2. A test stack with a fake sync plugin proves sync is called by bare apply.
3. A test stack with temporal enabled proves `temporal.env` is written by bare
   apply.
4. `go test ./...` passes.
