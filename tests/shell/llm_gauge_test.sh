#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

# shellcheck source=/dev/null
source "$ROOT_DIR/lib/palette.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/llm_gauge.sh"

# ── Helpers ────────────────────────────────────────────────

assert_eq() {
  local got=$1 want=$2 msg=$3
  if [[ "$got" != "$want" ]]; then
    printf 'FAIL assert_eq: %s\n  got:  %q\n  want: %q\n' "$msg" "$got" "$want" >&2
    exit 1
  fi
}

assert_contains() {
  local haystack=$1 needle=$2 msg=$3
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'FAIL assert_contains: %s\n  missing %q\n' "$msg" "$needle" >&2
    exit 1
  fi
}

# ── Setup ──────────────────────────────────────────────────

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

export OCA_CACHE_DIR="$WORK_DIR/cache"
mkdir -p "$OCA_CACHE_DIR"

# ── Tests ──────────────────────────────────────────────────

printf 'llm_gauge: single provider at 100%%... '
printf '100' > "$OCA_CACHE_DIR/zai"
RESULT=$(oca_llm_render_provider zai)
# 100% = 5 filled blocks
assert_contains "$RESULT" '█████' "100%% should show 5 filled blocks"
printf 'OK\n'

printf 'llm_gauge: single provider at 50%%... '
printf '50' > "$OCA_CACHE_DIR/copilot"
RESULT=$(oca_llm_render_provider copilot)
# 50% = round(50/20) = 3 filled blocks (using (value+9)/20)
assert_contains "$RESULT" '█' "50%% should show some filled blocks"
assert_contains "$RESULT" '░' "50%% should show some empty blocks"
printf 'OK\n'

printf 'llm_gauge: single provider at 0%%... '
printf '0' > "$OCA_CACHE_DIR/claude"
RESULT=$(oca_llm_render_provider claude)
assert_contains "$RESULT" '░░░░░' "0%% should show 5 empty blocks"
printf 'OK\n'

printf 'llm_gauge: missing provider file... '
RESULT=$(oca_llm_render_provider codex)
assert_contains "$RESULT" '░░░░░' "missing file should show empty gauge"
printf 'OK\n'

printf 'llm_gauge: empty file... '
: > "$OCA_CACHE_DIR/codex"
RESULT=$(oca_llm_render_provider codex)
assert_contains "$RESULT" '░░░░░' "empty file should show empty gauge"
printf 'OK\n'

printf 'llm_gauge: render all providers... '
printf '80' > "$OCA_CACHE_DIR/zai"
printf '60' > "$OCA_CACHE_DIR/copilot"
printf '40' > "$OCA_CACHE_DIR/claude"
printf '10' > "$OCA_CACHE_DIR/codex"
RESULT=$(oca_llm_render_all)
assert_contains "$RESULT" 'za' "render_all should include za prefix"
assert_contains "$RESULT" 'cop' "render_all should include cop prefix"
assert_contains "$RESULT" 'cla' "render_all should include cla prefix"
assert_contains "$RESULT" 'cod' "render_all should include cod prefix"
printf 'OK\n'

printf 'llm_gauge: gauge is 5 chars wide... '
printf '60' > "$OCA_CACHE_DIR/zai"
RESULT=$(oca_llm_render_gauge 60)
GAUGE_LEN=${#RESULT}
assert_eq "$GAUGE_LEN" "5" "gauge should be exactly 5 characters"
printf 'OK\n'

printf 'llm_gauge: color coding in truecolor mode... '
printf '85' > "$OCA_CACHE_DIR/zai"
OCA_FORCE_TTY=1 NO_COLOR= TERM=xterm-256color COLORTERM=truecolor \
  RESULT=$(oca_llm_render_provider zai)
# >80% should have green color sequence
assert_contains "$RESULT" $'\033[38;2;122;155;122m' "85%% should use green color (#7A9B7A)"
printf 'OK\n'

printf '\nAll llm_gauge tests passed.\n'
