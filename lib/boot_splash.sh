#!/usr/bin/env bash

# shellcheck source=./wordmark.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/wordmark.sh"

# ── Animation ──────────────────────────────────────────────

# oca_animate_splash_frames — generate animation frame content (for testing).
# Outputs each frame's color sequence + wordmark line, separated by newlines.
oca_animate_splash_frames() {
  local mode
  mode=$(oca_detect_color_mode)

  if [[ "$mode" != "truecolor" ]]; then
    return 0
  fi

  # Precompute RGB values for 8 frames interpolating INDIGO → INDIGO_BRIGHT
  # INDIGO:      108, 122, 184
  # INDIGO_BRIGHT: 139, 159, 224
  local i r g b
  for (( i = 0; i < 8; i++ )); do
    r=$(( 108 + (139 - 108) * i / 7 ))
    g=$(( 122 + (159 - 122) * i / 7 ))
    b=$(( 184 + (224 - 184) * i / 7 ))
    printf '\033[38;2;%d;%d;%dm' "$r" "$g" "$b"
    oca_render_wordmark full
    printf '\n'
  done

  # And back: INDIGO_BRIGHT → INDIGO
  for (( i = 6; i >= 0; i-- )); do
    r=$(( 108 + (139 - 108) * i / 7 ))
    g=$(( 122 + (159 - 122) * i / 7 ))
    b=$(( 184 + (224 - 184) * i / 7 ))
    printf '\033[38;2;%d;%d;%dm' "$r" "$g" "$b"
    oca_render_wordmark full
    printf '\n'
  done

  # Final frame in INDIGO_BRIGHT
  printf '\033[38;2;139;159;224m'
  oca_render_wordmark full
  printf '\n'
}

# oca_animate_splash — run the full boot splash animation.
oca_animate_splash() {
  local mode
  mode=$(oca_detect_color_mode)

  case "$mode" in
    mono)
      # No animation in mono — static output only
      oca_render_wordmark full
      ;;
    256)
      # Static colored wordmark in 256-color mode
      oca_render_wordmark full
      ;;
    truecolor)
      # Full animation
      local i r g b
      for (( i = 0; i < 8; i++ )); do
        r=$(( 108 + (139 - 108) * i / 7 ))
        g=$(( 122 + (159 - 122) * i / 7 ))
        b=$(( 184 + (224 - 184) * i / 7 ))
        printf '\033[2J\033[H'  # clear screen
        printf '\033[38;2;%d;%d;%dm' "$r" "$g" "$b"
        oca_render_wordmark full
        sleep 0.1
      done
      for (( i = 6; i >= 0; i-- )); do
        r=$(( 108 + (139 - 108) * i / 7 ))
        g=$(( 122 + (159 - 122) * i / 7 ))
        b=$(( 184 + (224 - 184) * i / 7 ))
        printf '\033[2J\033[H'
        printf '\033[38;2;%d;%d;%dm' "$r" "$g" "$b"
        oca_render_wordmark full
        sleep 0.1
      done
      printf '\033[2J\033[H'
      printf '\033[38;2;139;159;224m'
      oca_render_wordmark full
      printf '%s\n' "$OCA_COLOR_RESET"
      ;;
  esac
}

oca_boot_splash() {
  if [[ "${OCA_BOOT_SPLASH:-1}" == "0" ]]; then
    return 0
  fi

  if ! oca_is_tty; then
    return 0
  fi

  oca_animate_splash
  if [[ -n "${OCA_SPLASH_VERSION:-}" ]]; then
    if [[ -n "${OCA_SPLASH_DIR:-}" ]]; then
      printf '\nv%s · %s\n' "$OCA_SPLASH_VERSION" "$OCA_SPLASH_DIR"
    else
      printf '\nversion: %s\n' "$OCA_SPLASH_VERSION"
    fi
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  oca_boot_splash
fi
