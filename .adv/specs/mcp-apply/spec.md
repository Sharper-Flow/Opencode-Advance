# Capability: mcp-apply

Phase 1 capability spec for parsing `stack.toml` and rendering MCP configuration.

## Requirements

- `rq-mcp-parse01` — Parse `stack.toml` into typed config structures; known future sections are deferred, not fatal.
- `rq-mcp-validate01` — Validation errors include field paths and aggregate multiple failures.
- `rq-mcp-resolve01` — Expand `~/`, `$HOME`, `$XDG_*`, `${VAR}`, `${VAR:-default}`; preserve native OpenCode `{env:...}` / `{file:...}` tokens unchanged.
- `rq-mcp-render01` — Render deterministic `opencode.json` `.mcp` and authoritative `vision/servers.yaml` outputs.
- `rq-mcp-merge01` — Preserve user-added `.mcp` entries while overwriting declared MCP keys.
- `rq-mcp-atomic01` — Use same-dir temp write + rename, backup rotation, and correct file modes.
- `rq-mcp-health01` — Doctor checks use Vision `/version` and `/v1/servers` with per-server status output.
- `rq-mcp-isolation01` — All writes honor `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_CACHE_DIR`.
- `rq-mcp-debug01` — `oca debug plan` emits JSON plan; `oca debug validate` performs no writes.
- `rq-mcp-vision-integration01` — Vision V1-V5 admin endpoints are treated as the compatibility contract.
- `rq-mcp-concurrency01` — `oca apply` serializes concurrent runs with a host-local lock.
- `rq-mcp-daemon-type01` — `type = "daemon"` renders only an OpenCode `.mcp` entry and never a Vision managed server entry.
- `rq-mcp-env-file01` — `env_file` paths are recorded as-is; missing paths warn, not fail; file contents are never read.
- `rq-mcp-secret-scrub01` — Vision status surfaces scrub secrets from diagnostic strings before exposing them over HTTP.
- `rq-mcp-source-provenance01` — `source` on `[mcp.*]` and `[plugins.*]` is a provenance URL (documentation / traceability), not a package resolution directive. For `[plugins.*]`, package resolution uses `source` only when it is a `git+https://`, `https://...git`, or `npm:<pkg>` URI; other `source` values are informational and MUST NOT drive install behavior.
- `rq-mcp-restart-policy01` — `restart_policy` on `[mcp.*]` is an enum with exactly three legal values: `always`, `on-failure`, `never`. Any other value is a validation error surfaced with the field path. Unset defers to Vision default.
