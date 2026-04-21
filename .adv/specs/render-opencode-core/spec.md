# Capability: render-opencode-core

Phase 3 capability spec for core `opencode.json` rendering beyond MCP/plugins/instructions.

## Requirements

- `rq-core-render-providers01` — OCA renders `[providers.*]` into `opencode.json` `.provider` with shape translation `context/output -> limit.*` and `inputs/outputs -> modalities.*`, preserving undeclared user keys.
- `rq-core-render-permissions01` — OCA renders `[permissions]` into `opencode.json` `.permission`, translating `default` to `"*"` while preserving nested `external_directory` and `bash` maps plus undeclared user keys.
- `rq-core-render-watcher01` — OCA renders `[watcher].ignore` into `opencode.json` `.watcher.ignore`, preserving user-added ignore entries and other `.watcher` keys.
- `rq-core-render-lsp01` — OCA renders `[lsp.*]` into `opencode.json` `.lsp` with per-server preservation and passthrough for future server fields.
- `rq-core-apply-all01` — `oca apply` without `--target` composes all supported targets in dependency order through a running in-memory `opencode.json` document so later targets never clobber earlier ones.
- `rq-core-diff01` — `oca diff` is read-only, supports target filtering, and exits `0` when clean, `1` on drift, `2` on invalid config/target, and `3` on runtime planning failure.
