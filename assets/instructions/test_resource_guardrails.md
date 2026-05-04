# Test Resource Guardrails

Purpose: prevent local machine overload when multiple OpenCode instances run concurrently.

## Default Policy

1. Prefer `smoke` or `targeted` checks for normal iteration.
2. Run `full` test suites only when explicitly requested by the user or when preparing final signoff.
3. Never run full suites directly (`pytest`, `npm run test`, `npm run validate`, etc.) when a repo provides `bin/oc-test`.
4. Use command timeouts for all test invocations.

## Concurrency and Resource Limits

- Full suites must pass through a host-wide lock (`flock`) so only one full run is active per machine.
- Heavy test commands should be deprioritized with `nice` and `ionice` when available.
- If lock acquisition fails, report lock contention and continue with non-full validation when possible.

## Required Workflow

- If `bin/oc-test` exists in the repository:
  - Default: `bin/oc-test` (smoke)
  - Focused: `bin/oc-test targeted ...`
  - Full: `RUN_FULL_TESTS=1 bin/oc-test full`
- If `bin/oc-test` does not exist, emulate the same policy manually (timeout + low priority + no implicit full runs).

## Reporting

- State which tier was run (`smoke`, `targeted`, `full`).
- If full suite was skipped due to policy, say so explicitly and provide the exact opt-in command.
