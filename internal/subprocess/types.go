// Package subprocess is a generic external-command runner for OpenCode
// Advance. It has no git, pnpm, or Advance-specific behavior — domain
// wrappers (internal/plugin/git.go, internal/plugin/build.go,
// internal/sync/advance.go) call Run with appropriately shaped Cmd
// arguments.
//
// Design reference: design.md § K1 (subprocess is one generic package;
// domain wrappers call it). Reused by Temporal CLI health checks (Phase 5+).
package subprocess

import "time"

// ExitClass classifies how a subprocess invocation terminated. Callers use
// it to decide whether to retry, escalate, or treat the output as authoritative.
type ExitClass string

const (
	// ExitSuccess: the process ran to completion with exit code 0.
	ExitSuccess ExitClass = "success"
	// ExitNonZero: the process ran to completion with a non-zero exit code.
	// The caller owns interpretation — some tools signal warnings this way.
	ExitNonZero ExitClass = "non-zero-exit"
	// ExitTimeout: Cmd.Timeout elapsed before the process finished.
	// The runner sent SIGTERM, then SIGKILL after KillGrace.
	ExitTimeout ExitClass = "timeout"
	// ExitSignal: the process was terminated by a signal other than the
	// runner's own timeout-induced SIGTERM (e.g. context cancellation, OS kill).
	ExitSignal ExitClass = "signal"
)

// Cmd describes a single subprocess invocation. All fields are optional
// except Name.
type Cmd struct {
	// Name is the executable. Passed to exec.LookPath unless absolute.
	Name string
	// Args are argv[1:]. Never shell-interpreted — we do not invoke sh.
	// Callers that need shell features pass "sh" as Name with "-c" + script.
	Args []string
	// Dir sets the working directory. Empty = inherit caller's cwd.
	Dir string
	// Env is merged onto os.Environ() — explicit keys override inherited
	// values; inherited keys not listed in Env remain. Empty map = pure
	// inheritance.
	Env map[string]string
	// Timeout is the wall-clock budget. 0 = no timeout (context cancel
	// is still honored).
	Timeout time.Duration
	// KillGrace is the delay between SIGTERM and SIGKILL on timeout/cancel.
	// 0 = DefaultKillGrace.
	KillGrace time.Duration
}

// DefaultKillGrace is the SIGTERM → SIGKILL delay when Cmd.KillGrace is 0.
const DefaultKillGrace = 5 * time.Second

// Result carries the full outcome of one Run call. Output is always
// captured (combined stdout + stderr) regardless of ExitClass.
type Result struct {
	ExitClass ExitClass
	ExitCode  int // os.ProcessState.ExitCode(); -1 if the process was signalled or never started
	Output    []byte
	Duration  time.Duration
}
