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
  fs.writeFileSync(tmpFile, JSON.stringify(data, null, 2), { encoding: "utf8" });
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
  if (!config.enabled)
    return;
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

// src/index.ts
function stateFilePath2(input) {
  const paneId = process.env.TMUX_PANE;
  if (!paneId)
    return null;
  const socket = parseSocketFromTmux(process.env.TMUX);
  const sanitized = sanitizePaneId(paneId);
  return xdgStateHome("oca", "panes", socket, `${sanitized}.json`);
}
var plugin = async (input) => {
  initWatchdog(input);
  return {
    async event({ event }) {
      switch (event.type) {
        case "session.created": {
          const info = event.properties?.info;
          if (!info)
            break;
          const filePath = stateFilePath2(null);
          if (!filePath)
            break;
          atomicWriteJSON(filePath, {
            sessionID: info.id,
            directory: info.directory,
            ts: Date.now()
          });
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
        default:
          handleWatchdogEvent(event);
          break;
      }
    }
  };
};
var src_default = plugin;
export {
  src_default as default
};
