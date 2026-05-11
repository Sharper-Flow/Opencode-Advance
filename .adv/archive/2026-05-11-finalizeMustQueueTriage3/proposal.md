## Cross-Project Origin

This change was created as a follow-up from **opencode-advance**.

| Field | Value |
|-------|-------|
| Source project | opencode-advance |
| Source path | `/home/jrede/dev/opencodeadvance` |

> **Note:** The originating project should be consulted for context on why this change is needed.


## Success Criteria

### M4 — Instruction asset triage + parity check

- [ ] Each of `criteria-prioritizer.md`, `global-verify-policy.md`, `post_install_verification.md` has an explicit triage decision recorded in change wisdom: copied to `assets/instructions/` / retired from live dir / re-homed to skill or plugin.
- [ ] If copied: `assets/instructions/README.md` inventory updated and matches actual directory contents byte-for-byte.
- [ ] New parity test (Go or shell) fails if `assets/instructions/README.md` inventory drifts from actual contents of `assets/instructions/`.
- [ ] `oca apply --target instructions` against an isolated test config dir writes all currently-owned files without error.

### M5 — Migration TOML key sanitization

- [ ] `oca migrate from-open-chad --output <path>` against the operator's live open-chad state emits a TOML file that passes `cfg.Load()` / `oca debug validate` without hand-editing.
- [ ] Migrator strips npm version specs (`@<version>`, `@latest`, `@<sha>`) from plugin TOML keys; original spec preserved in the table body via `ref` or equivalent field where appropriate.
- [ ] New unit test in `internal/migrate/` covers npm-spec plugin name sanitization with fixtures for `@latest`, `@<semver>`, `@<sha>`, and plain (no-spec) cases.
- [ ] New integration test exercises the full `from-open-chad → cfg.Load() → debug validate` round-trip against a canned open-chad state snapshot.
- [ ] Other TOML key hazards (spaces, dots, brackets in plugin names) are scanned for and either sanitized or explicitly rejected with a clear error.

### M6 — Banned-term enforcement

- [ ] New docs test (Go test under `tests/` or similar) fails if any OCA-owned skill file contains banned legacy terms: `openchad`, `open-chad`, `oc switch` (excluding explicit historical-migration markers).
- [ ] Same test fails on banned legacy MCP function names in OCA-owned skills: `kagi_search_fetch`, `firecrawl_scrape`, `context7_resolve_library_id` (underscore form).
- [ ] Same test fails on claims of deleted ADV methodology skills: `adv-review-methodology`, `adv-apply-methodology`, `adv-harden-methodology`, `adv-discover-methodology`, `adv-prep-methodology` outside explicit inlined-and-deleted notes.
- [ ] `assets/skills/README.md` "Not in this directory" section optionally regenerable from a verifiable listing of `~/dev/oc-plugins/advance/skills/` (acceptable to defer regeneration tooling if the banned-term test already catches drift).

### Global

- [ ] `go test ./...` passes after all M4/M5/M6 work.
- [ ] `go vet ./...` clean.
- [ ] `gofmt -d .` clean.
- [ ] All file writes during tests use isolated config directories (no touches to `~/.config/opencode/`).

## Approach

1. **M5 first** (real bug, real blocker). Add red test reproducing the `@`-in-key parse failure. Fix `internal/migrate/openchad.go` plugin-key emitter to sanitize npm specs. Add round-trip test against a canned open-chad snapshot.
2. **M4 triage**. Read each of the three flagged files; decide copy/retire/re-home per content. For "copy" decisions, add to `assets/instructions/` and update README. Add parity test.
3. **M6 enforcement**. Add a Go test scanning `assets/skills/**/*.md` for banned terms/names/skill-claims. Confirm current content passes; assert future-proofing.
4. Run full `go test ./... && go vet ./... && gofmt -d .` clean.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| M4 triage decisions wrong (file actually still needed live) | Low | Medium | Each decision recorded in change wisdom with rationale; live file retired only after triage decision |
| M5 sanitization breaks unrelated plugin name forms | Low | Medium | New tests cover full matrix: plain name, `@latest`, `@<semver>`, `@<sha>`, dot-in-name, dash-in-name |
| M6 banned-term test produces false positives on legitimate references | Medium | Low | Allow-list explicit historical-migration markers; test is strict-fail with clear path-of-offense output |
| Consolidated scope tempts mid-change scope creep | Low | Low | Treat as three sub-tasks with TDD per task; do not expand into broader migration refactor or skill rewrite |