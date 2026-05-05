# Prepare OCA for signal-driven Advance workflows

## Problem
Advance is moving from project-keyed disk/Temporal dual-write workflows to global signal-driven change workflows. OCA currently documents and diagnoses Advance runtime surfaces that the new architecture deletes or changes. If left as-is, OCA will give stale guidance after the cutover.

## Scope
Current route: **docs + upstream checklist while waiting for upstream**.

In scope now:
- OCA-owned active docs/status guidance: `AGENTS.md`, `STATUS.md`, `NEXT_STEPS.md`, and architecture/design docs where they present current guidance.
- A concrete upstream-readiness checklist documenting when the larger OCA runtime refactor may proceed.
- Verification scans proving active docs no longer present the signal-driven refactor as landed or instruct users toward soon-deleted tools without timing context.

Deferred until Advance lands:
- Runtime code refactor in `internal/advruntime`, `internal/health`, `cmd/oca`, `lib/adv_status.sh`, `internal/dashboard`.
- Spec delta for `oca-workspace-projection`.
- Removing compatibility with current Advance tools/state.

Out of scope:
- Implementing Advance's signal-driven workflow refactor inside Advance.
- Changing ADV-owned agent/command/skill assets.
- Destructive cleanup of user `.adv` state.
- Mutating live OpenCode config outside dev/test overrides.

## Evidence
- User-provided upstream decision doc describes planned signal-driven architecture, but independent validator confirmed it is not landed on current Advance trunk.
- Current OCA active docs still reference current repair/cleanup era guidance; without timing context, those notes can mislead once upstream changes.

## Success Criteria
- Active OCA docs clearly distinguish current-Advance behavior from pending signal-driven cutover.
- OCA has an explicit upstream readiness checklist for when runtime refactor can proceed.
- No runtime behavior is changed before upstream lands.
- Verification scans prove no active docs imply signal-driven Advance is already landed.
- Go/docs verification appropriate for docs-only change passes.