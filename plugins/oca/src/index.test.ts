import { describe, test, expect, beforeEach, mock } from "bun:test";
import { deriveBranchSafety, deriveChangeID } from "./change";

// ── deriveChangeID tests ───────────────────────────────────────────────────

describe("deriveChangeID", () => {
  test("extracts changeID from change/ branch", () => {
    expect(deriveChangeID("change/myChange")).toBe("myChange");
  });

  test("handles multi-segment change ID", () => {
    expect(deriveChangeID("change/fix/auth-bug")).toBe("fix/auth-bug");
  });

  test("returns empty for non-change branch", () => {
    expect(deriveChangeID("trunk")).toBe("");
    expect(deriveChangeID("main")).toBe("");
    expect(deriveChangeID("feature/foo")).toBe("");
  });

  test("returns empty for empty string", () => {
    expect(deriveChangeID("")).toBe("");
  });

  test("returns empty for bare 'change/'", () => {
    expect(deriveChangeID("change/")).toBe("");
  });

  test("returns empty for non-string plugin-host input", () => {
    expect(deriveChangeID({ directory: "/repo" } as unknown as string)).toBe("");
  });
});

describe("deriveBranchSafety", () => {
  test("marks linked worktrees safe for WIP", () => {
    expect(
      deriveBranchSafety({
        isMainCheckout: false,
        isWorktree: true,
        branch: "change/foo",
        defaultBranch: "trunk",
      }),
    ).toBe("worktree");
  });

  test("marks non-default main checkout branch unsafe", () => {
    expect(
      deriveBranchSafety({
        isMainCheckout: true,
        isWorktree: false,
        branch: "feature/foo",
        defaultBranch: "trunk",
      }),
    ).toBe("unsafe_main_branch");
  });

  test("marks default branch main checkout safe", () => {
    expect(
      deriveBranchSafety({
        isMainCheckout: true,
        isWorktree: false,
        branch: "trunk",
        defaultBranch: "trunk",
      }),
    ).toBe("safe_main_checkout");
  });
});
