---
name: worktree
description: "Generic git worktree boundary guidance. For ADV-managed worktrees, defer to Advance's adv-worktree skill and adv_worktree_* tools."
keywords: ["worktree", "git-worktree", "branch-isolation", "feature-branch", "parallel-experiment", "merge-before-delete", "adv-worktree"]
license: MIT
metadata:
  priority: medium
  replaces: adv-worktree for ADV-managed flows
---

## When to Load This Skill

Load this skill for **generic git worktree boundary guidance** in OCA contexts.

For any ADV-managed change or ADV worktree lifecycle action, load
`skill("adv-worktree")` and follow Advance-owned guidance instead. Advance owns
ADV worktree registry, lifecycle gates, deletion safety, and multi-session
coordination.

## Ownership Boundary

| Context | Owner | Use |
|---|---|---|
| ADV-managed change worktree | Advance | `skill("adv-worktree")` |
| ADV worktree create/delete/triage | Advance | `adv_worktree_create`, `adv_worktree_delete`, `adv_worktree_cleanup`, `adv_worktree_triage` |
| Generic non-ADV git isolation | OCA guidance only | Manual git worktree flow, user-approved |

Backward-compatible aliases such as `worktree_create` may exist, but
`adv_worktree_*` names are canonical for ADV workflows.

## Generic Worktree Principles

- Use isolation for risky refactors, parallel experiments, feature branches, or
  exploratory spikes.
- Skip extra worktrees for small, low-risk, contained edits when current working
  tree isolation is already sufficient.
- After creating a worktree, run all file and command tools in the returned
  worktree path.
- Never delete a worktree until its branch is merged to the default branch and
  the worktree is clean.

## Merge-Before-Delete Invariant

Do not switch branches in the main checkout during cleanup. Resolve the main
checkout path, verify it is already on the default branch and clean, then merge
in place:

```bash
MAIN="$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")"
git -C "$MAIN" branch --show-current
git -C "$MAIN" status --porcelain
git -C "$MAIN" merge --ff-only change/{change-id}
git -C "$MAIN" log --oneline trunk..change/{change-id}
```

If the branch is not merged, do not delete the worktree.

## ADV Worktree Quick Reference

For ADV-managed worktrees, use Advance-owned tools:

```text
adv_worktree_create
adv_worktree_delete
adv_worktree_cleanup
adv_worktree_triage
```

Then continue with `workdir` set to the returned path. If details matter, load
`skill("adv-worktree")`; this OCA skill intentionally avoids duplicating that
workflow.

## Keywords

worktree, git worktree, branch isolation, parallel development, merge before
delete, adv-worktree, adv_worktree_create, adv_worktree_delete, worktree cleanup
