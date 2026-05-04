# MUST/SHOULD Proposal Index — Layer Strategy Follow-Up

**Status:** Index — OCA queue reviewed and accepted into roadmap  
**Date:** 2026-05-03  
**Purpose:** Map each MUST/SHOULD recommendation from the ADV/OCA layer strategy
investigation to a concrete proposal and the project to resume from.

---

## Resume Rule

Run each ADV workflow from the **target repo**, not from this index file's repo.
Some ADV-target proposals are staged here only because they were discovered while
investigating OCA.

Current caveat: do not start provider-variant ADV workflow until the runtime
provider prompt fix is resolved or an explicitly verified working ADV agent is
available.

---

## Project-Sorted Queues

Use these handoff files when asking each project's agent to review and add work
to its roadmap:

| Project | Handoff file | Resume directory |
|---|---|---|
| OCA | `2026-05-03-oca-roadmap-queue.md` | `~/dev/opencodeadvance` |
| Advance | `2026-05-03-advance-roadmap-queue.md` | `~/dev/oc-plugins/advance` |

---

## OCA Project Queue

Resume from `~/dev/opencodeadvance`.

### OCA MUST

| # | Proposal |
|---|---|
| M2 | `2026-05-03-oca-apply-lifecycle-parity.md` |
| M3 | `2026-05-03-oca-plugin-install.md` |
| M4 | `2026-05-03-oca-instruction-assets-real.md` |
| M5 | `2026-05-03-oca-starter-migration-validity.md` |
| M6 | `2026-05-03-oca-skill-guidance-refresh.md` |

### OCA SHOULD

| # | Proposal |
|---|---|
| S1 | `2026-05-03-oca-runtime-doctor-canaries.md` |
| S2 | `2026-05-03-context-budget-audit.md` |
| S3 | `2026-05-03-agent-permission-first-config.md` |
| S4 | `2026-05-03-mcp-tool-suite-profiles.md` |
| S5 | `2026-05-03-legacy-adv-state-doctor-warning.md` |

---

## Advance Project Queue

Resume from `~/dev/oc-plugins/advance`.

### Advance MUST

| # | Proposal |
|---|---|
| M1 | `2026-05-03-adv-provider-runtime-prompt-canary.md` |

### Advance SHOULD / Companion

| # | Proposal |
|---|---|
| S3-companion | Review `2026-05-03-agent-permission-first-config.md` after OCA decides permission-first shape; create ADV companion if ADV-owned agent files need migration. |

---

## Original MUST List

| # | Proposal | Target / resume project |
|---|---|---|
| M1 | `2026-05-03-adv-provider-runtime-prompt-canary.md` | `~/dev/oc-plugins/advance` |
| M2 | `2026-05-03-oca-apply-lifecycle-parity.md` | `~/dev/opencodeadvance` |
| M3 | `2026-05-03-oca-plugin-install.md` | `~/dev/opencodeadvance` |
| M4 | `2026-05-03-oca-instruction-assets-real.md` | `~/dev/opencodeadvance` |
| M5 | `2026-05-03-oca-starter-migration-validity.md` | `~/dev/opencodeadvance` |
| M6 | `2026-05-03-oca-skill-guidance-refresh.md` | `~/dev/opencodeadvance` |

## Original SHOULD List

| # | Proposal | Target / resume project |
|---|---|---|
| S1 | `2026-05-03-oca-runtime-doctor-canaries.md` | `~/dev/opencodeadvance` |
| S2 | `2026-05-03-context-budget-audit.md` | `~/dev/opencodeadvance` |
| S3 | `2026-05-03-agent-permission-first-config.md` | `~/dev/opencodeadvance` first; ADV companion if needed |
| S4 | `2026-05-03-mcp-tool-suite-profiles.md` | `~/dev/opencodeadvance` |
| S5 | `2026-05-03-legacy-adv-state-doctor-warning.md` | `~/dev/opencodeadvance` |

---

## Suggested Sequence

1. M1 — ADV provider prompt runtime fix / runtime canary passes in a fresh session.
2. M3 — install/register OCA umbrella plugin.
3. M2 — fix bare `oca apply` lifecycle.
4. M4 + M5 + M6 — make OCA source-of-truth assets valid.
5. S1 + S2 — add runtime/context observability.
6. S3 + S4 — trim/shape agent permissions and MCP exposure.
7. S5 — warn about legacy in-repo ADV mutable state.

OCA roadmap integration completed 2026-05-03 in:

- `docs/proposals/2026-05-03-oca-roadmap-queue.md`
- `docs/proposals/phases.md` § "Post-v1: OCA Reliability + Runtime Correctness Queue"
- `NEXT_STEPS.md`
- `STATUS.md`
