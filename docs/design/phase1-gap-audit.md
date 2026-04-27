# Phase 1 Completeness Audit

Post-planning audit to identify real gaps before execution. Scored against prep methodology cross-cutting concerns checklist + spec deltas + clarify warnings.

## Gap Inventory

### G1 — Missing spec deltas (HIGH)
Agreement lists 10 `rq-mcp-*01` requirements. None are recorded in `change.deltas` or `.adv/specs/`. Archive gate will fail without them.

**Fix:** Add spec deltas using `adv_change` delta mechanism (records them in change.json) and add a task to create the `mcp-apply` capability spec during archive.

### G2 — Ambiguous `type = "daemon"` semantics (HIGH)
`stack.example.toml` declares `[mcp.servers.vision] type = "daemon" port = 6275`. Discovery confirmed Vision has only `stdio`/`http`/`sse` transports — there is no `daemon`. But the live `opencode.json` shows vision as `{type: "remote", url: "http://localhost:6275/mcp"}`. OCA needs a rule for how to render the Vision entry.

**Fix:** Schema doc + design clarify:
- `type = "daemon"` in stack.toml is an OCA-only directive meaning "this is the Vision admin MCP itself"
- Render rule: writes `opencode.json` `.mcp.vision = {type: "remote", url: http://localhost:PORT/mcp, enabled, oauth, timeout}` AND does NOT write a `vision` entry to `servers.yaml` (Vision doesn't manage itself)
- Validation: exactly one server may have `type = "daemon"`; must be named `vision`; port must match Vision admin port

### G3 — `/v1/servers` authentication (HIGH)
Vision's `/mcp` is behind `SecurityMiddleware` (bearer token + origin check). `/health` and `/healthz` are NOT. New `/v1/servers` could go either way. This is an unresolved security decision.

**Fix:** Decide between:
- **Public** (like `/health`): no auth on admin status. Simpler; matches `/health`; low practical risk since admin binds `127.0.0.1`. Last-error messages may leak sensitive info (e.g. API keys in error strings).
- **Behind middleware**: OCA passes `Authorization: Bearer <token>` from `VISION_BEARER_TOKEN` env (same convention Vision uses).

**Recommendation:** Public. Vision admin is localhost-bound; `last_error` should be scrubbed of secrets before serialization. Add an `errorScrubber` helper in V1.

### G4 — Secret scrubbing in V1 responses (HIGH)
V1's `/v1/servers` returns `last_error` strings. If an MCP server fails with an error message containing `${API_KEY}`-expanded values, those leak through the admin endpoint.

**Fix:** V1 handler runs `last_error` through a scrubber that redacts `AUTHORIZATION`, `*_TOKEN`, `*_KEY`, `*_SECRET`, `password=` substrings.

### G5 — Variable resolution order of operations (MEDIUM)
Design says "Resolve at parse time". But validate runs BEFORE resolve in the Load() pipeline (planned order: Parse → Validate → Resolve). That means validation sees pre-resolved strings like `~/secrets/kagi.env` and has to know whether tokens are legal in each field. Port validation etc. works fine, but `env_file` validation can't check existence pre-resolution.

**Fix:** Change pipeline order to Parse → Resolve → Validate. Update design doc + Load() task.

### G6 — `env_file` existence check policy (MEDIUM)
Schema says "OCA does NOT read env_file paths; they are recorded as-is". But should validation check that the path exists at apply time? If it doesn't, user gets a confusing runtime error from Vision later. If it does, tests need to mock the filesystem.

**Fix:** Soft-warn at apply time if `env_file` path doesn't exist (non-fatal; print warning). Never read contents. Add this to doctor MCP scope.

### G7 — Unix permissions on written files (MEDIUM)
Vision's atomic write sets `0600` on `servers.yaml` (sensitive — may contain tokens). OCA's atomic write spec says nothing about permissions. `opencode.json` is typically `0644`.

**Fix:** Add permission policy to `WriteAtomic`:
- `servers.yaml` → `0600` (matches Vision's own Save)
- `opencode.json` → `0644`
- Generalize: `WriteAtomic(path, data, mode os.FileMode, maxBackups int)`

### G8 — Logging / observability (MEDIUM)
Cross-cutting concern missing from design. When `oca apply` runs, what does it log? How verbose? Structured or text? The architecture doc mentions `emit structured logs, traces, and errors for all significant actions` (rules.yaml P14).

**Fix:** Add logging policy:
- Use `log/slog` with text handler by default, JSON via `--output json`
- Log levels: INFO for high-level ops ("applying mcp target"), DEBUG for per-server decisions, ERROR for failures
- Include change-relevant context: target, path, server name

### G9 — Backup rotation edge cases (LOW)
"Keep 3 most recent `.bak.<epoch>` files". What if files have identical epochs (same-second apply)? What if epoch parsing fails? What about `.bak.<epoch>.tmp` orphans from interrupted writes?

**Fix:** Add in backup.go:
- Use monotonic epoch with `time.Now().UnixNano()` instead of `Unix()` to prevent collisions
- Ignore files that don't parse as epoch (don't crash)
- Clean up orphan `*.bak.*.tmp` on start of each apply

### G10 — Golden file fixture strategy (LOW)
Plan calls for goldens of every declared MCP variant. But each variant is stdio vs http vs with-env vs with-env_file vs with-required — combinatoric. Need explicit fixture list.

**Fix:** Write fixture index:
- `testdata/minimal.toml` — bare [meta] + 1 stdio server
- `testdata/full-stack.toml` — copy of stack.example.toml
- `testdata/with-env.toml` — server with `[server.env]`
- `testdata/with-env-file.toml` — server with `env_file`
- `testdata/required-true.toml` — server with `required=true`
- `testdata/daemon-type.toml` — [mcp.servers.vision] special case
- `testdata/availability-profile.toml` — server with `availability_profile = "networked"`
- `testdata/user-added-preserved.toml` — existing opencode.json with non-declared MCP entry, verify preservation
- `testdata/invalid-*.toml` — 5-6 validation failure fixtures (port out of range, both command+url, etc.)

Each fixture has corresponding `.opencode.json.golden` and `.servers.yaml.golden`.

### G11 — CLI ergonomics: `--target` without value (LOW)
`oca apply --target mcp` is clear. What about `oca apply --target` with no value, or `oca apply` (no target)? Phase 1 only implements mcp target; unknown/omitted targets should error cleanly.

**Fix:** Define behavior:
- `oca apply` (no target) → error: "--target is required in Phase 1 (only `mcp` supported)"
- `oca apply --target foo` → error: "unknown target `foo`; supported: mcp"
- Future phases expand supported targets.

### G12 — Version compatibility check (LOW)
`rq-mcp-vision-integration01` says "OCA detects Vision version mismatch and emits a clear upgrade hint". How? Vision doesn't expose a version endpoint today.

**Fix:** Two options:
- Probe `/v1/servers`; 404 → "Vision missing V1 endpoint; upgrade Vision to >= the version bundled with this OCA release"
- Add a `GET /version` endpoint to Vision as V5 (scope creep)

**Recommendation:** Option 1 — detect by probing. V1's presence IS the version marker.

### G13 — TOML strict decode policy for root sections (LOW)
Design says "unknown keys in root sections error". But stack.toml Phase 1 only implements `[meta]` and `[mcp]`. Running against a real `stack.example.toml` that contains `[plugins.*]`, `[providers.*]`, etc. will fail with "unknown section" errors.

**Fix:** Whitelist known-but-unimplemented top-level sections as "deferred":
- Known top-level: `meta`, `mcp`, `plugins`, `instructions`, `providers`, `agents`, `permissions`, `watcher`, `lsp`, `session`, `discord`, `skills`, `formatters`, `commands`, `opencode`
- `meta` and `mcp` → parsed into Stack
- All others → parsed into `Stack.Unimplemented map[string]toml.MetaData` and ignored (no error, no render)
- Truly unknown sections (typo'd, not in whitelist) → error with field path

### G14 — Error output format for scripting (LOW)
Exit codes are defined but stderr format is not. For CI integration, JSON output on `--output json` should apply to errors too.

**Fix:** When `--output json`, errors go to stderr as `{"level":"error", "code":"...", "path":"...", "message":"..."}`.

### G15 — Concurrent apply safety (LOW)
If two `oca apply` runs execute concurrently (e.g. user accidentally runs twice), they could corrupt backups or produce inconsistent state.

**Fix:** Acquire a file lock on `$OCA_CACHE_DIR/apply.lock` using `flock` (Linux/macOS). Mirror the test-resource-guardrails pattern. Timeout: 30s.

### G16 — CLARIFY_ASSUMPTION_HEAVY warning resolution (INFO)
The validator flagged "authentication/authorization without specifying the model" — this came from security-adjacent mentions in the proposal (env_file, secrets). G3 resolves the genuine auth question (Vision admin endpoint). No change needed beyond G3.

## Prep Methodology Cross-Cutting Concerns Checklist

| Concern | Addressed? | Location |
|---------|------------|----------|
| Error handling | ✓ | proposal §Error Handling; G5-G7 refinements |
| Logging | ✗ → ADD | G8 |
| Input validation | ✓ | validate.go task; G13 refinement |
| Security | Partial → ADD | G3, G4, G7 |
| Performance | ✓ | no perf-sensitive code paths in Phase 1 |
| Configuration | ✓ | env overrides, schema doc |
| Monitoring | ✓ | health.CheckMCP |
| Rollback | ✓ | .bak.<epoch> files |
| Concurrency | ✗ → ADD | G15 |
| Observability (traces) | ✗ | N/A for CLI; slog is enough (G8) |

## Decisions Needed

| ID | Decision | Default if no input |
|----|----------|---------------------|
| D1 | `/v1/servers` auth | Public (G3) |
| D2 | `env_file` existence check | Soft-warn at apply (G6) |
| D3 | Vision version detection | Probe `/v1/servers` for 404 (G12) |
| D4 | Concurrent apply lock | flock on `$OCA_CACHE_DIR/apply.lock` (G15) |
| D5 | Unknown stack.toml sections | Whitelist-and-defer (G13) |

## Task Delta (additions to planning)

15 existing tasks unchanged in substance. New tasks needed:

- **tk-spec-delta:** Create capability spec `mcp-apply` with rq-mcp-*01 requirements; add to `.adv/specs/mcp-apply/`
- **tk-secret-scrub:** Vision V1 handler includes error message scrubber for `last_error` field
- **tk-lock:** `internal/render`: implement `LockApply(cacheDir)` with flock + 30s timeout
- **tk-logging:** `internal/oca/log`: structured logging via slog; honors `--output`, `--verbose`, `--quiet`
- **tk-pipeline-order:** Update Load() to Parse → Resolve → Validate (fixes G5)
- **tk-perm-policy:** Update WriteAtomic signature to take `os.FileMode`; set 0600 for servers.yaml, 0644 for opencode.json
- **tk-daemon-type:** Render rule for `type = "daemon"` — writes opencode.json only, skips servers.yaml
- **tk-section-whitelist:** Parser accepts known-but-unimplemented sections without error (stack.example.toml compatibility)
- **tk-cli-errors:** `--target` required; unknown targets errored cleanly; JSON error format on `--output json`
