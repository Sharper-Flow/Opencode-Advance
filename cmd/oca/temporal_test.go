package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
	itemporal "github.com/Sharper-Flow/Opencode-Advance/internal/temporal"
)

func TestTemporalStatusJSONShape(t *testing.T) {
	tmp := t.TempDir()
	stackPath := writeTemporalStack(t, tmp)
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))
	withTemporalSupervisorFactory(t, func() itemporal.Supervisor {
		return itemporal.Supervisor{Reachable: func(context.Context, string) bool { return false }}
	})

	var stdout bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stdout, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--config", stackPath, "--output", "json", "temporal", "status"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got itemporal.Status
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json: %v output=%q", err, stdout.String())
	}
	if !got.Configured || !got.Enabled || got.Address != "127.0.0.1:7233" || got.Namespace != "default" || got.State != itemporal.StateStopped {
		t.Fatalf("status mismatch: %#v", got)
	}
	if got.LogPath == "" || got.DBPath == "" {
		t.Fatalf("paths missing: %#v", got)
	}
}

func TestTemporalStartTextOutput(t *testing.T) {
	tmp := t.TempDir()
	stackPath := writeTemporalStack(t, tmp)
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))
	started := false
	withTemporalSupervisorFactory(t, func() itemporal.Supervisor {
		return itemporal.Supervisor{
			DetectCLI: func(context.Context) (itemporal.DetectResult, error) {
				return itemporal.DetectResult{Path: "/bin/temporal", Version: "temporal version test"}, nil
			},
			StartBackground: func(context.Context, subprocess.Cmd, subprocess.StartOptions) (subprocess.BackgroundProcess, error) {
				started = true
				return subprocess.BackgroundProcess{PID: os.Getpid(), ProcessGroupID: os.Getpid()}, nil
			},
			Reachable: func(context.Context, string) bool { return started },
			Healthy:   func(context.Context, *cfg.Stack) bool { return true },
		}
	})

	var stdout bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stdout, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--config", stackPath, "temporal", "start"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(stdout.String(), "running") || !strings.Contains(stdout.String(), "127.0.0.1:7233") {
		t.Fatalf("start output = %q", stdout.String())
	}
}

func TestTemporalStopUnmanagedReturnsExitCodeOne(t *testing.T) {
	tmp := t.TempDir()
	stackPath := writeTemporalStack(t, tmp)
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))
	withTemporalSupervisorFactory(t, func() itemporal.Supervisor {
		return itemporal.Supervisor{Reachable: func(context.Context, string) bool { return true }}
	})

	var stdout bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stdout, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--config", stackPath, "temporal", "stop"})
	err := cmd.Execute()
	var coded exitCoder
	if !errors.As(err, &coded) || coded.ExitCode() != 1 {
		t.Fatalf("err=%v, want exit code 1", err)
	}
	if !strings.Contains(stdout.String(), "unmanaged") {
		t.Fatalf("stop output = %q", stdout.String())
	}
}

func writeTemporalStack(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "stack.toml")
	content := "[meta]\nversion = \"1.0.0\"\n\n[temporal]\nenabled = true\naddress = \"127.0.0.1:7233\"\nnamespace = \"default\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write stack: %v", err)
	}
	return path
}

func withTemporalSupervisorFactory(t *testing.T, factory func() itemporal.Supervisor) {
	t.Helper()
	previous := temporalSupervisorFactory
	temporalSupervisorFactory = factory
	t.Cleanup(func() { temporalSupervisorFactory = previous })
}
