# Advance Signal-Driven Workflow Cutover Readiness

> **Status:** waiting on upstream Advance  
> **Owner:** OCA tracks readiness; Advance owns implementation  
> **Related OCA change:** `prepareOcaSignalDrivenAdvance`

## Purpose

Advance has a planned signal-driven change workflow refactor that will replace
several current OCA-facing runtime assumptions. This note records the exact
readiness gates OCA must see before changing runtime behavior.

Until these gates pass, OCA should preserve current Advance compatibility:

- current Temporal search attributes remain valid
- current project/workflow projections remain valid
- current cleanup/repair tools remain current-era guidance
- current `oca-workspace-projection` spec remains law

## Upstream gates before OCA runtime refactor

Do **not** hard-cut OCA runtime code until all gates are satisfied.

| Gate | Required evidence |
|---|---|
| Advance landing point | Signal-driven workflow change merged to Advance `trunk` or released in a tag OCA can pin/test |
| Search attributes | New attributes registered and documented: `AdvChangeId`, `AdvChangeStatus`, `AdvChangeTitle`, `AdvAffectedProjects`, `AdvAffectedPaths`, `AdvCurrentGate`, `AdvCurrentBucket`, `AdvLastSignalAt`, `AdvCreatedAt` |
| Workflow queues | Global workflow queue / host activity queue contract documented, including how OCA should probe serviceability |
| Projection contract | Schema-v2 disk projection path and shape documented, including `schemaVersion === 2` and missing-file degradation rules |
| Deleted tools | Deleted/folded tool list confirmed in Advance tool registry or explicit transition/deprecation notes published |
| Archive output | Durable archive trinity confirmed: `.adv/specs/`, `.adv/wisdom.jsonl`, brief `.adv/archive/{change-id}.md`, plus transition handling for historical `change.json` bundles |
| Migration story | Advance migration command/process documented and dry-run verified on representative active changes |
| Verification target | OCA has a concrete Advance checkout/ref for integration verification |

## OCA follow-up when gates pass

Start a new ADV change or re-enter `prepareOcaSignalDrivenAdvance` for runtime
work. Expected OCA work:

1. Replace required search attributes in `internal/advruntime`.
2. Replace per-project queue classification with signal-era visibility checks.
3. Update shell/status projection readers to schema-v2 or silent degradation.
4. Update dashboard Temporal polling to map signal-era search attributes.
5. Revise `oca-workspace-projection` spec away from `snapshot.json`.
6. Update docs to remove current-era cleanup/repair guidance.

## Non-goals while waiting

- Do not remove current Advance compatibility.
- Do not update `oca-workspace-projection` yet.
- Do not delete historical `.adv/archive/*/change.json` bundles.
- Do not recommend signal-era commands or attributes as available until the
  landing gates pass.
