import { type Plugin } from "@opencode-ai/plugin";
import { xdgStateHome, atomicWriteJSON, readJSON, deleteStateFile } from "./state-file";
import { parseSocketFromTmux, sanitizePaneId } from "./tmux";
import { initWatchdog, handleWatchdogEvent } from "./watchdog";
import * as path from "path";
import * as fs from "fs";
import * as os from "os";

function stateFilePath(input: { directory?: string } | null): string | null {
  const paneId = process.env.TMUX_PANE;
  if (!paneId) return null;
  const socket = parseSocketFromTmux(process.env.TMUX);
  const sanitized = sanitizePaneId(paneId);
  return xdgStateHome("oca", "panes", socket, `${sanitized}.json`);
}

/**
 * Build v2 pane state enrichment. Best-effort: fields derived from env
 * are only included when the source data is available.
 */
function buildV2State(info: { id: string; directory?: string }): Record<string, unknown> {
  const now = Date.now();
  const state: Record<string, unknown> = {
    schemaVersion: 2,
    sessionID: info.id,
    directory: info.directory,
    ts: now,
    startedAt: now,
    lastSeenAt: now,
  };

  // Tmux metadata (best-effort)
  const paneId = process.env.TMUX_PANE;
  const socket = parseSocketFromTmux(process.env.TMUX);
  if (paneId) state.paneID = paneId;
  if (socket && socket !== "oca") state.socket = socket;
  else if (process.env.TMUX) state.socket = socket; // explicit tmux, even if default name

  // Agent (best-effort from env only; never inferred from chat)
  const agent = process.env.OPENCODE_AGENT;
  if (agent) state.agent = agent;

  // Git metadata (best-effort from directory)
  const dir = info.directory;
  if (dir) {
    try {
      // Try to find .git and extract project info
      const gitDir = path.join(dir, ".git");
      if (fs.existsSync(gitDir)) {
        state.gitRoot = dir;
        // Resolve gitCommonDir (handles worktrees where .git is a file)
        const gitStat = fs.statSync(gitDir);
        if (gitStat.isDirectory()) {
          state.gitCommonDir = gitDir;
        } else {
          // .git file in worktree — read gitdir pointer
          const content = fs.readFileSync(gitDir, "utf8").trim();
          const match = content.match(/^gitdir:\s*(.+)$/m);
          if (match) {
            const gitdir = match[1].trim();
            state.gitCommonDir = path.resolve(dir, gitdir);
          }
        }

        // Project ID from root commit SHA
        try {
          const { execSync } = require("child_process");
          const rootCommit = execSync("git rev-list --max-parents=0 HEAD", {
            cwd: dir,
            encoding: "utf8",
            timeout: 2000,
          }).trim();
          if (rootCommit) state.projectId = rootCommit;
        } catch { /* best-effort */ }

        // Worktree info
        try {
          const { execSync } = require("child_process");
          state.worktreePath = dir;
          const branch = execSync("git rev-parse --abbrev-ref HEAD", {
            cwd: dir,
            encoding: "utf8",
            timeout: 2000,
          }).trim();
          if (branch && branch !== "HEAD") state.worktreeBranch = branch;
        } catch { /* best-effort */ }
      }
    } catch { /* best-effort: ignore permission/ENOENT */ }
  }

  return state;
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
          atomicWriteJSON(filePath, buildV2State(info));
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
