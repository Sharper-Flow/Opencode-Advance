# Design

## Architecture Overview

This change takes the safe waiting-state route: **docs + upstream readiness checklist**. OCA will not alter runtime behavior while the Advance signal-driven architecture remains unlanded. Instead, OCA records the future dependency, clarifies current guidance, and defines precise gates for the later runtime refactor.

## Key Decisions

### KD-1 — Do not hard-cut runtime code now

`internal/advruntime`, `internal/health`, `cmd/oca`, `lib/adv_status.sh`, and dashboard runtime code stay compatible with current Advance. Current `snapshot.json`, current search attributes, project queue checks, and current cleanup tooling remain valid until Advance lands the replacement.

Rationale: validator confirmed current Advance trunk still exposes old contracts. Trunk must remain production-ready.

### KD-2 — Convert near-term work to docs + checklist

Update active docs/status guidance to say the current repair/cleanup/projection behavior is valid for the current Advance era, while signal-driven cutover is pending upstream.

Rationale: gives operators accurate context without breaking runtime behavior.

### KD-3 — Add upstream readiness checklist

Create or update an OCA-owned doc with concrete readiness gates:

- Advance signal-driven change merged to trunk or tagged.
- New Temporal search attributes registered and documented.
- Projection schema v2 path/shape documented, including absence/degradation contract.
- Deleted/folded tools removed from tool registry or explicit transition deprecation documented.
- Archive output switched to durable trinity or transition behavior documented.
- Migration command/story available and tested.
- OCA has a verification target: specific Advance checkout/ref or release.

Rationale: later runtime refactor should be mechanical and evidence-driven.

### KD-4 — Preserve existing spec until upstream lands

Do not update `oca-workspace-projection` in this change. Record that it must be revised in the later runtime refactor once `snapshot.json` is truly obsolete.

Rationale: specs are laws and current spec matches live behavior today.

## Implementation Strategy

1. Update active docs (`AGENTS.md`, `STATUS.md`, `NEXT_STEPS.md`, and possibly `docs/design/architecture.md`) to clarify current vs pending Advance behavior.
2. Add `docs/notes/2026-05-05-advance-signal-cutover-readiness.md` or equivalent checklist doc.
3. Add/update docs verification text where existing docs tests assert current status wording.
4. Run text scans for stale unqualified claims.
5. Run Go verification.

## LBP Analysis

Best long-term practice is to avoid coupling OCA to unlanded upstream internals. The clean greenfield approach is versioned/feature-detected contracts with clear readiness gates. Documentation now, runtime change after upstream lands, keeps trunk reliable while preserving momentum.

## Affected Components

- `AGENTS.md`
- `STATUS.md`
- `NEXT_STEPS.md`
- `docs/design/architecture.md` if current runtime section needs timing context
- new readiness checklist note under `docs/notes/`
- docs/status tests only if needed

## Risks / Mitigations

- **Risk: docs become stale again when upstream lands.** Mitigation: checklist names exact landing gates and follow-up runtime refactor trigger.
- **Risk: not enough momentum.** Mitigation: checklist makes future work mechanical; no runtime change blocked on speculation.
- **Risk: user expected code ripping now.** Mitigation: validator conflict showed code ripping would break current Advance; this route preserves trunk and prepares the cutover.

## Validator Result

`CONFLICT` resolved by route choice: user chose `docs and checklist` after validator found upstream not landed. No runtime hard-cut proceeds in this change.
