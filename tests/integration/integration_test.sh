#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$ROOT_DIR"
OCA_TEST_BIN_DIR=$(mktemp -d)
trap 'rm -rf "$OCA_TEST_BIN_DIR"' EXIT

echo "=== Integration Test: Phase 4 Richness ==="
echo

# 1. Verify all status bar scripts exist and are executable
echo "[1/5] Checking status bar scripts..."
for script in lib/status_bar.sh lib/llm_gauge.sh lib/boot_splash.sh lib/session_lifecycle.sh; do
  if [[ ! -x "$script" ]]; then
    echo "FAIL: $script not found or not executable"
    exit 1
  fi
done
echo "  OK — all scripts present and executable"

# 2. Build the binary
echo "[2/5] Building oca binary..."
if ! go build -o "$OCA_TEST_BIN_DIR/oca" ./cmd/oca >/dev/null 2>&1; then
  echo "FAIL: build failed"
  exit 1
fi
echo "  OK — binary builds"

# 3. Verify CLI commands are registered
echo "[3/5] Checking CLI command registration..."
OUTPUT=$("$OCA_TEST_BIN_DIR/oca" session --help 2>&1)
for cmd in attach switch kill killall restart reap; do
  if ! echo "$OUTPUT" | grep -q "$cmd"; then
    echo "FAIL: session $cmd not registered"
    exit 1
  fi
done
echo "  OK — all session subcommands registered"

OUTPUT=$("$OCA_TEST_BIN_DIR/oca" theme --help 2>&1)
for cmd in list apply; do
  if ! echo "$OUTPUT" | grep -q "$cmd"; then
    echo "FAIL: theme $cmd not registered"
    exit 1
  fi
done
echo "  OK — all theme subcommands registered"

# 4. Run unit tests
echo "[4/5] Running unit test suite..."
if ! go test ./cmd/oca/... ./internal/session/... >/dev/null 2>&1; then
  echo "FAIL: unit tests failed"
  exit 1
fi
echo "  OK — unit tests pass"

# 5. Verify shell script tests
echo "[5/5] Running shell script tests..."
PATH="$OCA_TEST_BIN_DIR:$PATH"
for test in tests/shell/llm_gauge_test.sh tests/shell/boot_splash_test.sh tests/shell/status_bar_test.sh tests/shell/session_lifecycle_test.sh; do
  if [[ -f "$test" ]]; then
    if ! bash "$test" >/dev/null 2>&1; then
      echo "FAIL: $test failed"
      exit 1
    fi
  fi
done
echo "  OK — all shell tests pass"

echo
echo "=== All integration checks passed ==="
