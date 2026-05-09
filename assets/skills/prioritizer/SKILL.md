---
name: prioritizer
description: "Context-aware tradeoff analysis — researches codebase and domain to draft criteria questions for user prioritization. Use when facing 2+ viable approaches with meaningful tradeoffs."
keywords: ["tradeoff", "prioritize", "criteria", "decision", "architecture-choice", "approach-comparison", "which-approach"]
license: MIT
metadata:
  priority: medium
  replaces: "prioritizer agent (was ~/.config/opencode/agents/prioritizer.md)"
---

## When to Load This Skill

Load this skill when facing a decision with **2+ viable approaches** that differ on real tradeoffs, and the "best" answer depends on what the user values most.

**Skip when:**
- The user gave explicit constraints ("make it fast", "keep it simple")
- There's only one reasonable approach
- The decision is trivial or easily reversible
- You already collected priorities earlier in this conversation
- The choice is constrained by security, API compatibility, or architecture

## Workflow

### Phase 1: Understand Context (30 seconds max)

Quickly scan the relevant code to understand:
- What patterns already exist in this area?
- What dependencies are involved?
- What's the current quality bar (tests, types, docs)?
- Are there existing conventions that constrain the choice?

Use `lgrep_search_semantic` for concept discovery, `lgrep_get_file_outline` for structure, and `read` for specific files.

### Phase 2: Research Tradeoff Space (30 seconds max)

For the specific decision domain, identify:
- What are the real tradeoffs between the approaches?
- What does the ecosystem recommend?
- Are there known pitfalls with any approach?

Use Context7 for library docs, Kagi for broader context, gh_grep_searchGitHub for real-world patterns.

### Phase 3: Draft Criteria Questions

Based on your analysis, produce **4-5 criteria questions** that capture the actual tradeoffs for THIS specific decision. Do NOT use generic criteria — tailor them.

### Phase 4: Present to User

Use the `question` tool with the drafted criteria. Format:

```json
{
  "questions": [
    {
      "header": "{Criterion 1 — domain-specific name}",
      "question": "{Context-aware question referencing the actual decision}",
      "options": [
        { "label": "Critical", "description": "{What Critical means for THIS criterion}" },
        { "label": "High", "description": "{What High means here}" },
        { "label": "Medium", "description": "{What Medium means here}" },
        { "label": "Low", "description": "{What Low means here}" }
      ]
    }
  ]
}
```

### Phase 5: Interpret and Recommend

After receiving user priorities:
1. Restate the priorities back to the user
2. Map priorities to approaches using the decision map
3. Recommend the approach that best fits their stated priorities
4. Note any tensions (e.g., "You rated X as Critical but Y as High — these conflict slightly")

## Rules

1. **Always 4 criteria + 1 constraints question** (5 total)
2. **Never use generic criteria** like "Correctness" or "Simplicity" when a domain-specific name is clearer (e.g., "Type safety" instead of "Correctness", "Bundle size" instead of "Performance")
3. **Option descriptions must reference the actual approaches** — "Prefer X even if Y" not "Optimize for this"
4. **The constraints question must offer context-specific constraints**, not generic ones
5. **Include a DECISION MAP** so you know how to interpret the answers
6. **Keep research fast** — 60 seconds total, not exhaustive
7. **If the codebase already has strong conventions**, note them — they may override any criterion

## Output Format

Before presenting questions to the user, document:

```
CONTEXT SUMMARY:
{2-3 sentences about current state and tradeoff space}

APPROACHES IDENTIFIED:
- {Approach A}: {1-line summary with key tradeoff}
- {Approach B}: {1-line summary with key tradeoff}

DECISION MAP:
- If {Criterion 1}=Critical → favors {Approach X}
- If {Criterion 2}=Critical → favors {Approach Y}
```

Then present the criteria questions via the `question` tool.
