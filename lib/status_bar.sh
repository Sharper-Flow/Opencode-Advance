#!/usr/bin/env bash
# status_bar.sh — Live tmux status bar content generator.
#
# Called by tmux #() with format variable expansion.
# Usage: status_bar.sh {row0|row1} <session_name> <pane_path> [pane_id]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=/dev/null
source "$SCRIPT_DIR/palette.sh"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/adv_status.sh"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/llm_gauge.sh"

# ── Color Helpers ──────────────────────────────────────────

_oca_status_color() {
  local hex="$1"
  printf '#[fg=%s]' "$hex"
}

_oca_status_reset() {
  printf '#[default]'
}

# ── Data Sources ───────────────────────────────────────────

_oca_status_git_branch() {
  local pane_path="$1"
  if [[ -z "$pane_path" || ! -d "$pane_path" ]]; then
    return 0
  fi
  if ! command -v git >/dev/null 2>&1; then
    return 0
  fi
  local branch
  branch=$(git -C "$pane_path" rev-parse --abbrev-ref HEAD 2>/dev/null)
  if [[ -n "$branch" ]]; then
    printf '%s' "$branch"
  fi
}

_oca_status_host() {
  printf '%s' "${HOSTNAME:-$(hostname -s 2>/dev/null || echo '?')}"
}

_oca_status_clock() {
  printf '%s' "$(date '+%H:%M')"
}

_oca_status_date() {
  printf '%s' "$(date '+%Y-%m-%d')"
}

_oca_cache_dir() {
  if [[ -n "${OCA_CACHE_DIR:-}" ]]; then
    printf '%s' "$OCA_CACHE_DIR"
    return 0
  fi
  if [[ -n "${XDG_RUNTIME_DIR:-}" ]]; then
    printf '%s/opencode-advance' "$XDG_RUNTIME_DIR"
    return 0
  fi
  printf '%s/opencode-advance-%s' "${TMPDIR:-/tmp}" "${USER:-unknown}"
}

_oca_discord_update() {
  local wrapper
  wrapper="$(_oca_cache_dir)/discord/discord-update.sh"
  if [[ -x "$wrapper" ]]; then
    "$wrapper" >/dev/null 2>&1 &
  fi
}

# ── Row Builders ───────────────────────────────────────────

# Row 0: [session_name] │ [git branch] │ [ADV state] │ [host] [time]
oca_status_row0() {
  local session_name="$1"
  local pane_path="$2"
  local pane_id="${3:-}"

  _oca_discord_update

  # Session name (left)
  printf '%s' "$(_oca_status_color '#8B9FE0')"
  printf ' %s' "$session_name"
  printf '%s' "$(_oca_status_reset)"

  # Git branch
  local branch
  branch=$(_oca_status_git_branch "$pane_path")
  if [[ -n "$branch" ]]; then
    printf ' %s' "$(_oca_status_color '#2D3138')│$(_oca_status_reset)"
    printf ' %s' "$(_oca_status_color '#A8A6A3')$branch$(_oca_status_reset)"
  fi

  # ADV state
  local adv_state
  adv_state=$(oca_adv_active_summary 2>/dev/null)
  if [[ -n "$adv_state" ]]; then
    printf ' %s' "$(_oca_status_color '#2D3138')│$(_oca_status_reset)"
    printf ' %s' "$(_oca_status_color '#6C7AB8')$adv_state$(_oca_status_reset)"
  fi

  # Occupancy (compact status from oca occupancy --status --pane)
  local occupancy_seg
  occupancy_seg=""
  if [[ -n "$pane_id" ]]; then
    occupancy_seg=$(timeout 0.5s oca occupancy --status --pane "$pane_id" 2>/dev/null)
  fi
  if [[ -n "$occupancy_seg" ]]; then
    printf ' %s' "$(_oca_status_color '#2D3138')│$(_oca_status_reset)"
    # Color warning (multiple occupants) differently
    if [[ "$occupancy_seg" == *"⚠"* ]]; then
      printf ' %s' "$(_oca_status_color '#E5A649')$occupancy_seg$(_oca_status_reset)"
    else
      printf ' %s' "$(_oca_status_color '#A8A6A3')$occupancy_seg$(_oca_status_reset)"
    fi
  fi

  # Temporal health
  local temporal_health
  temporal_health=$(oca_adv_temporal_health 2>/dev/null)
  if [[ -n "$temporal_health" ]]; then
    printf ' %s' "$(_oca_status_color '#2D3138')│$(_oca_status_reset)"
    printf ' %s' "$(_oca_status_color '#A8A6A3')$temporal_health$(_oca_status_reset)"
  fi

  # Right side: host + clock
  printf '%s' "#[align=right]"
  printf ' %s' "$(_oca_status_color '#A8A6A3')$(_oca_status_host)$(_oca_status_reset)"
  printf ' %s' "$(_oca_status_color '#A8A6A3')$(_oca_status_clock)$(_oca_status_reset)"
}

# Row 1: [window list] │ [LLM gauges] │ [date]
oca_status_row1() {
  local pane_path="$1"

  # Window list is handled by tmux's built-in #{W:...} format
  # We just provide the right side content
  printf '%s' "#[align=right]"

  # LLM gauges
  local gauges
  gauges=$(oca_llm_render_all 2>/dev/null)
  if [[ -n "$gauges" ]]; then
    printf ' %s' "$(_oca_status_color '#A8A6A3')$gauges$(_oca_status_reset)"
  fi

  # Date
  printf ' %s' "$(_oca_status_color '#A8A6A3')$(_oca_status_date)$(_oca_status_reset)"
}

# ── Main Entry ─────────────────────────────────────────────

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-}" in
    row0)
      oca_status_row0 "${2:-}" "${3:-}" "${4:-}"
      ;;
    row1)
      oca_status_row1 "${2:-}"
      ;;
    *)
      printf 'usage: %s {row0|row1} [args]\n' "$(basename "$0")" >&2
      exit 2
      ;;
  esac
fi
