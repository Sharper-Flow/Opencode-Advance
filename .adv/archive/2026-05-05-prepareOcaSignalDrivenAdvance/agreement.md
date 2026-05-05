## Discovery Findings

### Upstream Advance direction
- Planned source of truth: Temporal workflow event history; disk becomes eventually-consistent projection only for external readers.
- Planned queues: global workflow queue `advance-changes` plus per-host activity queues.
- Planned search attributes: `AdvChangeId`, `AdvChangeStatus`, `AdvChangeTitle`, `AdvAffectedProjects`, `AdvAffectedPaths`, `AdvCurrentGate`, `AdvCurrentBucket`, `AdvLastSignalAt`, `AdvCreatedAt`.
- Planned durable archive output: `.adv/specs/`, `.adv/wisdom.jsonl`, and brief `.adv/archive/{change-id}.md`.
- Planned deleted/folded tools include cleanup/repair/import/sweep/purge/mesh/task-run-status-era tooling.

### Validator Finding
Independent validator returned `CONFLICT`: the signal-driven design is not landed on current Advance trunk. Current Advance still exposes old tools, old search attributes, project-keyed queues, and `snapshot.json` projection. Runtime hard-cut now would break current OCA behavior.

## Chosen Route
`docs + upstream checklist`.

## Objectives
1. Clarify active OCA docs so current Advance behavior is described as current-only and not confused with the pending signal-driven cutover.
2. Add an upstream readiness checklist that defines exactly when the larger OCA runtime refactor may proceed.
3. Preserve current runtime compatibility; do not change OCA health/status/dashboard code yet.
4. Keep OCA non-authoritative and non-mutating.

## Acceptance Criteria
1. Active docs distinguish current Advance repair/cleanup/projection behavior from planned signal-era behavior.
2. A readiness checklist exists with concrete gates: Advance commit/tag, new search attributes registered, projection schema/path defined, deleted tools absent or deprecated, migration story known, and verification target available.
3. Runtime code is unchanged except if required for docs verification plumbing; no behavior changes to current OCA/Advance integration.
4. Existing `oca-workspace-projection` spec remains unchanged until upstream lands.
5. Verification scans show no active doc claims signal-driven architecture is landed.
6. `go test ./...`, `go vet ./...`, and `gofmt -d .` pass or are reported with exact blockers.

## Constraints
- OCA must not implement Advance internals.
- OCA must not delete user state.
- OCA must keep dev/test path isolation.
- No runtime hard-cut until upstream signal architecture lands and is verifiable.