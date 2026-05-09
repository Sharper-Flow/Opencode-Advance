// @bun
// src/state-file.ts
import * as fs from "fs";
import * as path from "path";
import * as os from "os";
function xdgStateHome(...subdirs) {
  const base = process.env.XDG_STATE_HOME || path.join(os.homedir(), ".local", "state");
  return path.join(base, ...subdirs);
}
function atomicWriteJSON(filePath, data) {
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
  const tmpFile = `${filePath}.${process.pid}.tmp`;
  fs.writeFileSync(tmpFile, JSON.stringify(data, null, 2), { encoding: "utf8", mode: 384 });
  fs.renameSync(tmpFile, filePath);
}
function readJSON(filePath) {
  try {
    const raw = fs.readFileSync(filePath, { encoding: "utf8" });
    return JSON.parse(raw);
  } catch {
    return null;
  }
}
function deleteStateFile(filePath) {
  try {
    fs.unlinkSync(filePath);
  } catch {}
}

// src/tmux.ts
function parseSocketFromTmux(tmuxEnv) {
  if (!tmuxEnv)
    return "oca";
  const parts = tmuxEnv.split(",");
  if (parts.length < 2)
    return "oca";
  const socketPath = parts[0];
  const basename = socketPath.split("/").pop();
  return basename || "oca";
}
function sanitizePaneId(paneId) {
  return paneId.replace(/^%/, "");
}

// src/watchdog.ts
function readConfig() {
  return {
    enabled: process.env.OCA_WATCHDOG_ENABLED === "1",
    idleTimeoutMs: parseInt(process.env.OCA_WATCHDOG_IDLE_TIMEOUT_MS || "0", 10),
    maxBumps: parseInt(process.env.OCA_WATCHDOG_MAX_BUMPS || "0", 10)
  };
}
function stateFilePath() {
  const paneId = process.env.TMUX_PANE;
  if (!paneId)
    return null;
  const socket = parseSocketFromTmux(process.env.TMUX);
  const sanitized = sanitizePaneId(paneId);
  return xdgStateHome("oca", "panes", socket, `${sanitized}.json`);
}
function readPaneState(filePath) {
  return readJSON(filePath);
}
function writePaneState(filePath, state) {
  atomicWriteJSON(filePath, state);
}
var trackers = new Map;
var checkInterval = null;
function stopWatchdog() {
  if (!checkInterval)
    return;
  clearInterval(checkInterval);
  checkInterval = null;
}
function bumpSession(filePath, paneState, config, $shell) {
  const wd = paneState.watchdog || {
    enabled: true,
    bump_count: 0,
    last_bump_at: 0,
    last_activity_at: Date.now(),
    status: "active"
  };
  if (wd.bump_count >= config.maxBumps) {
    wd.status = "exhausted";
    paneState.watchdog = wd;
    writePaneState(filePath, paneState);
    try {
      $shell`tmux display-message "\u26A0 OCA watchdog exhausted for session ${paneState.sessionID}"`.catch(() => {});
    } catch {}
    return;
  }
  wd.bump_count++;
  wd.last_bump_at = Date.now();
  wd.status = "active";
  paneState.watchdog = wd;
  writePaneState(filePath, paneState);
  try {
    $shell`oca pane restart-tui --force`.catch(() => {});
  } catch {}
}
function checkHang(config, $shell) {
  if (!config.enabled || config.idleTimeoutMs <= 0)
    return;
  const filePath = stateFilePath();
  if (!filePath)
    return;
  const now = Date.now();
  for (const [sessionId, tracker] of trackers) {
    if (tracker.currentStatus !== "busy")
      continue;
    const elapsed = now - tracker.lastActivityAt;
    if (elapsed < config.idleTimeoutMs)
      continue;
    const paneState = readPaneState(filePath);
    if (!paneState || paneState.sessionID !== sessionId)
      continue;
    bumpSession(filePath, paneState, config, $shell);
  }
}
function initWatchdog(input) {
  const config = readConfig();
  if (!config.enabled) {
    stopWatchdog();
    return;
  }
  stopWatchdog();
  checkInterval = setInterval(() => {
    checkHang(config, input.$);
  }, 60000);
  if (checkInterval.unref) {
    checkInterval.unref();
  }
}
function handleWatchdogEvent(event) {
  const config = readConfig();
  if (!config.enabled)
    return;
  switch (event.type) {
    case "session.status": {
      const { sessionID, status } = event.properties || {};
      if (!sessionID)
        break;
      let tracker = trackers.get(sessionID);
      if (!tracker) {
        tracker = { lastActivityAt: Date.now(), currentStatus: "unknown" };
        trackers.set(sessionID, tracker);
      }
      if (status?.type === "busy") {
        tracker.currentStatus = "busy";
        tracker.lastActivityAt = Date.now();
      } else if (status?.type === "idle") {
        tracker.currentStatus = "idle";
      }
      break;
    }
    case "message.part.updated": {
      const now = Date.now();
      for (const tracker of trackers.values()) {
        if (tracker.currentStatus === "busy") {
          tracker.lastActivityAt = now;
        }
      }
      break;
    }
    case "session.idle": {
      const { sessionID } = event.properties || {};
      if (!sessionID)
        break;
      const tracker = trackers.get(sessionID);
      if (tracker) {
        tracker.currentStatus = "idle";
      }
      break;
    }
    case "session.deleted": {
      const info = event.properties?.info;
      if (info?.id) {
        trackers.delete(info.id);
      }
      break;
    }
  }
}

// src/change.ts
function deriveChangeID(branch) {
  if (typeof branch !== "string" || !branch.startsWith("change/"))
    return "";
  const id = branch.slice("change/".length);
  return id || "";
}
function deriveBranchSafety(input) {
  if (input.isWorktree)
    return "worktree";
  if (!input.isMainCheckout)
    return "unknown";
  if (input.defaultBranch && input.branch && input.branch !== input.defaultBranch) {
    return "unsafe_main_branch";
  }
  return "safe_main_checkout";
}

// src/index.ts
import * as path2 from "path";
import { execFileSync } from "child_process";
function stateFilePath2(input) {
  const paneId = process.env.TMUX_PANE;
  if (!paneId)
    return null;
  const socket = parseSocketFromTmux(process.env.TMUX);
  const sanitized = sanitizePaneId(paneId);
  return xdgStateHome("oca", "panes", socket, `${sanitized}.json`);
}
function normalizeDefaultBranch(raw) {
  return raw.replace(/^origin\//, "").trim();
}
function buildV2State(info) {
  const now = Date.now();
  const state = {
    schemaVersion: 2,
    sessionID: info.id,
    directory: info.directory,
    ts: now,
    startedAt: now,
    lastSeenAt: now
  };
  const paneId = process.env.TMUX_PANE;
  const socket = parseSocketFromTmux(process.env.TMUX);
  if (paneId)
    state.paneID = paneId;
  if (socket && socket !== "oca")
    state.socket = socket;
  else if (process.env.TMUX)
    state.socket = socket;
  const agent = process.env.OPENCODE_AGENT;
  if (agent)
    state.agent = agent;
  const dir = info.directory;
  if (dir) {
    try {
      const git = (args) => execFileSync("git", args, {
        cwd: dir,
        encoding: "utf8",
        timeout: 2000,
        stdio: ["ignore", "pipe", "ignore"]
      }).trim();
      const gitRoot = git([
        "rev-parse",
        "--path-format=absolute",
        "--show-toplevel"
      ]);
      if (gitRoot) {
        state.gitRoot = gitRoot;
        state.worktreePath = gitRoot;
      }
      const commonDir = git([
        "rev-parse",
        "--path-format=absolute",
        "--git-common-dir"
      ]);
      if (commonDir) {
        const gitCommonDir = path2.resolve(dir, commonDir);
        const mainCheckoutPath = path2.dirname(gitCommonDir);
        state.gitCommonDir = gitCommonDir;
        state.mainCheckoutPath = mainCheckoutPath;
        state.isMainCheckout = gitRoot ? path2.resolve(gitRoot) === path2.resolve(mainCheckoutPath) : false;
      }
      let defaultBranch = "";
      try {
        defaultBranch = normalizeDefaultBranch(git(["symbolic-ref", "--short", "refs/remotes/origin/HEAD"]));
      } catch {
        try {
          defaultBranch = git([
            "rev-parse",
            "--abbrev-ref",
            "origin/HEAD"
          ]).replace(/^origin\//, "");
        } catch {}
      }
      if (defaultBranch)
        state.defaultBranch = defaultBranch;
      const rootCommit = git(["rev-list", "--max-parents=0", "HEAD"]);
      if (rootCommit) {
        state.projectId = rootCommit;
      }
      const branch = git(["rev-parse", "--abbrev-ref", "HEAD"]);
      if (branch && branch !== "HEAD") {
        state.worktreeBranch = branch;
        const cid = deriveChangeID(branch);
        if (cid)
          state.changeID = cid;
      }
      if (gitRoot) {
        state.branchSafety = deriveBranchSafety({
          isMainCheckout: state.isMainCheckout === true,
          isWorktree: state.isMainCheckout === false,
          branch: typeof state.worktreeBranch === "string" ? state.worktreeBranch : undefined,
          defaultBranch
        });
      }
    } catch {}
  }
  const role = process.env.OCA_PANE_ROLE || agent;
  if (role)
    state.role = role;
  return state;
}
var plugin = async (input) => {
  initWatchdog(input);
  return {
    async event({ event }) {
      handleWatchdogEvent(event);
      switch (event.type) {
        case "session.created": {
          const info = event.properties?.info;
          if (!info)
            break;
          const filePath = stateFilePath2(null);
          if (!filePath)
            break;
          atomicWriteJSON(filePath, buildV2State(info));
          break;
        }
        case "session.deleted": {
          const info = event.properties?.info;
          if (!info)
            break;
          const filePath = stateFilePath2(null);
          if (!filePath)
            break;
          const existing = readJSON(filePath);
          if (existing && existing.sessionID === info.id) {
            deleteStateFile(filePath);
          }
          break;
        }
      }
    }
  };
};
var src_default = plugin;
export {
  src_default as default
};
