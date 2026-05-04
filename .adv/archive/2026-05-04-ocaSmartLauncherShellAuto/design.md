## Architecture Overview

Three loosely-coupled components share one staleness primitive:

- **Producers** (`oca apply` / `install` / `update` / `pin` / plugin rebuild): write `~/.config/oca/env.sh` atomically, THEN touch `$OCA_CACHE_DIR/env.stamp`.
- **Consumer A — shell hook (per prompt):** stat `env.stamp`; on mtime advance source the env file + run drift surfacer; update marker.
- **Consumer B — update probe (explicit + cached):** `oca update --check` runs `git ls-remote` per plugin in parallel, writes `$OCA_CACHE_DIR/drift_cache.json`, reports.

Single shared primitive: the stamp. Hook also reads `drift_cache.json` (TTL-bounded). No daemon, no Temporal, no IPC.

## Key Decisions

| Q | Decision | Rationale |
|---|----------|-----------|
| Q1 stamp path | `$OCA_CACHE_DIR/env.stamp` | Beside existing `apply.lock`; respects `OCA_CACHE_DIR` test override; XDG_RUNTIME_DIR by default. |
| Q2 re-source target | `~/.config/oca/env.sh` (sibling) | Re-evaluating user rc risks user-code side effects. Sibling is OCA-only, safely idempotent. |
| Q3 write order (validator C1) | Env file FIRST, stamp LAST — hard constraint | TOCTOU guard. `RenderShellEnvWriteFile` writes file via `WriteAtomic`, only on success calls `WriteStamp`. Invariant `mtime(stamp) ≥ mtime(env.sh)` tested. |
| Q6 drift cache TTL | 5 min default, `update_probe.cache_ttl_minutes` configurable | LBP-typical short cache. |
| Q7 same-shell trigger | Trust mtime | `oca apply` synchronous; stamp touched before subprocess returns; next prompt cannot fire until then. |
| Q8 stack.toml schema | Two top-level sections: `[shell]`, `[update_probe]` | `[shell]` is shell-only; `[update_probe]` powers both CLI + shell surfacer. Naming `[shell.update_probe]` would mislead CLI consumer. |
| Naming | `oca update --check` (flag, not subcommand) | LBP: `apt list --upgradable`, `npm outdated`, `pip list --outdated`. Reuses existing infra. |
| Probe mechanism (validator dim 4) | `git ls-remote` (NOT `git fetch`) | Truly read-only; `fetch` would mutate `refs/remotes/*`. |
| Doctor reuse (validator C2) | `--check` extends, does not duplicate `rq-plugin-doctor01` | Doctor stays local-only; `--check` adds remote dimension. Both share `DriftStatus` types; CI enforces sync. |

## stack.toml

```toml
[shell]
auto_refresh        = true
auto_refresh_notice = "off"   # off | once | every

[update_probe]
default              = "passive"  # off | passive | warn
timeout_per_plugin_ms = 3000
timeout_global_ms     = 10000
cache_ttl_minutes     = 5
```

## Implementation Strategy

Five phases, smallest-first, each independently shippable.

**Phase α — Producer plumbing.** New: `internal/render/stamp.go` (`WriteStamp`), `internal/render/env_file.go` (`RenderShellEnv`, `RenderShellEnvWriteFile`). Wire into `Apply`, `cmd/oca/install.go`, `cmd/oca/update.go`, `cmd/oca/pin.go`, `internal/plugin/build.go:RunBuild`. Tests assert TOCTOU invariant.

**Phase β — Hook installation.** Extend `internal/install/shellprofile.go:shellProfileTemplate`: top-of-block interactive guard, source env.sh once, hook function (zsh `add-zsh-hook precmd` / bash `PROMPT_COMMAND` dedup), idempotent registration. Add `internal/config/shell.go` for `[shell]` parsing. Conditional render based on `Shell.AutoRefresh`.

**Phase γ — Update probe (CLI).** New `internal/plugin/probe.go`: `Probe`, `ProbeAll`, `DriftStatus`, `DriftResult`. Uses `git ls-remote` via `internal/subprocess`, parallel via `errgroup` with per-op + global timeout. `cmd/oca/update.go` gains `--check` flag → calls `ProbeAll`, writes `drift_cache.json` atomically, renders text/JSON. Add `internal/config/update_probe.go`.

**Phase δ — Drift surfacer.** Extend shell template with `_oca_drift_surfacer`. Per-shell guard prints once. Reads `drift_cache.json` via `awk`/`cut` (no jq dep). `warn` mode triggers `oca update --check --quiet` on cache miss with shell `timeout`.

**Phase ε — Uninstall + integration.** Verify `RemoveBlock` removes new content. Round-trip test (install→apply→uninstall). Benchmark hook overhead.

Sequencing: α → β; α → γ; (β + γ) → δ; δ → ε. α/β/γ individually releasable.

## LBP Analysis

Aligns with direnv/asdf/mise/starship pattern: stamp + precmd-stat + sibling env file. Probe matches `apt list --upgradable` shape. `git ls-remote` is the read-only-correct tool. External-solution check skipped (no third-party owns OCA's global managed-block territory; direnv is per-directory not a fit).

## Affected Components

**New:** `internal/render/{env_file,stamp}.go`, `internal/plugin/probe.go`, `internal/config/{shell,update_probe}.go`.
**Modified:** `internal/install/shellprofile.go`, `internal/render/{apply,plan}.go`, `cmd/oca/{apply,update,install,pin}.go`, `internal/plugin/build.go`, `internal/config/types.go`, `stack.example.toml`, `docs/design/{stack-toml-schema,cli-surface}.md`.
**Untouched (Pattern B owns):** `cmd/oca/{pane,session,watchdog}.go`, `$XDG_STATE_HOME/oca/panes/*.json` schema, `cmd/oca/main.go`.
**Specs:** new `oca-shell-auto-refresh` (rq-shellAutoRefresh01..06) + `oca-update-probe` (rq-updateProbe01..04). Probe spec cross-references `plugin-apply` rq-plugin-doctor01.

## Risks / Mitigations

| Risk | Mitigation |
|------|------------|
| Hook latency at 8+ shells per Pattern B session | One `stat(2)` budget; CI benchmark vs un-hooked baseline. |
| TOCTOU stamp-vs-envfile | Q3 hard constraint: env file FIRST, stamp LAST; invariant tested. |
| Hook composes badly with existing PROMPT_COMMAND | Bash detects + appends with `;`; zsh uses `add-zsh-hook -d` then re-add. Test covers pre-populated PROMPT_COMMAND. |
| WSL2 9P mtime 1-sec resolution | Default `OCA_CACHE_DIR` is XDG_RUNTIME_DIR (Linux ns resolution). 9P only for `/mnt/c`; OCA does not write there. |
| Drift surfacer noise in non-interactive contexts | Top-of-block interactive guard (AC #5). |
| Plugin probe credential prompt | `GIT_TERMINAL_PROMPT=0` + existing transport hardening; `ls-remote` is read-only. |
| Stamp race vs parallel apply | Existing `apply.lock` flock; stamp write inside lock. |
| `--check` vs doctor drift drift | Spec cross-reference + shared `DriftStatus` types + CI sync test. |
| Future Pattern B managed-block conflict | Same `WriteBlock`/`RemoveBlock` infra; sentinel canonicalization handles diff; coordinate via wisdom entry post-archive. |

## Validator Result

**Verdict: CAUTION** (clean pass; two technical refinements folded in before planning).

- C1 (Correctness, dim 1): write ordering — env file before stamp. → Resolved as Q3 hard constraint.
- C2 (Spec compliance, dim 3): `--check` should extend `rq-plugin-doctor01`, not duplicate. → Resolved via spec cross-reference + shared types + CI sync test.

Info-level confirmations: mtime-based staleness (one-stat budget); `git ls-remote` read-only-correct; two `stack.toml` sections justified; env-file split correct; three-tier mode coherent; precmd-stat over inotify; no spec conflicts.

Recommendation accepted. No contract compromise, no user-value tradeoff requiring re-checkpoint. Proceeding to planning.