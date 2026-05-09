/**
 * Derive the ADV change ID from a git branch name.
 * Returns the change ID if branch is "change/{id}", empty string otherwise.
 */
export function deriveChangeID(branch: unknown): string {
  if (typeof branch !== "string" || !branch.startsWith("change/")) return "";
  const id = branch.slice("change/".length);
  return id || "";
}
