#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

# shellcheck source=/dev/null
source "$ROOT_DIR/lib/session_lifecycle.sh"

# ── Helpers ────────────────────────────────────────────────

assert_eq() {
  local got=$1 want=$2 msg=$3
  if [[ "$got" != "$want" ]]; then
    printf 'FAIL assert_eq: %s\n  got:  %q\n  want: %q\n' "$msg" "$got" "$want" >&2
    exit 1
  fi
}

# ── Tests ──────────────────────────────────────────────────

printf 'session_lifecycle: kill with missing arg returns 2... '
(oca_session_kill "") >/dev/null 2>&1 && { printf 'FAIL: should exit 2\n' >&2; exit 1; }
printf 'OK\n'

printf 'session_lifecycle: killall handles no server... '
OCA_TMUX_SOCKET="oca-test-no-server-$$" OUTPUT=$(oca_session_killall 2>&1)
assert_eq "$OUTPUT" "" "killall with no server should return empty"
printf 'OK\n'

printf 'session_lifecycle: attach with missing arg returns 2... '
(oca_session_attach "") >/dev/null 2>&1 && { printf 'FAIL: should exit 2\n' >&2; exit 1; }
printf 'OK\n'

printf '\nAll session_lifecycle tests passed.\n'
