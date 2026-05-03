# ADV sync-global.sh — Prompt-Ref Single-File Concatenation Fix

**Status:** Proposal — ready for `/adv-proposal` · **Date:** 2026-05-03
**Target repo:** `Sharper-Flow/Advance` (oc-plugins/advance) — **NOT OpenCode Advance**
**Filing path:** When ready to start work, this proposal must be moved to (or re-drafted in) the ADV repo and `/adv-proposal` must run from inside `~/dev/oc-plugins/advance`. Drafted here in OCA repo because it was discovered while preparing the OCA-side 5-change split.
**Change ID (suggested):** `syncGlobalPromptRefSingleFile`
**Parent decision lock:** [`2026-05-03-session-and-resource-architecture.md`](./2026-05-03-session-and-resource-architecture.md) — discovered as blocker during decision-lock review
**Split position:** **Change #6 of 6 — BLOCKER for #1–5.** Must ship before `/adv-proposal` can be run on the other 5 changes (or any future ADV change in any project) using a Claude/GLM/GPT/Kimi provider variant.
**Estimated effort:** 1–2 hours
**Estimated blast radius:** ~6 files touched (`scripts/sync-global.sh`, two test files, three doc files; see §Companion files)
**Sequence:** **Ship first**, before any of the OCA #1–3 or ADV #4–5 changes can be drafted via `/adv-proposal`.

> **Link note:** the parent-doc link above (`./2026-05-03-session-and-resource-architecture.md`) resolves only when this file is in `opencodeadvance/docs/proposals/`. When this proposal is moved into the ADV repo to be filed via `/adv-proposal`, replace the relative link with an absolute path or a GitHub URL pointing to the OCA-repo source.

---

## Adaptation Note (read first)

This proposal was discovered late, during preparation of the OCA 5-change split (changes #1–5). It is not in the parent decision-lock doc but is a **prerequisite blocker** for the rest. When `/adv-proposal` and `/adv-discover` run on this change, the discovery phase MUST:

1. Re-read ADV's own `docs/provider-agent-assembly.md` — particularly §Design Principles point 3 ("native OpenCode `agent.adv-{provider}.prompt` contains `{file:./agent-parts/advance/adv.md}` plus `{file:./agent-parts/advance/providers/{provider}.md}`") and §Troubleshooting line 90 ("Provider session shows `[ADV:PROVIDER_STUB_UNEXPANDED]` — OpenCode did not expand `agent.adv-{provider}.prompt` file refs").
2. Re-verify the empirical finding: OpenCode 1.14.33's `{file:...}` resolver in `agent.X.prompt` only handles single-file form per [official docs](https://opencode.ai/docs/agents/#prompt). Multi-file concatenation with `\n\n` is not parsed.
3. Decide between two architectural directions (Option A vs B in §Resolution Options below); Option A is the recommended LBP fix.

---

## TL;DR

ADV's `sync-global.sh` writes `agent.adv-{provider}.prompt` in `opencode.json` as **two `{file:...}` references concatenated with `\n\n`**. OpenCode 1.14.33 (and per official docs, the documented behavior) only supports a single `{file:...}` per prompt. Result: the prompt-ref interpolation fails silently, opencode falls through to the markdown agent file body (which is the `[ADV:PROVIDER_STUB_UNEXPANDED]` diagnostic), and **all four provider variants — claude, gpt, glm, kimi — are broken in their orchestrator persona**. ADV operations still work via the instructions array (rules + ADV_INSTRUCTIONS.md load fine), but the orchestrator's persona / workflow guidance is missing. This blocks proper `/adv-proposal` runs for any new ADV change.

---

## Problem Statement

### Verified empirical state (2026-05-03)

- `~/.config/opencode/opencode.json` `agent.adv-claude.prompt` = `"{file:./agent-parts/advance/adv.md}\n\n{file:./agent-parts/advance/providers/claude.md}"` (verified via `python3 -c 'json.load(...)'`).
- Both target files exist and are readable: `~/.config/opencode/agent-parts/advance/adv.md` (16312 bytes) and `~/.config/opencode/agent-parts/advance/providers/claude.md` (316 bytes).
- `bash scripts/sync-global.sh --check` reports "config already correct" — sync layer believes everything is wired right.
- `opencode debug agent adv-claude` returns:
  ```json
  "prompt": "[ADV:PROVIDER_STUB_UNEXPANDED] Provider ADV stub did not expand through opencode.json prompt refs. Do NOT proceed with ADV workflow. Run `scripts/sync-global.sh --fix`, verify `agent.adv-claude.prompt` references `{file:./agent-parts/advance/adv.md}` and the provider hint, then restart OpenCode."
  ```
  — i.e. OpenCode used the markdown agent file body, NOT the `opencode.json` prompt field.
- All four provider variants (`adv-claude`, `adv-gpt`, `adv-glm`, `adv-kimi`) exhibit identical behavior.
- OpenCode official docs ([Agents page → Prompt section](https://opencode.ai/docs/agents/#prompt)) show only single-file form: `"prompt": "{file:./prompts/code-review.txt}"`. No documentation of multi-file concatenation support.

### Root cause

OpenCode's `{file:...}` resolver in `agent.X.prompt` interprets the entire prompt string as either a literal prompt OR a single file reference. The pattern `"{file:A}\n\n{file:B}"` is not a recognized format → OpenCode does not interpolate either ref → falls through to the markdown agent file body (which exists for tool/permission frontmatter and contains the stub diagnostic as a fallback warning).

### Secondary defect: `--check` false positive

`bash scripts/sync-global.sh --check` currently verifies that the prompt-ref string in `opencode.json` matches the expected literal (multi-`{file:}` form) and that source files exist. It does NOT verify that OpenCode actually expands the ref at runtime. Result: `--check` reports "config already correct" while the runtime is broken. This false-positive is in scope for this change — `--check` must be tightened so it catches both the legacy multi-ref form (post-migration drift) AND staleness of the new concatenated file vs canonical sources (see §Constraints).

### Impact

- All ADV changes drafted via `/adv-proposal` from a Claude/GLM/GPT/Kimi provider session run with **degraded orchestrator persona** — the agent has rules (instructions array works) but is missing the workflow contract, voice contract, gate machine guidance, sub-agent policy, and output contract from `adv.md`.
- ADV's own troubleshooting doc anticipates this failure mode (`docs/provider-agent-assembly.md` line 90) but doesn't solve it — the listed remediation ("Inspect prompt refs and files") is empty advice when refs are correct and files exist.
- No alert at session start beyond the agent body's stub diagnostic. Operators may not realize they're running degraded.
- Blocks the OCA 5-change split (#1–5) — `/adv-proposal` runs on those depend on a working orchestrator persona.

---

## Success Criteria

**Outcome-level (must be true after this change ships):**

- [ ] `sync-global.sh --fix` produces a single per-provider concatenated file under `~/.config/opencode/agent-parts/advance/adv-{provider}.md` containing canonical `adv.md` + provider hint joined.
- [ ] `opencode.json` `agent.adv-{provider}.prompt` is rewritten to a single ref: `{file:./agent-parts/advance/adv-{provider}.md}`.
- [ ] OpenCode resolves the agent prompt to the full canonical body + provider hint (no stub diagnostic in the resolved prompt) for all four provider variants.
- [ ] Tool drift detection (`check_tool_drift`) continues to verify canonical `adv.md` + each generated variant frontmatter; semantics unchanged.
- [ ] Skinny stub design preserved at the `agents/adv-{provider}.md` body level — frontmatter only, stub body kept as fallback diagnostic for any other expansion failure mode.
- [ ] `provider-eval.ts` metric `selected_agent_runtime_prompt` remains correct after switching to single-file source.
- [ ] Migration is transparent: an operator running `sync-global.sh --fix` against a config with the legacy multi-ref form gets it migrated to single-ref + the new concatenated file generated.
- [ ] **`--check` tightened** — detects (a) legacy multi-ref form in opencode.json, (b) missing concatenated file under `agent-parts/advance/adv-{provider}.md`, AND (c) staleness of the concatenated file vs current canonical `adv.md` or provider hint (mtime or content-hash compare).
- [ ] Concrete acceptance verification steps below all pass.

(Acceptance criteria below are the operator-runnable verification steps proving these outcomes.)

---

## Out of Scope

- Replacing OpenCode's `{file:...}` resolver — that's an upstream OpenCode change. This proposal works around the limitation, not fixes OpenCode itself.
- Changing the canonical `adv.md` content.
- Changing provider-hint content.
- OCA-side changes (this is purely ADV-internal).
- Bundling provider variants for non-supported providers (out of scope).

---

## Acceptance Criteria (operator-verifiable)

1. Run `bash ~/dev/oc-plugins/advance/scripts/sync-global.sh --fix` on a clean checkout post-change.
2. Verify `~/.config/opencode/agent-parts/advance/adv-claude.md` exists and equals `cat <canonical-adv.md> <providers/claude.md>` byte-for-byte (modulo a single `\n\n` separator). Expected size ≈ canonical `adv.md` size + provider hint size + 2 bytes.
3. Verify `opencode.json` `agent.adv-claude.prompt` is `"{file:./agent-parts/advance/adv-claude.md}"` (single ref).
4. Run `opencode debug agent adv-claude` and capture the resolved `prompt` field. Deterministic checks:
   - Does NOT contain the literal string `[ADV:PROVIDER_STUB_UNEXPANDED]`.
   - DOES contain stable section markers from canonical `adv.md`: `## Step 3: Gate Machine`, `## ADV State Access Policy`, `## Output Contract`.
   - DOES contain provider hint marker: `<!-- PROVIDER_HINT:claude -->`.
   - Resolved prompt length is within ±2% of `wc -c` of (canonical adv.md + provider hint).
5. Repeat step 4 for `adv-gpt`, `adv-glm`, `adv-kimi` (substituting the matching `<!-- PROVIDER_HINT:{provider} -->` marker).
6. Run `--check` against the post-fix state — passes cleanly.
7. Edit `opencode.json` to revert `agent.adv-claude.prompt` to the legacy multi-ref form; run `--check` — flags the broken state.
8. Touch the canonical `~/.config/opencode/agent-parts/advance/adv.md` (e.g. append a newline) so the concatenated `adv-claude.md` becomes stale; run `--check` — flags staleness via mtime or content-hash compare.
9. Re-run `--fix` after step 8 — concatenated file regenerates, `--check` passes again.

---

## Constraints

- MUST NOT change canonical `adv.md` source content.
- MUST NOT change provider-hint source content.
- MUST preserve "skinny stub" at `agents/adv-{provider}.md` (frontmatter only, body = stub diagnostic). This is the safety net for the case where opencode.json prompt fails for any other reason.
- MUST be backwards-compatible: an operator running `sync-global.sh --fix` against an old config (multi-ref prompt) gets it migrated to single-ref transparently.
- MUST keep size metrics (`generated_provider_file`, `selected_agent_runtime_prompt`) correct after the change.
- MUST NOT introduce per-provider canonical body forks — concatenation happens at sync time, not source.
- File generation MUST be idempotent (running `--fix` twice produces no diff the second time).
- `--check` MUST detect staleness of the concatenated file vs canonical sources. Mechanism (mtime compare, content hash, or both) is a discovery item; the requirement is that a stale concatenated file is reported as a failure, not a pass.

---

## Discovery Agenda

The discovery phase MUST address:

1. **Confirm OpenCode resolver behavior:** empirically verify single-`{file:...}` form works for ADV prompt refs (e.g. write a test single-ref to a temporary agent in opencode.json, run `opencode debug agent X`, observe prompt expansion). Rule out other failure modes before committing to the single-file fix.
2. **Source-of-truth ordering:** decide concatenation order in the generated file. Probably: canonical adv.md first, then provider hint (matches current opencode.json multi-ref order). Verify provider hint section header (`## Provider Hint`) renders correctly post-canonical-body.
3. **Source-file disposition (resolved by constraint):** canonical `agent-parts/advance/adv.md` and `agent-parts/advance/providers/{provider}.md` MUST remain on disk as the source-of-truth inputs to concatenation — required by the "no per-provider canonical body forks" constraint above. No discovery work needed; recorded here only to close the question explicitly.

3a. **Staleness-detection mechanism:** decide between mtime compare (cheaper, sensitive to touch), content-hash compare (more robust, costlier), or both. Trade-off discovery item only; the staleness-must-be-detected requirement itself is locked.
4. **Version detection:** determine if newer OpenCode versions support multi-`{file:...}` (check OpenCode changelog and release notes). If a future version adds support, the single-file workaround is still valid (matches docs) but the multi-ref form would also work.
5. **Test coverage:** existing `plugin/src/sync-global.test.ts` tests for `[ADV:PROVIDER_STUB_UNEXPANDED]`. Add tests that verify the single-concatenated-file output is correct + that opencode.json prompt is single-ref form.
6. **Documentation update:** `docs/provider-agent-assembly.md` Design Principles point 3 currently describes multi-ref; rewrite to describe single-concatenated-file mechanism + cite OpenCode official docs single-ref pattern.

---

## Resolution Options Considered

| Option | Approach | Verdict |
|---|---|---|
| **A — Single-file concatenation at sync time (Recommended)** | sync-global.sh produces `agent-parts/advance/adv-{provider}.md` per provider (canonical + hint joined); opencode.json prompt = single ref. Matches OpenCode docs pattern. Preserves skinny stub safety net. | **Adopt.** LBP, minimal sync-script change, no canonical forks. |
| B — Inline full body in agent stub | Put canonical adv.md content directly into `agents/adv-{provider}.md` body. No prompt-ref indirection. | Loses skinny stub design benefit; canonical body lives in 4 places (one per provider). Larger diff, harder to keep in sync. |
| C — Wait for OpenCode upstream support of multi-`{file:...}` | File issue with OpenCode; degraded mode in meantime. | Slowest; ecosystem timeline uncertain; ADV operators stuck in degraded mode. |
| D — Operator workaround only | Manually edit `agents/adv-{provider}.md` body to contain concatenated content. Survives until next sync. | Not a solution — workaround for an individual operator, doesn't fix the design. (NOTE: this workaround is what this user applied immediately to unblock the OCA #1–5 work.) |

---

## Risks

| Risk | Mitigation |
|---|---|
| Newer OpenCode adds multi-`{file:...}` support, making this change unnecessary | Single-file form per docs is supported NOW and will continue to be supported. Forward-compatible. |
| Concatenation file becomes stale between `--fix` runs (canonical adv.md edited, concatenated output not regenerated) | Two-layer mitigation: (1) `--check` detects staleness via mtime or content-hash compare against canonical sources and fails loudly; (2) `--fix` always regenerates idempotently from canonical sources. Operators are expected to run `--fix` after editing canonical sources; `--check` catches the case where they don't. |
| Operators with old multi-ref opencode.json get partial state during migration | `--fix` handles migration: detects old form, rewrites to new form, generates concatenation file. Single command transparent. |
| Provider hint section header (`## Provider Hint`) breaks markdown structure post-concatenation | Discovery item: verify rendering; canonical `adv.md` ends with `## ADV State Access Policy` section, provider hint adds new top-level `## Provider Hint` — should compose cleanly. |
| Skinny stub diagnostic still fires if user's OpenCode is even older / lacks `{file:...}` resolver entirely | Preserved as safety net at `agents/adv-{provider}.md` body level; this change doesn't remove it. |

---

## Composition with other changes

- **Blocker for OCA #1–5:** `/adv-proposal` runs on those depend on a working orchestrator persona. Fix this first.
- **Independent of OCA #1–3:** doesn't touch OCA code.
- **Independent of ADV #4 (idle worker reaper):** different subsystem (sync vs Temporal worker).
- **Independent of ADV #5 (coordinated session marker):** different subsystem (sync vs marker emission).

---

## Operator-Side Workaround (current state, post-2026-05-03)

This user applied workaround D to unblock the OCA work immediately:

```bash
# Backup current stub
cp ~/.config/opencode/agents/adv-claude.md \
   ~/.config/opencode/agents/adv-claude.md.bak.$(date +%Y%m%d-%H%M%S)

# Concatenate canonical + provider hint into the agent file body
python3 <<'EOF'
from pathlib import Path
agent_path = Path.home() / '.config/opencode/agents/adv-claude.md'
adv_md = (Path.home() / '.config/opencode/agent-parts/advance/adv.md').read_text()
provider_md = (Path.home() / '.config/opencode/agent-parts/advance/providers/claude.md').read_text()
current = agent_path.read_text()
parts = current.split('---\n')
frontmatter = parts[1]
new_body = adv_md.rstrip() + '\n\n' + provider_md.rstrip() + '\n'
agent_path.write_text('---\n' + frontmatter + '---\n\n' + new_body)
EOF
```

This survives until the next `sync-global.sh --fix` run, which will re-stub the body. After ADV change #6 (this proposal) ships, the workaround becomes unnecessary — `--fix` will produce a working state.

---

## Companion files to update on archive (in ADV repo)

- `scripts/sync-global.sh` — concatenation logic, single-ref prompt patching
- `plugin/src/sync-global.test.ts` — test coverage for new mechanism
- `plugin/src/overlay-sync-assets.test.ts` — adjust expectations if needed
- `docs/provider-agent-assembly.md` — rewrite §Design Principles point 3 + §Sync Behavior §Generation step 6
- `docs/provider-adv-smoke-checklist.md` — update verification steps
- `scripts/provider-eval.ts` — verify `selected_agent_runtime_prompt` measurement still correct
