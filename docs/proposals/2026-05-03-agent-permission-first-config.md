# Permission-First Agent Configuration

**Status:** Proposal — staged  
**Date:** 2026-05-03  
**Target repo:** `~/dev/opencodeadvance` first; `~/dev/oc-plugins/advance` companion follow-up for ADV-owned agents  
**Resume from:** `~/dev/opencodeadvance` for OCA agent/render work  
**Suggested change ID:** `permissionFirstAgentConfig`  
**Priority:** SHOULD

---

## Problem Statement

OpenCode docs mark agent `tools` as deprecated and recommend `permission`.
Live `opencode debug agent adv-gpt` shows large tool exposure and duplicated
permission blocks. OCA should move environment-level agents and rendered config
toward permission-first profiles.

ADV-owned agent files remain owned by the Advance repo; this proposal should
surface that boundary and create a companion ADV follow-up if needed.

---

## Success Criteria

- [ ] OCA docs and stack schema prefer `permission` over deprecated `tools`.
- [ ] OCA-owned agents are rendered or documented with permission-first shape.
- [ ] Environment-level agents have role-specific permissions:
      - `explore`: read/lgrep only
      - `librarian`: docs/web/examples only
      - `general`: broad but controlled
      - `mechanic`: infra/system tools
      - `build`: implementation tools
- [ ] Doctor reports deprecated `tools` usage in OCA-owned agent configs.
- [ ] ADV-owned agent migration is listed as a separate ADV follow-up if OCA
      cannot change those files.

---

## Out of Scope

- Removing ADV tools from the ADV orchestrator without an Advance repo change.
- Changing OpenCode's permission semantics.

---

## Implementation Sketch

1. Research current OpenCode permission schema and last-match semantics.
2. Add OCA agent config model if missing, or harden existing agent assets.
3. Render permission-first config for OCA-owned agents.
4. Add doctor check for deprecated `tools` usage in OCA-owned assets.
5. Create ADV companion proposal if ADV provider/frontmatter files still use
   deprecated `tools` after OCA work.

---

## Acceptance Criteria

1. OCA-owned agent configs no longer rely on `tools` for new/updated files.
2. Doctor identifies deprecated `tools` in OCA-owned files.
3. `go test ./...` passes.
