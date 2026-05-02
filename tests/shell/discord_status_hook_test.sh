#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

export OCA_CACHE_DIR="$TMP_DIR/cache"
mkdir -p "$OCA_CACHE_DIR/discord"
cat >"$OCA_CACHE_DIR/discord/discord-update.sh" <<'SH'
#!/usr/bin/env sh
printf invoked >"$OCA_CACHE_DIR/discord/hook-ran"
SH
chmod +x "$OCA_CACHE_DIR/discord/discord-update.sh"

bash "$ROOT_DIR/lib/status_bar.sh" row0 oca-test "$ROOT_DIR" >/dev/null

for _ in 1 2 3 4 5; do
  if [[ -f "$OCA_CACHE_DIR/discord/hook-ran" ]]; then
    break
  fi
  sleep 0.1
done

if [[ "$(cat "$OCA_CACHE_DIR/discord/hook-ran" 2>/dev/null || true)" != "invoked" ]]; then
  echo "FAIL: Discord update wrapper was not invoked by status bar" >&2
  exit 1
fi

echo "discord status hook test passed"
