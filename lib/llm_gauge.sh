#!/usr/bin/env bash
# llm_gauge.sh — LLM provider fuel gauge renderer.
#
# Reads provider quota files from $OCA_CACHE_DIR/{zai,copilot,claude,codex}
# and renders ASCII fuel gauge bars. Designed for tmux status bar integration.

# shellcheck source=./palette.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/palette.sh"

# ── Constants ──────────────────────────────────────────────

OCA_LLM_PROVIDERS="zai copilot claude codex"
OCA_LLM_GAUGE_FILLED="█"
OCA_LLM_GAUGE_EMPTY="░"
OCA_LLM_GAUGE_WIDTH=5

# ── Color Helpers ──────────────────────────────────────────

_oca_llm_color_for_value() {
  local value=$1
  local mode
  mode=$(oca_detect_color_mode)

  if [[ "$mode" == "mono" ]]; then
    printf ''
    return
  fi

  local color_name
  if (( value > 80 )); then
    color_name="success"
  elif (( value >= 20 )); then
    color_name="indigo"
  elif (( value > 0 )); then
    color_name="warning"
  else
    color_name="muted"
  fi

  # Map to actual palette colors
  local r g b
  case "$color_name" in
    success) r=122; g=155; b=122 ;;  # #7A9B7A
    indigo)  r=108; g=122; b=184 ;;  # #6C7AB8
    warning) r=212; g=168; b=67  ;;  # #D4A843
    muted)   r=168; g=166; b=163 ;;  # #A8A6A3
  esac

  if [[ "$mode" == "truecolor" ]]; then
    printf '\033[38;2;%d;%d;%dm' "$r" "$g" "$b"
  else
    # 256-color approximations
    local ansi
    case "$color_name" in
      success) ansi=108 ;;
      indigo)  ansi=103 ;;
      warning) ansi=178 ;;
      muted)   ansi=248 ;;
    esac
    printf '\033[38;5;%dm' "$ansi"
  fi
}

# ── Gauge Rendering ────────────────────────────────────────

# oca_llm_render_gauge — render a gauge bar for a numeric value (0-100).
# Output: N filled chars + (5-N) empty chars, optionally colorized.
oca_llm_render_gauge() {
  local value=${1:-0}

  # Validate: must be integer 0-100
  if ! [[ "$value" =~ ^[0-9]+$ ]]; then
    value=0
  fi
  if (( value > 100 )); then
    value=100
  fi

  # Map value to 0-5 blocks using (value + 9) / 20 (rounds up)
  local filled=$(( (value + 9) / 20 ))
  if (( filled > OCA_LLM_GAUGE_WIDTH )); then
    filled=$OCA_LLM_GAUGE_WIDTH
  fi
  local empty=$(( OCA_LLM_GAUGE_WIDTH - filled ))

  local color_seq
  color_seq=$(_oca_llm_color_for_value "$value")

  local result=""
  local i
  for (( i = 0; i < filled; i++ )); do
    result+="$OCA_LLM_GAUGE_FILLED"
  done
  for (( i = 0; i < empty; i++ )); do
    result+="$OCA_LLM_GAUGE_EMPTY"
  done

  if [[ -n "$color_seq" ]]; then
    printf '%s%s%s' "$color_seq" "$result" "$OCA_COLOR_RESET"
  else
    printf '%s' "$result"
  fi
}

# ── File Reading ───────────────────────────────────────────

_oca_llm_read_value() {
  local provider=$1
  local cache_dir="${OCA_CACHE_DIR:-${XDG_RUNTIME_DIR:-/tmp}/opencode-advance}"
  local file="$cache_dir/$provider"

  if [[ ! -f "$file" ]]; then
    printf ''
    return 1
  fi

  local content
  content=$(tr -d '[:space:]' < "$file" 2>/dev/null)

  if [[ -z "$content" ]]; then
    printf ''
    return 1
  fi

  printf '%s' "$content"
}

# ── Public API ─────────────────────────────────────────────

# oca_llm_render_provider — render gauge for a single provider.
# Usage: oca_llm_render_provider <provider_name>
# Output: <short_name><gauge_bar>
oca_llm_render_provider() {
  local provider=$1
  local short_name

  # Short names for compact display
  case "$provider" in
    zai)     short_name="za" ;;
    copilot) short_name="cop" ;;
    claude)  short_name="cla" ;;
    codex)   short_name="cod" ;;
    *)       short_name="${provider:0:3}" ;;
  esac

  local value
  if value=$(_oca_llm_read_value "$provider"); then
    local gauge
    gauge=$(oca_llm_render_gauge "$value")
    printf '%s%s' "$short_name" "$gauge"
  else
    local gauge
    gauge=$(oca_llm_render_gauge 0)
    printf '%s%s' "$short_name" "$gauge"
  fi
}

# oca_llm_render_all — render all provider gauges as a single string.
# Output: space-separated provider gauges: "za█████ cop██░░░ cla█░░░░ cod░░░░░"
oca_llm_render_all() {
  local result=""
  local first=1
  local provider
  for provider in $OCA_LLM_PROVIDERS; do
    if (( first )); then
      first=0
    else
      result+=" "
    fi
    result+=$(oca_llm_render_provider "$provider")
  done
  printf '%s' "$result"
}

# ── CLI Entry Point ────────────────────────────────────────

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-}" in
    all)
      oca_llm_render_all
      ;;
    provider)
      shift
      oca_llm_render_provider "$@"
      ;;
    gauge)
      shift
      oca_llm_render_gauge "$@"
      ;;
    *)
      printf 'usage: %s {all|provider <name>|gauge <value>}\n' "$(basename "$0")" >&2
      exit 2
      ;;
  esac
fi
