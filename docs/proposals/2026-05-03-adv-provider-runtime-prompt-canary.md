# ADV Provider Runtime Prompt Resolution + Canary

**Status:** Proposal — staged from OCA investigation  
**Date:** 2026-05-03  
**Target repo:** `~/dev/oc-plugins/advance`  
**Resume from:** `~/dev/oc-plugins/advance`  
**Suggested change ID:** `providerRuntimePromptCanary`  
**Priority:** MUST  
**Supersedes/extends:** `2026-05-03-adv-sync-prompt-ref-fix.md`

---

## Problem Statement

ADV provider agents still resolve to the fallback stub at runtime even after the
single-file prompt-ref shape exists in `opencode.json`.

Verified state from OCA investigation:

- `agent.adv-gpt.prompt` is `{file:./agent-parts/advance/adv-gpt.md}`.
- `~/.config/opencode/agent-parts/advance/adv-gpt.md` exists and contains the
  concatenated canonical prompt + provider hint.
- `opencode debug agent adv-gpt` still resolves `prompt` to
  `[ADV:PROVIDER_STUB_UNEXPANDED]` from `agents/adv-gpt.md`.

This suggests the earlier diagnosis was incomplete: OpenCode may prefer the
markdown agent body over the JSON `agent.<name>.prompt` override when both exist.
String-level sync checks are insufficient; ADV needs a runtime canary.

---

## Success Criteria

- [ ] `scripts/sync-global.sh --check` runs a runtime canary for each enabled
      provider variant or documents why it cannot.
- [ ] Canary verifies `opencode debug agent adv-{provider}` resolved prompt:
      - does **not** contain `[ADV:PROVIDER_STUB_UNEXPANDED]`
      - does contain canonical ADV prompt markers
      - does contain matching provider hint marker
- [ ] `--fix` produces a layout that passes the canary on a fresh OpenCode
      session.
- [ ] If JSON prompt refs cannot override markdown bodies, generated provider
      markdown agent bodies contain the real concatenated body while preserving
      frontmatter/tool allowlists.
- [ ] Provider variants `adv-claude`, `adv-gpt`, `adv-glm`, `adv-kimi` all pass.
- [ ] Existing provider assembly docs are updated with the actual runtime
      precedence model.

---

## Out of Scope

- Changing provider model choices.
- Changing canonical ADV workflow content except where needed to compose it.
- OCA-side rendering changes.

---

## Implementation Sketch

1. Reproduce with a minimal temporary agent to confirm whether JSON prompt refs
   override markdown bodies in OpenCode 1.14.33.
2. Update sync generation strategy based on actual precedence:
   - Preferred if JSON prompt wins: keep single `{file:...}` prompt ref.
   - If markdown body wins: write concatenated prompt into provider markdown
     body and treat stub-only body as invalid runtime state.
3. Add `opencode debug agent` checks to `--check`.
4. Add unit/smoke tests around generated provider files and check output.

---

## Acceptance Criteria

1. Run `scripts/sync-global.sh --fix`.
2. Restart OpenCode.
3. For each provider variant, run `opencode debug agent adv-{provider}`.
4. Confirm resolved prompt contains real ADV markers and no stub marker.
5. Run `scripts/sync-global.sh --check`; it fails if any provider resolves to
   stub.
