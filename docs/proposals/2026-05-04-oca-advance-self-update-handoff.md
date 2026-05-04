# OCA Advance Self-Update Handoff

**Status:** Draft roadmap item — OCA-owned companion to `Sharper-Flow/Advance#40`  
**Date:** 2026-05-04  
**Project:** OpenCode Advance (`~/dev/opencodeadvance`)  
**Priority:** SHOULD (`S7`)  

## Problem

When Advance changes its own plugin runtime code, the running OpenCode process keeps using the plugin bundle loaded at session startup. Advance can detect and explain the mismatch, but OCA owns the environment layer that rebuilds plugins, wires `opencode.json`, and starts/reuses sessions and windows.

Without an OCA-owned handoff, operators must guess whether to rebuild the main checkout or worktree, which plugin path OpenCode will load, and which session/window to restart.

## Ownership boundary

| Layer | Owner | Responsibility |
|---|---|---|
| Loaded runtime provenance | Advance | Report loaded plugin path, dist/source freshness, branch/HEAD, worker-bundle freshness, and cwd/worktree mismatch. |
| Build/update/install lifecycle | OCA | Run declared plugin build commands, update configured plugin path, and validate source/dist freshness. |
| Session/window handoff | OCA | Start or ensure an OpenCode session/window in the exact checkout/worktree that should load the rebuilt plugin. |

## Long-term solution

Add an OCA handoff path that pairs with Advance runtime diagnostics:

1. `oca doctor --scope plugins` warns when Advance `src/` is newer than `dist/` or configured plugin path points at a different checkout than the active worktree.
2. `oca update advance` / plugin prepare path rebuilds the configured Advance checkout deterministically using the stack build commands.
3. Add a documented handoff command or guidance path that opens/reuses an OCA project window in the intended worktree after rebuild.
4. Output explicitly distinguishes:
   - Temporal worker restart reloads worker bundle only.
   - OpenCode host/tool code reload requires a fresh OpenCode process/session.

## Acceptance criteria

- OCA can detect stale Advance `dist/index.js` relative to source files for the configured checkout.
- OCA can rebuild the configured Advance plugin checkout with existing `build` commands and report the rebuilt artifact path.
- OCA can print deterministic next-step guidance for opening/reusing a session/window in the correct checkout/worktree.
- OCA docs link to the Advance-side runtime provenance diagnostic and `Sharper-Flow/Advance#40`.
- OCA does not duplicate Advance-owned diagnostics or ADV assets.

## Related

- Advance issue: `Sharper-Flow/Advance#40`
- OCA issue: `Sharper-Flow/Opencode-Advance#9`
- Existing OCA seams: `internal/plugin/prepare.go`, `internal/plugin/build.go`, `internal/health/plugin.go`, `cmd/oca/session.go`, `internal/session/session.go`
