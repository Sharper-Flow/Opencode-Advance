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

OCA must not touch live user config during development. Phase 1 code explicitly honors:

- `OCA_OPENCODE_CONFIG_DIR`
- `OCA_VISION_CONFIG_DIR`
- `OCA_CACHE_DIR`

Default production paths remain:

- `~/.config/opencode/opencode.json`
- `~/.config/vision/servers.yaml`

Tests and local dev runs use temporary override directories instead.

## Future phases

The following remain planned, not shipped:

- plugin lifecycle and Advance sync delegation
- instruction/provider/agent/permission/watcher/LSP rendering
- session/theme UX
- migration from open-chad
- broader `oca doctor` scopes

See [`../proposals/phases.md`](../proposals/phases.md) for sequencing.
