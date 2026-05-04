# Worktree Usage Guide

You have access to canonical ADV worktree tools for isolated git worktree sessions:

- `adv_worktree_create`
- `adv_worktree_delete`
- `adv_worktree_cleanup`
- `adv_worktree_triage`

Backward-compatible aliases (`worktree_create`, `worktree_delete`, `worktree_cleanup`) may exist, but prefer `adv_worktree_*` names.

## When to Create a Worktree

Use `adv_worktree_create` when:
- **Risky refactors** — large structural changes that might break the codebase
- **Parallel experiments** — trying two different approaches to the same problem
- **Feature branches** — the user asks you to start a new feature in isolation
- **Exploratory work** — spiking on an idea without polluting the main branch

## When NOT to Create a Worktree

- Small, contained changes (bug fixes, config tweaks, single-file edits)
- When the user is already in a worktree session
- When the change is low-risk and easily reversible

## Behavior

- Default flow is inline: create worktree, then continue in the same agent session
- **CRITICAL: After creation, you MUST immediately switch ALL tool calls to use the returned worktree path as `workdir`.** Do not run any more commands against the original directory. This includes bash, read, edit, glob, grep — everything.
- On delete, all changes are auto-committed before cleanup. Pass the `branch` arg to `adv_worktree_delete`.
- You can have multiple worktrees running simultaneously

## Post-Change Cleanup (Merge Before Delete)

**Never delete a worktree until its branch is merged to the default branch (e.g. `main` or `trunk`).**

> **Invariant: main checkout stays on the default branch.** Cleanup never runs `git checkout` or `git switch` on either the main checkout or any worktree. The default branch is updated **in place** via `git -C "$MAIN" merge --ff-only`. The main checkout's HEAD is never touched indirectly. If main is on the wrong branch or has uncommitted changes, cleanup STOPS and asks you to restore main yourself — the agent does not mutate main on your behalf.

After implementation is complete and the change is archived/signed off:

### Step 1: Verify the branch is clean and resolve `$MAIN`

```bash
# In the worktree directory — no uncommitted changes
git status
# Should show "nothing to commit, working tree clean"

# Resolve the absolute path of the main checkout. Works from any worktree.
MAIN="$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")"
echo "$MAIN"   # sanity check — should be the main repo root
```

### Step 2: Verify main checkout invariant (HARD GATE)

```bash
# Detect default branch (main or trunk)
DEFAULT_BRANCH="$(git -C "$MAIN" symbolic-ref --short HEAD 2>/dev/null || true)"

# Main MUST already be on the default branch
git -C "$MAIN" branch --show-current
# Expected: main (or trunk). If anything else → STOP. Restore main to the
# default branch yourself (commit/stash any work in $MAIN, then
# `git -C "$MAIN" switch <default-branch>`) and rerun.

# Main MUST be clean
git -C "$MAIN" status --porcelain
# Expected: empty. If not → STOP. Commit or stash in $MAIN and rerun.
```

If either check fails: **stop and fix manually.** The cleanup flow assumes main is on the default branch and clean — it never switches branches or stashes for you.

### Step 3: Merge to the default branch (in place, no checkout)

```bash
# Fast-forward the default branch in $MAIN. No `git checkout` anywhere.
git -C "$MAIN" merge --ff-only change/{change-id}
```

Alternatively, if the project uses pull requests, push the branch from the worktree and open a PR:

```bash
git push -u origin change/{change-id}
gh pr create --title "Archive {change-id}" --body "Merges completed change."
```

Wait for the PR to be merged before proceeding to deletion.

### Step 4: Verify the merge

```bash
# Confirm every change-branch commit is reachable from the default branch
git -C "$MAIN" log --oneline {default-branch}..change/{change-id}
# Should return EMPTY (no commits ahead) — meaning everything is merged
```

### Step 5: Delete the worktree

Only after merge is confirmed:

```bash
adv_worktree_delete branch: "change/{change-id}" reason: "Change {change-id} merged to default branch"
```

If `adv_worktree_delete` is unavailable, fall back to the manual git equivalent (still no checkout):

```bash
git -C "$MAIN" worktree remove <worktree-path>
git -C "$MAIN" branch -D change/{change-id}
```

### Checklist

- [ ] All changes committed in the worktree branch
- [ ] Branch merged to default branch (direct merge or PR)
- [ ] Merge verified — no commits ahead of default branch
- [ ] `adv_worktree_delete` called with reason

**If the merge is not yet complete, do NOT delete the worktree.** The worktree protects unmerged work from being lost.

## Inline Mode (Default)

Worktrees default to **inline mode**: no new terminal or tmux window is opened.
After `adv_worktree_create` succeeds, use the returned path as `workdir` for all
subsequent tool calls (bash, read, edit, glob, grep, etc.).

When deleting an inline worktree, pass the `branch` argument to `adv_worktree_delete`
so the plugin knows which worktree to remove.

If a project sets `"inline": false` in `.opencode/worktree.jsonc`, the old
behavior is restored (new tmux window / terminal tab with a separate OpenCode
instance).

## Always Ask First

Before creating a worktree, briefly explain WHY you think isolation is needed and confirm with the user.
