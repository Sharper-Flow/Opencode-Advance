# First Boot — Initialize ADV in This Repo

This guide walks through the exact steps to initialize ADV state in the `opencodeadvance` repo and create the first ADV change from the scaffolded v1 implementation proposal.

## Prerequisites

Before you start, make sure:

1. ✅ You have OpenCode installed and working
2. ✅ The Advance plugin is installed and working in your global OpenCode config
3. ✅ You can successfully run `/adv-status` in other ADV-enabled repos
4. ✅ You are in a terminal, not inside an existing OpenCode session

## Step 1: Open OpenCode in this repository

From your terminal:

```bash
cd ~/dev/opencodeadvance
opencode
```

This launches OpenCode with the repo as the working directory. The Advance plugin (loaded globally) will detect `project.json` in the repo root and automatically initialize ADV state for this project.

## Step 2: Verify ADV is initialized

Run the status check as your first command:

```
/adv-status
```

Expected output:

```
============================================================
                    ADV PROJECT STATUS
============================================================

SPECS (The Laws)
----------------
Total: 0 capabilities

No specs defined yet. This is expected for a new project.

ACTIVE CHANGES
--------------
No active changes.

Suggestions:
- Create a new change: /adv-proposal "summary"
- Or see docs/proposals/v1-implementation.md for the planned v1.0 work

============================================================
```

If you see any errors about ADV state or project.json, stop and troubleshoot before continuing.

## Step 3: Create the first ADV change

The first change captures the entire v1.0 implementation work. The scaffolded proposal document at `docs/proposals/v1-implementation.md` contains all the content you need.

Run:

```
/adv-proposal OpenCode Advance v1.0 — clean rewrite of open-chad as a declarative Go-based configuration platform
```

The `/adv-proposal` command will:

1. Create a new change with the ID `opencodeAdvanceV1` (derived from the summary)
2. Walk you through a 2-phase flow: Problem Statement Agreement → Full Proposal
3. Ask you to confirm the problem statement
4. Ask for the full proposal body

When prompted for the proposal body, point the agent at the scaffolded file:

> Use the content from `docs/proposals/v1-implementation.md` as the full proposal. The problem statement, success criteria, constraints, approach, phases, risks, and deliverables are all already written there.

The agent will read that file, extract the relevant sections, and populate the ADV change accordingly.

## Step 4: Verify the change was created

```
/adv-status
```

Expected:

```
============================================================
                    ADV PROJECT STATUS
============================================================

SPECS (The Laws)
----------------
Total: 0 capabilities

ACTIVE CHANGES
--------------
- opencodeAdvanceV1 (proposal)
  OpenCode Advance v1.0 — clean rewrite of open-chad...

Suggestions:
- Discover context for this change: /adv-discover opencodeAdvanceV1
- Or see docs/proposals/phases.md for the phased work plan

============================================================
```

## Step 5: Create a dedicated Phase 0 change

The first real work is **Phase 0: Foundation + Brand**. See [`phases.md`](phases.md) for details.

Recommended structure:

- `opencodeAdvanceV1` = umbrella / tracking change only
- `phase0FoundationBrand` = first implementation change

Create a dedicated Phase 0 change:

```
/adv-proposal Phase 0: Foundation + brand — go.mod, wordmark render, palette, boot splash
```

Then walk the 7-gate workflow:

```
/adv-discover phase0FoundationBrand
/adv-design phase0FoundationBrand
/adv-prep phase0FoundationBrand
/adv-apply phase0FoundationBrand
```

Each phase follows the same pattern. The `opencodeAdvanceV1` change is the umbrella / tracking change; individual phase changes are where the work actually lives.

## Why this structure is recommended

- It keeps the long-range v1 narrative in one place
- It keeps implementation work archiveable in clean units
- It reduces task sprawl inside a single giant change
- It makes it easier to stop and resume between phases

**Recommendation:** keep Phase 0 separate too. Use the umbrella change for proposal context and progress tracking, not direct implementation.

## Working rules during implementation

Once the first change is active and you're implementing Phase 0:

### File safety

- **NEVER** write to `~/.config/opencode/`, `~/.config/vision/`, `~/.tmux.conf`, or any user-level config file
- Always use isolated test directories (`.dev/opencode/`, `.dev/vision/`, etc.) via environment overrides
- Run `oca --help` and `oca version` in the repo's own Go module, not a system-wide binary

### Commit discipline

- Atomic commits following conventional commit style (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`)
- Every commit should build (`go build ./...`) and pass (`go test ./...`)
- Commit TDD red and green phases separately where it makes sense

### ADV workflow discipline

- TDD is inline within each task (red → green → refactor → done)
- Use `adv_task_ready` (MCP tool) to get the next unblocked task
- Update task status as you work; never leave tasks in `in_progress` across sessions without notes
- Record wisdom entries (`/adv-wisdom` or `adv_wisdom_add`) for patterns, gotchas, and conventions discovered

### When to create new changes

- Each phase: new change
- Each significant refactor discovered during a phase: new change, with `discovered_from` link to the parent
- Trivial fixes within an active phase: add as a task to the current change
- Doc-only updates: commit directly (no ADV change needed)

## Troubleshooting

### ADV doesn't detect the project

If `/adv-status` reports the wrong project or no project at all:

```bash
# Verify project.json exists and is valid
cat project.json

# Verify you launched OpenCode from the repo root
pwd  # should print: /home/jrede/dev/opencodeadvance
```

### The plugin isn't loaded

If slash commands like `/adv-status` aren't recognized:

```bash
# Verify the plugin is in opencode.json
grep -A 3 plugin ~/.config/opencode/opencode.json

# Verify the plugin is built
ls ~/dev/oc-plugins/advance/plugin/dist/index.js
```

### The proposal is missing content

If `/adv-proposal` creates an empty or incomplete change:

- Re-read `docs/proposals/v1-implementation.md` and paste relevant sections directly into the proposal body when prompted
- Alternatively, use `adv_change_update` to edit the change's proposal.md directly

## After Phase 0

Once Phase 0 is archived and you have a working `go run ./cmd/oca version` that prints the wordmark, you're ready to start Phase 1: stack.toml + MCP apply.

From that point on, the project is self-sustaining — each phase is an ADV change, ADV drives the workflow, and the repo grows organically through the 7-gate lifecycle (proposal → discovery → design → planning → execution → acceptance → release) toward v1.0.
