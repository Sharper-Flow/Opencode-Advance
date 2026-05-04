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
state. In-repo `.adv/specs/` remains source-controlled, and valid
`.adv/archive/*/change.json` bundles are preserved archive artifacts. Legacy
mutable dirs such as `.adv/changes`, `.adv/db`, `.adv/agenda*`, and non-bundle
`.adv/archive` residue are migration debt.

OCA currently has legacy `.adv/changes` plus archive entries that need
bundle-aware handling. OCA must not delete them manually, but it should warn
operators and point to Advance cleanup tooling.

---

## Success Criteria

- [ ] `oca doctor` reports legacy in-repo ADV mutable state when present.
- [ ] `.adv/specs/` is explicitly treated as valid and never warned as legacy.
- [ ] Valid `.adv/archive/*/change.json` bundles are explicitly treated as valid
      and never warned as legacy.
- [ ] Warning points to `adv_migrate_cleanup` dry-run/execute workflow.
- [ ] Warning is non-fatal by default.
- [ ] Tests cover `.adv/specs` only, valid archive bundles, non-bundle archive
      residue, legacy dirs present, and no `.adv` dir.

---

## Out of Scope

- Running `adv_migrate_cleanup` automatically.
- Deleting files directly from OCA.
- Reading external ADV state files directly.

---

## Implementation Sketch

1. Add check under `adv-plugin`, `adv-assets`, or new `adv-state` doctor scope.
2. Scan only project-local `.adv` directory entries.
3. Warn for legacy mutable entries and non-bundle archive residue; preserve
   valid `.adv/archive/*/change.json` bundles.
4. Include remediation text:
   - run `adv_migrate_cleanup` dry-run from ADV-capable session
   - preserve `.adv/specs/` and valid `.adv/archive/*/change.json` bundles
   - execute only with user approval and backup

---

## Acceptance Criteria

1. Fixture with `.adv/specs` only passes.
2. Fixture with `.adv/changes` warns.
3. Fixture with valid `.adv/archive/<id>/change.json` bundle passes.
4. Fixture with non-bundle `.adv/archive` residue warns.
5. `go test ./...` passes.
