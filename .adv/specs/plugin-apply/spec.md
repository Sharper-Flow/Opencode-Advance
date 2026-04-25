# Capability: plugin-apply

Phase 2 capability spec for plugin lifecycle, plugin render surfaces, sync delegation, and plugin doctor coverage.

## Requirements

- `rq-plugin-clone01` — Git-source plugins clone/update into declared checkouts using subprocess-driven git operations, not embedded git libraries.
- `rq-plugin-build01` — Plugin build commands run non-interactively with inherited env plus explicit CI-safe overrides and return captured output on failure.
- `rq-plugin-npm01` — `npm:` plugin sources render as flat literal entries in `.plugin` and do not require local checkout/build steps.
- `rq-plugin-render01` — OCA renders deterministic `.plugin` entries into `opencode.json` while preserving user-added plugin entries.
- `rq-plugin-provides01` — `provides` categories suppress OCA-owned assets in those categories and preserve plugin ownership boundaries, including `adv-instructions` and reserved `adv-temporal`.
- `rq-plugin-sync-order01` — Plugin sync commands run only after successful render/apply commits; render failure blocks sync, sync failure does not roll back already-rendered files.
- `rq-plugin-instructions01` — Declared `[instructions].order` stays authoritative, plugin-provided instructions append deterministically, and `adv-instructions` entries are omitted from OCA render when ownership is plugin-patched.
- `rq-plugin-pin01` — `oca pin` captures current git HEAD SHAs into `stack.toml` for selected or all git plugins without mutating npm plugin declarations.
- `rq-plugin-doctor01` — `oca doctor --scope plugins` verifies checkout presence, built artifact presence, local git ref resolvability, rendered plugin entry presence, and classifies plugin drift.
- `rq-plugin-reserve-temporal01` — `[temporal]`, `adv-temporal`, and temporal CLI surfaces are validated and rendered by OCA (implemented in Phase 5). `provides = ["adv-temporal"]` in plugin config defers to OCA's rendering.
- `rq-plugin-subprocess01` — All plugin git/build/sync operations use the generic subprocess runner with explicit timeout, exit classification, and combined output capture.
- `rq-plugin-healthcheck-registry01` — Doctor scopes register through a shared health-check registry with builtin replay support for test resets and future scope expansion.
- `rq-plugin-isolation01` — Plugin apply, doctor, pin, and update flows honor `OCA_OPENCODE_CONFIG_DIR`, `OCA_VISION_CONFIG_DIR`, `OCA_CACHE_DIR`, and `OCA_PLUGIN_CHECKOUT_ROOT` for test/dev isolation.
