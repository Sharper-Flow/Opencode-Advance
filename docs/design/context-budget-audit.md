# Context Budget Audit

**Status:** Active
**Date:** 2026-05-09
**Tracked:** OCA #32

---

## Budget Summary

| Layer | Files | Words | Est. Tokens (~1.3 tok/word) | Load Mode |
|-------|-------|-------|----------------------------|-----------|
| AGENTS.md | 1 | 2,000 | ~2,600 | Always |
| OCA instructions | 11 | 4,949 | ~6,400 | Always |
| Orphan instructions | 3 | 1,807 | ~2,300 | Always |
| **Total always-on** | **15** | **8,756** | **~11,400** | — |
| Skills (on-demand) | 20 | 16,316 | ~21,200 | Per-trigger |
| ADV overlay (agent-specific) | — | — | ~5,000–15,000 | Per-agent |

**Pre-work context cost: ~11.4K tokens** (before user sends any message).

---

## Per-File Budget (Always-On)

### OCA Instructions (managed via `assets/instructions/`)

| File | Words | Verdict | Notes |
|------|-------|---------|-------|
| `rules.yaml` | 1,426 | ✅ Keep | Core policy layer; 33 rules |
| `mcp-tools.md` | 402 | ✅ Keep | Slim routing to skill; 12 lines |
| `shell_strategy.md` | 492 | ✅ Keep | Non-interactive execution rules |
| `lbp.md` | 309 | ✅ Keep | Long-term best practice stance |
| `worktree-guide.md` | 196 | ✅ Keep | Always-on worktree routing |
| `caveman.md` | 199 | ✅ Keep | Terse communication mode |
| `temp_directory.md` | 333 | ⚠️ Review | References "open-chad"; should be updated |
| `lgrep-tools.md` | 158 | ✅ Keep | Slim routing to skill |
| `test_resource_guardrails.md` | 196 | ✅ Keep | Test safety rules |
| `morph-tools.md` | 133 | ✅ Keep | Slim routing to skill |
| `identity.md` | 127 | ✅ Keep | OpenCode identity + config paths |

### Orphan Instructions (not in OCA assets — manually placed)

| File | Words | Verdict | Notes |
|------|-------|---------|-------|
| `criteria-prioritizer.md` | 1,250 | ❌ **Remove** | Duplicates `prioritizer` skill (578 words, loaded on-demand). Replace with 1-line routing instruction. Saves ~1,600 tokens. |
| `post_install_verification.md` | 529 | ❌ **Remove** | References "open-chad" (legacy). Post-install verification is a one-time action, not an always-on instruction. Saves ~700 tokens. |
| `global-verify-policy.md` | 28 | ✅ Keep | Slim verification nudge |

---

## Savings Opportunity

| Action | Token Savings |
|--------|--------------|
| Remove `criteria-prioritizer.md` | ~1,600 |
| Remove `post_install_verification.md` | ~700 |
| Update `temp_directory.md` references | 0 (cosmetic) |
| **Total** | **~2,300** |

Post-cleanup always-on budget: **~9,100 tokens** (down from ~11,400).

---

## Skills Budget (On-Demand)

Skills load per-trigger, not always. Only 1-3 skills typically active per session.

### OCA-Owned Skills

| Skill | Words | Load Trigger |
|-------|-------|-------------|
| `lgrep` | 1,533 | Code exploration queries |
| `mcp-selection` | 611 | Tool choice decisions |
| `prioritizer` | 578 | Tradeoff analysis |
| `caveman` | 441 | User requests / caveman mode |
| `caveman-review` | 438 | PR review tasks |
| `caveman-commit` | 374 | Commit message generation |
| `worktree` | 345 | Worktree operations |
| `morph` | 160 | Large file edits |

**Total OCA skills: 4,480 words (~5,800 tokens)**

### Advance-Owned Skills (via sync-global.sh)

| Skill | Words | Load Trigger |
|-------|-------|-------------|
| `eagle-ux-review` | 1,830 | UX evaluation tasks |
| `adv-ci-release` | 1,632 | CI/release commands |
| `pokeedge-visual-audit` | 1,449 | PokeEdge visual audit |
| `adv-backend-stack-eval` | 1,335 | Backend tech selection |
| `global-verify` | 1,033 | Verification workflows |
| `adv-worktree` | 935 | ADV worktree commands |
| `adv-slop-detection` | 883 | Slop scan commands |
| `adv-user-intuit` | 793 | User intuition protocol |
| `adv-tron` | 767 | Codebase recon |
| `sharperflow-web-standards` | 713 | Web stack standards |
| `adv-arch-detection` | 632 | Architecture scan |
| `adv-comp-research` | 359 | Competitive research |

**Total ADV skills: 11,361 words (~14,800 tokens)**

---

## Recommendations

1. **Remove `criteria-prioritizer.md`** from `~/.config/opencode/instructions/`. It duplicates the `prioritizer` skill which loads on-demand. Replace with a one-liner: "When facing tradeoff decisions with 2+ viable approaches, load `skill("prioritizer")`."

2. **Remove `post_install_verification.md`** from `~/.config/opencode/instructions/`. It references legacy "open-chad" branding and describes a one-time post-install action that doesn't need to load every session.

3. **Update `temp_directory.md`** to replace "open-chad" references with "OpenCode Advance" / "oca".

4. **Future optimization:** Consider lazy-loading non-critical instructions (worktree-guide, caveman, morph-tools) via skill triggers instead of always-on instruction loading. This would reduce the base budget by ~500 tokens but requires OCA template changes.

---

## Methodology

- Word counts via `wc -w` on source files
- Token estimate: 1.3 tokens per word (conservative average for mixed prose/code)
- Always-on = loaded via `opencode.json` `.instructions` array or AGENTS.md
- On-demand = loaded via `skill("name")` trigger
- ADV overlay budget estimated from ADV instruction size ranges
