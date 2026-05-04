#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

# shellcheck source=/dev/null
source "$ROOT_DIR/lib/palette.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/adv_status.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/llm_gauge.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/status_bar.sh"

# ── Helpers ────────────────────────────────────────────────

assert_contains() {
  local haystack=$1 needle=$2 msg=$3
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'FAIL assert_contains: %s\n  missing %q\n' "$msg" "$needle" >&2
    exit 1
  fi
}

assert_not_contains() {
  local haystack=$1 needle=$2 msg=$3
  if [[ "$haystack" == *"$needle"* ]]; then
    printf 'FAIL assert_not_contains: %s\n  unexpected %q\n' "$msg" "$needle" >&2
    exit 1
  fi
}

# ── Tests ──────────────────────────────────────────────────

printf 'status_bar: row0 includes session name... '
OUTPUT=$(oca_status_row0 "my-session" "/tmp")
assert_contains "$OUTPUT" "my-session" "row0 should contain session name"
printf 'OK\n'

printf 'status_bar: row0 includes git branch in repo... '
GIT_DIR=$(mktemp -d)
cd "$GIT_DIR" && git init && git config user.email "test@test" && git config user.name "Test" && git checkout -b my-branch >/dev/null 2>&1 || true && touch a && git add a && git commit -m "init" >/dev/null 2>&1 || true
OUTPUT=$(oca_status_row0 "test" "$GIT_DIR")
assert_contains "$OUTPUT" "my-branch" "row0 should contain git branch"
rm -rf "$GIT_DIR"
printf 'OK\n'

printf 'status_bar: row0 degrades outside git repo... '
OUTPUT=$(oca_status_row0 "test" "/tmp")
assert_not_contains "$OUTPUT" "master" "row0 should not show branch outside repo"
printf 'OK\n'

printf 'status_bar: row0 includes host... '
OUTPUT=$(oca_status_row0 "test" "/tmp")
assert_contains "$OUTPUT" "#[align=right]" "row0 should have right-aligned section"
printf 'OK\n'

printf 'status_bar: row1 includes date... '
OUTPUT=$(oca_status_row1 "/tmp")
assert_contains "$OUTPUT" "$(date '+%Y-%m-%d')" "row1 should contain date"
printf 'OK\n'

printf 'status_bar: row1 includes LLM gauges when data exists... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
mkdir -p "$OCA_CACHE_DIR"
printf '75' > "$OCA_CACHE_DIR/zai"
OUTPUT=$(oca_status_row1 "/tmp")
assert_contains "$OUTPUT" "za" "row1 should contain za gauge"
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'status_bar: row0 includes temporal health when configured... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance"
printf 'ADV_TEMPORAL_ADDRESS=127.0.0.1:1\n' > "$OCA_CACHE_DIR/temporal.env"
OUTPUT=$(oca_status_row0 "test" "/tmp")
assert_contains "$OUTPUT" "T:✗" "row0 should show temporal health when env exists"
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'status_bar: row0 omits temporal health when not configured... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
# No temporal.env, no ADV dir
OUTPUT=$(oca_status_row0 "test" "/tmp")
assert_not_contains "$OUTPUT" "T:" "row0 should not show temporal health when not configured"
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'status_bar: performance budget under 200ms... '
START=$(date +%s%N)
OUTPUT=$(oca_status_row0 "test" "/tmp")
END=$(date +%s%N)
ELAPSED=$(( (END - START) / 1000000 ))  # Convert ns to ms
if (( ELAPSED > 200 )); then
  printf 'FAIL: row0 took %dms, budget is 200ms\n' "$ELAPSED" >&2
  exit 1
fi
printf 'OK (%dms)\n' "$ELAPSED"

# ── Occupancy segment tests ─────────────────────────────────

printf 'occupancy: --status --pane returns ? for missing state... '
OCA_BIN="$ROOT_DIR/oca"
if [[ -x "$OCA_BIN" ]]; then
  STATUS_OUTPUT=$(TMUX_PANE="%9999" timeout 0.5s "$OCA_BIN" occupancy --status --pane "%9999" 2>/dev/null || true)
  if [[ "$STATUS_OUTPUT" != "?" ]]; then
    printf 'FAIL: expected "?", got %q\n' "$STATUS_OUTPUT" >&2
    exit 1
  fi
  printf 'OK\n'
else
  printf 'SKIP (oca binary not built)\n'
fi

printf 'occupancy: --status --pane returns ? without TMUX_PANE... '
if [[ -x "$OCA_BIN" ]]; then
  STATUS_OUTPUT=$(unset TMUX_PANE; timeout 0.5s "$OCA_BIN" occupancy --status 2>/dev/null || true)
  if [[ "$STATUS_OUTPUT" != "?" ]]; then
    printf 'FAIL: expected "?", got %q\n' "$STATUS_OUTPUT" >&2
    exit 1
  fi
  printf 'OK\n'
else
  printf 'SKIP (oca binary not built)\n'
fi

printf 'occupancy: timeout kills slow invocation... '
if [[ -x "$OCA_BIN" ]]; then
  # 0.1s timeout should still return quickly
  START=$(date +%s%N)
  timeout 0.1s "$OCA_BIN" occupancy --status --pane "%1" 2>/dev/null || true
  END=$(date +%s%N)
  ELAPSED=$(( (END - START) / 1000000 ))
  if (( ELAPSED > 500 )); then
    printf 'FAIL: timeout did not kill after %dms\n' "$ELAPSED" >&2
    exit 1
  fi
  printf 'OK (%dms)\n' "$ELAPSED"
else
  printf 'SKIP (oca binary not built)\n'
fi

printf '\nAll status_bar tests passed.\n'
