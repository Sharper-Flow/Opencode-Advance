# OCA Runtime Doctor Canaries

**Status:** Proposal — staged  
**Date:** 2026-05-03  
**Target repo:** `~/dev/opencodeadvance`  
**Resume from:** `~/dev/opencodeadvance`  
**Suggested change ID:** `runtimeDoctorCanaries`  
**Priority:** SHOULD

---

## Problem Statement

OCA doctor checks are useful but still too config/file oriented for failures
that only appear at OpenCode runtime.

Recent investigation found a live example: ADV provider prompt files and JSON
config looked correct, but `opencode debug agent adv-gpt` still resolved to the
stub fallback. A string-level check would pass while the agent is broken.

OCA should distinguish static configuration health from runtime integration
health.

---

## Success Criteria

- [ ] New or expanded doctor scope runs runtime canaries for key integrations.
- [ ] Canaries include ADV provider prompt resolution via `opencode debug agent`.
- [ ] Canaries include OCA plugin registration/pane-state write readiness.
- [ ] Canaries include Vision/MCP runtime status beyond rendered config.
- [ ] Canaries redact secrets and use strict timeouts.
- [ ] Failures include actionable remediation and target project ownership.

---

## Out of Scope

- Replacing existing static doctor scopes.
- Auto-fixing runtime failures.
- Running destructive or mutating checks by default.

---

## Implementation Sketch

1. Add `runtime` doctor scope or expand `cross` with runtime subchecks.
2. Add canary helpers:
   - `opencode debug agent adv-{provider}` prompt marker check.
   - plugin array contains OCA plugin and built artifact exists.
   - Vision `/version`, `/v1/servers`, and slot group APIs where supported.
3. Make checks time-bounded and best-effort so doctor remains responsive.
4. Include ownership in results: `OCA`, `Advance`, `Vision`, `OpenCode`.

---

## Acceptance Criteria

1. With ADV provider stub active, doctor emits a failure naming ADV prompt
   runtime canary.
2. With OCA plugin absent, doctor emits a warning/failure naming OCA plugin.
3. With Vision down, MCP runtime check reports unreachable.
4. `go test ./...` passes.
