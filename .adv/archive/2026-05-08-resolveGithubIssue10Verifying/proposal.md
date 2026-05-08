## Summary
Resolve GitHub issue #10: `AGENTS.md` contains transitional wording about stale Advance agent names (`scout -> plan`, `refine -> build`) and should instead document the current agent ownership boundary as stable reference material.

## Problem
OCA's agent ownership documentation must match the current Advance-shipped agent roster. Transitional notes about old `scout.md` / `refine.md` names create ongoing ambiguity in a reference doc, and there is no repo-local guard preventing those stale names from being reintroduced as current shipped agents.

## Success Criteria
- `AGENTS.md` no longer uses transitional "Recent Advance changes" wording for `scout` / `refine`.
- OCA documentation clearly states current Advance-owned/global/repo-local agent ownership.
- Repo verification includes a docs guard or checklist preventing `scout.md` / `refine.md` from being documented as current shipped Advance agents.
- GitHub issue #10 can be closed with evidence of audit and verification.

## Out of Scope
- Changing Advance's `scripts/sync-global.sh` behavior.
- Modifying live `~/.config/opencode` files.
- Renaming or deleting existing OCA runtime/migration stale-agent handling code.
- Broad rewrite of unrelated agent/instruction docs.