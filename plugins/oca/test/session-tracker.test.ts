import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";

import {
  xdgStateHome,
  atomicWriteJSON,
  readJSON,
  deleteStateFile,
} from "../src/state-file";
import {
  parseSocketFromTmux,
  sanitizePaneId,
} from "../src/tmux";

describe("state-file", () => {
  const tmpDir = path.join(os.tmpdir(), `oca-test-${Date.now()}`);

  beforeEach(() => {
    fs.mkdirSync(tmpDir, { recursive: true });
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it("writes and reads JSON atomically", () => {
    const file = path.join(tmpDir, "test.json");
    atomicWriteJSON(file, { sessionID: "ses_abc", directory: "/tmp", ts: 123 });
    const got = readJSON(file);
    expect(got).toEqual({ sessionID: "ses_abc", directory: "/tmp", ts: 123 });
  });

  it("deletes existing file", () => {
    const file = path.join(tmpDir, "del.json");
    atomicWriteJSON(file, { sessionID: "ses_abc" });
    expect(fs.existsSync(file)).toBe(true);
    deleteStateFile(file);
    expect(fs.existsSync(file)).toBe(false);
  });

  it("atomic write does not leave partial file on failure", () => {
    const file = path.join(tmpDir, "atomic.json");
    atomicWriteJSON(file, { sessionID: "ses_abc" });
    const tmpFiles = fs.readdirSync(tmpDir).filter((f) => f.endsWith(".tmp"));
    expect(tmpFiles.length).toBe(0);
  });

  it("readJSON returns null for missing file", () => {
    const got = readJSON(path.join(tmpDir, "missing.json"));
    expect(got).toBeNull();
  });

  it("xdgStateHome respects XDG_STATE_HOME env var", () => {
    const original = process.env.XDG_STATE_HOME;
    process.env.XDG_STATE_HOME = tmpDir;
    const got = xdgStateHome();
    expect(got).toBe(tmpDir);
    process.env.XDG_STATE_HOME = original;
  });

  it("xdgStateHome falls back to ~/.local/state", () => {
    const original = process.env.XDG_STATE_HOME;
    delete process.env.XDG_STATE_HOME;
    const got = xdgStateHome();
    const home = os.homedir();
    expect(got).toBe(path.join(home, ".local", "state"));
    process.env.XDG_STATE_HOME = original;
  });
});

describe("tmux", () => {
  it("parseSocketFromTmux extracts socket from $TMUX env var", () => {
    const socket = parseSocketFromTmux("/tmp/tmux-1000/oca,12345");
    expect(socket).toBe("oca");
  });

  it("parseSocketFromTmux handles default socket", () => {
    const socket = parseSocketFromTmux("/tmp/tmux-1000/default,12345");
    expect(socket).toBe("default");
  });

  it("parseSocketFromTmux returns 'oca' for empty input", () => {
    expect(parseSocketFromTmux("")).toBe("oca");
    expect(parseSocketFromTmux(undefined as any)).toBe("oca");
  });

  it("sanitizePaneId strips leading %", () => {
    expect(sanitizePaneId("%42")).toBe("42");
    expect(sanitizePaneId("%0")).toBe("0");
  });

  it("sanitizePaneId leaves clean IDs unchanged", () => {
    expect(sanitizePaneId("42")).toBe("42");
    expect(sanitizePaneId("0")).toBe("0");
  });
});

describe("session tracker hooks", () => {
  const tmpDir = path.join(os.tmpdir(), `oca-hooks-test-${Date.now()}`);
  const originalTmuxPane = process.env.TMUX_PANE;
  const originalTmux = process.env.TMUX;
  const originalXdg = process.env.XDG_STATE_HOME;

  beforeEach(() => {
    fs.mkdirSync(tmpDir, { recursive: true });
    process.env.XDG_STATE_HOME = tmpDir;
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
    process.env.TMUX_PANE = originalTmuxPane;
    process.env.TMUX = originalTmux;
    process.env.XDG_STATE_HOME = originalXdg;
  });

  it("sessionCreated writes state file when TMUX_PANE is set", async () => {
    process.env.TMUX_PANE = "%42";
    process.env.TMUX = "/tmp/tmux-1000/oca,12345";

    const mod = await import("../src/index");
    await mod.default.sessionCreated(
      { id: "ses_abc", directory: "/home/user/project" } as any,
      {} as any
    );

    const stateFile = path.join(tmpDir, "oca", "panes", "oca", "42.json");
    const got = readJSON(stateFile);
    expect(got.sessionID).toBe("ses_abc");
    expect(got.directory).toBe("/home/user/project");
    expect(typeof got.ts).toBe("number");
  });

  it("sessionCreated no-ops when TMUX_PANE is unset", async () => {
    delete process.env.TMUX_PANE;

    const mod = await import("../src/index");
    await mod.default.sessionCreated({ id: "ses_abc" } as any, {} as any);

    const stateDir = path.join(tmpDir, "oca", "panes");
    expect(fs.existsSync(stateDir)).toBe(false);
  });

  it("sessionDeleted clears matching state file", async () => {
    process.env.TMUX_PANE = "%42";
    process.env.TMUX = "/tmp/tmux-1000/oca,12345";

    const mod = await import("../src/index");
    await mod.default.sessionCreated({ id: "ses_abc", directory: "/tmp" } as any, {} as any);

    const stateFile = path.join(tmpDir, "oca", "panes", "oca", "42.json");
    expect(fs.existsSync(stateFile)).toBe(true);

    await mod.default.sessionDeleted({ id: "ses_abc" } as any, {} as any);
    expect(fs.existsSync(stateFile)).toBe(false);
  });

  it("sessionDeleted preserves non-matching state file", async () => {
    process.env.TMUX_PANE = "%42";
    process.env.TMUX = "/tmp/tmux-1000/oca,12345";

    const mod = await import("../src/index");
    await mod.default.sessionCreated({ id: "ses_abc", directory: "/tmp" } as any, {} as any);

    const stateFile = path.join(tmpDir, "oca", "panes", "oca", "42.json");
    await mod.default.sessionDeleted({ id: "ses_xyz" } as any, {} as any);
    expect(fs.existsSync(stateFile)).toBe(true);
  });
});
