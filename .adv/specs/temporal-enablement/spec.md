---
capability: temporal-enablement
version: "1.0.0"
status: active
---

# Capability: temporal-enablement

Phase 5 capability spec for Temporal infrastructure enablement: config parsing, env file rendering, doctor checks, and status bar integration.

## Requirements

- `rq-temp01` — **Temporal config parsing from stack.toml**
  - **Given** a `stack.toml` with `[temporal]` section,
  - **When** `oca` loads config,
  - **Then** `TemporalSection` is populated with:
    - `Address` (default `"127.0.0.1:7233"`),
    - `Namespace` (default `"default"`),
    - `AllowRemote` (bool),
    - `NodePath` (string, path to Node.js binary).

- `rq-temp02` — **Env file rendering with 4 ADV_TEMPORAL_* vars + atomic write**
  - **Given** a stack with temporal enabled,
  - **When** `oca apply --target temporal` runs,
  - **Then** `$OCA_CACHE_DIR/temporal.env` is written atomically (same-dir temp + rename) containing:
    - `ADV_TEMPORAL_ADDRESS=<value>`
    - `ADV_TEMPORAL_NAMESPACE=<value>`
    - `ADV_TEMPORAL_ALLOW_REMOTE=<value>` (only if non-empty)
    - `ADV_NODE_PATH=<value>` (only if non-empty).

- `rq-temp03` — **Node.js v20+ detection in default doctor scope**
  - **Given** `oca doctor` runs with default scope,
  - **When** the doctor registry executes health checks,
  - **Then** Node.js detection runs, emitting a **warning** (not fatal) if Node.js is missing or version `< v20`, with an install hint.

- `rq-temp04` — **Temporal CLI detection in default doctor scope**
  - **Given** `oca doctor` runs with default scope,
  - **When** the doctor registry executes health checks,
  - **Then** Temporal CLI (`temporal`) detection runs, emitting a **warning** (not fatal) if the binary is missing, with an install hint.

- `rq-temp05` — **`--scope temporal` dispatches to registry with server+namespace probe**
  - **Given** `oca doctor --scope temporal` runs,
  - **When** the temporal scope is dispatched,
  - **Then** two probes execute:
    1. TCP reachability check to the configured address (1s timeout),
    2. `temporal namespace describe` CLI probe (5s timeout).

- `rq-temp06` — **Non-loopback without allow_remote fails validation**
  - **Given** a `[temporal]` section with a non-loopback address and `allow_remote != true`,
  - **When** config is validated,
  - **Then** a validation error is surfaced on `temporal.allow_remote` requiring explicit opt-in for remote servers.

- `rq-temp07` — **Status bar row0 shows T:✓/T:✗ after ADV state segment**
  - **Given** `temporal.env` exists with an address,
  - **When** the status bar renders row0,
  - **Then**:
    - `T:✓` appears after the ADV state segment if the server is reachable,
    - `T:✗` appears if unreachable **and** the Advance state directory exists,
    - empty otherwise.

