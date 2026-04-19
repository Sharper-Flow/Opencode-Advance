# Phase 1 Design — stack.toml parser + MCP apply

Canonical design for change `phase1StackTomlParserMcpApply`. Validated inline by the agent (adv-researcher inline pass).

## Architecture

```
 stack.toml ──► config.Load() ──► Stack ──► render.Plan() ──► Plan ──► render.Apply() ──► files
                    │                                │                          │
                    ▼                                ▼                          ▼
             config.Validate()               template execution         atomic.Write()
                    │                                │                          │
                    ▼                                ▼                          ▼
             config.Resolve()                  stable sort keys           .bak.<epoch> rotate
```

`oca doctor --scope mcp` is a separate path: `config.Load()` → `health.CheckMCP()` (hits Vision `/v1/servers`) → report.

## Packages

### `internal/config`

```go
type Stack struct {
    Meta Meta                  `toml:"meta"`
    MCP  MCPSection            `toml:"mcp"`
}

type Meta struct {
    Version     string `toml:"version"`
    Name        string `toml:"name"`
    Description string `toml:"description"`
}

type MCPSection struct {
    Servers map[string]Server `toml:"servers"`
}

// Server mirrors Vision's ServerConfig with OCA-only additions.
// Unknown Vision fields decode into ExtraFields for forward-compat passthrough.
type Server struct {
    // Core transport (Vision)
    Port      int               `toml:"port"`
    Transport string            `toml:"transport,omitempty"`      // "stdio"|"http"|"sse"
    Command   string            `toml:"command,omitempty"`
    Args      []string          `toml:"args,omitempty"`
    Env       map[string]string `toml:"env,omitempty"`
    URL       string            `toml:"url,omitempty"`
    Headers   map[string]string `toml:"headers,omitempty"`

    // Lifecycle (Vision)
    Autostart           bool   `toml:"autostart,omitempty"`
    RestartPolicy       string `toml:"restart_policy,omitempty"`
    MaxRestarts         int    `toml:"max_restarts,omitempty"`
    Stateful            bool   `toml:"stateful,omitempty"`
    AvailabilityProfile string `toml:"availability_profile,omitempty"`
    SessionTimeout      string `toml:"session_timeout,omitempty"` // e.g. "30m"
    MaxSessions         int    `toml:"max_sessions,omitempty"`
    SessionTTL          string `toml:"session_ttl,omitempty"`
    HealthCheckInterval string `toml:"health_check_interval,omitempty"`
    RequestTimeout      string `toml:"request_timeout,omitempty"`

    // Resilience (Vision)
    Retry          *RetryConfig    `toml:"retry,omitempty"`
    CircuitBreaker *CircuitBreaker `toml:"circuit_breaker,omitempty"`

    // Sharing (Vision)
    SharedReadOnlyTools   []string `toml:"shared_read_only_tools,omitempty"`
    SharedResultCacheTTL  string   `toml:"shared_result_cache_ttl,omitempty"`
    SharedResultCacheSize int      `toml:"shared_result_cache_size,omitempty"`
    MaxInFlightRequests   int      `toml:"max_in_flight_requests,omitempty"`

    // OCA-only / Vision (after V2+V3)
    Required    bool   `toml:"required,omitempty"`    // Vision respects after V3
    Source      string `toml:"source,omitempty"`      // Vision accepts after V2
    Description string `toml:"description,omitempty"` // Vision accepts after V2
    Enabled     *bool  `toml:"enabled,omitempty"`     // nil = default true; tri-state for explicit disable

    // OCA-side only (not written out)
    EnvFile string `toml:"env_file,omitempty"`

    // Passthrough for unknown Vision fields
    ExtraFields map[string]any `toml:"-" yaml:"-"`
}

type RetryConfig struct {
    MaxAttempts     int      `toml:"max_attempts,omitempty"`
    InitialDelay    string   `toml:"initial_delay,omitempty"`
    MaxDelay        string   `toml:"max_delay,omitempty"`
    RetryableErrors []string `toml:"retryable_errors,omitempty"`
}

type CircuitBreaker struct {
    FailureThreshold int    `toml:"failure_threshold,omitempty"`
    RecoveryTimeout  string `toml:"recovery_timeout,omitempty"`
}
```

Functions:

```go
func Load(path string) (*Stack, error)                  // read+parse+validate+resolve
func Parse(data []byte) (*Stack, error)                 // parse-only
func (s *Stack) Validate() error                        // aggregate errors
func (s *Stack) Resolve(env Env) error                  // in-place var resolution
type Env struct { Home, XDGConfig, XDGData string; Lookup func(string) (string, bool) }
type ValidationError struct { Path string; Msg string; Cause error }
type ValidationErrors []ValidationError // Error() pretty-prints all
```

**Validation rules (mirroring Vision):**
- `[meta].version` must be `"1.0.0"` (only supported schema version)
- `[mcp.servers.<name>].port` in `[6276, 6300]`
- Transport inference: explicit `transport` wins, else `stdio` if `command` set, else `http` if `url` ends with `/mcp`, else `sse` if url set, else error
- Cannot set both `command` and `url`
- `http` transport requires `url` ending in `/mcp`
- Port uniqueness across declared servers
- Unknown `restart_policy` / `availability_profile` rejected
- Native OpenCode tokens (`{env:...}`, `{file:...}`) pass through; unknown OCA tokens (starts with `{` but not a native token) error with field path

**Variable resolution:**
- `$HOME`, `~/...` → user home dir
- `$XDG_CONFIG_HOME`, `$XDG_DATA_HOME` → env or XDG defaults
- `${ENV_VAR}` / `${ENV_VAR:-default}` → env lookup (mirrors Vision's pattern)
- Native OpenCode tokens NOT resolved — written literally to output

### `internal/render`

```go
type Plan struct {
    Source  string       // path to stack.toml
    Targets []TargetOp   // ordered operations
}

type TargetOp struct {
    Path      string     // destination
    Op        string     // "write" | "merge" | "noop"
    Before    []byte     // existing content (may be nil)
    After     []byte     // final content
    BackupPath string    // ".bak.<epoch>" if backup created
    Reason    string     // human-readable what-and-why
}

func PlanMCP(stack *config.Stack, paths Paths) (*Plan, error)
func Apply(plan *Plan, opts ApplyOptions) (*ApplyResult, error)

type Paths struct {
    OpencodeJSON string // $OCA_OPENCODE_CONFIG_DIR/opencode.json
    VisionYAML   string // $OCA_VISION_CONFIG_DIR/servers.yaml
}

type ApplyOptions struct {
    DryRun   bool
    MaxBackups int  // default 3
}
```

**JSON merge algorithm for `opencode.json`:**

1. Read existing file, parse as `map[string]any`, preserve all top-level keys
2. If `.mcp` missing or not an object, set it to `{}`
3. For each declared server in stack.toml:
   - Render the OCA-shape entry `{type: "remote", url: "http://localhost:<port>/mcp", enabled, oauth: false, timeout}`
   - Write to `.mcp[<name>]`, overwriting any existing value
4. Leave other `.mcp` keys untouched (user-added servers)
5. Sort `.mcp` keys alphabetically before marshalling
6. Marshal with `json.MarshalIndent(obj, "", "  ")` + trailing newline
7. If marshalled bytes equal existing bytes → `Op: noop`, no backup, no write

**YAML write for `vision/servers.yaml`:**

1. Build Vision-shape `map[string]any` with `servers:` key + per-server entries
2. Strip OCA-only fields (`required`, `source`, `description`, `env_file`, `enabled` when nil) — BUT after V2+V3 land in Vision, pass them through
3. Emit keys in stable alphabetical order
4. Include a header comment: `# vision/servers.yaml — generated by oca apply\n# source: <stack.toml>\n\n`
5. If bytes equal existing → `Op: noop`

**Atomic write (`internal/render/atomic.go`):**

```go
func WriteAtomic(path string, data []byte, maxBackups int) error {
    // 1. stat existing file; if exists and content == data → return nil (noop at I/O level)
    // 2. rotate backups: keep newest maxBackups-1 .bak.<epoch> files, delete older
    // 3. if existing file present, rename it to .bak.<now>
    // 4. write temp file in same directory (os.CreateTemp with prefix)
    // 5. os.Rename temp → target
    // 6. on any failure after temp write, best-effort remove temp
}
```

**Determinism tests:** run `PlanMCP` twice on the same stack.toml; assert byte-equal outputs.

### `internal/health`

```go
type Status string
const (
    StatusPass Status = "pass"
    StatusWarn Status = "warn"
    StatusFail Status = "fail"
)

type Check struct {
    Name    string
    Status  Status
    Message string
    Hint    string
    Elapsed time.Duration
}

func CheckMCP(ctx context.Context, stack *config.Stack, opts Options) ([]Check, error)

type Options struct {
    VisionAdminURL string        // default http://localhost:6275
    Timeout        time.Duration // per-request, default 5s
    HTTPClient     *http.Client  // injectable for tests
}
```

**MCP check flow:**
1. GET `{VisionAdminURL}/v1/servers` (new endpoint from Vision V1)
2. Decode `[]{name, port, transport, state, last_error, uptime_seconds}`
3. For each declared server in stack.toml:
   - If present in response with `state == "running"` → pass
   - If present with `state == "failed"|"crashed"` → fail with last_error
   - If present but `state == "stopped"` and `autostart: false` → pass (intentionally stopped)
   - If absent from response → warn "declared but not registered with Vision"
4. If Vision unreachable → single warn check "Vision admin unreachable — run `vision start`"

### `cmd/oca`

- `apply.go` — `oca apply [--target mcp] [--dry-run]`. Flag wiring to `render.PlanMCP` + `render.Apply`.
- `doctor.go` — `oca doctor [--scope mcp] [--timeout DUR] [--output text|json]`.
- `debug.go` — `oca debug plan` (prints Plan JSON), `oca debug validate` (prints validation result, no writes).

All commands share:
- `--config <path>` — defaults to `./stack.toml` or `$XDG_CONFIG_HOME/opencode-advance/stack.toml`
- `--output text|json` — output format
- Respect `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR` env overrides

### `templates/`

- `opencode.json.mcp.gotmpl` — renders one `.mcp[<name>]` entry per declared server
- `vision-servers.yaml.gotmpl` — renders the full servers.yaml

Template engine: stdlib `text/template` with safe JSON escaping for the JSON template. Each template is self-contained; no shared includes.

### Paths resolution (`internal/config/paths.go`)

```go
type Paths struct {
    OpencodeConfigDir string // default ~/.config/opencode
    VisionConfigDir   string // default ~/.config/vision
}

func ResolvePaths() Paths {
    // honor OCA_OPENCODE_CONFIG_DIR, OCA_VISION_CONFIG_DIR
    // fall back to defaults
}
```

## Test strategy

- **Unit tests** per package: parse, validate, resolve, plan, merge, atomic, health
- **Golden tests** under `internal/render/testdata/`:
  - Each MCP server variant has input TOML + expected `opencode.json.mcp` + expected `servers.yaml` fragment
  - Full `stack.example.toml` → full golden output
  - `-update` flag regenerates goldens
- **E2E test** under `tests/apply_test.go`:
  - Load `stack.example.toml`
  - Set env overrides to `t.TempDir()`
  - Run `oca apply --target mcp --dry-run` → inspect plan
  - Run `oca apply --target mcp` → compare output dir to golden dir
  - Run again → assert no-op (byte-identical, no new backup)
- **Doctor test** with `httptest.Server` emulating Vision `/v1/servers`

## Cross-repo Vision work (V1-V4)

Tasks in prep phase will carry `metadata.target_repo: vision`, `metadata.target_path: /home/jrede/dev/vision`. ADV switches workdir per task. Sequencing: V1-V4 land first, then OCA code integrates against them.

**V1 concrete shape:**
```go
// GET /v1/servers
// Response: { servers: [ { name, port, transport, state, autostart, last_error, uptime_seconds, restart_count } ] }
// GET /v1/servers/{name}
// Response: { name, port, transport, state, autostart, last_error, uptime_seconds, restart_count, args, env_keys }
```

**V2**: add `Source string `yaml:"source,omitempty"`` and `Description string `yaml:"description,omitempty"`` to `ServerConfig` in `internal/config/schema.go`.

**V3**: in Vision's server startup path, if `Required=true && Autostart=true` and initial start fails after max_restarts, daemon returns non-nil startup error.

**V4**: doc block in Vision README titled "External configuration tools" explaining the contract.

## Validator verdict (inline adv-researcher pass)

**Architecture soundness:** ✓ Clean separation of config (parse/validate/resolve), render (plan/merge/atomic), health (checks). Mirrors Vision's own package layout. Pure-function plan stage is unit-testable independently.

**Simplicity:** ✓ No ORM, no reflection-heavy codegen, no DI framework. Stdlib + two small deps (BurntSushi/toml, yaml.v3). Template engine is stdlib.

**Potential pitfalls considered:**
- **Map iteration non-determinism** — mitigated by sorting keys before marshal
- **JSON pretty-print rendering drift** — fixed indent (2 spaces), trailing newline, determinism test enforces
- **Cross-filesystem temp+rename** — temp is created in target dir via `os.CreateTemp(dir, prefix)`, so same filesystem guaranteed
- **Vision version drift** — doctor+apply emit clear "Vision V1 required" hint on 404
- **TOML decoder strictness** — BurntSushi/toml `DisallowUnknownFields` via `MetaData` + post-decode check; unknown fields in root cause errors; unknown fields under `[mcp.servers.*]` land in `ExtraFields` (passthrough)
- **BOM handling** — BurntSushi handles by default; add a regression test

**LBP check:** BurntSushi/toml + yaml.v3 + stdlib testing is the 2026 go-to for CLI config tools. No deprecated pattern used.

**Verdict:** PROCEED — no CONFLICT found.
