#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

# shellcheck source=/dev/null
source "$ROOT_DIR/lib/palette.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/wordmark.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/boot_splash.sh"

assert_eq() {
  local got=$1
  local want=$2
  local msg=$3
  if [[ "$got" != "$want" ]]; then
    printf 'assert_eq failed: %s\n  got:  %q\n  want: %q\n' "$msg" "$got" "$want" >&2
    exit 1
  fi
}

assert_contains() {
  local haystack=$1
  local needle=$2
  local msg=$3
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'assert_contains failed: %s\nmissing %q in %q\n' "$msg" "$needle" "$haystack" >&2
    exit 1
  fi
}

mode=$(NO_COLOR=1 OCA_FORCE_TTY=1 TERM=xterm-256color COLORTERM=truecolor oca_detect_color_mode)
assert_eq "$mode" "mono" "NO_COLOR should force mono mode"

mode=$(OCA_FORCE_TTY=1 TERM=xterm-256color COLORTERM=truecolor oca_detect_color_mode)
assert_eq "$mode" "truecolor" "truecolor tty should use truecolor mode"

mode=$(OCA_FORCE_TTY=1 TERM=xterm-256color COLORTERM= oca_detect_color_mode)
assert_eq "$mode" "256" "256-color tty should use 256 mode"

medium=$(OCA_FORCE_TTY=0 NO_COLOR=1 oca_render_wordmark medium)
assert_eq "$medium" "OpenCode *ADVANCE*" "mono medium wordmark should use explicit ADVANCE marker"

full=$(OCA_FORCE_TTY=1 TERM=xterm-256color COLORTERM= oca_render_wordmark full)
assert_contains "$full" $'\033[38;5;255m' "full render should colorize OpenCode half"
assert_contains "$full" $'\033[38;5;103m' "full render should colorize ADVANCE half"

assert_eq "$OCA_TMUX_IVORY" "#E8E6E3" "palette should load tmux ivory from shared asset"
assert_contains "$OCA_COLOR_INDIGO" $'\033[38;2;108;122;184m' "palette should derive truecolor indigo from shared asset"

if split_output=$(oca_split_wordmark_line "malformed" 2>&1); then
  printf 'expected malformed line helper path to fail\n' >&2
  exit 1
else
  assert_contains "$split_output" "malformed wordmark line" "malformed full wordmark lines should fail loudly"
fi

disabled=$(OCA_FORCE_TTY=1 OCA_BOOT_SPLASH=0 oca_boot_splash)
assert_eq "$disabled" "" "boot splash should be silent when disabled"

non_interactive=$(OCA_FORCE_TTY=0 oca_boot_splash)
assert_eq "$non_interactive" "" "boot splash should be silent when stdout is not interactive"

echo "brand helper tests passed"
