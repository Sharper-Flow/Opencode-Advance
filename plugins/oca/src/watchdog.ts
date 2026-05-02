import { type PluginInput, type Hooks } from "@opencode-ai/plugin";
import { xdgStateHome, atomicWriteJSON, readJSON } from "./state-file";
import { parseSocketFromTmux, sanitizePaneId } from "./tmux";

// ── Types ──────────────────────────────────────────────────────────────────

interface WatchdogState {
  enabled: boolean;
  bump_count: number;
  last_bump_at: number;
  last_activity_at: number;
  status: "active" | "exhausted" | "disabled";
}

interface PaneState {
  sessionID: string;
  directory: string;
  ts: number;
  watchdog?: WatchdogState;
}

interface SessionTracker {
  lastActivityAt: number;
  currentStatus: "idle" | "busy" | "unknown";
}

// ── Config from env ────────────────────────────────────────────────────────

function readConfig(): {
  enabled: boolean;
  idleTimeoutMs: number;
  maxBumps: number;
} {
  return {
    enabled: process.env.OCA_WATCHDOG_ENABLED === "1",
    idleTimeoutMs: parseInt(process.env.OCA_WATCHDOG_IDLE_TIMEOUT_MS || "0", 10),
    maxBumps: parseInt(process.env.OCA_WATCHDOG_MAX_BUMPS || "0", 10),
  };
}

// ── State file helpers ─────────────────────────────────────────────────────

function stateFilePath(): string | null {
  const paneId = process.env.TMUX_PANE;
  if (!paneId) return null;
  const socket = parseSocketFromTmux(process.env.TMUX);
  const sanitized = sanitizePaneId(paneId);
  return xdgStateHome("oca", "panes", socket, `${sanitized}.json`);
}

function readPaneState(filePath: string): PaneState | null {
  return readJSON(filePath) as PaneState | null;
}

function writePaneState(filePath: string, state: PaneState): void {
  atomicWriteJSON(filePath, state);
}

// ── Watchdog core ──────────────────────────────────────────────────────────

const trackers = new Map<string, SessionTracker>();
let checkInterval: ReturnType<typeof setInterval> | null = null;

function bumpSession(
  filePath: string,
  paneState: PaneState,
  config: { maxBumps: number },
  $shell: any,
): void {
  const wd = paneState.watchdog || {
    enabled: true,
    bump_count: 0,
    last_bump_at: 0,
    last_activity_at: Date.now(),
    status: "active" as const,
  };

  if (wd.bump_count >= config.maxBumps) {
    // Exhausted — write status, emit tmux warning, stop checking.
    wd.status = "exhausted";
    paneState.watchdog = wd;
    writePaneState(filePath, paneState);
    // Non-blocking tmux notification.
    try {
      $shell`tmux display-message "⚠ OCA watchdog exhausted for session ${paneState.sessionID}"`.catch(
        () => {},
      );
    } catch {}
    return;
  }

  // Bump: respawn the TUI preserving the session.
  wd.bump_count++;
  wd.last_bump_at = Date.now();
  wd.status = "active";
  paneState.watchdog = wd;
  writePaneState(filePath, paneState);

  try {
    $shell`oca pane restart-tui --force`.catch(() => {});
  } catch {}
}

function checkHang(
  config: { enabled: boolean; idleTimeoutMs: number; maxBumps: number },
  $shell: any,
): void {
  if (!config.enabled || config.idleTimeoutMs <= 0) return;

  const filePath = stateFilePath();
  if (!filePath) return;

  const now = Date.now();

  for (const [sessionId, tracker] of trackers) {
    if (tracker.currentStatus !== "busy") continue;

    const elapsed = now - tracker.lastActivityAt;
    if (elapsed < config.idleTimeoutMs) continue;

    // Hang detected for this session.
    const paneState = readPaneState(filePath);
    if (!paneState || paneState.sessionID !== sessionId) continue;

    bumpSession(filePath, paneState, config, $shell);
  }
}

// ── Init ───────────────────────────────────────────────────────────────────

export function initWatchdog(input: PluginInput): void {
  const config = readConfig();
  if (!config.enabled) return;

  // Start periodic check (60s interval).
  checkInterval = setInterval(() => {
    checkHang(config, input.$);
  }, 60_000);

  // Don't prevent process exit.
  if (checkInterval.unref) {
    checkInterval.unref();
  }
}

// ── Event handler ──────────────────────────────────────────────────────────

export function handleWatchdogEvent(event: any): void {
  const config = readConfig();
  if (!config.enabled) return;

  switch (event.type) {
    case "session.status": {
      const { sessionID, status } = event.properties || {};
      if (!sessionID) break;

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
      // Any message part activity resets the timer for ALL tracked sessions.
      // This is conservative — we don't know which session the message belongs to
      // from the event alone, but message activity means the system is alive.
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
      if (!sessionID) break;
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

// ── Test-only diagnostics ──────────────────────────────────────────────────

/** @internal For test verification only. */
export function __testClearTrackers(): void {
  trackers.clear();
}
/** @internal For test verification only. */
export function __testTrackerCount(): number {
  return trackers.size;
}

// ── Hook export ────────────────────────────────────────────────────────────

export function watchdogHooks(input: PluginInput): Pick<Hooks, "event"> {
  initWatchdog(input);

  return {
    async event({ event }) {
      handleWatchdogEvent(event);
    },
  };
}
