import { describe, test, expect, mock, beforeEach } from "bun:test";
import {
  handleWatchdogEvent,
  initWatchdog,
} from "./watchdog";
import * as stateFile from "./state-file";

// ── Helpers ────────────────────────────────────────────────────────────────

function setEnv(vars: Record<string, string>) {
  for (const [k, v] of Object.entries(vars)) {
    process.env[k] = v;
  }
}

function clearWatchdogEnv() {
  delete process.env.OCA_WATCHDOG_ENABLED;
  delete process.env.OCA_WATCHDOG_IDLE_TIMEOUT_MS;
  delete process.env.OCA_WATCHDOG_MAX_BUMPS;
}

// ── Tests ──────────────────────────────────────────────────────────────────

describe("handleWatchdogEvent", () => {
  beforeEach(clearWatchdogEnv);

  test("ignores events when watchdog disabled", () => {
    setEnv({ OCA_WATCHDOG_ENABLED: "0" });
    // Should not throw.
    handleWatchdogEvent({ type: "session.status", properties: { sessionID: "s1", status: { type: "busy" } } });
  });

  test("tracks busy status when enabled", () => {
    setEnv({
      OCA_WATCHDOG_ENABLED: "1",
      OCA_WATCHDOG_IDLE_TIMEOUT_MS: "300000",
      OCA_WATCHDOG_MAX_BUMPS: "3",
    });

    handleWatchdogEvent({
      type: "session.status",
      properties: { sessionID: "s1", status: { type: "busy" } },
    });

    // Internal state not directly observable, but subsequent idle should clear it.
    handleWatchdogEvent({
      type: "session.idle",
      properties: { sessionID: "s1" },
    });

    // No crash = success.
  });

  test("resets activity on message.part.updated", () => {
    setEnv({
      OCA_WATCHDOG_ENABLED: "1",
      OCA_WATCHDOG_IDLE_TIMEOUT_MS: "300000",
      OCA_WATCHDOG_MAX_BUMPS: "3",
    });

    // Set busy first.
    handleWatchdogEvent({
      type: "session.status",
      properties: { sessionID: "s1", status: { type: "busy" } },
    });

    // Message part updated should reset activity timer.
    handleWatchdogEvent({
      type: "message.part.updated",
      properties: { part: { id: "p1" } },
    });

    // No crash = success.
  });

  test("cleans up on session.deleted", () => {
    setEnv({
      OCA_WATCHDOG_ENABLED: "1",
      OCA_WATCHDOG_IDLE_TIMEOUT_MS: "300000",
      OCA_WATCHDOG_MAX_BUMPS: "3",
    });

    handleWatchdogEvent({
      type: "session.status",
      properties: { sessionID: "s1", status: { type: "busy" } },
    });

    handleWatchdogEvent({
      type: "session.deleted",
      properties: { info: { id: "s1" } },
    });

    // Tracker should be removed. No crash = success.
  });

  test("no-ops on unknown event type", () => {
    setEnv({
      OCA_WATCHDOG_ENABLED: "1",
      OCA_WATCHDOG_IDLE_TIMEOUT_MS: "300000",
      OCA_WATCHDOG_MAX_BUMPS: "3",
    });

    handleWatchdogEvent({ type: "unknown.event", properties: {} });
  });
});

describe("initWatchdog", () => {
  beforeEach(clearWatchdogEnv);

  test("no-ops when disabled", () => {
    setEnv({ OCA_WATCHDOG_ENABLED: "0" });
    const mockInput = { $: {} } as any;
    initWatchdog(mockInput);
    // No interval set when disabled.
  });
});
