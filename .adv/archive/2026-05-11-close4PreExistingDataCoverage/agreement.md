# Agreement — close4PreExistingDataCoverage

## Objectives

Close 4 data-coverage gaps in `internal/migrate/` so the live smoke test `oca migrate from-open-chad → oca debug validate` returns **0 errors** against the operator's actual `~/.config/opencode/opencode.json` + `~/.config/vision/servers.yaml` state. After archive, v1.0.0 tag publication and live cutover are unblocked.

## Discovery findings

| Question | Resolution |
|---|---|
| Are `MinSlots`/`MaxSlots` referenced outside migrate? | **No.** 6 refs total, all in `internal/migrate/{openchad.go,emit.go}`. Safe to delete. |
| What MCP type values appear in operator's live `opencode.json`? | **Only `"remote"`** (verified by grep). Vision yaml has no `type:` field. Translation surface = single `remote → http` mapping. |
| Does `local:` source require tilde expansion at migrate-emit time? | **No.** `IsLocalSource()` is checked at `internal/plugin/prepare.go:29` and returns early — local sources are never cloned/built. The literal string (with or without `~`) is preserved through to runtime. Migrate emit can use either absolute or `~/...` form. Prefer absolute for clarity. |
| Will `local:<path>` validate without `checkout`? | Plugin schema (`internal/config/types.go:236-237`): `source` is required, `checkout` is optional. Local sources don't need `checkout` (Prepare skips). Validator should pass. |

## Scope per gap (carries forward from proposal verbatim)

### Gap 1 — MCP type translation

- Read-time map at `openchad.go:320-321`: `remote` → `http` (only translation needed; warning-passthrough for any other unknown values).
- Servers with `type=remote` and a `url` but no Vision-side port → parse port from URL (`http://host:NNN/...`); fall back to warning + `port=0`.

### Gap 2 — Plugin source fallback

- Path entries not matched by `discoverPlugins` get fallback resolution in `classifyPluginEntry` or a follow-up reader step:
  1. If `<path>/.git` exists → `readGitRemoteURL(path)` → set Source.
  2. Otherwise → `source = "local:<absolute-path>"`.
- Absolute path resolution: expand `~/...` via `os.UserHomeDir()` at classify time so emitted local: values are unambiguous.

### Gap 3 — Slot group schema alignment

- Rewrite `readVisionServers` yaml struct: `{base_port int, count int, group_port int, template string}` (drop `min_slots/max_slots`).
- Update `SlotGroupState`: replace `MinSlots/MaxSlots` with `BasePort/Count` (6 internal callers).
- Update emit (`emit.go:114-130`): write `base_port = N`, `count = N` instead of `min_slots/max_slots`.

### Gap 4 — Path-plugin name derivation

- `classifyPluginEntry` detects trailing `/plugin` suffix in path entries:
  - `/x/foo/plugin` → name=`foo` (use parent dir name)
  - `/x/foo` → name=`foo` (existing `filepath.Base` behavior)
- Aligns Name derivation with `pluginCheckoutMatches` semantics already used by `discoverPlugins`.

## Acceptance gate

Single hard criterion: live smoke test against operator state returns 0 errors.

```sh
oca migrate from-open-chad --output /tmp/oca-cutover-v2.toml
oca debug validate --config /tmp/oca-cutover-v2.toml
# → expected exit 0, "ok" output
```

Plus standard `go test ./... && go vet ./... && gofmt -d .` clean.

## B/F/S/M ambiguity scan

| Category | Coverage | Findings |
|---|---|---|
| B (Boundaries) | C | Out-of-scope explicit (no schema changes, no apply behavior, no v1 tag). Touched scope = `internal/migrate/` + tests only |
| F (Functional Scope) | C | Each gap has explicit success criteria; 17 total |
| S (Completion Signals) | C | Live smoke test = single hard acceptance signal; supplemental coverage via unit + integration tests |
| M (Missing Information) | C | All resolved during discovery (MCP type matrix, MinSlots/MaxSlots dead-code confirmation, local: semantics, tilde-expand policy) |

**No CRITICAL or HIGH ambiguity. Proceeding to design.**

## Constraints

- Worktree-only writes (trunk firewall enforced)
- All tests use isolated test config dirs except final smoke test which intentionally runs against operator state
- No changes to `internal/config/` or `internal/apply/`
- No expansion into npm-source pinning, slot group `servers` semantics, or `migrate init` — all out-of-scope