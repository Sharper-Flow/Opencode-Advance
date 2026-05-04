# OCA Skill Guidance Refresh

**Status:** Proposal — staged  
**Date:** 2026-05-03  
**Target repo:** `~/dev/opencodeadvance`  
**Resume from:** `~/dev/opencodeadvance`  
**Suggested change ID:** `skillGuidanceRefresh`  
**Priority:** MUST

---

## Problem Statement

Several OCA-owned skill files still contain stale pre-OCA or stale ADV guidance.

Observed examples:

- `assets/skills/worktree/SKILL.md` mentions `openchad` and `oc switch`.
- `assets/skills/README.md` lists deleted/stale ADV methodology skills such as
  `adv-review-methodology`, `adv-apply-methodology`, and others that are no
  longer current.
- `assets/skills/mcp-selection/SKILL.md` uses stale MCP tool names and omits
  current OpenCode function-name caveats.

Skills are on-demand methodology. If stale, they actively misroute agents.

---

## Success Criteria

- [ ] No OCA-owned skill contains `openchad` or `open-chad` except historical
      migration notes, if explicitly marked.
- [ ] Skill README lists current ownership boundaries: OCA skills vs ADV skills.
- [ ] `mcp-selection` matches current MCP tool/function naming rules and known
      Context7 caveat.
- [ ] `worktree` matches OCA session/tmux behavior and current ADV worktree tool
      names.
- [ ] Tests or docs checks catch stale banned terms in OCA-owned skills.

---

## Out of Scope

- Rewriting ADV-owned skills.
- Changing skill loading behavior.

---

## Implementation Sketch

1. Audit `assets/skills/*/SKILL.md` and README files.
2. Replace stale openchad references with OCA/current OpenCode terminology.
3. Align MCP tool examples with actual function names exposed in this runtime.
4. Add a simple docs test for banned stale references in OCA-owned skill assets.

---

## Acceptance Criteria

1. `lgrep_search_text`/test scan finds no stale openchad references in
   OCA-owned skill assets.
2. Skill docs name current ADV skills only.
3. `go test ./...` passes.
