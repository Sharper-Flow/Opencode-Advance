# oca-update-probe

OCA exposes read-only plugin update awareness separate from local plugin doctor health.

## Requirements

- `rq-updateProbe01` — `oca update --check` MUST compare local plugin HEAD/ref to remote state using `git ls-remote`, not `git fetch`.
- `rq-updateProbe02` — The probe MUST set `GIT_TERMINAL_PROMPT=0`, use existing git hardening flags, and avoid mutating local checkouts or remote-tracking refs.
- `rq-updateProbe03` — Probe results MUST be written to `$OCA_CACHE_DIR/drift_cache.json` with generated timestamp and explicit statuses.
- `rq-updateProbe04` — `[update_probe]` MUST support `off`, `passive`, and `warn`; `passive` is default and MUST NOT initiate network activity from shell startup.

Cross-reference: plugin local health remains covered by `plugin-apply` / `rq-plugin-doctor01`; this capability adds remote comparison only.
