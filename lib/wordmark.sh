#!/usr/bin/env bash

# shellcheck source=./palette.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/palette.sh"

oca_wordmark_asset() {
  local variant=$1
  printf '%s/%s.txt\n' "$(oca_brand_assets_dir)" "$variant"
}

oca_colorize() {
  local color_name=$1
  local text=$2
  local mode=${3:-$(oca_detect_color_mode)}
  local seq
  seq=$(oca_color_sequence "$color_name" "$mode")
  if [[ -z "$seq" ]]; then
    printf '%s' "$text"
    return
  fi
  printf '%s%s%s' "$seq" "$text" "$OCA_COLOR_RESET"
}

oca_split_wordmark_line() {
  local line=$1
  if [[ "$line" != *"   "* ]]; then
    printf 'malformed wordmark line: %s\n' "$line" >&2
    return 1
  fi

  printf '%s\n%s\n' "${line%%   *}" "${line#*   }"
}

oca_render_wordmark() {
  local variant=${1:-full}
  local mode
  mode=$(oca_detect_color_mode)

  case "$variant" in
    short)
      if [[ "$mode" == "mono" ]]; then
        cat "$(oca_wordmark_asset short)"
      else
        oca_colorize INDIGO "$(cat "$(oca_wordmark_asset short)")" "$mode"
      fi
      ;;
    medium)
      if [[ "$mode" == "mono" ]]; then
        printf 'OpenCode *ADVANCE*'
      else
        printf '%s %s' \
          "$(oca_colorize IVORY 'OpenCode' "$mode")" \
          "$(oca_colorize INDIGO 'ADVANCE' "$mode")"
      fi
      ;;
    full|*)
      if [[ "$mode" == "mono" ]]; then
        cat "$(oca_wordmark_asset full)"
        return
      fi

      local line left right
      while IFS= read -r line; do
        mapfile -t parts < <(oca_split_wordmark_line "$line") || return 1
        left=${parts[0]}
        right=${parts[1]}
        printf '%s   %s\n' \
          "$(oca_colorize IVORY "$left" "$mode")" \
          "$(oca_colorize INDIGO "$right" "$mode")"
      done < "$(oca_wordmark_asset full)"
      ;;
  esac
}
