package subprocess

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Run executes c and returns a populated Result. The returned error is
// non-nil for any non-success ExitClass, so callers can use a single
// error check while still inspecting Result for the precise outcome.
//
// Design reference: design.md § K1. Canonical 2026 pattern
// (exec.CommandContext + CombinedOutput + Cmd.Cancel) per agent decision A2.
func Run(ctx context.Context, c Cmd) (Result, error) {
	start := time.Now()

	// Build the exec context. When Cmd.Timeout is set we derive a bounded
	// child ctx so the runner controls cancellation; callers still keep
	// their parent ctx responsive.
	execCtx := ctx
	var cancelTimeout context.CancelFunc
	if c.Timeout > 0 {
		execCtx, cancelTimeout = context.WithTimeout(ctx, c.Timeout)
		defer cancelTimeout()
	}

	cmd := exec.CommandContext(execCtx, c.Name, c.Args...)
	if c.Dir != "" {
		cmd.Dir = c.Dir
	}
	cmd.Env = mergeEnv(os.Environ(), c.Env)

	// Combined stdout+stderr capture, per AC17/AC18. A single bytes.Buffer
	// is safe because exec serializes writes from the child process's
	// OS-level handles into the Go io.Writer.
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	// Cmd.Cancel fires when the derived context is cancelled OR the
	// timeout elapses. Send SIGTERM first for clean shutdown.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	// Cmd.WaitDelay is the SIGTERM → SIGKILL grace window enforced by Go's
	// os/exec. 0 means kill immediately after Cancel; a positive value
	// allows the child to exit cleanly first.
	grace := c.KillGrace
	if grace == 0 {
		grace = DefaultKillGrace
	}
	cmd.WaitDelay = grace

	// Classify errors from cmd.Run() into ExitClass. Order matters:
	//   1. context cancellation vs timeout (cmd.ProcessState may be nil)
	//   2. ExitError (process ran, non-zero code)
	//   3. exec.Error (command not found, permission denied)
	//   4. other (uncategorised; treat as ExitSignal)
	runErr := cmd.Run()
	duration := time.Since(start)

	res := Result{
		Output:   buf.Bytes(),
		Duration: duration,
	}

	// Extract exit code defensively — ProcessState is nil if the process
	// never started (missing executable).
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		res.ExitCode = -1
	}

	switch {
	case runErr == nil:
		res.ExitClass = ExitSuccess
		return res, nil

	case errors.Is(execCtx.Err(), context.DeadlineExceeded):
		res.ExitClass = ExitTimeout
		return res, fmt.Errorf("subprocess %s: timeout after %v", c.Name, c.Timeout)

	case errors.Is(execCtx.Err(), context.Canceled):
		// Parent cancellation propagates into execCtx (derived via
		// WithTimeout), so a single check on execCtx.Err() covers both
		// direct cancellation of execCtx and inherited cancellation from
		// the caller's ctx. Distinguished from DeadlineExceeded above.
		res.ExitClass = ExitSignal
		return res, fmt.Errorf("subprocess %s: cancelled", c.Name)

	default:
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			// Process ran to completion with non-zero status. Detect
			// signalled exit by examining Sys() on platforms that support it.
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				res.ExitClass = ExitSignal
				return res, fmt.Errorf("subprocess %s: terminated by signal %v", c.Name, ws.Signal())
			}
			res.ExitClass = ExitNonZero
			return res, fmt.Errorf("subprocess %s: exit %d", c.Name, res.ExitCode)
		}

		// exec.Error is returned for cases like ENOENT / EACCES before
		// the process starts. Treat as non-zero with a synthetic code
		// so callers can distinguish from process-initiated failures.
		var execErr *exec.Error
		if errors.As(runErr, &execErr) {
			res.ExitClass = ExitNonZero
			return res, fmt.Errorf("subprocess %s: %w", c.Name, runErr)
		}

		res.ExitClass = ExitSignal
		return res, fmt.Errorf("subprocess %s: %w", c.Name, runErr)
	}
}

// mergeEnv folds explicit key/value pairs onto a parent env slice. When
// a key in extra already exists in parent, the extra value wins; keys
// only present in parent are preserved. Empty extra returns parent as-is
// (avoid allocation on the common inheritance-only case).
func mergeEnv(parent []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return parent
	}
	out := make([]string, 0, len(parent)+len(extra))
	// Build a set of keys we're overriding so we can skip them from parent.
	override := make(map[string]struct{}, len(extra))
	for k := range extra {
		override[k] = struct{}{}
	}
	for _, pair := range parent {
		// KEY=VALUE — split on first '='; malformed entries are passed through unchanged.
		for i := 0; i < len(pair); i++ {
			if pair[i] == '=' {
				if _, skip := override[pair[:i]]; skip {
					goto next
				}
				break
			}
		}
		out = append(out, pair)
	next:
	}
	for k, v := range extra {
		out = append(out, k+"="+v)
	}
	return out
}
