# Archive: Fix bare `oca apply` lifecycle parity so the default apply path prepares/builds plugins, runs sync, renders Temporal env, and matches sequential target applies.

**Change ID:** fixBareOcaApplyLifecycleParity
**Archived:** 2026-05-04T19:07:25.302Z
**Created:** 2026-05-04T18:27:45.012Z

## Tasks Completed

- ✅ [TDD inline] Add failing bare-apply plugin lifecycle test, then implement shared default-target dispatcher so bare `oca apply` invokes plugin prepare/render/sync. Preserve target-specific plugin behavior.
  > Added defaultApplyTargets derived from render.AllTargets + temporal and applyTargetsInOrder loop/switch helper. No-target apply now reuses target-specific lifecycle dispatch. Added bare apply plugin sync regression test; target-specific plugin sync still passes.
- ✅ [TDD inline] Add failing bare-apply Temporal test, then include `temporal` after render targets so bare `oca apply` writes `temporal.env` when enabled.
  > Added TestApplyCommand_BareApplyRendersTemporal verifying bare apply writes temporal.env and emits the path. Existing target-specific Temporal test still passes.
- ✅ [TDD inline] Add failing dry-run read-only/lifecycle-output test, then ensure bare `oca apply --dry-run` shows side-effect lifecycle stages without clone/build/sync/temporal writes.
  > Added bare apply dry-run regression test proving plugin lifecycle stages are reported while plugin checkout and temporal.env are not created. applyPlugins now prints text-mode `would prepare plugins` and `would sync plugins` during dry-run without side effects.
- ✅ [separate verification] Run targeted apply lifecycle tests and `go test ./...`; verify target-specific apply behavior remains unchanged and full suite passes.
  > Integration tests updated to use isolated stack fixtures (writeIsolatedStackExample helper) so bare apply triggers fake plugin remotes instead of real GitHub clones. Full suite go test ./... green. Checkpoint: 385c70b.

## Specs Modified

