# Architecture Overview

This document is the canonical high-level architecture reference for the code that is **actually implemented today**.

Current shipped scope is **Phase 1** only:

- `stack.toml` parsing for `[meta]` and `[mcp]`
- deferred acceptance of future top-level sections
- `oca apply --target mcp`
- `oca doctor --scope mcp`
- `oca debug plan` and `oca debug validate`

Future phases are tracked in [`../proposals/phases.md`](../proposals/phases.md).

## Core principle: single source of truth

The user owns one file: `stack.toml`. In Phase 1, OCA renders only the MCP slice from that file.

```text
stack.toml
  └─► oca apply --target mcp
        ├─► opencode.json (.mcp merge only)
        └─► vision/servers.yaml (authoritative full write)
```

OCA does **not** yet render plugins, instructions, providers, agents, permissions, watcher, LSP, session, or theme configuration. Those sections are accepted as deferred input for later phases.

## Subsystems

### 1. Configuration (`internal/config/`)

Phase 1 config is intentionally small and strict.

- Parses TOML with `github.com/BurntSushi/toml`
- Decodes typed Phase 1 sections: `Meta`, `MCP`
- Preserves known future sections in `Stack.DeferredSections`
- Resolves shell-style paths and env variables in-place
- Emits aggregated field-path validation errors
- Collects non-fatal `env_file` warnings

Current core types:

```go
type Stack struct {
    Meta             Meta
    MCP              MCPSection
    DeferredSections map[string]map[string]any
    Warnings         []Warning
}
```

Current pipeline:

```text
ParseFile -> Resolve -> Validate -> collect env_file warnings
```

Not implemented in Phase 1:

- project-local override merge
- typed plugin/provider/agent/session structures
- migration import logic

### 2. Rendering (`internal/render/`)

Phase 1 rendering is **programmatic**, not template-based.

- `RenderMCPFragment` builds one OpenCode `.mcp` entry
- `MergeMCP` overwrites declared MCP keys and preserves user-added keys
- `RenderVisionServers` writes authoritative `servers.yaml`
- `PlanMCP` computes deterministic target operations
- `Apply` executes the plan with dry-run support, lock acquisition, atomic writes, and backup rotation

Current write guarantees:

- `opencode.json` mode: `0644`
- `vision/servers.yaml` mode: `0600`
- same-directory temp-file + rename atomicity
- `.bak.<UnixNano>` backup rotation
- orphan `*.bak.*.tmp` cleanup before write
- host-local `flock` on `$OCA_CACHE_DIR/apply.lock`

Phase 1 special rule:

- `type = "daemon"` is an OCA-only directive
- it still renders an OpenCode remote MCP URL
- it is **not** written into `vision/servers.yaml`

### 3. Health (`internal/health/`)

Phase 1 health is MCP-only.

- `CheckMCP` probes Vision `GET /version`
- requires `api.v1_servers = true`
- then probes `GET /v1/servers`
- classifies declared servers as pass / warn / fail
- treats required-but-not-running servers as failures
- still reports local `env_file` and command-path advisory checks when Vision is down or incompatible

Current defaults:

- Vision admin URL: `http://127.0.0.1:6275`
- per-request timeout: caller-controlled, defaulted by CLI
- execution model: sequential, not worker-pooled

### 4. CLI (`cmd/oca/`)

Current shipped commands:

- `oca version`
- `oca apply --target mcp`
- `oca doctor --scope mcp`
- `oca debug plan`
- `oca debug validate`

Shared shipped flags:

- `--config`
- `--output text|json`
- `--verbose` / `-v`
- `--quiet` / `-q`

Not implemented yet:

- `install`, `diff`, `pin`, `update`, `uninstall`
- migration commands beyond scaffolding
- session / theme command groups

## Data flow: `oca apply --target mcp`

```text
1. Load stack.toml                (config.Load)
2. Build render plan             (render.PlanMCP)
3. --dry-run? print plan, exit
4. Acquire apply lock            (render.AcquireApplyLock)
5. Merge opencode.json .mcp      (render.MergeMCP)
6. Render vision/servers.yaml    (render.RenderVisionServers)
7. Write targets atomically      (render.WriteAtomic)
8. Report per-target result      (CLI JSON/text output)
```

## Data flow: `oca doctor --scope mcp`

```text
1. Load stack.toml               (config.Load)
2. Probe Vision /version         (health.fetchVersion)
3. Probe Vision /v1/servers      (health.fetchServers)
4. Classify server states        (health.CheckMCP)
5. Add local advisory checks     (env_file, command-path)
6. Render report                 (CLI JSON/text output)
```

## Clean cutover policy during development

OCA must not touch live user config during development. Phase 1 + Phase 2 code explicitly honors:

- `OCA_OPENCODE_CONFIG_DIR`
- `OCA_VISION_CONFIG_DIR`
- `OCA_CACHE_DIR`
- `OCA_PLUGIN_CHECKOUT_ROOT` (Phase 2 — base dir for `{checkout}` token expansion)

Default production paths remain:

- `~/.config/opencode/opencode.json`
- `~/.config/vision/servers.yaml`
- `~/dev/oc-plugins/<plugin>/` (plugin checkouts)

Tests and local dev runs use temporary override directories instead.

## Phase 2 subsystems (shipped)

Phase 2 added three internal packages and one render primitive. All external commands route through `internal/subprocess`; no `exec.Command` calls live outside that package.

### `internal/subprocess`

Generic command runner: `Run(ctx, Cmd) (Result, error)`. Implements the canonical Go 2026 pattern — `exec.CommandContext` with `Cmd.Cancel` (SIGTERM) and `Cmd.WaitDelay` (SIGTERM→SIGKILL grace). Classifies exits as `ExitSuccess | ExitNonZero | ExitTimeout | ExitSignal`. Merges a caller-provided env map onto the inherited process environment. Single `bytes.Buffer` for combined stdout+stderr (the kernel serializes writes at the OS level). Zero knowledge of `git`, `npm`, or `sh` — callers supply argv and timeout.

Design references: K1, AC17, AC18, AC19.

### `internal/plugin`

Thin domain wrappers over `internal/subprocess`:

- `git.go` — `Clone`, `Fetch`, `Checkout`, `RevParseHEAD`, `StatusClean`, `RemoteOriginURL`. Every invocation is prefixed with `-c protocol.file.allow=user -c protocol.ext.allow=false` to defuse transport-confusion attack surface (CVE-2022-39253 class). Ref names pass through `validateGitRef` (regex `^[A-Za-z0-9._/-]+$`, rejects empty / leading `-` / `..`).
- `build.go` — `RunBuild` executes each declared build step via `sh -c` with `CI=true`, `DEBIAN_FRONTEND=noninteractive`, `npm_config_yes=true` merged into the environment. Failures report step index and captured output for remediation.
- `npm.go` — literal-version handler for npm-source plugins. No clone, no build.
- `prepare.go` — clone-or-update orchestrator. Uses `os.Lstat` to refuse symlinked checkout paths, compares `remote.origin.url` against stack.toml `source` to detect drift, aborts on dirty worktrees.
- `pin.go` — `CapturePin` (reads HEAD SHA) + `WritePin` (atomic temp-file + rename write of the `ref` field). The caller (`cmd/oca/pin.go`) holds the apply lock so concurrent `oca apply` / `oca pin` cannot corrupt `stack.toml`.

Design references: K1, K2, jc-pnpm1, jc-update1.

### `internal/sync`

`InvokeAdvance(ctx, plugin)` runs the plugin's configured `sync` shell command from its checkout directory and returns combined stdout/stderr filtered through `render.Redact`. The redaction is best-effort pattern matching (GitHub PATs, OpenAI keys, generic `key=value` forms); it is NOT a security boundary and is documented as such.

### `internal/render` — `MergeArray`

New primitive for flat-array sections (`instructions` ordering). Preserves user-declared paths, dedups against declared + plugin-provided entries via `canonicalPath` (which refuses to normalize traversal above the reference point), and prunes stale worktree paths whose base dir is gone. Companion: `.bak.<epoch>` rotation with per-target `SuppressBackup` (see wisdom `ws-GcnRm8`).

### Health registry

Pluggable health checks with a `ResetForTesting()` contract so within-package tests remain serial but don't leak across test binaries.

## Future phases

The following remain planned, not shipped:

- provider/agent/permission/watcher/LSP rendering
- session/theme UX
- migration from open-chad
- broader `oca doctor` scopes (temporal reserved for Phase 6.5)

See [`../proposals/phases.md`](../proposals/phases.md) for sequencing.
