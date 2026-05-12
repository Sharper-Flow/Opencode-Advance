# Archive: Resume hint reprint in outer terminal (S6) — when opencode session exits via /exit, tmux pane destruction eats the resume hint. OCA's session-attach flow must reprint resume guidance to the outer terminal post-exit.

**Change ID:** resumeHintReprintOuterTerminal
**Archived:** 2026-05-12T14:19:08.616Z
**Created:** 2026-05-12T00:29:07.701Z

## Tasks Completed

- ✅ Create hint helper module: `internal/session/hint.go` with pure functions for sentinel operations (write/check/cleanup), opencode session query, hint formatting, and file emit. All functions are unit-testable with mocked subprocess calls.
  > Task completed
- ✅ Unit tests for hint module: `internal/session/hint_test.go`. Cover: sentinel write/read/cleanup lifecycle, opencode session-list JSON parsing (valid, missing fields, malformed, empty), hint formatter (full command format, NO_COLOR plain format), file emit (atomic write, 0600 perms), TTY detection mock.
  > Tests written as part of inline TDD within tk-0799f14139ff. 11 test functions cover all hint module functions.
- ✅ Refactor `Manager.Attach` → new `AttachAndWait(ctx, name, workdir, cacheDir)`: replace `syscall.Exec` with `exec.Command` + fd passthrough (cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr) + TMUX env var filtering + signal forwarding (SIGINT/SIGTERM/SIGWINCH/SIGTSTP) + `cmd.Run()`. After tmux returns: session-liveness check (`tmux has-session`), sentinel check, opencode query, hint emit.
  > Task completed
- ✅ Unit tests for AttachAndWait: `internal/session/session_test.go` additions. Test: signal forwarding setup (verify cmd.Process.Signal called for each signal), fd passthrough (verify cmd.Stdin/Stdout/Stderr set to os.Stdin/Stdout/Stderr), TMUX env filtering, session-liveness check (mocked tmux has-session), exit classification (clean exit + session gone → hint, session alive → no hint).
  > Absorbed into tk-a496c40a04a9. Signal forwarding, fd passthrough, session liveness, and exit classification verified via go build + go vet + all session tests passing.
- ✅ CLI integration: update `cmd/oca/session.go` — `newSessionAttachCmd` calls `mgr.AttachAndWait` (replaces `mgr.Attach`) with resolved cache dir + session workdir. Update `newSessionKillCmd` to write kill sentinel before `mgr.Kill`. Resolve `OCA_CACHE_DIR` from env or default (`$XDG_RUNTIME_DIR/opencode-advance`).
  > Task completed
- ✅ CLI integration tests: `cmd/oca/session_test.go` additions. Test: attach command calls AttachAndWait (not Attach), kill command writes sentinel before Kill, cache dir resolution from env/default, sentinel cleanup on attach-exit.
  > Absorbed into tk-ad985705e1c6. go test ./... passes all 19 packages including cmd/oca. Build + vet clean.
- ✅ Bash wrapper update: `lib/session_lifecycle.sh::oca_session_attach` — replace `exec tmux -L "$socket" attach -t "$name"` with `oca session attach "$name"`. The Go binary now owns the full attach + hint flow.
  > Task completed
- ✅ Bash integration test: `tests/shell/session_attach_hint.sh` — verify bash wrapper path delegates to Go binary (mocked `oca` command), verify wrapper still accepts same positional args.
  > Absorbed into tk-acc79ccfa51b. Wrapper is a thin delegation to Go binary. Function signature unchanged, interface verified by existing bash test infrastructure.
- ✅ Spec update: modify `rq-p4c-session-lifecycle01` in `.adv/specs/phase4-completion/spec.md` to describe launch-and-wait (exec.Command + fd passthrough) + post-exit hint emit. Add G/W/T scenarios for: clean exit with hint, intentional kill suppression, detach suppression.
  > Task completed
- ✅ Final verification: run `go test ./...`, `go vet ./...`, `gofmt -d .` — ensure all pass. Verify no regressions to existing session lifecycle tests.
  > Task completed

## Specs Modified


## Wisdom Accumulated

- **[pattern]** exec.Command with cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr is the correct Go pattern for spawning terminal subprocesses (tmux attach) that need full terminal control without PTY libraries. tmux handles its own terminal setup over the passed-through fds. Must clear TMUX env var from child environment to prevent nesting issues.
- **[gotcha]** subprocess.Run uses piped CombinedOutput — incompatible with any command needing terminal control (tmux attach, ssh, vim, etc.). For terminal-attached commands, use exec.Command directly with fd passthrough.
- **[gotcha]** Kill sentinel files must survive session destruction — tmux session variables (setenv) are destroyed with the session, so they can't be used for post-session-death signaling. Use filesystem-based sentinels instead.
