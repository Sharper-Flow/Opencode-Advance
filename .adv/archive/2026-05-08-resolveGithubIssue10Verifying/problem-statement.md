## Problem Statement

GitHub issue #10 reports that `AGENTS.md` still contains transitional context about Advance's historical shared-agent consolidation (`scout -> plan`, `refine -> build`). The current documentation should be a stable ownership reference, not a migration note, and should be guarded so stale agent filenames are not reintroduced as current shipped Advance assets.

## Evidence Gathered
- Issue #10 title: `AGENTS.md references stale Advance agent names (scout.md, refine.md)`.
- Current `AGENTS.md` line 207 contains transitional wording: `Recent Advance changes consolidated shared agents: scout -> plan and refine -> build.`
- Current Advance `.opencode/agents/` contains `adv.md`, `plan.md`, `build.md`, `adv-researcher.md`, `adv-engineer.md`, and repo-local `adv-tron.md`; no `scout.md` or `refine.md`.
- Advance `scripts/sync-global.sh` marks `scout.md` and `refine.md` as legacy stale global agent files and removes them during sync.
- Live global agents include provider variants plus `adv-engineer.md`, `adv-researcher.md`, `build.md`, `plan.md`, and OCA-owned environment agents; no live `scout.md` / `refine.md`.

## Impact
Low severity documentation accuracy issue. No functional runtime change required, but stale reference docs can mislead future work and should have automated protection where practical.