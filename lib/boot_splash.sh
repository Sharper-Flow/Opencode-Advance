#!/usr/bin/env bash

# shellcheck source=./wordmark.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/wordmark.sh"

oca_boot_splash() {
  if [[ "${OCA_BOOT_SPLASH:-1}" == "0" ]]; then
    return 0
  fi

  if ! oca_is_tty; then
    return 0
  fi

  oca_render_wordmark full
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
