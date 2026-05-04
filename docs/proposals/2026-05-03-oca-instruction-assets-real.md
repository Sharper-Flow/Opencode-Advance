# OCA-Owned Instruction Assets Are Real Files

**Status:** Proposal — staged  
**Date:** 2026-05-03  
**Target repo:** `~/dev/opencodeadvance`  
**Resume from:** `~/dev/opencodeadvance`  
**Suggested change ID:** `instructionAssetsReal`  
**Priority:** MUST

---

## Problem Statement

OCA declares ownership of environment-level instructions, but
`assets/instructions/` contains only `README.md`.

The README lists instruction files OCA owns, including `identity.md`,
`rules.yaml`, `shell_strategy.md`, `mcp-tools.md`, `lgrep-tools.md`, and others.
Those files are not present in the repo, so OCA cannot cleanly reproduce the
global OpenCode instruction set during cutover.

This leaves the live environment dependent on pre-OCA global files rather than
OCA as source of truth.

---

## Success Criteria

- [ ] `assets/instructions/` contains all OCA-owned instruction files listed in
      its README.
- [ ] OCA does not copy or own `ADV_INSTRUCTIONS.md` or ADV cost-governance
      instruction files.
- [ ] `oca apply --target instructions` renders deployed instruction paths that
      exist.
- [ ] `oca doctor --scope cross` resolves all declared instruction paths.
- [ ] Tests cover missing source assets and successful instruction deployment.

---

## Out of Scope

- Shortening instruction content for token budget. That belongs to context
  budget audit.
- Changing ADV instruction ownership.

---

## Implementation Sketch

1. Copy canonical current non-ADV instruction files into `assets/instructions/`.
2. Add planning/apply logic to deploy these files into
   `~/.config/opencode/instructions/` or the configured OCA config dir.
3. Ensure stack examples use deployed global paths, not nonexistent repo-local
   placeholders.
4. Update README inventory to match actual files.

---

## Acceptance Criteria

1. Fresh temp config dir + `oca apply --target instructions` writes all listed
   OCA instruction files.
2. Rendered `opencode.json.instructions` entries point at existing files.
3. `go test ./...` passes.
