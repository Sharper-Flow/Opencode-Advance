# Context Budget Audit + Instruction Loading Diet

**Status:** Proposal — ready for future `/adv-proposal` once ADV provider prompt refs are healthy  
**Date:** 2026-05-03  
**Target repo:** `opencodeadvance` (OCA), with possible follow-up in Advance  
**Change ID (suggested):** `contextBudgetAudit`  
**Sequence:** Ship after the ADV provider prompt-ref fix; can run independently of Pattern B/hibernation.  
**Estimated effort:** 1–3 days for OCA audit/reporting; additional ADV follow-up if lazy instruction loading needs plugin/agent changes.

---

## TL;DR

Add a first-class context-budget audit so OCA can measure, report, and reduce baseline prompt load without weakening strict tool, shell, security, MCP, lgrep, morph, or ADV safety instructions.

The goal is not “shorter rules.” The goal is **right-sized loading**: keep mandatory tool/safety guidance detailed and always available where needed, while moving phase-specific or agent-specific methodology out of every session's baseline context.

---

## Problem Statement

The current OpenCode + Advance setup loads a large baseline instruction stack:

- OCA/global instructions (`rules.yaml`, shell strategy, MCP selection, lgrep, morph, worktree, temp/cache, LBP, caveman, etc.)
- Advance workflow instructions (`ADV_INSTRUCTIONS.md`)
- Agent prompts and provider hints
- Tool schemas for all tools exposed to the active agent
- Slash-command and skill metadata

This creates token pressure before user/project context enters the conversation. The strict parts are valuable and should remain detailed, but not every agent/session needs every workflow phase, methodology, or operational reference in baseline prompt.

Pain points:

- Baseline context is hard to see or quantify.
- Largest prompt contributors are not surfaced by `oca doctor`.
- Tool-schema load varies by agent but is not reported.
- Non-ADV sessions can still inherit ADV-heavy workflow detail.
- Phase-specific ADV guidance often belongs in commands/skills rather than always-loaded global instructions.
- Users cannot make informed tradeoffs between strictness and prompt budget because OCA does not show the budget.

---

## Success Criteria

- [ ] `oca doctor --scope context` reports estimated baseline context cost for the active rendered OpenCode config.
- [ ] Report separates at least: instructions, agent prompts, plugin-provided instruction refs, command metadata, skill metadata, and exposed tool schemas.
- [ ] Report includes top contributors by file/tool/agent with path/name and estimated tokens.
- [ ] Report compares agent profiles, e.g. `adv`, `build`, `explore`, `librarian`, `mechanic`, `general`, showing tool count and estimated schema load per agent.
- [ ] Report flags likely avoidable baseline load, such as phase-specific methodology in always-loaded instructions or unused MCP/tool exposure.
- [ ] Strict tool/safety instructions remain present where required: shell non-interactive policy, permission/destructive-action policy, MCP invocation rules, lgrep/morph routing rules, TDD/ADV safety rules for ADV agents.
- [ ] OCA docs explain the intended split: global safety/tool law vs on-demand methodology skills/commands.
- [ ] If OpenCode config supports it, OCA renders lighter non-ADV instruction profiles so routine non-ADV sessions do not load full ADV workflow detail. If not supported, the change documents the upstream limitation and files a follow-up.

---

## Non-Goals

- Do **not** weaken or summarize strict tool instructions in a way that loses enforceability.
- Do **not** remove security/destructive-action confirmation guidance.
- Do **not** remove ADV workflow law from ADV sessions.
- Do **not** rely on model memory or undocumented behavior for safety-critical rules.
- Do **not** optimize purely for smallest prompt. Optimize for reliable behavior per token.

---

## Proposed User Experience

```text
$ oca doctor --scope context
Context budget
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ rendered config          ~/.config/opencode/opencode.json
  ✓ instructions             42k est tokens across 12 files
  ✓ agent prompts            18k est tokens across 9 agents
  ✓ tool schemas             68k est tokens for adv, 14k for explore
  ⚠ always-loaded ADV detail  ADV_INSTRUCTIONS.md loaded for non-ADV default agent

Top contributors
  18k  ADV_INSTRUCTIONS.md
   7k  mcp-tools.md
   5k  rules.yaml
   4k  worktree-guide.md
  31k  adv tool schemas

Agent profile comparison
  adv        74 tools  68k schema tokens  78k instructions/prompts
  build      42 tools  39k schema tokens  52k instructions/prompts
  explore     8 tools   9k schema tokens  31k instructions/prompts
  librarian   7 tools   8k schema tokens  28k instructions/prompts

Recommendations
  - Keep shell_strategy.md global: safety-critical.
  - Keep mcp-tools.md global until MCP invocation bugs are resolved.
  - Move worktree-guide.md to on-demand skill for non-worktree sessions.
  - Split ADV global law from phase-specific command methodology.
  - Disable unused MCP servers from build/general agent tool allowlists.

1 warning, 0 errors. context audit complete.
```

---

## Design Direction

### 1. Measure before changing

Add read-only audit logic first. Count estimated tokens for:

- files in `opencode.json.instructions`
- plugin-appended instructions, including Advance refs
- rendered agent prompts/provider prompts
- command definitions if included in runtime context
- skill manifests/metadata if included in runtime context
- tool schemas per agent allowlist

Use a deterministic approximation if a provider-specific tokenizer is unavailable. The report must label estimates as estimates.

### 2. Separate law from methodology

Classify instruction content into:

| Class | Loading posture |
|---|---|
| Safety law | Always loaded for relevant agents. Must remain strict. |
| Tool invocation law | Always loaded where tool is exposed. Must remain detailed. |
| Agent role contract | Loaded for that agent only. |
| Workflow phase methodology | Command/skill loaded on demand. |
| Domain methodology | Skill loaded on demand. |
| Reference docs | Not loaded; read/fetch when needed. |

### 3. Prefer agent-specific exposure

Tool schema load should follow agent role:

- `explore`: lgrep/read/search only; no write or shell.
- `librarian`: web/docs/examples; no local shell.
- `mechanic`: infra/system tools.
- `adv-engineer`: implementation/evidence tools, narrow ADV reads.
- `adv`: full workflow orchestration only when ADV work is active.

The context audit should make violations visible.

### 4. Keep strict tool docs intact

Do not compress the content that prevents known failure modes:

- shell non-interactive rules
- MCP native-tool invocation rules
- Context7/OpenCode naming caveats
- lgrep-first exploration policy
- morph-edit selection policy
- destructive git/shell guardrails
- ADV cancellation/doom-loop/TDD/gate rules for ADV sessions

Savings should come from conditional loading and tool allowlist trimming, not from removing detail from critical rules.

---

## Discovery Agenda

1. Determine exactly what OpenCode injects into model context for instructions, agents, tools, commands, and skills.
2. Determine whether OpenCode supports per-agent/per-command instruction profiles today.
3. Identify which OCA instructions are truly global vs tool-specific vs agent-specific.
4. Identify which Advance instruction sections are global ADV law vs phase-specific command methodology.
5. Confirm whether tool schema size can be estimated from rendered OpenCode config alone or needs runtime introspection.
6. Decide token estimator: provider tokenizer, OpenAI-compatible tokenizer, or deterministic byte/4 approximation with explicit caveat.
7. Define warning thresholds for baseline context load and per-agent tool-schema load.

---

## Acceptance Criteria

1. Running `oca doctor --scope context` on the operator's real config prints a context budget report with top contributors and agent comparison.
2. The command is read-only and does not require network access.
3. The report clearly distinguishes exact counts from estimates.
4. At least one recommendation is generated for the current OCA/Advance setup.
5. Tests cover missing files, plugin-appended instruction refs, duplicate instruction refs, and disabled MCP/tool profiles.
6. Documentation states that strict tool/safety instructions remain detailed by design.

---

## Risks

| Risk | Mitigation |
|---|---|
| Token estimates differ from provider tokenizer | Label as estimate; allow provider-specific estimator later. |
| OpenCode does not expose enough context assembly detail | Report known rendered config inputs and document unknown runtime additions. |
| Overzealous trimming weakens agent behavior | Gate reductions through explicit classification; safety/tool law cannot be trimmed. |
| ADV split requires upstream plugin changes | Treat OCA audit/reporting as first step; file ADV follow-up only with evidence. |

---

## Companion files to update on archive

- `docs/design/cli-surface.md` — document `oca doctor --scope context`
- `docs/design/stack-toml-schema.md` — document any new instruction/profile knobs if added
- `assets/instructions/README.md` — classify OCA-owned instructions by loading posture
- `assets/skills/README.md` — identify methodology intended for on-demand loading
- `docs/proposals/phases.md` — mark context-budget track shipped/updated
