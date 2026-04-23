# tests/

This directory holds tests that span multiple Go packages or require shell integration.

Go unit tests live next to their source code (e.g., `internal/config/types_test.go`), not here. This directory is for:

## Integration tests (Go)

| File             | Scope                                                       |
| ---------------- | ----------------------------------------------------------- |
| `apply_test.go`    | End-to-end `oca apply` against an isolated test environment |
| `doctor_test.go`   | End-to-end `oca doctor` with mock MCP servers             |
| `migrate_test.go`  | Migration from a canned open-chad state snapshot           |
| `plugin_test.go`   | Plugin lifecycle (clone from local mock git remote)        |

## Shell integration tests

| File                        | Scope                                              |
| --------------------------- | -------------------------------------------------- |
| `shell/brand_helpers_test.sh` | Palette, wordmark rendering across terminal types  |
| `shell/verification_workflow_test.sh` | CI verification workflow                      |
| `shell/session_test.sh`       | tmux session lifecycle (planned, Phase 4 foundation) |
| `shell/boot_splash_test.sh`   | Boot splash rendering on different terminal types (planned, Phase 4 richness) |
| `shell/status_bar_test.sh`    | Status bar renderer output (planned, Phase 4 richness) |

## Test isolation policy

Every test MUST use isolated config directories. Never touch `~/.config/opencode/`, `~/.config/vision/`, or `~/.tmux.conf`.

Use `t.TempDir()` in Go tests, `mktemp -d` in bash tests, and honor the `OCA_*_DIR` environment overrides throughout.

## Status

Populated starting in Phase 0 (shell brand helper tests) and Phase 1 (Go integration tests). Phase 4 foundation adds session integration tests (Go, in `internal/session/session_test.go` and `cmd/oca/` tests). Shell-level session/boot-splash tests are planned for Phase 4 richness.
