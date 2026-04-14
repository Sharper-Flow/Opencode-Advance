#!/usr/bin/env bash
set -euo pipefail

output=$(make -n check 2>&1)

if [[ "$output" != *"bash tests/shell/brand_helpers_test.sh"* ]]; then
  printf 'make check did not include shell verification\n%s\n' "$output" >&2
  exit 1
fi

echo "verification workflow test passed"
