import { type Plugin } from "@opencode-ai/plugin";

/**
 * OCA (OpenCode Advance) umbrella plugin.
 *
 * Feature modules:
 * - session-tracker: captures per-pane opencode session IDs for restart continuity
 */

export default {
  sessionCreated: async (session, _output) => {
    // TODO: write per-pane state file in session-tracker module
    console.log("[oca] sessionCreated:", session.id);
  },
  sessionDeleted: async (session, _output) => {
    // TODO: clear per-pane state file in session-tracker module
    console.log("[oca] sessionDeleted:", session.id);
  },
} satisfies Plugin;
