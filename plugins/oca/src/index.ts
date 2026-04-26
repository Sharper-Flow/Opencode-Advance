import { type Plugin } from "@opencode-ai/plugin";
import { xdgStateHome, atomicWriteJSON, readJSON, deleteStateFile } from "./state-file";
import { parseSocketFromTmux, sanitizePaneId } from "./tmux";
import * as path from "path";

function stateFilePath(): string | null {
  const paneId = process.env.TMUX_PANE;
  if (!paneId) return null;
  const socket = parseSocketFromTmux(process.env.TMUX);
  const sanitized = sanitizePaneId(paneId);
  return xdgStateHome("oca", "panes", socket, `${sanitized}.json`);
}

export default {
  sessionCreated: async (session, _output) => {
    const filePath = stateFilePath();
    if (!filePath) return;
    atomicWriteJSON(filePath, {
      sessionID: session.id,
      directory: session.directory,
      ts: Date.now(),
    });
  },
  sessionDeleted: async (session, _output) => {
    const filePath = stateFilePath();
    if (!filePath) return;
    const existing = readJSON(filePath);
    if (existing && existing.sessionID === session.id) {
      deleteStateFile(filePath);
    }
  },
} satisfies Plugin;
