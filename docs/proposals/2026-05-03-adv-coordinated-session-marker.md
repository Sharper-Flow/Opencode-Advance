# ADV Peer-Session Topology Distinction (formerly: Coordinated Session Marker Reframe)

**Status:** Proposal — ready for `/adv-proposal` · **Date:** 2026-05-03 (rescoped 2026-05-03 post-reconnaissance)
**Target repo:** `Sharper-Flow/Advance` (oc-plugins/advance) — **NOT OpenCode Advance**
**Filing path:** When ready to start work, this proposal must be moved to (or re-drafted in) the ADV repo and `/adv-proposal` must run from inside `~/dev/oc-plugins/advance`. Drafted here in OCA repo for split coherence.
**Change ID (suggested):** `peerSessionTopologyDistinction`
**Parent decision lock:** [`2026-05-03-session-and-resource-architecture.md`](./2026-05-03-session-and-resource-architecture.md) §3.8 + §9 + §10.4
**Split position:** Change #5 of 7 — small enhancement; not a blocker
**Estimated effort:** 1–2 hours (rescoped down from 0.5–1 day)
**Sequence:** Land any time after Pattern B (#1) is in operator hands.

---

## Adaptation Note (read first)

**This proposal was rescoped on 2026-05-03 after reconnaissance discovered the originally proposed marker reframe is largely already shipped by ADV.** When `/adv-proposal` and `/adv-discover` run on this change, the discovery phase MUST:

1. Re-verify the empirical finding: ADV plugin already emits `[ADV:PEER_SESSIONS] N peer session(s) active in this project.` at info level (not warn) via `plugin/src/index.ts:380`. Detection in `plugin/src/utils/peer-sessions.ts` uses git common-dir + ADV project-id (not CWD-equality). Tests at `plugin/src/adv-instructions-assets.test.ts:30` explicitly forbid the old "Concurrent OpenCode sessions detected" wording.
2. Re-read parent §3.8 — note that the warning-vs-info reframe assumption is satisfied; what remains is the topology distinction.
3. Re-scope: this change adds **only** the worktree-topology check that distinguishes "coordinated" (each peer in own worktree, safe) from "conflict" (≥2 peers on shared working tree, real safety risk).

---

## TL;DR

**Originally proposed:** replace `[ADV:WARN] Concurrent OpenCode sessions detected` with `[ADV:COORDINATED]` info-level marker.

**Already shipped (prior change, 2026-04 era):** the marker has been reframed to `[ADV:PEER_SESSIONS] N peer session(s) active` at info level. Detection uses superior topology basis (git common-dir + project-id, not CWD). Tests forbid the old wording.

**This change adds (rescoped):** topology-aware classification — when ≥2 peers detected, run `git worktree list` per peer workdir; if all peers are on distinct worktrees, emit unchanged `[ADV:PEER_SESSIONS]` (info, safe). If any 2 peers share a working tree, ALSO emit `[ADV:WARN] Shared-tree conflict: PIDs <a,b> on tree <path>. Git operations from any of those affect all.` Add an explicit warn signal for the real-conflict case while preserving the existing info marker for the safe case.

---

## Problem Statement

### Verified empirical state (2026-05-03)

- `plugin/src/index.ts:367-387` emits `[ADV:PEER_SESSIONS] N peer session(s) active in this project.` at `hooksLogger.info` level when ≥1 peer detected.
- Detection in `plugin/src/utils/peer-sessions.ts` uses Linux `/proc` enumeration filtered to opencode processes, then matches via `git common-dir` OR ADV project-id. Privacy-defensive schema (PID + cwd internal only).
- Tests in `plugin/src/adv-instructions-assets.test.ts:29-30` explicitly assert: `expect(content).not.toMatch(/Concurrent OpenCode sessions detected/)`.
- The marker is **purely informational** — every peer gets counted equally regardless of whether they're in distinct worktrees (safe) or sharing a working tree (real safety risk).

### Remaining gap

The current detection collapses two distinct scenarios into one info marker:

| Scenario | Safety | Current emit | Should emit |
|---|---|---|---|
| 8 peers, each in own ADV worktree | SAFE — Temporal serializes state writes; per-worktree git isolation eliminates working-tree races | `[ADV:PEER_SESSIONS] 8 peer session(s) active` (info) | unchanged |
| 2 peers in same trunk checkout | UNSAFE — git ops from one affect the other; race on uncommitted changes | `[ADV:PEER_SESSIONS] 2 peer session(s) active` (info) | **also** `[ADV:WARN] Shared-tree conflict: PIDs <a,b> on tree <path>` |

Operators currently can't distinguish from the marker which scenario they're in. The safe case is the expected operating point under Pattern B (each ADV change worktree → own peer); the unsafe case is operator error or race condition.

---

## Success Criteria

- [ ] When ≥2 peers detected, ADV runs topology check: `git worktree list` per peer workdir → derive distinct vs shared trees.
- [ ] If all peers on distinct worktrees → existing `[ADV:PEER_SESSIONS]` info emit unchanged.
- [ ] If any 2 peers share a working tree → ALSO emit `[ADV:WARN] Shared-tree conflict: PIDs <a,b> on tree <path>. Git operations from any of those affect all.` (warn level via `hooksLogger.warn`).
- [ ] Topology check is cached with TTL (default 30 sec, configurable) to avoid `git worktree list` fanout under heavy peer-detection load.
- [ ] Existing info-level emit remains for the no-conflict case (no regression for the expected operating point).
- [ ] No false positives: peers on distinct worktrees within the same project must NOT trigger the warn.
- [ ] No false negatives: 2+ peers in same trunk MUST trigger the warn.
- [ ] Non-git projects: skip topology check entirely (existing info emit unchanged).
- [ ] Test coverage: integration test simulates both scenarios + verifies correct marker emission.

---

## Out of Scope

- Marker reframe from `[ADV:WARN]` → `[ADV:PEER_SESSIONS]` — already shipped.
- OCA-side status-bar rendering (covered by OCA #3, which already plans forward-compatible parsing).
- Pattern B implementation (covered by OCA #1).
- Cross-project session coordination (parent §3.8 scoped to within-project).
- Replacement of existing peer-session detection mechanism (only adds topology layer on top).

---

## Acceptance Criteria (operator-verifiable)

1. Setup: 8 OCA tmux sessions in same project, each with opencode in its own ADV worktree (Pattern B's expected operating point per OCA #1).
2. Verify `[ADV:PEER_SESSIONS] 8 peer session(s) active in this project.` info emit; NO `[ADV:WARN]` emit.
3. Setup: open a second opencode session in the same trunk checkout (no worktree switch).
4. Verify BOTH `[ADV:PEER_SESSIONS] 9 peer session(s) active` info emit AND `[ADV:WARN] Shared-tree conflict: PIDs <a,b> on tree <trunk-path>` warn emit.
5. Topology cache: trigger 5 rapid `adv_status` calls within 30 sec; verify `git worktree list` invocation count is ≤ 1 per peer (cache hits prevent fanout).
6. Non-git project: cd to a non-git directory; verify peer detection skips topology check; existing emit unchanged.
7. No regression: existing test `adv-instructions-assets.test.ts` still asserts old wording is forbidden.

---

## Constraints

- MUST NOT modify the existing `[ADV:PEER_SESSIONS]` info marker for the safe case.
- MUST NOT regress existing tests forbidding the old `Concurrent OpenCode sessions detected` wording.
- MUST cache `git worktree list` results — fanout at 8+ peers is unacceptable.
- MUST coordinate with OCA #3 (status-bar parser) to ensure new `[ADV:WARN] Shared-tree conflict` marker decode renders correctly. OCA #3's forward-compat parser handles unknown markers gracefully → safe to land OCA #3 first or this change first.
- MUST handle non-git projects gracefully (skip topology check, no error).
- Cache TTL: default 30 sec, configurable via `coordination_topology_cache_ttl_seconds` (or similar) in plugin config.

---

## Discovery Agenda

The discovery phase MUST address:

1. **Cache strategy:** mtime-based, content-hash-based, or fixed-TTL (default 30 sec)? Discovery decision; the must-cache requirement is locked.
2. **Topology check trigger cadence:** every peer-detection emit? Or rate-limited? Tie to existing peer-detection invocation site at `plugin/src/index.ts:373`.
3. **Multi-peer pair detection:** efficient pairing algorithm to identify all ≥2-peer-shared-tree groups (not just first pair). For 8 peers all on same trunk, marker should list all 8 PIDs, not just 2.
4. **Marker shape stability:** confirm OCA #3's parser handles `[ADV:WARN] Shared-tree conflict: PIDs <a,b> on tree <path>` correctly. Coordinate marker-shape decision with OCA #3 before either ships.
5. **Test fixture:** mockable peer-detection + topology check needed. Reuse `plugin/src/utils/peer-sessions.test.ts` patterns.

---

## Locked Design Direction (from parent §9 + post-recon rescope)

| Aspect | Decision (locked) |
|---|---|
| Marker for safe case | Existing `[ADV:PEER_SESSIONS]` info (already shipped) |
| Marker for shared-tree case | NEW `[ADV:WARN] Shared-tree conflict: PIDs <a,b> on tree <path>` warn |
| Topology basis | `git worktree list` per peer workdir |
| Cache | Required (TTL strategy = discovery item) |
| Owner | ADV plugin |
| Coordination with OCA #3 | Forward-compat parser already planned; either change can land first |

---

## Risks

| Risk | Mitigation |
|---|---|
| `git worktree list` fanout at 8+ peers | Cache with TTL; fail-loud test that fanout count ≤ 1 per cache window. |
| OCA status-bar parser doesn't recognize new warn marker | OCA #3 forward-compat parser. Either change can land first. |
| Operator misses the safe-case info emit because warn fires alongside | Both markers emit (info + warn). Status-bar surfaces both distinctly. |
| Non-git project edge case crashes detection | Discovery item — explicit fallback path tested. |
| False positive: 2 peers on same worktree path but different repos (rare) | Use `git rev-parse --git-common-dir` for strict equality, not path string compare. |

---

## Composition with other changes

- **Independent of OCA #1, #2:** doesn't touch OCA code.
- **Coordinates with OCA #3:** new warn marker shape needs forward-compat parser; OCA #3 already plans this.
- **Independent of ADV #4 (idle worker reaper):** different subsystem.
- **Independent of ADV #6 (sync prompt fix):** different subsystem.
- **No prerequisite on other changes** — can ship in isolation.

---

## Companion files to update on archive (in ADV repo)

- `plugin/src/utils/peer-sessions.ts` — add topology classification function
- `plugin/src/index.ts` — call topology check when ≥2 peers; emit warn alongside info if conflict
- `plugin/src/utils/peer-sessions.test.ts` — coverage for both scenarios
- `plugin/src/adv-instructions-assets.test.ts` — verify old assertions still hold
- `plugin/src/types.ts` — `coordination_topology_cache_ttl_seconds` config field
- ADV `ADV_INSTRUCTIONS.md § Multi-Session Coordination` — document new warn marker
- ADV `docs/peer-session-topology.md` — new design note
