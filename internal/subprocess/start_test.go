package subprocess

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStartBackgroundCapturesOutputArgsEnvAndProcessGroup(t *testing.T) {
	tmp := t.TempDir()
	recorder := filepath.Join(tmp, "recorder")
	argsFile := filepath.Join(tmp, "args.txt")
	logFile := filepath.Join(tmp, "process.log")
	script := `#!/bin/sh
printf 'args:%s|%s\n' "$1" "$2" > "$RECORD_ARGS"
printf 'env:%s\n' "$OCA_TEST_VALUE" >> "$RECORD_ARGS"
printf 'stdout-line\n'
printf 'stderr-line\n' >&2
sleep 30
`
	if err := os.WriteFile(recorder, []byte(script), 0o755); err != nil {
		t.Fatalf("write recorder: %v", err)
	}

	proc, err := StartBackground(context.Background(), Cmd{
		Name: recorder,
		Args: []string{"literal space", "semi;colon"},
		Env: map[string]string{
			"RECORD_ARGS":    argsFile,
			"OCA_TEST_VALUE": "kept",
		},
	}, StartOptions{OutputPath: logFile, SetProcessGroup: true})
	if err != nil {
		t.Fatalf("StartBackground: %v", err)
	}
	t.Cleanup(func() { _ = syscall.Kill(-proc.PID, syscall.SIGKILL) })

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(argsFile); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("recorder did not write args file")
		}
		time.Sleep(25 * time.Millisecond)
	}

	argsBytes, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	argsText := string(argsBytes)
	if !strings.Contains(argsText, "args:literal space|semi;colon") {
		t.Fatalf("argv not preserved literally: %q", argsText)
	}
	if !strings.Contains(argsText, "env:kept") {
		t.Fatalf("env not propagated: %q", argsText)
	}

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	logText := string(logBytes)
	if !strings.Contains(logText, "stdout-line") || !strings.Contains(logText, "stderr-line") {
		t.Fatalf("combined output missing stdout/stderr: %q", logText)
	}

	pgid, err := syscall.Getpgid(proc.PID)
	if err != nil {
		t.Fatalf("Getpgid: %v", err)
	}
	if pgid != proc.PID || proc.ProcessGroupID != proc.PID {
		t.Fatalf("process group mismatch: kernel=%d returned=%d pid=%d", pgid, proc.ProcessGroupID, proc.PID)
	}
}

func TestStartBackgroundMissingExecutableReturnsError(t *testing.T) {
	_, err := StartBackground(context.Background(), Cmd{Name: filepath.Join(t.TempDir(), "missing")}, StartOptions{})
	if err == nil {
		t.Fatal("StartBackground missing executable err=nil")
	}
}
