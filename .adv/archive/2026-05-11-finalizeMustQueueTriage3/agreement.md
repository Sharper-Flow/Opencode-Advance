# Agreement — finalizeMustQueueTriage3

## Objectives

Ship the last three pre-cutover correctness fixes (M4 instruction triage, M5 migration npm-spec bug, M6 banned-term enforcement) as a single 7-gate ADV change. After archive, OCA is ready for `v1.0.0` tag + open-chad replacement.

## Triage decisions (resolved during discovery)

Discovery peeked at the 3 candidate live files and resolved their ownership:

| File | Lines | Content | Decision | Rationale |
|---|---|---|---|---|
| `criteria-prioritizer.md` | 219 | Prioritizer sub-agent trigger guide | **retire** | Subsumed by `assets/skills/prioritizer/SKILL.md` which is the canonical OCA-owned skill |
| `global-verify-policy.md` | 1 | `/check` policy one-liner | **copy to `assets/instructions/`** | Actively referenced; OCA-owned environment policy |
| `post_install_verification.md` | 103 | Open-chad-specific install verification | **retire** | Open-chad legacy; `oca doctor` + `oca install` supersede this |

Retired files are removed from the live `~/.config/opencode/instructions/` on the next `oca apply --target instructions` (since they no longer exist in OCA's source-of-truth). No proactive deletion required from this change.

## M5 root cause (resolved during discovery)

- `internal/migrate/openchad.go:251-258` — `readOpenCodeJSON` skips `@scoped` npm packages but does NOT handle `name@version` form. `filepath.Base("opencode-openai-codex-auth@latest")` returns the full string verbatim.
- `internal/migrate/emit.go:138` — emits `[plugins.%s]` directly with no key sanitization.

**Chosen fix:** sanitize at emit boundary. Strip `@<spec>` from the TOML table key; preserve original spec in `ref` field if version-pin was meaningful. Single chokepoint catches all plugin sources (current + future). Also covers other TOML-hostile chars (spaces, dots in name, brackets) by adding a `sanitizePluginKey` helper that errors loudly if a name cannot be made TOML-safe.

## Acceptance criteria

Same 17 success criteria from the proposal carry forward verbatim. Discovery did not adjust scope.

## B/F/S/M ambiguity scan

| Category | Coverage | Findings |
|---|---|---|
| B (Boundaries) | C | Out-of-scope explicit (no cutover, no migrate refactor, no v1 tag, no ADV-owned skill changes). Touched scope = `internal/migrate/`, `assets/instructions/`, `assets/skills/`, `tests/` |
| F (Functional Scope) | C | Each M-item has explicit success criteria; 17 total |
| S (Completion Signals) | C | All measurable; `go test ./...`, `go vet`, `gofmt -d .` clean; specific file/test paths cited |
| M (Missing Information) | C | All resolved during discovery (3 triage decisions + M5 root cause) |

**No CRITICAL or HIGH ambiguity. Proceeding to design.**

## Constraints

- Worktree-only writes (trunk firewall enforced)
- All tests use isolated test config dirs (no touches to `~/.config/opencode/`)
- No changes to ADV-owned files
- No regressions in existing migration behavior for non-npm plugins (plain paths, scoped npm, git checkouts)