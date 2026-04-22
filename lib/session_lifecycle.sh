#!/usr/bin/env bash
# session_lifecycle.sh — OCA tmux session primitives.
# Called by the Go CLI (cmd/oca/session.go) via subprocess.Run or tmux send-keys.
#
# Exit codes: 0 success, 1 tmux error, 2 invalid args

# shellcheck source=./palette.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/palette.sh"
# shellcheck source=./boot_splash.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/boot_splash.sh"

oca_session_socket() {
  printf '%s\n' "${OCA_TMUX_SOCKET:-oca}"
}

# session_new — create a detached tmux session.
# Usage: session_new <name> <working_dir> [tmux_conf_path]
oca_session_new() {
  local name="$1"
  local workdir="$2"
  local conf="${3:-}"

  if [[ -z "$name" || -z "$workdir" ]]; then
    printf 'usage: session_new <name> <working_dir> [tmux_conf]\n' >&2
    return 2
  fi

  if [[ ! -d "$workdir" ]]; then
    printf 'session_new: working directory %s does not exist\n' "$workdir" >&2
    return 2
  fi

  local socket
  socket=$(oca_session_socket)
  local args=(-L "$socket")

  if [[ -n "$conf" && -f "$conf" ]]; then
    args+=(-f "$conf")
  fi

  args+=(new-session -d -s "$name" -c "$workdir")

  tmux "${args[@]}" || {
    printf 'session_new: tmux failed (exit %d)\n' "$?" >&2
    return 1
  }

  return 0
}

# session_list — list OCA-managed tmux sessions.
# Output: <name>\t<attached> per line (filtered by oca- prefix).
oca_session_list() {
  local socket
  socket=$(oca_session_socket)

  local output
  output=$(tmux -L "$socket" list-sessions -F '#{session_name}	#{session_attached}' 2>/dev/null) || {
    # No server or no sessions — return empty
    return 0
  }

  while IFS= read -r line; do
    local name
    name="${line%%	*}"
    if [[ "$name" == oca-* ]]; then
      printf '%s\n' "$line"
    fi
  done <<< "$output"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-}" in
    new)
      shift
      oca_session_new "$@"
      ;;
    list)
      shift
      oca_session_list
      ;;
    *)
      printf 'usage: %s {new|list}\n' "$(basename "$0")" >&2
      exit 2
      ;;
  esac
fi
