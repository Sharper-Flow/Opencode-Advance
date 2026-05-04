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

The README lists 10 instruction files OCA owns, including `identity.md`,
`rules.yaml`, `shell_strategy.md`, `mcp-tools.md`, `lgrep-tools.md`, and others.
Those files are not present in the repo, so OCA cannot cleanly reproduce the
global OpenCode instruction set during cutover.

This leaves the live environment dependent on pre-OCA global files rather than
OCA as source of truth.

### Pre-flight finding (2026-05-04)

Verification against the live `~/.config/opencode/instructions/` directory
shows **14 files**, not 10. The four files present live but missing from the
README inventory:

| Live file | Notes |
|---|---|
| `caveman.md` | Caveman-mode instructions (related to caveman skill family). |
| `criteria-prioritizer.md` | 8.9 KB; possibly older asset; likely superseded by the `prioritizer/` skill — confirm before copying. |
| `global-verify-policy.md` | Added 2026-05-03; references `/check` workflow. |
| `post_install_verification.md` | Added 2026-05-03; installer verification doc. |

Discovery MUST triage each: (a) OCA-owned → copy into `assets/instructions/` and
add to README; (b) stale → retire from live dir on next `oca apply`; (c) move
to a different ownership (skill/plugin). Do not silently drop files the live
environment depends on today.

See [`../notes/2026-05-04-m-queue-preflight-verification.md`](../notes/2026-05-04-m-queue-preflight-verification.md).

---

## Success Criteria

- [ ] `assets/instructions/` contains all OCA-owned instruction files listed in
      its README, **after the live-vs-README discrepancy above is triaged**.
- [ ] README inventory matches the set of files in `assets/instructions/`
      byte-for-byte (no live files missing from README; no README entries
      pointing at absent files).
- [ ] Each of the 4 pre-flight-flagged files (`caveman.md`,
      `criteria-prioritizer.md`, `global-verify-policy.md`,
      `post_install_verification.md`) has an explicit ownership decision
      recorded in the change wisdom: copied / retired / re-homed elsewhere.
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

1. Triage the 4 pre-flight-flagged files (decision per file: copy / retire /
   re-home).
2. Copy canonical current non-ADV instruction files (the 10 originally listed
   plus any kept from step 1) into `assets/instructions/`.
3. Add planning/apply logic to deploy these files into
   `~/.config/opencode/instructions/` or the configured OCA config dir.
4. Ensure stack examples use deployed global paths, not nonexistent repo-local
   placeholders.
5. Update README inventory to match actual files (drop retired entries; add
   any newly-owned entries).
6. Add a docs check that fails if the README inventory drifts from the
   actual contents of `assets/instructions/`.

---

## Acceptance Criteria

1. Fresh temp config dir + `oca apply --target instructions` writes all listed
   OCA instruction files.
2. Rendered `opencode.json.instructions` entries point at existing files.
3. README-vs-asset-dir parity check (new docs test) passes.
4. Each of the 4 pre-flight-flagged files has a recorded triage decision
   visible in the change's wisdom or proposal addendum.
5. `go test ./...` passes.
