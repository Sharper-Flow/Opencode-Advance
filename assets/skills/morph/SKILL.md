---
name: morph
description: "morph_edit usage guidance - prefer it for large, scattered, or whitespace-sensitive edits in existing files."
keywords: ["morph_edit", "large-file-edit", "scattered-edits", "whitespace-sensitive", "refactor", "existing-file-edit"]
license: MIT
metadata:
  priority: medium
  replaces: edit
---

## When to Load This Skill

Load this skill when you need to edit an existing file and the change is large,
scattered, or fragile for exact string replacement.

## Prefer `morph_edit` For

- Large files (roughly 300+ lines)
- Multiple non-adjacent edits in one file
- Whitespace-sensitive changes
- Refactors where exact old/new matching is brittle

## Prefer Native Tools For

- Small exact replacements
- Single-line fixes
- New file creation

## Format

Use `// ... existing code ...` markers at the start, between edits, and at the
end so unchanged code is preserved.

Include 1-2 unique context lines around each change and give specific
instructions describing the intent of the edit.

## Fallback

If `morph_edit` fails, fall back to native exact editing tools.
