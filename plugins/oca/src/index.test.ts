import { describe, test, expect, beforeEach, mock } from "bun:test";
import { deriveChangeID } from "./index";

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
});
