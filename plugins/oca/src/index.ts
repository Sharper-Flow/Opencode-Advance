import { type Plugin } from "@opencode-ai/plugin";
import {
  xdgStateHome,
  atomicWriteJSON,
  readJSON,
  deleteStateFile,
} from "./state-file";
import { parseSocketFromTmux, sanitizePaneId } from "./tmux";
import { initWatchdog, handleWatchdogEvent } from "./watchdog";
import * as path from "path";
import { execFileSync } from "child_process";

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
/**
 * Derive the ADV change ID from a git branch name.
 * Returns the change ID if branch is "change/{id}", empty string otherwise.
 */
export function deriveChangeID(branch: string): string {
  if (!branch || !branch.startsWith("change/")) return "";
  const id = branch.slice("change/".length);
  return id || "";
}

export function deriveBranchSafety(input: {
  isMainCheckout: boolean;
  isWorktree: boolean;
  branch?: string;
  defaultBranch?: string;
}): "worktree" | "safe_main_checkout" | "unsafe_main_branch" | "unknown" {
  if (input.isWorktree) return "worktree";
  if (!input.isMainCheckout) return "unknown";
  if (
    input.defaultBranch &&
    input.branch &&
    input.branch !== input.defaultBranch
  ) {
    return "unsafe_main_branch";
  }
  return "safe_main_checkout";
}

function normalizeDefaultBranch(raw: string): string {
  return raw.replace(/^origin\//, "").trim();
}

function buildV2State(info: {
  id: string;
  directory?: string;
}): Record<string, unknown> {
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

  // Git metadata (best-effort from directory). Use git directly instead of
  // checking for <directory>/.git so subdirectories of a worktree are handled.
  const dir = info.directory;
  if (dir) {
    try {
      const git = (args: string[]) =>
        execFileSync("git", args, {
          cwd: dir,
          encoding: "utf8",
          timeout: 2000,
          stdio: ["ignore", "pipe", "ignore"],
        }).trim();

      const gitRoot = git([
        "rev-parse",
        "--path-format=absolute",
        "--show-toplevel",
      ]);
      if (gitRoot) {
        state.gitRoot = gitRoot;
        state.worktreePath = gitRoot;
      }

      const commonDir = git([
        "rev-parse",
        "--path-format=absolute",
        "--git-common-dir",
      ]);
      if (commonDir) {
        const gitCommonDir = path.resolve(dir, commonDir);
        const mainCheckoutPath = path.dirname(gitCommonDir);
        state.gitCommonDir = gitCommonDir;
        state.mainCheckoutPath = mainCheckoutPath;
        state.isMainCheckout = gitRoot
          ? path.resolve(gitRoot) === path.resolve(mainCheckoutPath)
          : false;
      }

      let defaultBranch = "";
      try {
        defaultBranch = normalizeDefaultBranch(
          git(["symbolic-ref", "--short", "refs/remotes/origin/HEAD"]),
        );
      } catch {
        try {
          defaultBranch = git([
            "rev-parse",
            "--abbrev-ref",
            "origin/HEAD",
          ]).replace(/^origin\//, "");
        } catch {
          /* best-effort: no remote default branch */
        }
      }
      if (defaultBranch) state.defaultBranch = defaultBranch;

      const rootCommit = git(["rev-list", "--max-parents=0", "HEAD"]);
      if (rootCommit) {
        state.projectId = rootCommit;
      }

      const branch = git(["rev-parse", "--abbrev-ref", "HEAD"]);
      if (branch && branch !== "HEAD") {
        state.worktreeBranch = branch;
        // Derive changeID from ADV worktree branch convention.
        const cid = deriveChangeID(branch);
        if (cid) state.changeID = cid;
      }

      if (gitRoot) {
        state.branchSafety = deriveBranchSafety({
          isMainCheckout: state.isMainCheckout === true,
          isWorktree: state.isMainCheckout === false,
          branch:
            typeof state.worktreeBranch === "string"
              ? state.worktreeBranch
              : undefined,
          defaultBranch,
        });
      }
    } catch {
      /* best-effort: ignore permission/ENOENT */
    }
  }

  // Role: explicit from env, or derived from agent.
  const role = process.env.OCA_PANE_ROLE || agent;
  if (role) state.role = role;

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
