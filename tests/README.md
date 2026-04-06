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
| `shell/install_test.sh`       | `oca install` against a chroot'd fresh Ubuntu       |
| `shell/session_test.sh`       | tmux session lifecycle                             |
| `shell/boot_splash_test.sh`   | Boot splash rendering on different terminal types  |
| `shell/status_bar_test.sh`    | Status bar renderer output                         |

## Test isolation policy

Every test MUST use isolated config directories. Never touch `~/.config/opencode/`, `~/.config/vision/`, or `~/.tmux.conf`.

Use `t.TempDir()` in Go tests, `mktemp -d` in bash tests, and honor the `OCA_*_DIR` environment overrides throughout.

## Status

Empty — populated progressively in each phase. Phase 1 adds the first integration tests for `oca apply --target mcp`.
