# Design: fixBareOcaApplyLifecycleParity

## Verdict

VALIDATED by independent validator (`adv-researcher`). Minor refinements incorporated.

## Current Architecture

`cmd/oca/apply.go` has two paths:

1. **No-target path**: `render.ComposeApplyPlan(..., render.AllTargets)` → one pure render plan → `render.Apply` once.
2. **Target-specific path**: loops over targets and dispatches to target functions (`applyPlugins`, `applyTemporal`, etc.).

The no-target path cannot represent side effects:

- plugin prepare/build before render
- plugin sync after render
- Temporal env file render/write

`render.ComposeApplyPlan` remains valid for pure render composition and `oca diff`; it must not become side-effectful.

## Implementation Strategy

### 1. Add lifecycle-aware default target dispatch

Add helper in `cmd/oca/apply.go`:

```go
func defaultApplyTargets() []string {
    targets := make([]string, 0, len(render.AllTargets)+1)
    for _, t := range render.AllTargets {
        targets = append(targets, string(t))
    }
    return append(targets, "temporal")
}

func applyTargetsInOrder(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool, targets []string) error {
    for _, target := range targets {
        switch target { ... existing target switch ... }
    }
    return nil
}
```

Use `defaultApplyTargets()` when no `--target` supplied. Keep explicit target validation unchanged.

**Why:** derives from `render.AllTargets` to avoid ordering drift, appends `temporal` because Temporal is not a render target.

### 2. Replace no-target pure render fast path

Current no-target path loads stack/warnings, builds `ComposeApplyPlan`, and applies it once.

New no-target path:

1. load stack
2. print warnings
3. call `applyTargetsInOrder(ctx, state, stack, paths, dryRun, defaultApplyTargets())`

Targeted path reuses the same helper after validation.

### 3. Preserve NoRollback intent with per-target rollback

Agreement says prior successful writes remain on partial failure.

Per-target dispatch achieves this:

- previous successful targets stay written
- failed target uses its existing target-local rollback behavior
- no rollback-all behavior introduced

This is slightly safer than monolithic `NoRollback: true` because a failed target does not leave its own partial writes when the target-local apply can roll back.

### 4. Dry-run lifecycle reporting

Dry-run remains read-only:

- no plugin prepare/build
- no plugin sync
- no temporal.env write

Add lightweight text output for side-effectful targets where needed, for example:

```text
would prepare plugins
would render plugin config
would sync plugins
rendered /path/to/temporal.env
```

Implementation should keep JSON output compatibility simple: text lifecycle lines only in text mode; JSON continues to use redacted plan output from target functions.

### 5. Tests

Add/extend integration tests using existing helpers:

- Bare apply with fake git plugin/sync proves sync marker exists.
- Bare apply with `[temporal]` enabled proves `temporal.env` exists.
- Bare apply `--dry-run` proves no sync marker and no `temporal.env`.
- Target-specific tests continue passing.
- Full suite: `go test ./...`.

## Files Expected

- `cmd/oca/apply.go`
- `cmd/oca/apply_test.go` or `tests/*` lifecycle integration test
- Docs only if test/design shows user-facing behavior drift; not required by current agreement.

## Rejected Alternatives

- Make `ComposeApplyPlan` side-effectful — rejected. It is a render/diff primitive.
- Keep no-target fast path and special-case plugins/temporal around it — rejected. Duplicates lifecycle semantics and invites drift.
- Rollback all previous target writes on later failure — rejected by user; current NoRollback spirit remains.

## Design Validator Refinements Incorporated

1. Derive default targets from `render.AllTargets` + `temporal`.
2. Use loop+switch helper instead of function dispatch table because `applyTemporal` has different signature.
3. Clarify per-target rollback semantics as preserving prior-target NoRollback behavior while avoiding partial failed target writes.
4. Accept repeated `WriteShellEnvAndStamp` calls as idempotent and non-blocking.