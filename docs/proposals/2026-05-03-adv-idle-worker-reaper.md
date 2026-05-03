# ADV Idle Temporal Worker Reaper

**Status:** Proposal — ready for `/adv-proposal` · **Date:** 2026-05-03 (updated 2026-05-03 post-reconnaissance)
**Target repo:** `Sharper-Flow/Advance` (oc-plugins/advance) — **NOT OpenCode Advance**
**Filing path:** When ready to start work, this proposal must be moved to (or re-drafted in) the ADV repo and `/adv-proposal` must run from inside `~/dev/oc-plugins/advance`. Drafted here in OCA repo for split coherence.
**Change ID (suggested):** `idleWorkerReaper`
**Parent decision lock:** [`2026-05-03-session-and-resource-architecture.md`](./2026-05-03-session-and-resource-architecture.md) §3.4 + §3.7 + §9 + §10
**Split position:** Change #4 of 7 — ADV-repo work, parallelizable with OCA changes
**Estimated effort:** 1–2 days
**Prerequisites:** ADV #6 (sync prompt fix). Independent of OCA #0/#1/#2/#3 and ADV #5.
**Sequence:** Land any time after worktree-shared-worker invariant is verified at 8-agent scale (parent §3.4 P7 verification task).

---

## Adaptation Note (read first)

This proposal carries the **decision-locked** direction from the parent doc. When `/adv-proposal` and `/adv-discover` run on this change, the discovery phase MUST:

1. Re-read parent §3.4 (worker singleton — already shipped via `rq-workerSingleton01`); this change extends, not replaces, the singleton.
2. Re-read parent §3.7 (idle worker reaper design + memory math).
3. Re-read parent §9.2 Q6 — 60 min default idle aligns with hibernation idle threshold.
4. Re-read parent §10.3 ("ADV idle worker reaper (#4) — discovery items") and treat each item as a discovery objective.
5. Verify worker-lock handover under 8-agent load (parent §10.3, P7 verification task) — read of `worker-lock.ts` (282 lines) suggests reclaim path is correct, but needs validation under real concurrent load.

Discovery findings may refine the design but MUST NOT reverse decision-locked answers without explicit operator re-approval.

---

## TL;DR

After 60 minutes of zero Temporal activity (no inbound tool calls, no inflight workflows), the worker-lock owner shuts down its Temporal worker, releases the lock, stays as a Temporal client only. On next ADV tool call, any opencode on the project re-acquires the lock (existing reclaim path) and respawns the worker. Per-project RAM win: ~350 MB. With 3 idle projects: ~1 GB. Composes with OCA hibernation (change #2) — fully-hibernated project drops to zero RAM cost until first user input.

---

## Problem Statement

Each ADV project runs one Temporal worker (per `rq-workerSingleton01`, parent §3.4) consuming ~350 MB regardless of agent count. At 5 active projects with mixed usage patterns, idle projects keep workers alive indefinitely. Memory math (parent §1.3): 5 idle workers × 350 MB = 1.75 GB held against the 42 GB WSL2 ceiling for zero work.

The worker-lock infrastructure already supports clean handover (atomic O_EXCL filesystem create, PID-liveness reclaim via `process.kill(pid, 0)`, ESRCH-triggered respawn). Adding an idle-shutdown path is a small extension, not a redesign.

---

## Success Criteria

- [ ] After `worker.idle_shutdown_minutes` minutes (default 60) of no inbound Temporal activity AND no `Running` workflow executions for the project, worker-lock owner shuts down its Temporal worker and releases the lock.
- [ ] On next ADV tool call from any opencode on the project, worker is respawned via existing lock-acquire + spawn path.
- [ ] First-call-after-idle latency documented (target ~3–5 sec; measure and validate).
- [ ] Configurable via `worker.idle_shutdown_minutes` in plugin config; default 60, 0 = disabled.
- [ ] Hysteresis: worker shuts down at most once per hour to avoid flapping.
- [ ] Inflight guard: do NOT shut down if `ListWorkflowExecutions(Running)` returns non-empty for project's task queues, OR any activity is mid-execution.
- [ ] Telemetry: log shutdown + respawn events at INFO level for operator visibility.
- [ ] Composes with OCA hibernation: when all opencodes on a project are hibernated, no client traffic → reaper shuts down worker → project at zero RAM cost.

---

## Out of Scope

- Vision daemon idle reaping (separate concern; Vision is host-wide singleton).
- Temporal dev-server idle reaping (parent §1.2: shared host-wide; not per-project).
- Worker pool sharing across projects (parent §3.4 already-shipped singleton is per-project; cross-project sharing is out of scope and would break project isolation).
- OCA-side detection of "all opencodes hibernated" (OCA change #2 owns hibernation; reaper just reacts to absence of Temporal traffic).
- Pre-warm hooks for predictable resume latency (deferred follow-up).

---

## Acceptance Criteria (operator-verifiable)

1. Open opencode in a project, run any ADV tool call to spawn worker, leave idle 60+ minutes (or temporarily lower threshold), confirm worker process disappears (`ps` check).
2. From the same opencode (still alive), run an ADV tool call; worker respawns within ~5 sec; tool call succeeds.
3. From a different opencode on same project, run an ADV tool call; worker respawns; tool call succeeds.
4. Inflight workflow scenario: start a long-running workflow, leave opencode idle past threshold; worker does NOT shut down while workflow is `Running`.
5. Hysteresis: rapidly toggle activity around the threshold; worker shuts down at most once per hour.
6. Composition with OCA change #2: hibernate all 8 opencodes on a project; observe worker shutdown after threshold; resume one opencode; first ADV tool call respawns worker.

---

## Constraints

- MUST NOT break `rq-workerSingleton01` invariant (one worker per project, lock-arbitrated).
- MUST NOT shut down worker mid-workflow (inflight guard required).
- MUST NOT race with concurrent lock-acquire from peer opencodes (use existing lock semantics).
- MUST emit clear log lines on shutdown + respawn for operator debugging.
- Default threshold (60 min) MUST align with OCA hibernation default (parent §9.2 Q6) for predictable composed behavior.
- Project-id basis: existing `git rev-list --max-parents=0 HEAD` (parent §3.4 + §9.1 Q11 verification).

---

## Discovery Agenda (per parent §10.3)

The discovery phase MUST address:

1. **Inflight guard exhaustiveness:** `ListWorkflowExecutions(Running)` covers active workflows; verify it also catches activities mid-flight, signal handlers, in-flight queries. Missing dimension = worker shutdown mid-work. Discovery item: enumerate all in-flight states and confirm guard covers each.
2. **Lock-handover under 8-agent load (parent P7):** confirm rapid lock-handover when worker-lock owner exits while peer opencodes immediately reclaim. Empirical test: simulate 8-opencode load, force shutdown, measure reclaim latency + correctness. Reconnaissance read of `plugin/src/temporal/worker-lock.ts` (312 lines, 2026-05-03) confirms the existing primitive is suitable for extension:
   - `acquireWorkerLock` returns `WorkerLockResult` (`{owned, ownerPid, workerId, lockPath}`); ESRCH triggers reclaim, EPERM treats lock as held.
   - `releaseWorkerLock` is best-effort: rename → remove, fallback direct remove.
   - Atomic create via tmp+rename + O_EXCL canonical write.
   - **Reaper extension surface:** add `releaseWorkerLockGraceful(projectStateDir, opts)` that (a) calls `ListWorkflowExecutions(Running)` first — refuse if non-empty — (b) calls existing `releaseWorkerLock`. Reclaim path post-graceful-release works via existing `acquireWorkerLock` (canonical doesn't exist → atomic create succeeds for next caller).
3. **First-call-after-idle latency measurement:** measure cold-start of Temporal SDK + Node startup + connection + workflow registration. Document P50/P95.
4. **Vision daemon worst-case fanout (parent §3.9):** 8 agents × 23 MCP servers = up to 184 parallel calls. Run `vision_metrics` baseline under simulated load; identify any single-server bottleneck. (Out of scope for fixing here, but in scope for measurement to confirm reaper is not masking a different problem.)
5. **Telemetry surface:** decide log level + structured-log fields for shutdown/respawn events. Operator needs visibility without grep noise.

---

## Locked Design Direction (from parent §9)

| Aspect | Decision (locked) |
|---|---|
| Mechanism | Worker shuts down after idle threshold; client mode persists; respawn on next call via existing lock semantics |
| Idle threshold | 60 min default; configurable via `worker.idle_shutdown_minutes` |
| Inflight guard | Required — no shutdown while workflow Running or activity mid-exec |
| Hysteresis | Once per hour max |
| Composition with hibernation | When all opencodes hibernated → no client traffic → reaper triggers → zero RAM project |

---

## Risks

| Risk | Mitigation |
|---|---|
| Worker shutdown mid-workflow | Inflight guard with full coverage of `Running` + activities + queries; discovery item to enumerate. |
| Lock-handover race at 8 agents | Empirical validation in discovery; existing `worker-lock.ts` reclaim path appears correct but needs load test. |
| First-call latency degrades operator UX | Measurement-first; if >5s, consider warm-pool or reduce shutdown aggressiveness. |
| Reaper masks a Vision/MCP saturation problem | Discovery item runs `vision_metrics` under load to rule out. |

---

## Composition with other changes

- **Composes with OCA change #2 (hibernation):** when all opencodes on a project hibernate, no client traffic → reaper triggers → worker dies → project at zero RAM cost until next user input. Combined memory win: per parent §3.5.2 + §3.7 = ~16 GB (hibernation) + ~1 GB (reaper at 3 idle projects) = ~17 GB headroom.
- **Independent of OCA change #1 (Pattern B):** reaper logic is purely ADV-internal; does not require Pattern B.
- **Independent of OCA change #3 (resurrect):** unrelated.

---

## Companion files to update on archive (in ADV repo)

- `plugin/src/temporal/worker-multi.ts` — idle detection + shutdown logic
- `plugin/src/temporal/worker-lock.ts` — possibly minor extension for graceful release path
- `plugin/src/types.ts` — `worker.idle_shutdown_minutes` config field
- `docs/idle-worker-shutdown.md` — new design note (parent §8 mentions)
- ADV `ADV_INSTRUCTIONS.md` — minor mention if operator-facing behavior change warrants
