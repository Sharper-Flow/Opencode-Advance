# Advance Roadmap Queue — OCA Layer Strategy Follow-Up

**Status:** Project handoff queue  
**Date:** 2026-05-03  
**Project:** Advance plugin (`~/dev/oc-plugins/advance`)  
**Purpose:** Feed Advance-owned MUST/SHOULD follow-ups discovered during OCA layer
strategy investigation into the Advance roadmap.

---

## Resume Instructions

Open a fresh agent session in:

```bash
cd ~/dev/oc-plugins/advance
```

Ask the Advance agent to review this queue, reconcile it with Advance's roadmap,
and create proper ADV changes from the referenced proposal text.

Important caveat: provider variants currently may resolve to
`[ADV:PROVIDER_STUB_UNEXPANDED]`. Use a verified working ADV agent/session before
running `/adv-proposal` or other ADV workflows.

---

## MUST — Advance-Owned

| Priority | Proposal staged in OCA | Why |
|---|---|---|
| M1 | `~/dev/opencodeadvance/docs/proposals/2026-05-03-adv-provider-runtime-prompt-canary.md` | Runtime canary shows provider agents can still resolve to stub despite single-file prompt refs; static sync checks are not enough. |

Related prior proposal:

- `~/dev/opencodeadvance/docs/proposals/2026-05-03-adv-sync-prompt-ref-fix.md`

The new runtime-canary proposal supersedes/extends the older single-file prompt
ref proposal because the empirical failure persisted after single-ref files
existed.

---

## SHOULD / Companion — Advance-Owned If Needed

| Priority | Trigger | Action |
|---|---|---|
| S3-companion | After OCA completes `2026-05-03-agent-permission-first-config.md` | Review ADV-owned agent files/provider variants. If they still use deprecated `tools` or over-broad exposure, create an Advance companion change for permission-first agent config. |

---

## Roadmap Integration Notes

- Treat M1 as a blocker for reliable provider-specific ADV sessions.
- Runtime verification must use `opencode debug agent adv-{provider}`, not only
  file/string checks.
- If OpenCode precedence makes markdown agent body win over JSON prompt, adapt
  sync-global generation accordingly and update provider assembly docs.
