# OCA Roadmap Queue — Layer Strategy Follow-Up

**Status:** Reviewed and accepted into OCA roadmap  
**Date:** 2026-05-03  
**Project:** OpenCode Advance (`~/dev/opencodeadvance`)  
**Purpose:** Feed OCA-owned MUST/SHOULD proposals into the OCA roadmap.

---

## Resume Instructions

Open a fresh agent session in:

```bash
cd ~/dev/opencodeadvance
```

Ask the OCA agent to review this queue, reconcile it with `NEXT_STEPS.md`,
`STATUS.md`, and `docs/proposals/phases.md`, then add accepted items to the OCA
roadmap in the right sequence.

---

## MUST — OCA-Owned

| Priority | Proposal | Why |
|---|---|---|
| M3 | `2026-05-03-oca-plugin-install.md` | OCA pane/session state writer is absent from live config; Pattern B, hibernation, watchdog depend on it. |
| M2 | `2026-05-03-oca-apply-lifecycle-parity.md` | Bare `oca apply` must be trustworthy as source-of-truth apply command. |
| M4 | `2026-05-03-oca-instruction-assets-real.md` | OCA claims instruction ownership but assets are missing. |
| M5 | `2026-05-03-oca-starter-migration-validity.md` | Generated starter/migration stacks must validate and apply. |
| M6 | `2026-05-03-oca-skill-guidance-refresh.md` | OCA-owned skills contain stale openchad/old-ADV/tool-name guidance. |

Suggested order: M3 → M2 → M4/M5/M6.

---

## SHOULD — OCA-Owned

| Priority | Proposal | Why |
|---|---|---|
| S1 | `2026-05-03-oca-runtime-doctor-canaries.md` | Static config checks missed runtime broken ADV prompt resolution. |
| S2 | `2026-05-03-context-budget-audit.md` | Make prompt/tool-schema load visible before trimming. |
| S3 | `2026-05-03-agent-permission-first-config.md` | OpenCode docs deprecate `tools`; OCA should move to permission-first profiles. |
| S4 | `2026-05-03-mcp-tool-suite-profiles.md` | Manage MCP as per-agent tool suites, not only server installs. |
| S5 | `2026-05-03-legacy-adv-state-doctor-warning.md` | Warn about legacy `.adv/{changes,db,agenda*}` and non-bundle archive residue while preserving `.adv/specs` and valid archive bundles. |
| S6 | `2026-05-04-oca-resume-hint-outer-terminal.md` | `/exit` resume hint prints inside dying tmux pane; outer terminal sees nothing. Fix: `bin/oc` captures and reprints after tmux exits. |
| S7 | `2026-05-04-oca-advance-self-update-handoff.md` | OCA owns the deterministic rebuild + fresh-session handoff for Advance self-updates (`Opencode-Advance#9`); ADV owns runtime provenance diagnostics. |

Suggested order: S1/S2 → S3/S4 → S5/S6/S7.

---

## Cross-Project Dependencies

OCA work should account for the Advance queue:

- `2026-05-03-adv-provider-runtime-prompt-canary.md` must land in Advance or a
  verified non-stub ADV agent must be available before relying on provider ADV
  sessions for new ADV workflows.
- `2026-05-03-agent-permission-first-config.md` may create an ADV companion if
  ADV-owned agent frontmatter/config still needs migration after OCA work.
- `2026-05-04-oca-advance-self-update-handoff.md` composes with
  Advance issue `Sharper-Flow/Advance#40` and OCA issue
  `Sharper-Flow/Opencode-Advance#9`. Keep runtime provenance in Advance; keep
  rebuild/session lifecycle in OCA.

---

## Roadmap Integration Notes

- Add accepted MUST items before broad post-v1 session architecture work.
- Keep live-config writes isolated until cutover policy says otherwise.
- Do not let OCA own ADV assets; route ADV-owned changes to the Advance queue.

---

## Review Result — 2026-05-03

All OCA-owned queue items are accepted as valid roadmap entries. Each proposal
already exists as a standalone ADV proposal draft in `docs/proposals/` and is
now reflected in `docs/proposals/phases.md`, `NEXT_STEPS.md`, and `STATUS.md`.

Validation summary:

| Priority | Proposal | Result | Evidence |
|---|---|---|---|
| M3 | `2026-05-03-oca-plugin-install.md` | Accepted | `plugins/oca/` exists; live `opencode.json` plugin array still lacks the OCA plugin; pane-state dependency blocks Pattern B/hibernation. |
| M2 | `2026-05-03-oca-apply-lifecycle-parity.md` | Accepted | `cmd/oca/apply.go` no-target path still uses `render.ComposeApplyPlan(...)`, bypassing `applyPlugins(...)` prepare/sync and `applyTemporal(...)`. |
| M4 | `2026-05-03-oca-instruction-assets-real.md` | Accepted | `assets/instructions/` contains only `README.md` while OCA docs claim ownership of real instruction assets. |
| M5 | `2026-05-03-oca-starter-migration-validity.md` | Accepted | `internal/migrate/init.go` emits `[plugins.advance]` without `path`; validator requires `checkout` and `path` for git plugins. |
| M6 | `2026-05-03-oca-skill-guidance-refresh.md` | Accepted | `assets/skills/worktree/SKILL.md` contains stale `openchad`/`oc switch`; `assets/skills/README.md` still names deleted ADV methodology skills. |
| S1 | `2026-05-03-oca-runtime-doctor-canaries.md` | Accepted | `opencode debug agent adv-gpt` can still resolve to the provider stub even when static config references the single concatenated prompt file, proving static checks are insufficient. |
| S2 | `2026-05-03-context-budget-audit.md` | Accepted | Current config exposes broad instruction and tool-schema load; OpenCode docs confirm permission/tool exposure is per-agent and measurable from config/runtime debug output. |
| S3 | `2026-05-03-agent-permission-first-config.md` | Accepted | OpenCode agent docs mark `tools` as deprecated and prefer `permission`; current runtime debug output still shows legacy `tools`. |
| S4 | `2026-05-03-mcp-tool-suite-profiles.md` | Accepted | OpenCode permissions support wildcard gating for built-ins and MCP tools; OCA currently models servers more strongly than per-agent tool exposure. |
| S5 | `2026-05-03-legacy-adv-state-doctor-warning.md` | Accepted | Project has `.adv/changes` and archive entries alongside valid `.adv/specs`; OCA should warn only on legacy/non-bundle residue and point to Advance cleanup tooling, not delete directly. |
| S6 | `2026-05-04-oca-resume-hint-outer-terminal.md` | Accepted | `/exit` pane death hides the resume hint from the outer terminal; OCA shell wrapper is the correct layer to capture/reprint it. |
| S7 | `2026-05-04-oca-advance-self-update-handoff.md` | Accepted | Advance can report loaded plugin provenance, but only OCA owns plugin rebuild/update wiring and project/worktree session launch. |

Operational caveat: ADV change creation was not performed in this review session
because provider ADV runtime prompt resolution still needs a fresh OpenCode
restart/runtime canary. Use the proposal files as `/adv-proposal` bodies after a
fresh session verifies a non-stub ADV agent.
