/**
 * Derive the ADV change ID from a git branch name.
 * Returns the change ID if branch is "change/{id}", empty string otherwise.
 */
export function deriveChangeID(branch: unknown): string {
  if (typeof branch !== "string" || !branch.startsWith("change/")) return "";
  const id = branch.slice("change/".length);
  return id || "";
}

export function deriveBranchSafety(input: {
  isMainCheckout: boolean;
  isWorktree: boolean;
  branch?: string;
  defaultBranch?: string;
}): "worktree" | "safe_main_checkout" | "unsafe_main_branch" | "unknown" {
  if (input.isWorktree) return "worktree";
  if (!input.isMainCheckout) return "unknown";
  if (
    input.defaultBranch &&
    input.branch &&
    input.branch !== input.defaultBranch
  ) {
    return "unsafe_main_branch";
  }
  return "safe_main_checkout";
}
