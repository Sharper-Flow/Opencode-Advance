# Agreement

## Objectives

1. Add offline `oca maintain` for deterministic ADV/OCA self-update recovery when no sessions/workers are active.
2. Detect plugin source-vs-dist drift and expose exact checkout/dist/build-marker evidence.
3. Make verified worktree fixes available to the next OpenCode session by merging safe branches and rebuilding main-checkout plugin dist.
4. Preserve trunk-is-prod by merging only archived changes with full release verification.
5. Keep Temporal runtime mutation conservative: report health/recovery in v1 unless safe standalone execute surfaces exist.
6. Clean only merged, clean, session-free worktrees.
7. Provide stable text and JSON output for operators/scripts.

## Acceptance Criteria

1. `oca maintain --dry-run` reports session/worker blockers, merge candidates, rebuild needs, cleanup candidates, and Temporal/ADV health checks without mutation.
2. `oca maintain --execute` refuses before mutation if any OpenCode session, OCA session, or ADV/Temporal worker for the project is active.
3. Execute mode merges only branches with archived ADV state and release gate complete (`release` gate status `done`), treated as the full verification proof for v1.
4. Branches missing release completion, with merge conflicts, or not archived are skipped with actionable diagnostics.
5. Runtime plugin source/dist drift is detected and reported with exact checkout/dist paths.
6. Main-checkout plugin dist rebuild writes/reports a git SHA build marker stronger than mtime.
7. Next OpenCode session can verify loaded plugin path/build marker.
8. Worktree cleanup removes only merged, clean, session-free worktrees.
9. Temporal recovery is report-only in v1 unless a safe standalone execute surface exists.
10. Text and JSON output are stable enough for scripting.
11. OCA tests pass with `go test ./...`; Advance tests pass for any added drift/diagnostic code.

## Constraints

- No hot reload or mid-session plugin reload.
- No OpenCode core plugin-path changes.
- No auto-killing sessions/workers.
- No non-archived or release-incomplete branch merges.
- No force-push, amend, or destructive git history mutation.
- OCA must not duplicate ADV state authority; consume ADV-safe standalone surfaces or report recovery actions.
- Temporal Worker Versioning is deferred; v1 uses offline quiescence + build marker.

## Avoidances

- Do not treat Temporal worker restart as plugin tool-code reload.
- Do not rely on mtime alone for runtime freshness.
- Do not delete dirty, unmerged, or session-active worktrees.
- Do not run broad Temporal repair mutations as part of first version.

## Decisions

### User Decisions

1. Session/worker gate: hard refuse when OpenCode/OCA/ADV/Temporal activity is detected. Why: Temporal docs say worker shutdown can interrupt in-flight Workflow Tasks/Activities.
2. Merge authority: dry-run by default; explicit `--execute` required for merge/rebuild/cleanup mutation.
3. Verification proof: require archived ADV state plus release gate complete as the v1 full-verification proof before merge to trunk.

### Agent Decisions (LBP)

1. Use quiescent offline maintenance for v1 instead of hot reload or auto-kill.
2. Use git SHA/build marker for plugin dist freshness; reserve full Temporal Worker Versioning for a later change.
3. Keep Temporal recovery advisory/report-only unless an existing safe standalone execute surface is available.
4. Reuse OCA plugin build/update primitives, ADV recovery/triage surfaces, occupancy/session scanners, and worktree census patterns.

## Deferred Questions

- Exact build marker storage format can be refined in implementation but must include schema version, source root, git SHA, built time, and build command hash.
- Full Temporal Worker Versioning integration is deferred to a later change.

## Sign-Off

User approved acceptance criteria with reply `approve` on 2026-05-04.