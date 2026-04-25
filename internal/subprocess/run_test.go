package subprocess

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestContractGenericity is the anti-Advance-specific-drift contract test.
// It exercises Run with a synthetic non-build command (echo + sleep via sh),
// proving the runner has no git, pnpm, or Advance-specific behavior baked in.
// Covers AC19.
func TestContractGenericity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("contract test uses POSIX shell")
	}
	ctx := context.Background()
	res, err := Run(ctx, Cmd{
		Name: "sh",
		Args: []string{"-c", "echo hello; echo err >&2; sleep 0"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.ExitClass != ExitSuccess {
		t.Fatalf("ExitClass=%q want=%q output=%q", res.ExitClass, ExitSuccess, res.Output)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode=%d want=0", res.ExitCode)
	}
	if !strings.Contains(string(res.Output), "hello") {
		t.Fatalf("Output missing stdout: %q", res.Output)
	}
	if !strings.Contains(string(res.Output), "err") {
		t.Fatalf("Output missing stderr (combined capture required): %q", res.Output)
	}
	if res.Duration <= 0 {
		t.Fatalf("Duration not recorded: %v", res.Duration)
	}
}

// TestEnvMergeOntoInherited proves Cmd.Env merges onto os.Environ() rather
// than replacing it — a key reuse property for Temporal CLI (Phase 5+).
// Covers AC17.
func TestEnvMergeOntoInherited(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell")
	}
	t.Setenv("OCA_SUBPROC_INHERITED", "inherited-value")
	ctx := context.Background()
	res, err := Run(ctx, Cmd{
		Name: "sh",
		Args: []string{"-c", "echo inherited=$OCA_SUBPROC_INHERITED; echo explicit=$OCA_SUBPROC_EXPLICIT"},
		Env:  map[string]string{"OCA_SUBPROC_EXPLICIT": "explicit-value"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := string(res.Output)
	if !strings.Contains(out, "inherited=inherited-value") {
		t.Fatalf("inherited env missing: %q", out)
	}
	if !strings.Contains(out, "explicit=explicit-value") {
		t.Fatalf("explicit env missing: %q", out)
	}
}

// TestNonZeroExit classifies a non-zero exit as ExitNonZero. Covers AC18.
func TestNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell")
	}
	ctx := context.Background()
	res, err := Run(ctx, Cmd{
		Name: "sh",
		Args: []string{"-c", "echo boom >&2; exit 17"},
	})
	// Run returns the Result even on non-zero exit; err is non-nil for ExitNonZero.
	if err == nil {
		t.Fatalf("expected error for non-zero exit, got nil")
	}
	if res.ExitClass != ExitNonZero {
		t.Fatalf("ExitClass=%q want=%q", res.ExitClass, ExitNonZero)
	}
	if res.ExitCode != 17 {
		t.Fatalf("ExitCode=%d want=17", res.ExitCode)
	}
	if !strings.Contains(string(res.Output), "boom") {
		t.Fatalf("Output missing stderr: %q", res.Output)
	}
}

// TestTimeoutClassification proves Timeout → ExitTimeout and kills the
// process rather than hanging. Covers AC18.
func TestTimeoutClassification(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell")
	}
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}
	ctx := context.Background()
	start := time.Now()
	res, err := Run(ctx, Cmd{
		Name:    "sleep",
		Args:    []string{"30"},
		Timeout: 200 * time.Millisecond,
	})
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("runner did not enforce timeout: elapsed=%v", elapsed)
	}
	if err == nil {
		t.Fatalf("expected error for timeout, got nil")
	}
	if res.ExitClass != ExitTimeout {
		t.Fatalf("ExitClass=%q want=%q (elapsed=%v)", res.ExitClass, ExitTimeout, elapsed)
	}
}

// TestCancellationViaContext proves ctx cancellation terminates the process.
// Covers AC17 (context honored).
func TestCancellationViaContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell")
	}
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	res, err := Run(ctx, Cmd{
		Name: "sleep",
		Args: []string{"30"},
	})
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("runner ignored context cancellation: elapsed=%v", elapsed)
	}
	if err == nil {
		t.Fatalf("expected error for cancelled ctx, got nil")
	}
	if res.ExitClass != ExitSignal && res.ExitClass != ExitTimeout {
		// Accept either depending on whether the ctx deadline or cancel propagated first.
		t.Fatalf("ExitClass=%q want ExitSignal or ExitTimeout", res.ExitClass)
	}
}

// TestMissingExecutable returns a structured error, not a panic.
func TestMissingExecutable(t *testing.T) {
	ctx := context.Background()
	res, err := Run(ctx, Cmd{
		Name: "this-command-should-not-exist-on-any-host-xyzzy",
	})
	if err == nil {
		t.Fatalf("expected error for missing executable")
	}
	if res.ExitClass == ExitSuccess {
		t.Fatalf("ExitClass must not be success when executable missing: %q", res.ExitClass)
	}
}
