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

# ── tmux Helpers ───────────────────────────────────────────

# Extract socket name from $TMUX env (format: /tmp/tmux-UID/NAME,PID)
parseSocketFromTmux() {
  local tmux_env="${1:-}"
  if [[ -z "$tmux_env" ]]; then
    printf 'oca'
    return 0
  fi
  local path="${tmux_env%%,*}"
  local basename="${path##*/}"
  if [[ -z "$basename" || "$basename" == "." ]]; then
    printf 'oca'
  else
    printf '%s' "$basename"
  fi
}

# ── Window Glyph Renderer ───────────────────────────────────

# _oca_status_window_glyphs_from_list — Pure function: compact window glyph
# renderer. Takes tmux list-windows output, column budget, and optional project_id.
# Input format: "index:name:active" per line (active=1 for current window)
# Output: compact "N:glyph" pairs with tmux color escapes, truncated to budget.
# When project_id is given, enriches change/ windows with workspace status suffix.
_oca_status_window_glyphs_from_list() {
  local window_list="$1"
  local budget="${2:-160}"
  local project_id="${3:-}"

  # Build workspace status lookup if project_id given
  local ws_lookup=""
  if [[ -n "$project_id" && -n "$window_list" ]]; then
    local snapshot
    snapshot=$(_oca_adv_snapshot_read "$project_id" 2>/dev/null)
    if [[ -n "$snapshot" ]] && _oca_adv_has_jq; then
      ws_lookup=$(printf '%s' "$snapshot" | jq -r '
        .worktree_registry // {} | to_entries[] |
        "\(.value.changeId // (.value.branch | sub("^change/"; ""))):\(.value.status)"
      ' 2>/dev/null) || true
    fi
  fi

  local result=""
  local result_len=0
  local truncated=0

  while IFS=: read -r idx name active; do
    [[ -n "$idx" ]] || continue

    # Map window name to glyph
    local glyph=""
    local ws_suffix=""
    if [[ "$name" == "trunk" || "$name" == "main" ]]; then
      glyph="T"
    elif [[ "$name" == change/* ]]; then
      # Extract change ID, abbreviate to 4 chars
      local change_id="${name#change/}"
      glyph="${change_id:0:4}"
      # Check workspace status for this change
      if [[ -n "$ws_lookup" ]]; then
        local ws_status
        ws_status=$(printf '%s\n' "$ws_lookup" | grep "^${change_id}:" 2>/dev/null | cut -d: -f2) || true
        case "$ws_status" in
          setup_failed) ws_suffix="✗" ;;
          stale)        ws_suffix="ѻ" ;;
          merged)       ws_suffix="✓" ;;
          pending_delete) ws_suffix="␡" ;;
        esac
      fi
    else
      # Plain window name, truncate to 4 chars
      glyph="${name:0:4}"
    fi

    # Build segment: "N:glyph+suffix " (trailing space)
    local segment="${idx}:${glyph}${ws_suffix}"
    local seg_len=$(( ${#segment} + 1 ))  # +1 for separator space

    # Check budget
    local new_len=$(( result_len + seg_len ))
    if (( new_len > budget )); then
      truncated=1
      break
    fi

    # Color: indigo for active, muted for inactive
    local color_reset="$(_oca_status_reset)"
    if [[ "$active" == "1" ]]; then
      result="${result}$(_oca_status_color '#6C7AB8')${segment}${color_reset} "
    else
      result="${result}$(_oca_status_color '#A8A6A3')${segment}${color_reset} "
    fi
    result_len=$new_len
  done <<< "$window_list"

  # Append truncation marker if needed
  if (( truncated )); then
    result="${result}$(_oca_status_color '#A8A6A3')…$(_oca_status_reset)"
  fi

  printf '%s' "$result"
}

# oca_status_window_glyphs — Production wrapper: reads current tmux session
# windows and renders compact glyph display.
oca_status_window_glyphs() {
  local session_name="$1"
  local budget="${2:-120}"  # Default 120 to leave room for gauges/date

  if [[ -z "$TMUX" ]]; then
    return 0
  fi

  local socket
  socket=$(parseSocketFromTmux "$TMUX" 2>/dev/null || echo "oca")

  local window_list
  window_list=$(tmux -L "$socket" list-windows -t "$session_name" -F '#{window_index}:#{window_name}:#{window_active}' 2>/dev/null) || return 0

  _oca_status_window_glyphs_from_list "$window_list" "$budget"
}

# ── Row Builders ───────────────────────────────────────────

# Row 0: [session_name] │ [git branch] [⚡] │ [ADV state] │ [ws state] │ [host] [time]
oca_status_row0() {
  local session_name="$1"
  local pane_path="$2"
  local pane_id="${3:-}"

  _oca_discord_update

  # Session name (left)
  printf '%s' "$(_oca_status_color '#8B9FE0')"
  printf ' %s' "$session_name"
  printf '%s' "$(_oca_status_reset)"

  # Git branch + branch safety
  local branch
  branch=$(_oca_status_git_branch "$pane_path")
  if [[ -n "$branch" ]]; then
    printf ' %s' "$(_oca_status_color '#2D3138')│$(_oca_status_reset)"
    printf ' %s' "$(_oca_status_color '#A8A6A3')$branch$(_oca_status_reset)"

    # Branch safety indicator (⚡ when on default branch with active changes)
    local project_id
    project_id=$(_oca_status_resolve_project_id "$pane_path")
    local safety
    safety=$(oca_status_branch_safety "$pane_path" "$project_id" 2>/dev/null)
    if [[ -n "$safety" ]]; then
      printf ' %s' "$safety"
    fi
  fi

  # ADV state
  local adv_state
  adv_state=$(oca_adv_active_summary 2>/dev/null)
  if [[ -n "$adv_state" ]]; then
    printf ' %s' "$(_oca_status_color '#2D3138')│$(_oca_status_reset)"
    printf ' %s' "$(_oca_status_color '#6C7AB8')$adv_state$(_oca_status_reset)"
  fi

  # Workspace state indicator (setup_failed, stale, merged, etc.)
  local project_id
  project_id=$(_oca_status_resolve_project_id "$pane_path")
  local ws_state
  ws_state=$(oca_status_workspace_state "$project_id" 2>/dev/null)
  if [[ -n "$ws_state" ]]; then
    printf ' %s' "$(_oca_status_color '#2D3138')│$(_oca_status_reset)"
    printf ' %s' "$ws_state"
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

# Resolve project ID from pane path using git root commit SHA.
_oca_status_resolve_project_id() {
  local pane_path="$1"
  if [[ -z "$pane_path" || ! -d "$pane_path" ]]; then
    printf ''
    return 0
  fi
  if ! command -v git >/dev/null 2>&1; then
    printf ''
    return 0
  fi
  local sha
  sha=$(git -C "$pane_path" rev-list --max-parents=0 HEAD 2>/dev/null) || return 0
  if [[ -n "$sha" ]]; then
    printf '%s' "$sha"
  fi
}

# Row 1: window glyphs | LLM gauges | date
oca_status_row1() {
  local pane_path="$1"
  # Row 1 now delegates to row1-windows for unified rendering
  oca_status_row1_windows "" "$pane_path"
}

# Row 1 variant: compact window glyphs (Pattern B) | LLM gauges | date
oca_status_row1_windows() {
  local session_name="$1"
  local pane_path="$2"

  # Compact window glyphs (left side)
  if [[ -n "$session_name" && -n "$TMUX" ]]; then
    local glyphs
    glyphs=$(oca_status_window_glyphs "$session_name" 120)
    if [[ -n "$glyphs" ]]; then
      printf '%s' "$glyphs"
    fi
  fi

  # Right side
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
    row1-windows)
      oca_status_row1_windows "${2:-}" "${3:-}"
      ;;
    *)
      printf 'usage: %s {row0|row1} [args]\n' "$(basename "$0")" >&2
      exit 2
      ;;
  esac
fi
