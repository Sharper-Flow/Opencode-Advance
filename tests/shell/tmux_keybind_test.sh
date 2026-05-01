#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMUX_CONF="$REPO_ROOT/assets/themes/obsidian.tmux.conf"

echo "=== tmux keybind test ==="

# 1. Keybind exists
if ! grep -q 'bind r' "$TMUX_CONF"; then
    echo "FAIL: bind r not found in obsidian.tmux.conf"
    exit 1
fi
echo "PASS: bind r found"

# 2. Guard uses if-shell with oca- prefix check
if ! grep -q 'if-shell.*echo.*session_name.*grep.*oca-' "$TMUX_CONF"; then
    echo "FAIL: session-name guard not found"
    exit 1
fi
echo "PASS: session-name guard found"

# 3. OCA branch runs oca pane restart-tui
if ! grep -q 'oca pane restart-tui' "$TMUX_CONF"; then
    echo "FAIL: oca pane restart-tui command not in keybind"
    exit 1
fi
echo "PASS: oca pane restart-tui in keybind"

# 4. Non-OCA branch sends literal r
if ! grep -q "send-keys.*r" "$TMUX_CONF"; then
    echo "FAIL: fallback send-keys not found"
    exit 1
fi
echo "PASS: fallback send-keys found"

# 5. Window option: remain-on-exit off (prevents dead-pane accumulation when
#    opencode crashes or is killed; AC1 of sessionSafetyHardeningConfig).
if ! grep -qE '^[[:space:]]*setw[[:space:]]+-g[[:space:]]+remain-on-exit[[:space:]]+off' "$TMUX_CONF"; then
    echo "FAIL: 'setw -g remain-on-exit off' not found in obsidian.tmux.conf"
    exit 1
fi
echo "PASS: setw -g remain-on-exit off found"

echo "=== all tests passed ==="
