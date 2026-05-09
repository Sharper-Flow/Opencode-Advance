# Temporal Dashboard Research

**Date:** 2026-04-23
**Status:** Research complete — not yet scoped as a phase
**Prerequisite:** Phase 6.5 (Temporal Enablement)

## Problem

ADV agents run autonomous workflows through Temporal, but visibility into change progress, gate status, task graphs, agent activity, and cost tracking requires running `/adv-status` or reading Temporal CLI output directly. A web dashboard would provide:

1. **Change progress at a glance** — which gate, how many tasks done, what's blocked
2. **Multi-change orchestration** — all active changes, relative progress, dependencies
3. **Agent activity / cost tracking** — retries, time invested, doom-loop state
4. **Operational health** — Temporal server, worker, namespace, workflow health

## Advance Temporal API Surface

Advance exposes a rich query/update API through two durable workflows:

### changeWorkflow (per-change state machine)

**Queries (read-only):**

| Wire Name | Returns |
|-----------|---------|
| `adv.change.bootstrap` | Change metadata (projectId, changeId, title, initializedAt) |
| `adv.change.state` | Full state (tasks, wisdom, gates, artifacts, closure) |
| `adv.change.tasks` | Tasks filtered by status and optional text filter |
| `adv.change.ready` | Tasks with no blocking dependencies |
| `adv.change.task` | Single task by ID |

**Updates (write, atomic):**

| Wire Name | Purpose |
|-----------|---------|
| `adv.change.addTask` | Create and add a task |
| `adv.change.updateTask` | Update task status, notes, error recovery |
| `adv.change.recordTaskEvidence` | Record TDD red/green evidence |
| `adv.change.setTaskPhase` | Set TDD phase (none/red/green/refactor/complete) |
| `adv.change.cancelTask` | Cancel a task with approval evidence |
| `adv.change.reclassifyTaskTdd` | Reclassify TDD intent after prep gate |
| `adv.change.completeGate` | Mark a gate complete |
| `adv.change.reopenFromGate` | Reset gates from specified point (re-entry) |
| `adv.change.addWisdom` | Add wisdom entry to change |
| `adv.change.updateArtifactMetadata` | Update artifact path and hash |
| `adv.change.closeChange` | Mark workflow closed (terminal) |

**Signals (fire-and-forget):**

| Wire Name | Purpose |
|-----------|---------|
| `adv.change.applyChangeSummary` | Propagate change summary to project workflow |

### projectWorkflow (per-project singleton)

**Queries:**

| Wire Name | Returns |
|-----------|---------|
| `adv.project.bootstrap` | Project metadata |
| `adv.project.state` | Full state (agenda, wisdom, migration ledger, change summaries) |
| `adv.project.agenda` | Agenda items filtered by status |
| `adv.project.wisdom` | Project-level wisdom filtered by type |
| `adv.project.migrationLedger` | State import audit trail |

**Updates:**

| Wire Name | Purpose |
|-----------|---------|
| `adv.project.addAgendaItem` | Create agenda item |
| `adv.project.updateAgendaItem` | Update agenda item status/priority/notes |
| `adv.project.addWisdom` | Add promoted wisdom entry |
| `adv.project.recordMigrationEntry` | Record state migration event |

### Search Attributes

Workflows register custom search attributes for external querying:

- `AdvChangeId` — change filter (Keyword)
- `AdvChangeStatus` — status filter (Keyword)
- `AdvChangeTitle` — change title (Keyword)
- `AdvAffectedProjects` — project filter (KeywordList)
- `AdvCurrentGate` — current gate (Keyword)
- `AdvCurrentBucket` — priority bucket (Keyword)
- `AdvLastSignalAt` — last signal timestamp (Datetime)
- `AdvCreatedAt` — creation timestamp (Datetime)
- `AdvWorktreeBranches` — worktree branches (KeywordList)
- `AdvWorktreePaths` — worktree paths (KeywordList)

Queryable via `client.workflow.list({ query: 'AdvChangeStatus = "active"' })`.

### Environment Variables

| Var | Default | Purpose |
|-----|---------|---------|
| `ADV_TEMPORAL_ADDRESS` | `127.0.0.1:7233` | Server address |
| `ADV_TEMPORAL_NAMESPACE` | `default` | Namespace on server |
| `ADV_TEMPORAL_ALLOW_REMOTE` | `false` | Allow non-loopback address |
| `ADV_NODE_PATH` | (PATH lookup) | Explicit Node binary path |
| `ADV_DISABLE_TEMPORAL` | (unset) | Force file-backed fallback |
| `ADV_TEMPORAL_CHANGE_HISTORY_THRESHOLD` | `2000` | Continue-as-new trigger |
| `ADV_TEMPORAL_PROJECT_HISTORY_THRESHOLD` | `10000` | Continue-as-new trigger |

### Worker Model

- In-process on Node.js hosts (single process, multiple task queues)
- Out-of-process Node.js child on Bun hosts (via `ADV_NODE_PATH`)
- Task queue naming: `advance-{projectId}`
- Workflow ID naming: `adv/change/{projectId}/{changeId}`, `adv/project/{projectId}`

## Temporal Web UI Extensibility

**No plugin system.** Temporal Web (`temporalio/ui`) is a monolithic Svelte SPA with no extension API for custom panels, domain views, or runtime component injection.

Configuration supports operational toggles (disable actions, auth) but not visual customization. See [Web UI Configuration Reference](https://docs.temporal.io/references/web-ui-configuration).

Community request for custom columns: [temporalio/ui#771](https://github.com/temporalio/ui/issues/771) — still open.

### Third-party tools

- **[Temporal Flow](https://github.com/itaisoudry/temporal-flow-web)** — Electron app with DAG visualization, live mode, fuzzy search. Active, MIT-licensed. Best existing visualization tool.
- **[Temporalio.Graphs](https://github.com/oleg-shilo/Temporalio.Graphs)** — C#/React workflow structure visualization.
- **[Temporal Code Exchange](https://temporal.io/code-exchange)** — Official community tools catalog.

## Browser → Temporal Feasibility

**Direct browser → Temporal Server: not possible.**

- Temporal Server exposes gRPC on port 7233 — no gRPC-Web support
- HTTP API on port 7243 (v1.22+) has no CORS headers — browser blocks requests
- No auth on raw frontend — designed for SDK/CLI clients

### Viable paths

| Path | How | Effort | Tradeoff |
|------|-----|--------|----------|
| **ui-server sidecar** | Temporal's Go proxy (bundled with `temporal server start-dev`) handles HTTP→gRPC + CORS + auth | Low | Extra service dependency; already bundled in dev server |
| **Custom Node proxy** | Thin Express app wrapping `@temporalio/client` | Medium | Own the code, but more to maintain |
| **OCA embedded proxy** | Go binary serves SPA + embeds HTTP→gRPC proxy | Medium-high | Single binary, cleanest UX, but Go gRPC proxy to write |

### Recommended architecture

For OCA (single-user, local-dev context):

```
OCA Go binary
├── Serves static SPA (dashboard UI)
├── Embedded HTTP→gRPC proxy (or reuse ui-server)
│   ├── CORS configured for localhost
│   └── Translates ADV query/update calls
└── Temporal Server (7233 gRPC)
```

The dev-server scenario is simplest: `temporal server start-dev` already bundles ui-server on port 8233 with CORS support. An OCA-hosted SPA could point to it directly for development.

## Architecture Options

| Option | Approach | Fits OCA? | Recommendation |
|--------|---------|-----------|---------------|
| **Temporal Web + static overlay** | Run Temporal Web as-is; build separate static SPA for ADV domain views alongside it | ✅ Minimal OCA changes | Good starting point |
| **OCA embedded dashboard** | OCA serves SPA + embeds thin Go HTTP→gRPC proxy | ✅ Single binary | Best long-term UX |
| **Temporal Web fork** | Fork `temporalio/ui`, add ADV panels | ❌ Maintenance burden | Not recommended |
| **Temporal Flow adaptation** | Use/adapt Temporal Flow for visualization | ⚠️ Electron stack | Research only |

## Control Surface

If the dashboard supports write actions (full control surface), it would need to expose:

- Approve/reject gate completions
- Retry failed tasks
- Cancel changes (with approval flow)
- Re-enter from a specific gate
- Add/update agenda items
- Force-checkpoint task progress

This maps directly to the Temporal update handlers listed above.

## Open Questions

1. **Phase 6.5 is a prerequisite** — OCA needs Temporal infra managed before dashboard is useful
2. **Auth model** — localhost-only (no auth) vs. Tailscale-exposed (need auth)
3. **SPA framework** — separate from OCA's Go codebase; choice TBD
4. **Real-time updates** — Temporal has no native SSE/websocket; dashboard needs polling or event subscriptions
5. **Action surface scope** — which `/adv-*` commands to replicate in the UI
6. **Mobile experience** — responsive design vs. dedicated mobile views vs. just "works on phone browser"
7. **Timeline** — post-v1.0? v1.1? v2.0? Depends on Phase 6.5 completion and user demand

## Sources

- [Temporal Web UI Configuration](https://docs.temporal.io/references/web-ui-configuration)
- [Temporal HTTP API (v1.22+)](https://github.com/temporalio/temporal/releases/tag/v1.22.0)
- [Temporal API Proto Definitions](https://github.com/temporalio/api)
- [Temporal UI Server](https://github.com/temporalio/ui-server)
- [Temporal Flow](https://github.com/itaisoudry/temporal-flow-web)
- [Temporal Code Exchange](https://temporal.io/code-exchange)
- [Community: Embed Temporal UI](https://community.temporal.io/t/embed-temporal-ui-into-other-web-host/6350)
- [Community: Custom Columns](https://github.com/temporalio/ui/issues/771)
