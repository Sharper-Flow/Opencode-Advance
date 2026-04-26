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

// src/index.ts
function stateFilePath() {
  const paneId = process.env.TMUX_PANE;
  if (!paneId)
    return null;
  const socket = parseSocketFromTmux(process.env.TMUX);
  const sanitized = sanitizePaneId(paneId);
  return xdgStateHome("oca", "panes", socket, `${sanitized}.json`);
}
var src_default = {
  sessionCreated: async (session, _output) => {
    const filePath = stateFilePath();
    if (!filePath)
      return;
    atomicWriteJSON(filePath, {
      sessionID: session.id,
      directory: session.directory,
      ts: Date.now()
    });
  },
  sessionDeleted: async (session, _output) => {
    const filePath = stateFilePath();
    if (!filePath)
      return;
    const existing = readJSON(filePath);
    if (existing && existing.sessionID === session.id) {
      deleteStateFile(filePath);
    }
  }
};
export {
  src_default as default
};
