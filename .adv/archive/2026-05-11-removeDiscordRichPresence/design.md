## Design Strategy

This change is **mechanical removal with one architectural decision**: how to preserve backward-compat parsing of legacy `[discord]` blocks.

**Validator status: VALIDATED** by `adv-researcher` (3 notes incorporated below).

### Architecture Decision: Backward-compat parse strategy

**Decision:** Reuse the existing `DeferredSections` machinery in `internal/config/parse.go`.

**Mechanism:**
- Keep `knownSections["discord"] = true` (line 36 of `parse.go`)
- Delete the `case "discord":` typed dispatch (lines 243–248 of `parse.go`)
- Delete `DiscordSection` type and `Stack.Discord` field from `internal/config/types.go`
- `[discord]` then falls through to `default:` (line 264) → lands in `stack.DeferredSections["discord"]`
- `Validate()` at `internal/config/validate.go:145-152` already tolerates known-deferred names without error

**Why this approach:**
- Zero new code paths — uses an already-tested, already-validated mechanism (`agents` is the canonical precedent: in `knownSections`, no typed dispatch, lands in `DeferredSections`)
- Validator confirmed: `parse_test.go:81` already asserts `agents` stays deferred — same pattern works for `discord`
- One-line schema change (delete dispatch case) + one-line field deletion
- No code reads `Stack.Discord` outside the deleted Discord package (validator grep confirmed)

**Rejected alternatives:**
- **Remove from `knownSections`** — hard-fails legacy configs, violates silent-tolerance decision
- **Explicit `case "discord": /* no-op */`** — duplicates default arm
- **Emit deprecation warning** — out of scope

### Test surgery strategy (validator-refined)

| File | Action |
|---|---|
| `cmd/oca/discord_test.go` | **Delete entire file** |
| `internal/discord/presence_test.go` | **Delete entire file** |
| `internal/discord/taglines_test.go` | **Delete entire file** |
| `tests/shell/discord_status_hook_test.sh` | **Delete entire file** |
| `tests/polish_docs_test.go` | **Surgical edit:** delete `TestPolishDocsRemoveDeferredDiscordReferences`; update `TestPhase8RoadmapDocsShowReleaseCandidate` to assert `phase8ExtrasPolish`; keep `TestDocsNoStaleAgentNames` |
| `tests/phase3_types_test.go` | **Surgical edit:** ① delete `stack.Discord` assertions (lines 374–376); ② update the `remaining` list (line 368) from `[]string{"agents"}` to `[]string{"agents", "discord"}` so the test now asserts `[discord]` lands in `DeferredSections` — gives positive AC3 evidence in-place; ③ keep `[discord]` block in the test fixture as the legacy-stack proof |
| `tests/deferred_sections_test.go` | **Surgical edit:** remove `discord` from typed-sections list (line 19); delete `stack.Discord` assertions (lines 24–31); add positive assertion that `[discord]` from `stack.example.toml`'s legacy form lives in `DeferredSections` after removal — note: this requires keeping a legacy `[discord]` example somewhere parseable, see Stack-example note below |
| `tests/readme_release_test.go` | **Surgical edit:** remove `"oca discord enable"` line (line 22) |

### Stack example handling

`stack.example.toml` currently has `[discord]` (line 406+). Per the proposal, this block is **removed** from the example. Backward-compat test evidence must therefore use a hand-rolled in-test TOML literal (already the case in `phase3_types_test.go`'s fixture) — not `stack.example.toml`. Update `tests/deferred_sections_test.go` accordingly: it currently reads `stack.example.toml`; once `[discord]` is gone from that file, the test's positive assertion needs an in-test fixture instead.

### Migration code cleanup (P21 campsite + P25 related-scan)

Validator identified `internal/migrate/openchad.go` `DiscordState` (lines 49, 132–135, 515–517) and its test (`openchad_test.go:140-145`) as dead code post-removal: the migration reads Discord config from `open-chad` source but never emits a `[discord]` section to the output `stack.toml`. Including this cleanup in the same change because:

- **Adjacent scope** — same domain (Discord), same change concept, identifiable by `rg -i discord`
- **P21 campsite-rule** — opportunistic cleanup of clearly-stale code that is local and safe
- **P25 related-scan** — leaving the migration code orphaned guarantees a follow-up "remove dead Discord code" change

**Action:** Delete `DiscordState` type, its assignment in `openchad.go:132-135`, the function reading it (around line 515-517), and the corresponding `openchad_test.go` assertions. Confirm `internal/migrate/emit.go` is unaffected (validator already verified it never writes `[discord]`).

### Documentation strategy

| File | Edit class |
|---|---|
| `README.md` | Remove CLI table rows + `[discord]` block from example |
| `STATUS.md` | Remove Phase 0 brand-list tagline mention + Phase 8 Discord deliverable line |
| `AGENTS.md` | Remove `internal/discord/` and `assets/discord/` repo-tree entries |
| `lib/README.md` | Remove "Discord runtime code now lives in…" sentence |
| `docs/design/architecture.md` | Remove from typed-section list; remove `Discord` rendering description |
| `docs/design/stack-toml-schema.md` | Remove `[discord]` row from header table; remove `## [discord]` section + body |
| `docs/proposals/phases.md` | Rename `phase8ExtrasPolishDiscord` → `phase8ExtrasPolish`; strike Discord deliverables; preserve release-pipeline + final-README + polish items |
| `docs/proposals/v1-implementation.md` | Update phase reference; remove "Discord taglines are rewritten" criterion; remove brand-narrative Discord taglines mention; remove Phase 8 checklist Discord items |

### Status bar surgery (R3 resolved)

- `_oca_discord_update()` (lines 68–74 of `lib/status_bar.sh`) is fire-and-forget background invocation with `>/dev/null 2>&1 &` — no stdout, no visible status content
- Call site at line 208 inside `oca_status_row0()` is a side-effect call
- Removing both does not change a single character of rendered tmux output

### Dependency cleanup

- Delete `internal/discord/` package → `go mod tidy` removes `github.com/hugolgst/rich-go` direct dep + likely `gopkg.in/natefinch/npipe.v2` indirect

### Asset cleanup

- Delete `assets/discord/` directory (only contains `taglines.toml`)

### Verification approach (AC1–AC8)

1. `go build ./...` — clean
2. `go vet ./...` — clean
3. `gofmt -d .` — clean
4. `go test ./...` — full suite pass
5. Shell test suite — `tests/shell/*_test.sh` (minus deleted `discord_status_hook_test.sh`)
6. `oca --help | grep -i discord` — zero hits
7. Backward-compat test in `phase3_types_test.go` + `deferred_sections_test.go` proves AC3
8. `rg -i discord .` (excluding `.adv/archive/`, `.git/`, `node_modules/`, change-metadata) — zero hits in tracked source

### Implementation order (keeps build green between commits)

Sequencing to minimize the broken-build window across granular commits. (A single atomic-PR merge to trunk satisfies P32 regardless; this ordering is to make individual task checkpoints viable.)

1. Test edits to mixed-concern files (drop assertions that depend on `Stack.Discord`)
2. Delete pure-Discord test files (`cmd/oca/discord_test.go`, `internal/discord/*_test.go`, `tests/shell/discord_status_hook_test.sh`)
3. Delete `cmd/oca/discord.go` + remove `newDiscordCmd` registration in `cmd/oca/root.go`
4. Delete `internal/discord/` package + `assets/discord/`
5. Remove `DiscordSection` type + `Stack.Discord` field in `internal/config/types.go`
6. Remove typed dispatch case in `internal/config/parse.go` (keep entry in `knownSections`)
7. Update `tests/phase3_types_test.go` `remaining` list (`agents` → `agents, discord`); update `tests/deferred_sections_test.go` positive backward-compat assertion
8. Delete migration `DiscordState` in `internal/migrate/openchad.go` + corresponding test assertions
9. Remove `_oca_discord_update` from `lib/status_bar.sh`
10. Remove `[discord]` block from `stack.example.toml`
11. Doc edits (README, STATUS, AGENTS, lib/README, architecture, stack-toml-schema)
12. Phase 8 rename in `docs/proposals/phases.md` + `docs/proposals/v1-implementation.md`
13. `go mod tidy` + verify go.sum
14. Final verification: `go test ./...`, shell tests, `rg -i discord`

### Final risk register

| ID | Risk | Status |
|---|---|---|
| R1 | Mixed-concern test files | Resolved — surgical-edit strategy per file |
| R2 | Transitive deps | Resolved — `go mod tidy` |
| R3 | Status bar layout | Resolved — fire-and-forget hook, no visible output |
| R4 | External identifier consumers | Resolved — zero external refs |
| R5 | Atomic mid-removal build break | Mitigated by implementation order; final PR is atomic |
| R6 (new from validator) | `phase3_types_test.go` `remaining` list misses `discord` | Incorporated — explicit fix in step 7 |
| R7 (new from validator) | `internal/migrate/openchad.go` `DiscordState` orphaned | Incorporated — explicit deletion in step 8 |

### Out of scope (re-affirmed)

No replacement system. No Phase 8 re-numbering. No `$XDG_RUNTIME_DIR/.../discord/` user-cache cleanup. No `oca apply` deprecation-warning emitter.