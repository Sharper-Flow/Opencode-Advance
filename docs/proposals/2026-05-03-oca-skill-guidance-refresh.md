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

Confirmed by pre-flight verification 2026-05-04:

- `assets/skills/README.md` "Not in this directory" table still lists deleted
  ADV methodology skills: `adv-review-methodology`, `adv-apply-methodology`,
  `adv-harden-methodology`. ADV inlined these into command files and deleted
  the skill files (per `ADV_INSTRUCTIONS.md` § stale-reference note). The
  README claims they are still Advance-owned skills synced via
  `sync-global.sh` — false.
- `assets/skills/mcp-selection/SKILL.md` uses stale MCP function names: lines
  30, 48, 57 reference `kagi_search_fetch`, `gh_grep_searchGitHub`,
  `firecrawl_scrape`. Current schema names are `kagi_kagi_search_fetch`,
  `firecrawl_firecrawl_scrape` (and the `gh_grep_*` form needs verification
  against runtime).

Pre-flight finding: the previously-reported `assets/skills/worktree/SKILL.md`
staleness (`openchad`, `oc switch`) is **already fixed** — the file already
references `adv_worktree_*` canonical names. Worktree skill is OUT of scope
for this change.

Skills are on-demand methodology. If stale, they actively misroute agents.

See [`../notes/2026-05-04-m-queue-preflight-verification.md`](../notes/2026-05-04-m-queue-preflight-verification.md).

---

## Success Criteria

- [ ] No OCA-owned skill contains `openchad` or `open-chad` except historical
      migration notes, if explicitly marked.
- [ ] `assets/skills/README.md` "Not in this directory" table no longer lists
      `adv-review-methodology`, `adv-apply-methodology`,
      `adv-harden-methodology`, or any other ADV skill that has been deleted
      from the ADV plugin source.
- [ ] `assets/skills/README.md` "Not in this directory" table is regenerated
      from a verifiable source (e.g. listing of `~/dev/oc-plugins/advance/skills/`)
      rather than hand-curated.
- [ ] `assets/skills/mcp-selection/SKILL.md` uses MCP function names that
      match the current runtime schema: `kagi_kagi_search_fetch`,
      `firecrawl_firecrawl_scrape`, plus correct forms for Context7 / lgrep /
      gh_grep / vision tools as exposed today.
- [ ] mcp-selection notes the schema-name discovery caveat (server-prefixed
      names, Context7 hyphens, Vision double-prefix) consistent with
      `instructions/mcp-tools.md` § Tool Name Discovery.
- [ ] Tests or docs checks catch stale banned terms in OCA-owned skills,
      including the deleted-ADV-skill list and the legacy MCP tool-name set.

---

## Out of Scope

- Rewriting ADV-owned skills.
- Changing skill loading behavior.
- `assets/skills/worktree/SKILL.md` content refresh — already fixed
  (no `openchad`/`oc switch` refs; uses canonical `adv_worktree_*` names).

---

## Implementation Sketch

1. Audit `assets/skills/*/SKILL.md` and README files (skip worktree per
   Out of Scope above; pre-flight confirmed clean).
2. Refresh `assets/skills/README.md` "Not in this directory" table from
   the actual ADV-plugin skill listing (`~/dev/oc-plugins/advance/skills/`).
3. Refresh `assets/skills/mcp-selection/SKILL.md` against the current runtime
   tool schema:
   - Replace `kagi_search_fetch` → `kagi_kagi_search_fetch`.
   - Replace `firecrawl_scrape` → `firecrawl_firecrawl_scrape`.
   - Verify `gh_grep_*` form against runtime; correct if drifted.
   - Add the Tool-Name-Discovery note pointing at `instructions/mcp-tools.md`.
4. Replace any stale `openchad` references found in other skills (none
   expected based on pre-flight, but re-scan during implementation).
5. Add a docs test for banned stale references in OCA-owned skill assets:
   - Banned terms: `openchad`, `open-chad`, `oc switch`.
   - Banned MCP names: `kagi_search_fetch`, `firecrawl_scrape`,
     `context7_resolve_library_id` (underscore form).
   - Banned ADV-skill claims: `adv-review-methodology`,
     `adv-apply-methodology`, `adv-harden-methodology`.

---

## Acceptance Criteria

1. `lgrep_search_text`/test scan finds no stale `openchad`/`open-chad`/
   `oc switch` references in OCA-owned skill assets.
2. `assets/skills/README.md` "Not in this directory" table has zero entries
   for skills that no longer exist in the ADV plugin source.
3. `assets/skills/mcp-selection/SKILL.md` uses only schema-current MCP
   function names (verified by grepping for the banned legacy forms).
4. New docs test enforces the banned-term and banned-name lists above.
5. `go test ./...` passes.
