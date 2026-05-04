# M-Queue Pre-Flight Verification (2026-05-04)

Verified the current-state assertions in each of the 5 OCA Reliability MUST proposals (M3/M2/M4/M5/M6) one day after they were drafted (2026-05-03). All proposals remain accurate enough to enter the queue. Three refinements recorded below — the implementing agent should apply them during `/adv-discover`.

Companion to: `docs/proposals/phases.md` § Post-v1: OCA Reliability + Runtime Correctness Queue.

---

## M3 — OCA umbrella plugin install — VERIFIED ✓

All four claims hold as of 2026-05-04 02:45 UTC:

- `~/.config/opencode/opencode.json` `plugin` array has **5 entries**, none referencing `~/dev/opencodeadvance/plugins/oca`. Live entries: ADV plugin, claude-max, morph-fast-apply, vision, opencode-openai-codex-auth.
- `~/.local/state/oca/panes/` does not exist.
- Plugin source present: `plugins/oca/{src/index.ts, dist/index.js, package.json}`.
- `~/dev/opencodeadvance/stack.toml` does not exist (only `stack.example.toml`).

No refinement needed. Proposal is filing-ready.

---

## M2 — OCA apply lifecycle parity — VERIFIED ✓

Source claims confirmed in `cmd/oca/apply.go`:

- Line 51 — bare apply uses `render.ComposeApplyPlan(stack, paths, state.configPath, render.AllTargets)` directly.
- Line 167 — `pluginpkg.Prepare(ctx, plugin)` is reachable only from the target-specific switch path.
- Line 135 — `applyTemporal(state, stack, dryRun)` is called only when `temporal` is the target; bare apply never reaches it.

No refinement needed. Proposal is filing-ready.

---

## M4 — OCA-owned instruction assets are real files — VERIFIED with REFINEMENT ⚠

`assets/instructions/` contains only `README.md`. The README inventory lists 10 files; the live `~/.config/opencode/instructions/` directory contains **14 files**. The four files present live but missing from the README:

| Live file | Notes |
|---|---|
| `caveman.md` | Caveman-mode instructions (related to caveman skill family). |
| `criteria-prioritizer.md` | 8.9 KB; possibly older asset; likely superseded by the `prioritizer/` skill — confirm before copying. |
| `global-verify-policy.md` | New 2026-05-03; references `/check` workflow. |
| `post_install_verification.md` | New 2026-05-03; installer verification doc. |

**Refinement for /adv-discover:** decide for each whether it's OCA-owned and should be copied into `assets/instructions/`, or stale and should be retired from the live dir on next `oca apply`. Update the README inventory accordingly. Do not silently drop files that the live env depends on today.

---

## M5 — Starter and migration stack validity — VERIFIED ✓

Confirmed in `internal/migrate/init.go` line 47-51 and `stack.example.toml` line 130-139:

`init.go` `[plugins.advance]` block emits: `source`, `checkout`, `build`, `provides`.

`stack.example.toml` `[plugins.advance]` block has all of the above **plus**: `ref`, `subdir`, `path`, `sync`, `instructions`. The missing `path` is the specific field validation requires for git plugins.

No refinement needed. Proposal is filing-ready.

---

## M6 — OCA skill guidance refresh — VERIFIED with REFINEMENT ⚠

Three claims checked. Two confirmed, one stale:

| Claim | Status | Detail |
|---|---|---|
| `assets/skills/README.md` lists deleted ADV methodology skills (`adv-review-methodology`, `adv-apply-methodology`, `adv-harden-methodology`) | **CONFIRMED** | All three appear in the README's "Not in this directory" table at lines 27, 28, 32. ADV inlined these and deleted the skill files (per `ADV_INSTRUCTIONS.md` § stale-reference note) — README must drop them. |
| `assets/skills/mcp-selection/SKILL.md` uses stale MCP function names | **CONFIRMED** | Lines 30, 48, 57 use `kagi_search_fetch`, `gh_grep_searchGitHub`, `firecrawl_scrape` — current schema names are `kagi_kagi_search_fetch`, `firecrawl_firecrawl_scrape` (and `gh_grep_searchGitHub` may also drift; verify against runtime). |
| `assets/skills/worktree/SKILL.md` mentions `openchad`, `oc switch` | **STALE — already fixed** | No `openchad`/`oc switch` references found. Skill already references `adv_worktree_*` canonical names. |

**Refinement for /adv-discover:** drop the worktree-skill claim from M6 scope. Focus on (a) README inventory cleanup of deleted ADV skills, (b) mcp-selection function-name refresh against current schema. Add a banned-term docs check that asserts the README does not list `adv-*-methodology` skills that no longer exist in the ADV repo.

---

## Doctor: stale OpenCode session debt

`bun ~/dev/oc-plugins/advance/scripts/opencode-session-doctor.ts --dry-run` would delete 3 stale blank assistant message rows (5 total blank, 2 live in flight). All 3 stale rows are >12 minutes old in three different sessions. Threshold 5 min. Safe to delete; pure hygiene. Run without `--dry-run` to apply.

---

## ADV #6 status

Filed 2026-05-04 in ADV-plugin project as `syncglobalpromptrefsinglefile` (cross-project from OCA, target_confirmed). Drafted source preserved at `docs/proposals/2026-05-03-adv-sync-prompt-ref-fix.md` for provenance. Operator-side workaround D (manual concatenation in `~/.config/opencode/agents/adv-claude.md`) is currently in effect; survives until next `sync-global.sh --fix` overwrites the stub body. After ADV #6 archives, workaround becomes unnecessary.

---

## Cross-project filing pattern (captured here because adv_wisdom_add hit a Temporal workflow issue this turn)

When filing an ADV change in a different project than the current session:

1. Verify target path exists and `adv_change_list target_path: "<path>"` resolves the project.
2. Read the source draft in current repo's `docs/proposals/`.
3. Synthesize: clean problemStatement (extracted Problem Statement section) + proposal body (full draft minus filing-path preamble; fix relative links that won't resolve cross-repo).
4. `adv_change_create target_path: "<target>" target_confirmed: true confirmationEvidence: "<user approval quote>"` plus `summary`, `problemStatement`, `proposal`.
5. Tool returns `cross_project_origin` block with provenance recorded.
6. Switch to target-project session for /adv-discover onward (`cd <target> && opencode`); cross-session ADV mutation via `opencode run --dir` works but is slow.
