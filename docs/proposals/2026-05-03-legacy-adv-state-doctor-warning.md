# Legacy In-Repo ADV State Doctor Warning

**Status:** Proposal — staged  
**Date:** 2026-05-03  
**Target repo:** `~/dev/opencodeadvance`  
**Resume from:** `~/dev/opencodeadvance`  
**Suggested change ID:** `legacyAdvStateDoctorWarning`  
**Priority:** SHOULD

---

## Problem Statement

Current Advance stores mutable state externally through Temporal-backed project
state. In-repo `.adv/specs/` remains source-controlled; legacy mutable dirs such
as `.adv/changes`, `.adv/archive`, `.adv/db`, and `.adv/agenda*` are migration
debt.

OCA currently has `.adv/changes` and `.adv/archive`. OCA must not delete them
manually, but it should warn operators and point to Advance cleanup tooling.

---

## Success Criteria

- [ ] `oca doctor` reports legacy in-repo ADV mutable state when present.
- [ ] `.adv/specs/` is explicitly treated as valid and never warned as legacy.
- [ ] Warning points to `adv_migrate_cleanup` dry-run/execute workflow.
- [ ] Warning is non-fatal by default.
- [ ] Tests cover `.adv/specs` only, legacy dirs present, and no `.adv` dir.

---

## Out of Scope

- Running `adv_migrate_cleanup` automatically.
- Deleting files directly from OCA.
- Reading external ADV state files directly.

---

## Implementation Sketch

1. Add check under `adv-plugin`, `adv-assets`, or new `adv-state` doctor scope.
2. Scan only project-local `.adv` directory entries.
3. Warn for legacy mutable entries.
4. Include remediation text:
   - run `adv_migrate_cleanup` dry-run from ADV-capable session
   - preserve `.adv/specs/`
   - execute only with user approval and backup

---

## Acceptance Criteria

1. Fixture with `.adv/specs` only passes.
2. Fixture with `.adv/changes` warns.
3. Fixture with `.adv/archive` warns.
4. `go test ./...` passes.
