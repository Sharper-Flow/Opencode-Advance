# Future Better Session Management

## Goal

Make ADV/OCA session management branch-aware while keeping git worktrees as the safe execution boundary for agents.

## Current Constraint

Agents need a real filesystem path for edits, tests, diffs, and tooling. A git branch ref alone is not enough. Worktrees stay required for mutating work.

Current model optimizes for safety:

- one change branch per worktree: `change/<change-id>`
- all agent tools run with `workdir` set to worktree path
- git owns real branch/worktree facts
- ADV Temporal state owns change workflow, tasks, gates, session/worktree registry
- OCA surfaces session UX through tmux/status/pane state

Pain: user/agent thinks in changes and branches, but system often forces thinking in checked-out directories.

## Competitive Validation

Every serious local AI coding tool converges on git worktrees as the isolation primitive. Research (2025-05) across eight tools:

| Tool | Strategy | Maturity signals | Known problems |
|---|---|---|---|
| Claude Code | `--worktree` flag; auto-worktree for desktop sessions; subagent `isolation: worktree` | `.worktreeinclude` for gitignored files; orphan auto-sweep | Subagent `git switch` escapes isolation (#55708); dirty worktree data loss on cleanup (#46470) |
| Cursor | Agents Window auto-creates worktrees; `/worktree <task>` | `.cursor/worktrees.json` setup hooks; `worktreeMaxCount` + cleanup interval | Cannot migrate running chat to worktree |
| Windsurf | Per-conversation worktree; toggle at session start | `post_setup_worktree` hook; 20 worktree cap per workspace | Same limitation on migration |
| Copilot CLI | Worktree isolation mode for background agents | Auto-commit per turn in worktree | **1,526 worktrees/day bug** (#296194); data loss on `git worktree remove` (#289973) |
| Devin | VM isolation (not git-native) | Manager → worker delegation | Cloud-only; not local git |
| Codex CLI | Manual worktrees only | No native management | Recognized gap |
| Aider | No worktree support | Single-session pair programmer | Not designed for parallelism |

### Cross-cutting lessons

1. **Worktrees are universal** for local agent isolation. No tool uses copy-on-write or containerized git.
2. **Cleanup is the #1 operational gap.** Every tool that supports worktrees has filed bugs about proliferation, stale worktrees, or data loss during auto-cleanup. Claude Code's "empty → auto-remove, dirty → prompt" is the most mature pattern. Copilot CLI is the cautionary tale.
3. **Branch-switching inside worktrees escapes isolation.** Claude Code #55708: `git switch` inside a worktree-isolated subagent can mutate the parent session's HEAD. This is a fundamental risk that must be architecturally prevented, not just instructed against.
4. **The "fresh checkout" problem is universal.** New worktrees lack `.env`, `node_modules`, DB state, etc. Every mature tool solves this with setup hooks (copy files, run install, migrate DB). ADV has `postCreate` hooks in `worktree.jsonc` already — this is ahead of Codex and Aider.
5. **Merge conflicts surface at merge time, not editing time.** Worktrees prevent file-level collisions by design. No tool reports branch conflicts between concurrent agents during editing. The real operational problems are cleanup and stale state, not concurrent edits.

## Current Safety Gap

ADV enforces worktree isolation for change-managed workflows, but nothing prevents:

1. **Branch-switching in the main checkout.** A non-ADV agent (or ADV agent outside a change workflow) can `git checkout feature-X` in the shared main checkout, conflicting with another session using that branch.
2. **New session landing on a non-default branch.** OpenCode starts in whatever directory state the main checkout is in. If another agent left it on a feature branch, the new session inherits that state silently.
3. **No session-startup branch detection.** Plugin init emits `[ADV:PEER_SESSIONS]` for process awareness but does not check whether the working directory is on the default branch.

These are the exact scenario that triggered this investigation: a new session opened on another agent's feature branch with no warning.

### Proposed Safety Rules (add to existing)

- **Hard rule: agents MUST NOT `git checkout` or `git switch` in the main checkout.** If a different branch is needed, create a worktree. This is what Claude Code, Cursor, and Windsurf enforce architecturally.
- **Session startup detection:** at plugin init, if working directory is not on the default branch, emit `[ADV:ATTN] Working directory is on branch X, not the default branch (Y). Another agent may be using this branch. Create a worktree or restore default branch.`
- **Main checkout is shared infrastructure.** Treat it like a shared database connection — no session owns it exclusively. Default branch is the only safe shared state.

## Proposed Model: Branch Registry + Materialized Worktrees

Split “branch state exists” from “branch is checked out as worktree”.

### Core Concepts

| Concept | Authority | Purpose |
|---|---|---|
| Branch ref | Git | Source of truth for commits and merge status |
| Worktree path | Git | Materialized filesystem for execution |
| Branch registry | ADV Temporal | Coordination metadata and lifecycle |
| Session projection | OCA | Fast UI/status display, rebuildable cache |

### Branch States

```text
unmaterialized   branch exists, no active worktree
materializing    worktree create in progress
active           worktree exists and may have sessions
idle             worktree exists, no live session
pending_delete   cleanup queued, blocked by process or merge safety
merged           branch fully integrated into default branch
deleted          branch/worktree removed
stale            git metadata points to missing/invalid path
```

### Registry Record

```ts
interface BranchRecord {
  branch: string
  changeId?: string
  status: BranchState
  baseRef?: string
  headSha?: string
  aheadBehind?: { ahead: number; behind: number }
  materializedPath?: string
  lastSeenAt: string
  lastSessionId?: string
  source: "git_scan" | "adv_worktree_create" | "user_import"
}
```

Do not store file contents or large diffs here. Recompute git facts from git.

## Optimized Flow

### 1. Start or Resume Change

1. User says: “resume change X”.
2. ADV looks up change and branch registry.
3. If worktree exists: reuse path.
4. If branch exists but no worktree: materialize via `adv_worktree_create`.
5. If neither exists: create branch + worktree from default branch.
6. Agent switches all tools to returned `workdir`.

### 2. Park Work Without Losing State

When session ends or user switches away:

- keep branch registry record
- keep worktree if useful and clean/active
- mark session inactive
- OCA status shows “parked” branch/change
- no need to keep tmux pane open

### 3. Lazy Materialization

Branch can exist without checkout until agent needs files.

Useful for:

- reviewing queue of changes
- planning future work
- showing status in OCA UI
- archive/cleanup triage
- avoiding worktree sprawl

### 4. Git-First Reconciliation

Periodic or on-demand scan:

```text
git branch --format=...
git worktree list --porcelain
git merge-base / rev-list --left-right --count
```

Then reconcile registry:

- missing worktree path → `stale`
- branch merged into default → `merged`
- worktree path exists → `active` or `idle`
- registry-only record missing git branch → `deleted` or `stale`

Git wins on facts. Temporal wins on coordination metadata.

## Temporal Role

Temporal should not virtualize git. Use it for durable coordination:

- branch registry updates
- session registry updates
- materialization lifecycle
- pending delete queue
- human gates and task state
- crash recovery after interrupted create/delete

Avoid Temporal for:

- high-frequency UI polling
- file-level state
- replacing `git branch` / `git worktree list`
- simple display cache

## OCA Role

OCA should own user-facing session UX:

- `oca session list` shows active sessions plus parked ADV branches
- tmux status shows current branch/change/worktree state
- `oca session resume <change-id>` asks ADV to materialize/reuse worktree
- `oca session park` detaches pane/session but preserves registry state
- cache/projection stored under OCA state, rebuildable from ADV + git

OCA cache must not be authority. If cache is wrong, rebuild.

## ADV Role

ADV should own workflow semantics:

- enforce worktree before mutating tasks
- map `changeId -> branch -> worktreePath`
- update BranchRecord on create/reuse/delete/archive
- block delete until merge verification passes
- surface stale/diverged branch states during resume/archive

## Safety Rules

- Mutating agent work always uses real worktree path.
- Never edit default branch for ADV tasks.
- Never delete worktree until branch is merged or explicitly force-approved.
- Git facts override registry facts.
- Registry/cache is repairable; git history is canonical.
- Temporal failure must not hide existing git worktrees; reuse git worktree first when possible.

## Benefits

- Agents can reason in changes/branches instead of paths.
- Worktrees remain safe execution sandboxes.
- Session UI can show parked/future work without checking everything out.
- Existing worktrees get reused before expensive recovery paths.
- Cleanup becomes safer: merged, stale, idle, active states are explicit.
- Temporal remains valuable without becoming fake git.

## Implementation Sketch

### Phase 1: Read-Only Branch Inventory

- Add git scanner that returns branches + worktrees + merge state.
- Add OCA display command/status section.
- No mutation yet.

### Phase 2: ADV Branch Registry

- Extend project workflow with `branch_registry` or evolve `worktree_registry` into branch-aware records.
- Add reconciliation update fed by git scanner.
- Keep existing `worktree_registry` compatibility.

### Phase 3: Resume by Change/Branch

- Add command/tool path: `resume(changeId)` → reuse/materialize worktree.
- OCA session command calls that path.
- Agent always receives concrete `workdir`.

### Phase 1.5: Session Startup Safety (new, immediate)

- Add default-branch detection at plugin init.
- If main checkout is not on default branch, emit `[ADV:ATTN]` with branch name and remediation options.
- Block or warn agent from proceeding on non-default branch unless it owns the matching change worktree.
- This addresses the "new session lands on another agent's branch" scenario directly.

### Phase 4: Park / Cleanup UX

- Add "parked" state.
- Add safe cleanup for idle merged worktrees.
- Add stale registry repair flow.

**Cleanup lessons from competitors:**

| Lesson | Source | Pattern for ADV |
|---|---|---|
| Empty worktrees → auto-remove, dirty → prompt before removal | Claude Code | Extend `adv_worktree_triage` with dirty-state awareness; auto-remove only clean idle worktrees |
| Hard cap on worktree count per project | Cursor (20), Windsurf (20) | Configurable `maxWorktreesPerProject` with oldest-first eviction for merged/idle worktrees |
| Worktree proliferation bug: 1,526/day | Copilot CLI #296194 | Triage runs at session start + periodic; alert when count exceeds threshold |
| `git worktree remove` deletes working directory including uncommitted changes | Copilot CLI #289973 | ADV's 3-condition deletion gate already prevents this; keep it strict |
| Cleanup interval vs. on-demand | Cursor (6h interval), Claude Code (startup sweep) | Both: startup sweep + configurable interval; startup sweep catches crashes |

### Phase 5: Setup Hook Reliability (new, LBP direction)

- Validate `postCreate` hooks from `.opencode/worktree.jsonc` succeed before marking worktree `active`.
- If hooks fail, surface error and prevent agent from using the worktree until resolved.
- Add `.worktreeinclude`-style file copy (Claude Code pattern) as a simpler alternative to full hook scripts for common cases (`.env`, `.env.local`, config files).
- Document the "fresh checkout" problem and recommended hook patterns.

## Non-Goals

- No virtual filesystem.
- No editing branch refs without worktree.
- No Temporal replacement for git.
- No global multi-user branch lock system.
- No forced cleanup of unmerged branches.

## Open Questions

1. **Auto-worktree per session (Cursor/Windsurf pattern):** Every new session gets its own worktree automatically. Strongest isolation but requires setup hooks to be reliable. Worth pursuing after Phase 5 validates hook reliability.
2. **Agent instruction vs. architectural enforcement for branch-switching:** Claude Code's #55708 shows instruction-level rules can be violated (agent ran `git switch` inside a worktree). Should ADV architecturally block `git checkout`/`git switch` in the main checkout (bash guard extension), or is instruction-level sufficient?
3. **Main checkout as read-only for agents:** If we go to auto-worktree-per-session, the main checkout becomes effectively read-only for all agents. This is a UX shift — users currently expect to work in main for quick tasks.

## Recommendation

Build branch-aware session management as a hybrid:

```text
Git = truth
Temporal = durable coordination
OCA = session UX and cache
Worktrees = execution boundary
Main checkout = shared default-branch infrastructure (never branch-switched by agents)
```

This gives better session ergonomics without weakening safety. The immediate priority is Phase 1.5 (session startup safety) — it's cheap, addresses a real operational pain, and requires no architectural changes.
