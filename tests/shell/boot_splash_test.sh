#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

# shellcheck source=/dev/null
source "$ROOT_DIR/lib/palette.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/boot_splash.sh"

# ── Helpers ────────────────────────────────────────────────

assert_eq() {
  local got=$1 want=$2 msg=$3
  if [[ "$got" != "$want" ]]; then
    printf 'FAIL assert_eq: %s\n  got:  %q\n  want: %q\n' "$msg" "$got" "$want" >&2
    exit 1
  fi
}

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

printf 'boot_splash: animation generates frames in truecolor... '
FRAMES=$(OCA_FORCE_TTY=1 TERM=xterm-256color COLORTERM=truecolor oca_animate_splash_frames 2>&1)
# Should contain multiple color sequences (one per frame)
FRAME_COUNT=$(printf '%s' "$FRAMES" | grep -c $'\033\[38;2;' || true)
if (( FRAME_COUNT < 8 )); then
  printf 'FAIL: expected at least 8 frames with truecolor sequences, got %d\n' "$FRAME_COUNT" >&2
  exit 1
fi
printf 'OK (%d frames)\n' "$FRAME_COUNT"

printf 'boot_splash: animation uses correct color range... '
# First frame should use INDIGO (108,122,184)
assert_contains "$FRAMES" $'\033[38;2;108;122;184m' "first frame should use INDIGO color"
# Last frame should use INDIGO_BRIGHT (139,159,224)
assert_contains "$FRAMES" $'\033[38;2;139;159;224m' "last frame should use INDIGO_BRIGHT color"
printf 'OK\n'

printf 'boot_splash: 256 mode shows colored wordmark... '
OUTPUT=$(OCA_FORCE_TTY=1 TERM=xterm-256color COLORTERM= oca_boot_splash 2>&1)
# Should contain color sequences (256-color mode)
assert_contains "$OUTPUT" $'\033[38;5;' "256 mode should use 256-color sequences"
# Output should be non-empty
[[ -n "$OUTPUT" ]] || { printf 'FAIL: 256 mode output is empty\n' >&2; exit 1; }
printf 'OK\n'

printf 'boot_splash: mono mode is plain text... '
OUTPUT=$(OCA_FORCE_TTY=1 NO_COLOR=1 oca_boot_splash 2>&1)
# Should contain wordmark content without escape sequences
[[ -n "$OUTPUT" ]] || { printf 'FAIL: mono mode output is empty\n' >&2; exit 1; }
assert_not_contains "$OUTPUT" $'\033[' "mono mode should have no escape sequences"
printf 'OK\n'

printf 'boot_splash: OCA_BOOT_SPLASH=0 disables splash... '
OUTPUT=$(OCA_BOOT_SPLASH=0 oca_boot_splash 2>&1)
assert_eq "$OUTPUT" "" "OCA_BOOT_SPLASH=0 should produce no output"
printf 'OK\n'

printf '\nAll boot_splash tests passed.\n'
