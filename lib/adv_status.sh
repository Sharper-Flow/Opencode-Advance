#!/usr/bin/env bash
# adv_status.sh — Parse ADV external state from filesystem for status bar.
#
# Operates OUTSIDE the OpenCode agent/MCP context (invoked by tmux #()).
# Exempt from ADV_INSTRUCTIONS.md direct-read prohibition — MCP tools
# are unavailable in this context.
#
# Requires: jq for JSON parsing. Graceful degradation when missing.

# ── ADV State Directory Resolution ────────────────────────

_oca_adv_data_root() {
  local xdg="${XDG_DATA_HOME:-$HOME/.local/share}"
  printf '%s/opencode/plugins/advance\n' "$xdg"
}

_oca_adv_changes_dir() {
  local project_id="$1"
  printf '%s/%s/changes\n' "$(_oca_adv_data_root)" "$project_id"
}

# ── Cache ──────────────────────────────────────────────────

_oca_adv_cache_file() {
  local cache_dir="${OCA_CACHE_DIR:-${XDG_RUNTIME_DIR:-/tmp}/opencode-advance}"
  printf '%s/adv_status\n' "$cache_dir"
}

_oca_adv_cache_read() {
  local cache_file
  cache_file=$(_oca_adv_cache_file)
  if [[ ! -f "$cache_file" ]]; then
    return 1
  fi

  # Check TTL (10 seconds, aligned with tmux status-interval)
  local now
  now=$(date +%s)
  local cache_mtime
  cache_mtime=$(stat -c %Y "$cache_file" 2>/dev/null || echo 0)
  local age=$(( now - cache_mtime ))
  if (( age > 10 )); then
    return 1
  fi

  cat "$cache_file"
}

_oca_adv_cache_write() {
  local content="$1"
  local cache_file
  cache_file=$(_oca_adv_cache_file)
  local cache_dir
  cache_dir=$(dirname "$cache_file")
  mkdir -p "$cache_dir"
  printf '%s' "$content" > "$cache_file"
}

# ── jq Availability ────────────────────────────────────────

_oca_adv_has_jq() {
  command -v jq >/dev/null 2>&1
}

# ── Public API ─────────────────────────────────────────────

# oca_adv_find_active_changes — List non-archived change IDs for a project.
# Usage: oca_adv_find_active_changes <project_changes_dir>
# Output: one change ID per line
oca_adv_find_active_changes() {
  local changes_dir="$1"

  if [[ ! -d "$changes_dir" ]]; then
    return 0
  fi

  if ! _oca_adv_has_jq; then
    return 0
  fi

  local change_dir change_json ch_status
  for change_dir in "$changes_dir"/*/; do
    [[ -d "$change_dir" ]] || continue
    change_json="$change_dir/change.json"
    [[ -f "$change_json" ]] || continue

    ch_status=$(jq -r '.status // "unknown"' "$change_json" 2>/dev/null)
    if [[ "$ch_status" != "archived" && "$ch_status" != "closed" ]]; then
      basename "$change_dir"
    fi
  done
}

# oca_adv_change_summary — Parse a single change and return formatted summary.
# Usage: oca_adv_change_summary <change_dir>
# Output: {shortId}:{current_gate} or {shortId}:✓
oca_adv_change_summary() {
  local change_dir="$1"
  local change_json="$change_dir/change.json"

  if [[ ! -f "$change_json" ]]; then
    printf ''
    return 0
  fi

  if ! _oca_adv_has_jq; then
    printf 'adv:?'
    return 0
  fi

  # Try cache first
  local cache_key="summary:$(basename "$change_dir")"
  local cached
  if cached=$(_oca_adv_cache_read 2>/dev/null); then
    local cached_line
    cached_line=$(printf '%s' "$cached" | grep "^$cache_key:" 2>/dev/null || true)
    if [[ -n "$cached_line" ]]; then
      printf '%s' "${cached_line#*:}"
      return 0
    fi
  fi

  # Parse change.json
  local change_id current_gate gates_complete gates_total

  change_id=$(jq -r '.id // "unknown"' "$change_json" 2>/dev/null)

  # Find first non-done gate
  current_gate=$(jq -r '
    [.gates | to_entries[] | select(.value.status != "done") | .key][0] // "done"
  ' "$change_json" 2>/dev/null)

  if [[ "$current_gate" == "done" ]]; then
    current_gate="✓"
  fi

  # Shorten change ID (truncate to ~12 chars if long)
  local short_id="$change_id"
  if (( ${#change_id} > 15 )); then
    short_id="${change_id:0:12}..."
  fi

  local result="${short_id}:${current_gate}"

  # Write to cache (append to existing cache)
  local cache_content
  if cache_content=$(_oca_adv_cache_read 2>/dev/null); then
    # Replace existing entry or append
    local new_content
    new_content=$(printf '%s\n' "$cache_content" | grep -v "^$cache_key:" 2>/dev/null || true)
    _oca_adv_cache_write "${new_content}${cache_key}:${result}"$'\n'
  else
    _oca_adv_cache_write "${cache_key}:${result}"$'\n'
  fi

  printf '%s' "$result"
}

# oca_adv_active_summary — Convenience: find first active change and summarize.
# Usage: oca_adv_active_summary [project_changes_dir]
# Scans all ADV projects if no dir given.
oca_adv_active_summary() {
  local changes_dir="$1"

  if [[ -n "$changes_dir" ]]; then
    local change_ids
    change_ids=$(oca_adv_find_active_changes "$changes_dir")
    if [[ -n "$change_ids" ]]; then
      local first_id
      first_id=$(printf '%s' "$change_ids" | head -1)
      oca_adv_change_summary "$changes_dir/$first_id"
      return 0
    fi
  fi

  # Scan all projects
  local adv_root
  adv_root=$(_oca_adv_data_root)
  if [[ ! -d "$adv_root" ]]; then
    printf ''
    return 0
  fi

  local project_dir
  for project_dir in "$adv_root"/*/; do
    [[ -d "$project_dir" ]] || continue
    local project_changes="$project_dir/changes"
    [[ -d "$project_changes" ]] || continue

    local change_ids
    change_ids=$(oca_adv_find_active_changes "$project_changes")
    if [[ -n "$change_ids" ]]; then
      local first_id
      first_id=$(printf '%s' "$change_ids" | head -1)
      oca_adv_change_summary "$project_changes/$first_id"
      return 0
    fi
  done

  printf ''
}

# ── Temporal Health ────────────────────────────────────────

# oca_adv_temporal_health — Probe Temporal server reachability for status bar.
# Reads $OCA_CACHE_DIR/temporal.env for address, probes with bash /dev/tcp.
# Returns: "T:✓" on success, "T:✗" if unreachable AND Advance state dir exists,
#          empty string otherwise (no temporal.env or no ADV state dir).
# Caches result for 10s.
oca_adv_temporal_health() {
  local cache_dir="${OCA_CACHE_DIR:-${XDG_RUNTIME_DIR:-/tmp}/opencode-advance}"
  local env_file="$cache_dir/temporal.env"
  local cache_file="$cache_dir/temporal_health"

  # Need temporal.env to know what to probe
  if [[ ! -f "$env_file" ]]; then
    return 0
  fi

  # Need ADV state dir to care about Temporal
  local xdg="${XDG_DATA_HOME:-$HOME/.local/share}"
  if [[ ! -d "$xdg/opencode/plugins/advance" ]]; then
    return 0
  fi

  # Parse address from temporal.env
  local addr
  addr=$(grep '^ADV_TEMPORAL_ADDRESS=' "$env_file" 2>/dev/null | cut -d= -f2- | tr -d '[:space:]')
  if [[ -z "$addr" ]]; then
    return 0
  fi

  # Extract host and port
  local host port
  if [[ "$addr" == *:* ]]; then
    host="${addr%:*}"
    port="${addr##*:}"
  else
    host="$addr"
    port="7233"
  fi
  # Handle IPv6 brackets
  if [[ "$host" == \[* ]]; then
    host="${host#[}"
    host="${host%]}"
  fi

  # Check cache (10s TTL)
  if [[ -f "$cache_file" ]]; then
    local now cache_mtime age
    now=$(date +%s)
    cache_mtime=$(stat -c %Y "$cache_file" 2>/dev/null || echo 0)
    age=$(( now - cache_mtime ))
    if (( age <= 10 )); then
      cat "$cache_file"
      return 0
    fi
  fi

  # Validate host/port before expansion (prevent injection)
  if [[ ! "$host" =~ ^[0-9a-fA-F.:]+$ ]] || [[ ! "$port" =~ ^[0-9]+$ ]]; then
    printf ''
    return 0
  fi

  # Probe
  local result
  if timeout 1 bash -c "cat < /dev/tcp/${host}/${port}" >/dev/null 2>&1; then
    result="T:✓"
  else
    result="T:✗"
  fi

  # Write cache
  mkdir -p "$cache_dir"
  printf '%s' "$result" > "$cache_file"

  printf '%s' "$result"
}

# ── CLI Entry Point ────────────────────────────────────────

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-}" in
    summary)
      shift
      oca_adv_active_summary "$@"
      ;;
    find)
      shift
      oca_adv_find_active_changes "$@"
      ;;
    parse)
      shift
      oca_adv_change_summary "$@"
      ;;
    *)
      printf 'usage: %s {summary|find|parse} [args]\n' "$(basename "$0")" >&2
      exit 2
      ;;
  esac
fi
