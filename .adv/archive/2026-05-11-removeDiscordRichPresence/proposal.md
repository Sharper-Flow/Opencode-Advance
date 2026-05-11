## Proposed Approach

Single ADV change, full removal across all layers. Backward-compatible config parse via unknown-section tolerance.

### Phases of work

1. **Wiring removal** — delete `cmd/oca/discord.go`, `internal/discord/`, `assets/discord/`, remove `newDiscordCmd` registration from `cmd/oca/root.go`, drop `rich-go` from `go.mod`
2. **Config schema** — remove `DiscordSection` type and `Discord` field from `Stack`, remove `discord` case from `parse.go` typed dispatch, ensure `[discord]` falls through to unknown-section path (verify or add tolerance)
3. **Status bar** — remove `_oca_discord_update()` function from `lib/status_bar.sh` and its call site
4. **Test cleanup** — delete `cmd/oca/discord_test.go`, `tests/shell/discord_status_hook_test.sh`, `tests/polish_docs_test.go` (entire file appears Discord-centric — verify); update `tests/phase3_types_test.go`, `tests/deferred_sections_test.go`, `tests/readme_release_test.go` to drop Discord assertions
5. **Stack example** — remove `[discord]` block from `stack.example.toml`
6. **Docs** — update README (CLI table + example), STATUS.md, AGENTS.md (tree), `lib/README.md`, `docs/design/architecture.md`, `docs/design/stack-toml-schema.md`, `docs/proposals/v1-implementation.md`
7. **Phase plan rename** — `docs/proposals/phases.md`: rename `phase8ExtrasPolishDiscord` → `phase8ExtrasPolish`, strike Discord deliverables, preserve release-pipeline/README/polish items
8. **Verify** — `go mod tidy`, `gofmt`, `go vet`, `go test ./...`, shell test suite, final `rg -i discord` sweep

### Risk register

- **R1** Test files may contain assertions wedged into surrounding logic (e.g., `phase3_types_test.go` covers more than Discord) — discovery phase must inspect each test before deleting vs editing
- **R2** `rich-go` removal may leave transitive deps stranded — `go mod tidy` handles, but verify
- **R3** Status bar may have brittle layout that breaks when one segment is yanked — visually verify or check tests
- **R4** Phase 8 rename may break ADV/external tracking that keys off the literal `phase8ExtrasPolishDiscord` string — discovery phase to grep for tooling consumers

### Open questions for discovery

- Is `tests/polish_docs_test.go` purely Discord-centric, or does it cover other Phase 8 polish guarantees that must survive?
- Does any CI workflow, dashboard, or external tool key off the `phase8ExtrasPolishDiscord` identifier?
- Does `rich-go` show up in any other code paths (search beyond `internal/discord/`)?