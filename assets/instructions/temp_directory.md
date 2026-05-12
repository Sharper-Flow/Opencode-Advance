# OpenCode Advance: Temp & Cache Directory Policy

## Where temp/cache files live

OpenCode Advance uses a dedicated cache directory instead of scattering files across `/tmp`.

The location is set by the `OCA_CACHE_DIR` environment variable, resolved by the
`oca` CLI and shell integration:

| Condition | Resolved Path |
|-----------|---------------|
| `$XDG_RUNTIME_DIR` is set | `$XDG_RUNTIME_DIR/opencode-advance/` |
| `$XDG_RUNTIME_DIR` is unset (macOS, containers) | `/tmp/opencode-advance-$USER/` |
| `OCA_CACHE_DIR` is pre-set | That value is used as-is (no override) |

The directory is created with owner-only permissions (`0700`) and is guaranteed
to exist before any OCA operation that needs it.

## Files in the cache directory

| File | Written by | Read by | Content |
|------|-----------|---------|---------|
| `temporal.env` | `internal/render/temporal.go` | `oca adv-status` (`internal/advstatus`) | `ADV_TEMPORAL_ADDRESS=...` |

No `adv_status` or `temporal_health` cache files are written. The tmux status
bar reads ADV state on demand through `oca adv-status`.

All writes are atomic: data is written to a temp file then `mv`'d to the final
path to prevent partial reads.

## Cleanup

Cache files are short-lived (10s TTL for status bar polling). Stale files are
overwritten on the next poll cycle. No periodic cleanup daemon is needed.

## Why not `/tmp` directly?

- `/tmp` is world-writable and full of unrelated process artifacts
- `$XDG_RUNTIME_DIR` is user-private (`0700`) and auto-cleaned on logout
- A single subtree can be approved once in OpenCode's file access policy

## For AI agents

When creating temporary files during a session, prefer:
- `$TMPDIR` (if set, points to a per-session safe location)
- `$(mktemp)` or `$(mktemp -d)` (respects `$TMPDIR` automatically)
- `/tmp/opencode` (pre-approved for OpenCode agent temporary work)

Do not hardcode `/tmp/` paths when `$TMPDIR` or `$OCA_CACHE_DIR` are available.
