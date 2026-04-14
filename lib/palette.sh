#!/usr/bin/env bash

oca_repo_root() {
  local script_dir
  script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
  cd "$script_dir/.." && pwd
}

oca_brand_assets_dir() {
  printf '%s/brand/assets\n' "$(oca_repo_root)"
}

oca_is_tty() {
  if [[ -n "${OCA_FORCE_TTY:-}" ]]; then
    [[ "${OCA_FORCE_TTY}" != "0" ]]
    return
  fi
  [[ -t 1 ]]
}

oca_detect_color_mode() {
  if [[ -n "${NO_COLOR:-}" ]] || ! oca_is_tty; then
    printf 'mono\n'
    return
  fi

  local color_term=${COLORTERM:-}
  local term=${TERM:-}
  if [[ "$color_term" == *truecolor* ]] || [[ "$color_term" == *24bit* ]]; then
    printf 'truecolor\n'
    return
  fi

  if [[ "$term" == *256color* ]]; then
    printf '256\n'
    return
  fi

  printf 'mono\n'
}

_oca_load_palette() {
  local palette_file
  palette_file="$(oca_brand_assets_dir)/palette.env"

  while IFS='=,' read -r name hex ansi r g b; do
    [[ -z "$name" ]] && continue
    [[ -z "$hex" || -z "$ansi" || -z "$r" || -z "$g" || -z "$b" ]] && continue
    case "$name" in
      IVORY)
        export OCA_TMUX_IVORY="$hex"
        export OCA_COLOR_IVORY="$(printf '\033[38;2;%s;%s;%sm' "$r" "$g" "$b")"
        export OCA_COLOR256_IVORY="$(printf '\033[38;5;%sm' "$ansi")"
        ;;
      INDIGO)
        export OCA_TMUX_INDIGO="$hex"
        export OCA_COLOR_INDIGO="$(printf '\033[38;2;%s;%s;%sm' "$r" "$g" "$b")"
        export OCA_COLOR256_INDIGO="$(printf '\033[38;5;%sm' "$ansi")"
        ;;
      INDIGO_BRIGHT)
        export OCA_TMUX_INDIGO_BRIGHT="$hex"
        export OCA_COLOR_INDIGO_BRIGHT="$(printf '\033[38;2;%s;%s;%sm' "$r" "$g" "$b")"
        export OCA_COLOR256_INDIGO_BRIGHT="$(printf '\033[38;5;%sm' "$ansi")"
        ;;
      INDIGO_GLOW)
        export OCA_TMUX_INDIGO_GLOW="$hex"
        export OCA_COLOR_INDIGO_GLOW="$(printf '\033[38;2;%s;%s;%sm' "$r" "$g" "$b")"
        export OCA_COLOR256_INDIGO_GLOW="$(printf '\033[38;5;%sm' "$ansi")"
        ;;
    esac
  done < "$palette_file"

  if [[ -z "${OCA_TMUX_IVORY:-}" || -z "${OCA_TMUX_INDIGO:-}" || -z "${OCA_TMUX_INDIGO_BRIGHT:-}" || -z "${OCA_TMUX_INDIGO_GLOW:-}" ]]; then
    printf 'missing required palette entries in %s\n' "$palette_file" >&2
    return 1
  fi

  export OCA_COLOR_RESET=$'\033[0m'
}

oca_color_sequence() {
  local color_name=$1
  local mode=${2:-$(oca_detect_color_mode)}

  case "$mode:$color_name" in
    truecolor:IVORY) printf '%s' "$OCA_COLOR_IVORY" ;;
    truecolor:INDIGO) printf '%s' "$OCA_COLOR_INDIGO" ;;
    truecolor:INDIGO_BRIGHT) printf '%s' "$OCA_COLOR_INDIGO_BRIGHT" ;;
    truecolor:INDIGO_GLOW) printf '%s' "$OCA_COLOR_INDIGO_GLOW" ;;
    256:IVORY) printf '%s' "$OCA_COLOR256_IVORY" ;;
    256:INDIGO) printf '%s' "$OCA_COLOR256_INDIGO" ;;
    256:INDIGO_BRIGHT) printf '%s' "$OCA_COLOR256_INDIGO_BRIGHT" ;;
    256:INDIGO_GLOW) printf '%s' "$OCA_COLOR256_INDIGO_GLOW" ;;
    *) printf '' ;;
  esac
}

_oca_load_palette
