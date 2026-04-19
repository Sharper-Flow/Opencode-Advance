# Phase 1 Design — stack.toml parser + MCP apply

Canonical design for change `phase1StackTomlParserMcpApply`, updated to match the code shipped on the Phase 1 release branch.

## Architecture

```text
stack.toml
  └─► config.Load()
        ├─► ParseFile()
        ├─► Resolve()
        └─► Validate()
              ├─► render.PlanMCP()
              │     ├─► MergeMCP()
              │     └─► RenderVisionServers()
              └─► render.Apply()
```

`oca doctor --scope mcp` is a separate path:

```text
config.Load() -> health.CheckMCP() -> CLI report
```

Phase 1 render logic is **programmatic**. No templates are used.

## `internal/config`

### Current core types

```go
type Stack struct {
    Meta             Meta
    MCP              MCPSection
    DeferredSections map[string]map[string]any
    Warnings         []Warning
}

type MCPSection struct {
    Servers map[string]Server
}
```

`Server` mirrors the Phase 1 Vision-facing fields plus OCA-owned convenience fields:

- transport and connection: `Port`, `Type`, `Transport`, `Command`, `Args`, `Env`, `URL`, `Headers`
- lifecycle: `Autostart`, `RestartPolicy`, `MaxRestarts`, `Stateful`, `AvailabilityProfile`, `SessionTimeout`, `MaxSessions`, `SessionTTL`, `HealthCheckInterval`, `RequestTimeout`
- resilience: `Retry`, `CircuitBreaker`
- sharing: `SharedReadOnlyTools`, `SharedResultCacheTTL`, `SharedResultCacheSize`, `MaxInFlightRequests`
- Vision-pass-through fields now supported: `Required`, `Source`, `Description`
- OCA-only fields: `Enabled`, `Timeout`, `EnvFile`

Phase 1 does **not** capture arbitrary per-server unknown keys into `ExtraFields`. Forward compatibility is handled at the deferred top-level section layer, not per-server passthrough.

### Current functions

```go
func Load(path string) (*Stack, error)
func Parse(data []byte) (*Stack, error)
func ParseFile(path string) (*Stack, error)
func (s *Stack) Resolve()
func (s *Stack) Validate() error

type ValidationError struct {
    Path    string
    Message string
}

type ValidationErrors []ValidationError
```

### Validation rules

- `meta.version` must be `"1.0.0"`
- server `port` required and must be within `[6275, 6300]`
- ports must be unique
- `type` must be one of `stdio|http|sse|daemon` when set
- `type = "daemon"`:
  - only one allowed
  - must be named `vision`
  - must not set `command`, `args`, or `url`
- non-daemon transport inference:
  - explicit `transport`
  - else explicit non-daemon `type`
  - else `stdio` if `command` is set
  - else `http` if `url` ends with `/mcp`
  - else `sse` if `url` is set
  - else `stdio`
- cannot set both `command` and `url`
- `http` URLs must end with `/mcp`
- `restart_policy` must be `always|on-failure|never`
- `availability_profile` currently accepts `networked`
- `request_timeout`, when set, must be a positive duration string
- known future top-level sections are accepted as deferred
- truly unknown top-level sections error with field path

### Resolution rules

`Resolve()` expands:

- `$HOME`
- `~/...`
- `$XDG_CONFIG_HOME`, `$XDG_DATA_HOME`
- `${VAR}`
- `${VAR:-default}`

Native OpenCode tokens such as `{env:...}` and `{file:...}` are preserved literally.

### Load pipeline

Actual order is:

```text
Parse -> Resolve -> Validate -> collect env_file warnings
```

That order matters because validation runs on resolved values.

## `internal/render`

### Current plan/apply types

```go
type Plan struct {
    Source   string
    LockPath string
    Targets  []TargetOp
}

type TargetOp struct {
    Name       string
    Path       string
    Op         string
    Mode       os.FileMode
    Reason     string
    Before     []byte
    After      []byte
    BackupPath string
}

type ApplyOptions struct {
    DryRun     bool
    MaxBackups int
}
```

### `opencode.json` MCP merge

`RenderMCPFragment` emits:

```json
{
  "type": "remote",
  "url": "http://localhost:<port>/mcp",
  "enabled": true,
  "oauth": false,
  "timeout": <ms>
}
```

Timeout precedence:

1. `Server.Timeout`
2. parsed positive `Server.RequestTimeout`
3. default `5000`

`MergeMCP`:

- preserves all non-MCP top-level keys
- preserves undeclared `.mcp` entries
- overwrites declared `.mcp` entries
- sorts `.mcp` keys alphabetically
- emits indented JSON with trailing newline

### `vision/servers.yaml` render

`RenderVisionServers` writes the full authoritative file.

- skips `type = "daemon"` entries entirely
- writes stable sorted server names
- passes through Vision-supported fields including `required`, `source`, and `description`
- strips OCA-only behavior (`Type`, `EnvFile`, OpenCode-only `Enabled`)
- converts integer `Timeout` to `request_timeout: <N>ms` when needed

### Atomic writes and locking

Actual write signature:

```go
func WriteAtomic(path string, data []byte, mode os.FileMode, maxBackups int) (string, error)
```

Behavior:

1. clean orphan `*.bak.*.tmp`
2. short-circuit on byte-identical no-op
3. rotate existing backups
4. create same-directory temp file
5. apply explicit mode
6. rename into place atomically
7. return backup path when one was created

`Apply()` acquires one lock for the whole run via:

```text
$OCA_CACHE_DIR/apply.lock
```

using `flock` on Unix with a 30s timeout.

## `internal/health`

### Current types

```go
type Check struct {
    Name    string
    Status  Status
    Message string
    Hint    string
    Elapsed time.Duration
}

type Options struct {
    VisionAdminURL string
    Timeout        time.Duration
    HTTPClient     *http.Client
}
```

Default Vision URL is:

```text
http://127.0.0.1:6275
```

### Current MCP check flow

1. GET `/version`
2. require `api.v1_servers = true`
3. GET `/v1/servers`
4. classify declared servers:
   - `running` -> pass
   - `failed|crashed` -> fail
   - `stopped` -> warn unless `required`, then fail
   - missing from response -> warn
5. add local `env_file` and command-path advisory checks
6. if Vision is unreachable or incompatible, return warn checks plus the local advisories

## `cmd/oca`

Shipped Phase 1 commands:

- `oca apply --target mcp [--dry-run] [--config PATH]`
- `oca doctor --scope mcp [--timeout DUR] [--output text|json]`
- `oca debug plan`
- `oca debug validate`

Shared shipped flags:

- `--config`
- `--output text|json`
- `--verbose` / `-v`
- `--quiet` / `-q`

Phase 1 intentionally rejects unimplemented targets/scopes with clear errors.

## Cross-repo Vision work (V1-V5)

Phase 1 depends on the following landed Vision deltas:

- **V1** `GET /v1/servers` and `GET /v1/servers/{name}`
- **V1 refinement** `scrubSecrets()` on serialized `last_error`
- **V2** `source` / `description` fields on `ServerConfig`
- **V3** required-server semantics surfaced for OCA doctor classification
- **V4** README external-tool contract
- **V5** `GET /version` with API capability flags

## Test strategy

Current coverage includes:

- package tests for config parse / validate / resolve / paths
- package tests for render atomic / merge / redact / vision YAML / MCP fragment timeout precedence
- package tests for health version/server classification
- e2e tests for apply, doctor, concurrency, deferred sections, and JSON output shape
- golden outputs under `internal/render/testdata/`

## Notes

- Phase 1 uses **programmatic renderers**, not template files
- broader v1 command surface still lives in planning docs, not in shipped code
- schema reference remains canonical for field definitions, with Phase 1 deferral notes called out there
