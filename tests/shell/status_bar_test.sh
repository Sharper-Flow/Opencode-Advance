#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

# shellcheck source=/dev/null
source "$ROOT_DIR/lib/palette.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/adv_status.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/llm_gauge.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/lib/status_bar.sh"

# ── Helpers ────────────────────────────────────────────────

assert_contains() {
  local haystack=$1 needle=$2 msg=$3
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'FAIL assert_contains: %s\n  missing %q\n' "$msg" "$needle" >&2
    exit 1
  fi
}

assert_not_contains() {
  local haystack=$1 needle=$2 msg=$3
  if [[ "$haystack" == *"$needle"* ]]; then
    printf 'FAIL assert_not_contains: %s\n  unexpected %q\n' "$msg" "$needle" >&2
    exit 1
  fi
}

# Strip tmux format escapes: #[...] sequences
_strip_tmux_escapes() {
  local input="$1"
  printf '%s' "$input" | sed 's/#\[[^]]*\]//g'
}

# ── Tests ──────────────────────────────────────────────────

printf 'status_bar: row0 includes session name... '
OUTPUT=$(oca_status_row0 "my-session" "/tmp")
assert_contains "$OUTPUT" "my-session" "row0 should contain session name"
printf 'OK\n'

printf 'status_bar: row0 includes git branch in repo... '
GIT_DIR=$(mktemp -d)
cd "$GIT_DIR" && git init && git config user.email "test@test" && git config user.name "Test" && git checkout -b my-branch >/dev/null 2>&1 || true && touch a && git add a && git commit -m "init" >/dev/null 2>&1 || true
OUTPUT=$(oca_status_row0 "test" "$GIT_DIR")
assert_contains "$OUTPUT" "my-branch" "row0 should contain git branch"
rm -rf "$GIT_DIR"
printf 'OK\n'

printf 'status_bar: row0 degrades outside git repo... '
OUTPUT=$(oca_status_row0 "test" "/tmp")
assert_not_contains "$OUTPUT" "master" "row0 should not show branch outside repo"
printf 'OK\n'

printf 'status_bar: row0 includes host... '
OUTPUT=$(oca_status_row0 "test" "/tmp")
assert_contains "$OUTPUT" "#[align=right]" "row0 should have right-aligned section"
printf 'OK\n'

printf 'status_bar: row1 includes date... '
OUTPUT=$(oca_status_row1 "/tmp")
assert_contains "$OUTPUT" "$(date '+%Y-%m-%d')" "row1 should contain date"
printf 'OK\n'

printf 'status_bar: row1 includes LLM gauges when data exists... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
mkdir -p "$OCA_CACHE_DIR"
printf '75' > "$OCA_CACHE_DIR/zai"
OUTPUT=$(oca_status_row1 "/tmp")
assert_contains "$OUTPUT" "za" "row1 should contain za gauge"
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'status_bar: row0 includes temporal health when configured... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance"
printf 'ADV_TEMPORAL_ADDRESS=127.0.0.1:1\n' > "$OCA_CACHE_DIR/temporal.env"
OUTPUT=$(oca_status_row0 "test" "/tmp")
assert_contains "$OUTPUT" "T:✗" "row0 should show temporal health when env exists"
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'status_bar: row0 omits temporal health when not configured... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
# No temporal.env, no ADV dir
OUTPUT=$(oca_status_row0 "test" "/tmp")
assert_not_contains "$OUTPUT" "T:" "row0 should not show temporal health when not configured"
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'status_bar: performance budget under 200ms... '
START=$(date +%s%N)
OUTPUT=$(oca_status_row0 "test" "/tmp")
END=$(date +%s%N)
ELAPSED=$(( (END - START) / 1000000 ))  # Convert ns to ms
if (( ELAPSED > 200 )); then
  printf 'FAIL: row0 took %dms, budget is 200ms\n' "$ELAPSED" >&2
  exit 1
fi
printf 'OK (%dms)\n' "$ELAPSED"

# ── Occupancy segment tests ─────────────────────────────────

printf 'occupancy: --status --pane returns ? for missing state... '
OCA_BIN="$ROOT_DIR/oca"
if [[ -x "$OCA_BIN" ]]; then
  STATUS_OUTPUT=$(TMUX_PANE="%9999" timeout 0.5s "$OCA_BIN" occupancy --status --pane "%9999" 2>/dev/null || true)
  if [[ "$STATUS_OUTPUT" != "?" ]]; then
    printf 'FAIL: expected "?", got %q\n' "$STATUS_OUTPUT" >&2
    exit 1
  fi
  printf 'OK\n'
else
  printf 'SKIP (oca binary not built)\n'
fi

printf 'occupancy: --status --pane returns ? without TMUX_PANE... '
if [[ -x "$OCA_BIN" ]]; then
  STATUS_OUTPUT=$(unset TMUX_PANE; timeout 0.5s "$OCA_BIN" occupancy --status 2>/dev/null || true)
  if [[ "$STATUS_OUTPUT" != "?" ]]; then
    printf 'FAIL: expected "?", got %q\n' "$STATUS_OUTPUT" >&2
    exit 1
  fi
  printf 'OK\n'
else
  printf 'SKIP (oca binary not built)\n'
fi

printf 'occupancy: row0 passes explicit pane id to oca... '
FAKE_BIN_DIR=$(mktemp -d)
cat > "$FAKE_BIN_DIR/oca" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" > "${OCA_FAKE_ARGS_FILE:?}"
printf '1× trunk'
EOF
chmod +x "$FAKE_BIN_DIR/oca"
ARGS_FILE=$(mktemp)
OLD_PATH="$PATH"
export OCA_FAKE_ARGS_FILE="$ARGS_FILE"
PATH="$FAKE_BIN_DIR:$PATH"
hash -r
OUTPUT=$(TMUX_PANE="%wrong" oca_status_row0 "test" "/tmp" "%expected")
PATH="$OLD_PATH"
hash -r
unset OCA_FAKE_ARGS_FILE
ARGS=$(cat "$ARGS_FILE")
rm -rf "$FAKE_BIN_DIR" "$ARGS_FILE"
if [[ "$ARGS" != *"--pane %expected"* ]]; then
  printf 'FAIL: expected explicit pane id, got %q\n' "$ARGS" >&2
  exit 1
fi
printf 'OK\n'

printf 'occupancy: timeout kills slow invocation... '
if [[ -x "$OCA_BIN" ]]; then
  # 0.1s timeout should still return quickly
  START=$(date +%s%N)
  timeout 0.1s "$OCA_BIN" occupancy --status --pane "%1" 2>/dev/null || true
  END=$(date +%s%N)
  ELAPSED=$(( (END - START) / 1000000 ))
  if (( ELAPSED > 500 )); then
    printf 'FAIL: timeout did not kill after %dms\n' "$ELAPSED" >&2
    exit 1
  fi
  printf 'OK (%dms)\n' "$ELAPSED"
else
  printf 'SKIP (oca binary not built)\n'
fi

# ── Compact Window Glyph Tests ───────────────────────────────

printf 'window glyphs: renders 11 windows within 160-column budget... '
MOCK_WINDOWS="1:trunk:0
2:change/abc123def:0
3:change/def456ghi:0
4:change/ghi789jkl:1
5:change/jkl012mno:0
6:change/mno345pqr:0
7:change/pqr678stu:0
8:change/stu901vwx:0
9:change/vwx234yza:0
10:change/yza567bcd:0
11:change/bcd890efg:0"
OUTPUT=$(_oca_status_window_glyphs_from_list "$MOCK_WINDOWS" 160)
# Strip tmux escape sequences for length check
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
LEN=${#PLAIN}
if (( LEN > 160 )); then
  printf 'FAIL: output %d chars exceeds 160 budget\n' "$LEN" >&2
  exit 1
fi
# Must contain at least 11 window indices
for i in 1 2 3 4 5 6 7 8 9 10 11; do
  if [[ "$PLAIN" != *"$i:"* ]]; then
    printf 'FAIL: missing window index %d\n' "$i" >&2
    exit 1
  fi
done
printf 'OK (%d chars for 11 windows)\n' "$LEN"

printf 'window glyphs: maps trunk to T glyph... '
MOCK_WINDOWS="1:trunk:0
2:change/abc:1"
OUTPUT=$(_oca_status_window_glyphs_from_list "$MOCK_WINDOWS" 160)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" != *"1:T"* ]]; then
  printf 'FAIL: expected "1:T" in output, got %q\n' "$PLAIN" >&2
  exit 1
fi
printf 'OK\n'

printf 'window glyphs: maps change windows to abbreviated ID... '
MOCK_WINDOWS="1:trunk:0
2:change/patternBSessionTopologyOne:1"
OUTPUT=$(_oca_status_window_glyphs_from_list "$MOCK_WINDOWS" 160)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" != *"2:patt"* ]]; then
  printf 'FAIL: expected "2:patt" (abbreviated change ID) in output, got %q\n' "$PLAIN" >&2
  exit 1
fi
printf 'OK\n'

printf 'window glyphs: highlights active window with indigo... '
MOCK_WINDOWS="1:trunk:0
2:change/abc:1"
OUTPUT=$(_oca_status_window_glyphs_from_list "$MOCK_WINDOWS" 160)
if [[ "$OUTPUT" != *"6C7AB8"* ]]; then
  printf 'FAIL: active window should have indigo (#6C7AB8) color, got %q\n' "$OUTPUT" >&2
  exit 1
fi
printf 'OK\n'

printf 'window glyphs: truncates when budget exceeded... '
# 30 windows with long names, small budget
MOCK_WINDOWS=""
for i in $(seq 1 30); do
  ACTIVE=0
  if (( i == 15 )); then ACTIVE=1; fi
  MOCK_WINDOWS="${MOCK_WINDOWS}${i}:change/veryLongChangeName${i}:${ACTIVE}"$'\n'
done
OUTPUT=$(_oca_status_window_glyphs_from_list "$MOCK_WINDOWS" 80)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
LEN=${#PLAIN}
if (( LEN > 80 )); then
  printf 'FAIL: output %d chars exceeds 80 budget\n' "$LEN" >&2
  exit 1
fi
# Should end with truncation marker
if [[ "$PLAIN" != *"…" ]]; then
  printf 'FAIL: expected truncation marker … at end, got %q\n' "$PLAIN" >&2
  exit 1
fi
printf 'OK (%d chars, truncated)\n' "$LEN"

printf 'window glyphs: single trunk window renders cleanly... '
MOCK_WINDOWS="1:trunk:1"
OUTPUT=$(_oca_status_window_glyphs_from_list "$MOCK_WINDOWS" 160)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" != *"1:T"* ]]; then
  printf 'FAIL: expected "1:T" for single trunk, got %q\n' "$PLAIN" >&2
  exit 1
fi
printf 'OK\n'

printf 'window glyphs: handles windows with plain names... '
MOCK_WINDOWS="1:bash:0
2:editor:1"
OUTPUT=$(_oca_status_window_glyphs_from_list "$MOCK_WINDOWS" 160)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" != *"1:bash"* ]]; then
  printf 'FAIL: expected "1:bash" for plain name, got %q\n' "$PLAIN" >&2
  exit 1
fi
printf 'OK\n'

# ── Branch Safety & Workspace State Tests ─────────────────────

printf 'branch safety: unsafe glyph when on default branch with active change... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
# Create fake ADV state with an active change and a worktree record showing trunk
PROJECT_ID="testproj01"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID"
cat > "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID/snapshot.json" <<'EOF'
{
  "worktree_registry": {
    "change/testChange01": {
      "branch": "change/testChange01",
      "path": "",
      "materialized": false,
      "changeId": "testChange01",
      "status": "active",
      "setupReady": true
    }
  }
}
EOF
# Create git repo on trunk
GIT_DIR=$(mktemp -d)
cd "$GIT_DIR" && git init && git config user.email "t@t" && git config user.name "T"
touch a && git add a && git commit -m "init" >/dev/null 2>&1
git checkout -b trunk >/dev/null 2>&1 || true
OUTPUT=$(oca_status_branch_safety "$GIT_DIR" "$PROJECT_ID" 2>/dev/null || true)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" != *"⚡"* ]]; then
  printf 'FAIL: expected ⚡ (unsafe trunk) glyph, got %q\n' "$PLAIN" >&2
  rm -rf "$GIT_DIR" "$WORK_DIR"
  exit 1
fi
rm -rf "$GIT_DIR" "$WORK_DIR"
printf 'OK\n'

printf 'branch safety: safe when on change branch... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
PROJECT_ID="testproj02"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID"
cat > "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID/snapshot.json" <<'EOF'
{
  "worktree_registry": {
    "change/testChange02": {
      "branch": "change/testChange02",
      "path": "/tmp/worktree",
      "materialized": true,
      "changeId": "testChange02",
      "status": "active",
      "setupReady": true
    }
  }
}
EOF
GIT_DIR=$(mktemp -d)
cd "$GIT_DIR" && git init && git config user.email "t@t" && git config user.name "T"
touch a && git add a && git commit -m "init" >/dev/null 2>&1
git checkout -b change/testChange02 >/dev/null 2>&1
OUTPUT=$(oca_status_branch_safety "$GIT_DIR" "$PROJECT_ID" 2>/dev/null || true)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" == *"⚡"* ]]; then
  printf 'FAIL: should NOT show unsafe glyph on change branch, got %q\n' "$PLAIN" >&2
  rm -rf "$GIT_DIR" "$WORK_DIR"
  exit 1
fi
rm -rf "$GIT_DIR" "$WORK_DIR"
printf 'OK\n'

printf 'branch safety: safe when no active changes... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
PROJECT_ID="testproj03"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID"
# No snapshot.json — no ADV state
GIT_DIR=$(mktemp -d)
cd "$GIT_DIR" && git init && git config user.email "t@t" && git config user.name "T"
touch a && git add a && git commit -m "init" >/dev/null 2>&1
git checkout -b trunk >/dev/null 2>&1 || true
OUTPUT=$(oca_status_branch_safety "$GIT_DIR" "$PROJECT_ID" 2>/dev/null || true)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" == *"⚡"* ]]; then
  printf 'FAIL: should not show unsafe glyph with no ADV state, got %q\n' "$PLAIN" >&2
  rm -rf "$GIT_DIR" "$WORK_DIR"
  exit 1
fi
rm -rf "$GIT_DIR" "$WORK_DIR"
printf 'OK\n'

printf 'workspace glyph: setup_failed indicator... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
PROJECT_ID="testproj04"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID"
cat > "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID/snapshot.json" <<'EOF'
{
  "worktree_registry": {
    "change/brokenChange": {
      "branch": "change/brokenChange",
      "path": "/tmp/wt-broken",
      "materialized": true,
      "changeId": "brokenChange",
      "status": "setup_failed",
      "setupReady": false,
      "setupFailureReason": "postCreate hook timeout"
    }
  }
}
EOF
OUTPUT=$(oca_status_workspace_state "$PROJECT_ID" 2>/dev/null || true)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" != *"✗"* ]]; then
  printf 'FAIL: expected ✗ for setup_failed, got %q\n' "$PLAIN" >&2
  rm -rf "$WORK_DIR"
  exit 1
fi
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'workspace glyph: stale indicator... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
PROJECT_ID="testproj05"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID"
cat > "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID/snapshot.json" <<'EOF'
{
  "worktree_registry": {
    "change/oldChange": {
      "branch": "change/oldChange",
      "path": "",
      "materialized": false,
      "changeId": "oldChange",
      "status": "stale",
      "setupReady": true
    }
  }
}
EOF
OUTPUT=$(oca_status_workspace_state "$PROJECT_ID" 2>/dev/null || true)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" != *"ѻ"* ]]; then
  printf 'FAIL: expected ѻ (stale) indicator, got %q\n' "$PLAIN" >&2
  rm -rf "$WORK_DIR"
  exit 1
fi
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'workspace glyph: merged/idle shows no warning... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
PROJECT_ID="testproj06"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID"
cat > "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID/snapshot.json" <<'EOF'
{
  "worktree_registry": {
    "change/doneChange": {
      "branch": "change/doneChange",
      "path": "/tmp/wt-done",
      "materialized": true,
      "changeId": "doneChange",
      "status": "merged",
      "setupReady": true
    }
  }
}
EOF
OUTPUT=$(oca_status_workspace_state "$PROJECT_ID" 2>/dev/null || true)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" == *"✗"* ]] || [[ "$PLAIN" == *"ѻ"* ]] || [[ "$PLAIN" == *"⚡"* ]]; then
  printf 'FAIL: merged state should show no warning glyph, got %q\n' "$PLAIN" >&2
  rm -rf "$WORK_DIR"
  exit 1
fi
# Should show ✓ for merged
if [[ "$PLAIN" != *"✓"* ]]; then
  printf 'FAIL: expected ✓ for merged state, got %q\n' "$PLAIN" >&2
  rm -rf "$WORK_DIR"
  exit 1
fi
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'workspace glyph: empty when no ADV state... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
PROJECT_ID="testproj07"
# No snapshot.json
OUTPUT=$(oca_status_workspace_state "$PROJECT_ID" 2>/dev/null || true)
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ -n "$PLAIN" ]]; then
  printf 'FAIL: expected empty output with no ADV state, got %q\n' "$PLAIN" >&2
  rm -rf "$WORK_DIR"
  exit 1
fi
rm -rf "$WORK_DIR"
printf 'OK\n'

printf 'window glyphs: enriches with workspace state... '
WORK_DIR=$(mktemp -d)
export OCA_CACHE_DIR="$WORK_DIR/cache"
export XDG_DATA_HOME="$WORK_DIR/xdg-data"
mkdir -p "$OCA_CACHE_DIR"
PROJECT_ID="testproj08"
mkdir -p "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID"
cat > "$XDG_DATA_HOME/opencode/plugins/advance/$PROJECT_ID/snapshot.json" <<'EOF'
{
  "worktree_registry": {
    "change/brokenWT": {
      "branch": "change/brokenWT",
      "path": "/tmp/wt-broken",
      "materialized": true,
      "changeId": "brokenWT",
      "status": "setup_failed",
      "setupReady": false
    }
  }
}
EOF
MOCK_WINDOWS="1:trunk:0
2:change/brokenWT:1"
OUTPUT=$(_oca_status_window_glyphs_from_list "$MOCK_WINDOWS" 160 "$PROJECT_ID")
PLAIN=$(_strip_tmux_escapes "$OUTPUT")
if [[ "$PLAIN" != *"✗"* ]]; then
  printf 'FAIL: window glyph for setup_failed worktree should show ✗, got %q\n' "$PLAIN" >&2
  rm -rf "$WORK_DIR"
  exit 1
fi
rm -rf "$WORK_DIR"
printf 'OK\n'

# ── Restore env for remaining tests ──
unset XDG_DATA_HOME OCA_CACHE_DIR

printf '\nAll status_bar tests passed.\n'
