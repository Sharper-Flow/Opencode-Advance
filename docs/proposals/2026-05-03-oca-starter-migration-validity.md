# Starter and Migration Stack Validity

**Status:** Proposal — staged  
**Date:** 2026-05-03  
**Target repo:** `~/dev/opencodeadvance`  
**Resume from:** `~/dev/opencodeadvance`  
**Suggested change ID:** `starterMigrationValidity`  
**Priority:** MUST

---

## Problem Statement

`oca migrate init` and migration output must create a stack that can be loaded,
validated, and applied without hand repair.

Current risk observed in source:

- `internal/migrate/init.go` emits `[plugins.advance]` without a `path`, while
  validation requires `path` for git plugins.
- starter instructions reference `{assets}/instructions/...`, but current OCA
  instruction assets are absent.
- migrated plugin entries may omit fields required by current validation/render
  semantics.

Even if a narrow test currently passes, this must be protected as a product
contract: generated stacks are valid stacks.

---

## Success Criteria

- [ ] `oca migrate init --output stack.toml` emits a stack that passes
      `cfg.Load()` immediately.
- [ ] `oca migrate from-open-chad --output stack.toml` emits a stack that passes
      `cfg.Load()` for representative fixtures.
- [ ] Generated Advance plugin declaration includes `checkout`, `subdir`,
      `path`, `build`, `sync`, `provides`, and instruction handling consistent
      with `stack.example.toml`.
- [ ] Generated OCA plugin declaration is included or explicitly documented as
      opt-in with a warning.
- [ ] Tests fail if generated examples drift from validation rules.

---

## Out of Scope

- Performing live cutover.
- Removing the legacy migration command.

---

## Implementation Sketch

1. Update `EmitInit()` to mirror the minimum valid plugin/instruction/temporal
   structure from `stack.example.toml`.
2. Update migration emitter so plugin entries include resolved `path` when
   source is git.
3. Add round-trip tests for both init and migrated full-state output:
   emit → parse/load → diff/apply dry-run.
4. Keep all writes isolated by env overrides in tests.

---

## Acceptance Criteria

1. `oca migrate init --output /tmp/stack.toml` then `oca debug validate --config
   /tmp/stack.toml` passes.
2. Migration fixtures round-trip through `cfg.Load()` and render plan.
3. `go test ./...` passes.
