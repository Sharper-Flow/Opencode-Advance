import { type Plugin } from "@opencode-ai/plugin";
import { xdgStateHome, atomicWriteJSON, readJSON, deleteStateFile } from "./state-file";
import { parseSocketFromTmux, sanitizePaneId } from "./tmux";
import { initWatchdog, handleWatchdogEvent } from "./watchdog";
import * as path from "path";

function stateFilePath(input: { directory?: string } | null): string | null {
  const paneId = process.env.TMUX_PANE;
  if (!paneId) return null;
  const socket = parseSocketFromTmux(process.env.TMUX);
  const sanitized = sanitizePaneId(paneId);
  return xdgStateHome("oca", "panes", socket, `${sanitized}.json`);
}

const plugin: Plugin = async (input) => {
  // Initialize watchdog if enabled via env vars.
  initWatchdog(input);

  return {
    async event({ event }) {
      // Route all lifecycle events to the watchdog so tracker state stays
      // consistent with pane state.  The watchdog ignores events it does not
      // recognise (e.g. session.created), so this is safe.
      handleWatchdogEvent(event);

      switch (event.type) {
        case "session.created": {
          const info = event.properties?.info;
          if (!info) break;
          const filePath = stateFilePath(null);
          if (!filePath) break;
          atomicWriteJSON(filePath, {
            sessionID: info.id,
            directory: info.directory,
            ts: Date.now(),
          });
          break;
        }

        case "session.deleted": {
          const info = event.properties?.info;
          if (!info) break;
          const filePath = stateFilePath(null);
          if (!filePath) break;
          const existing = readJSON(filePath);
          if (existing && existing.sessionID === info.id) {
            deleteStateFile(filePath);
          }
          break;
        }
      }
    },
  };
};

export default plugin;
