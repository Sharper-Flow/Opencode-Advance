package temporal

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

func TestSupervisorStartBuildsStartDevCommandAndMetadata(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	paths := RuntimePathsFromEnv()
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(true), Address: "127.0.0.1:7234", Namespace: "adv-test"}}
	now := time.Date(2026, 5, 2, 1, 2, 3, 0, time.UTC)

	var gotCmd subprocess.Cmd
	var gotOpts subprocess.StartOptions
	started := false
	sup := Supervisor{
		Paths: paths,
		DetectCLI: func(context.Context) (DetectResult, error) {
			return DetectResult{Path: "/bin/temporal", Version: "temporal version test"}, nil
		},
		StartBackground: func(_ context.Context, cmd subprocess.Cmd, opts subprocess.StartOptions) (subprocess.BackgroundProcess, error) {
			gotCmd = cmd
			gotOpts = opts
			started = true
			return subprocess.BackgroundProcess{PID: os.Getpid(), ProcessGroupID: os.Getpid()}, nil
		},
		Reachable: func(context.Context, string) bool { return started },
		Healthy:   func(context.Context, *cfg.Stack) bool { return true },
		Now:       func() time.Time { return now },
	}

	status, err := sup.Start(context.Background(), stack)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if status.State != StateRunning || !status.Managed || !status.Running || !status.Reachable || !status.Healthy {
		t.Fatalf("status mismatch: %#v", status)
	}
	wantArgs := []string{"server", "start-dev", "--ip", "127.0.0.1", "--port", "7234", "--namespace", "adv-test", "--db-filename", paths.DB, "--log-level", "warn", "--headless"}
	if gotCmd.Name != "/bin/temporal" {
		t.Fatalf("Name = %q", gotCmd.Name)
	}
	if !reflect.DeepEqual(gotCmd.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v", gotCmd.Args, wantArgs)
	}
	if gotOpts.OutputPath != paths.Log || !gotOpts.SetProcessGroup {
		t.Fatalf("StartOptions = %#v", gotOpts)
	}

	meta, ok, err := ReadMetadata(paths)
	if err != nil || !ok {
		t.Fatalf("ReadMetadata ok=%v err=%v", ok, err)
	}
	if meta.PID != os.Getpid() || meta.StartedAt != now || meta.DBPath != paths.DB || meta.LogPath != paths.Log || meta.Port != 7234 || meta.Namespace != "adv-test" {
		t.Fatalf("metadata mismatch: %#v", meta)
	}
}

func TestSupervisorStartIsIdempotentForManagedRunningServer(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	paths := RuntimePathsFromEnv()
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(true), Address: "127.0.0.1:7233", Namespace: "default"}}
	if err := WriteMetadata(paths, Metadata{PID: os.Getpid(), Address: "127.0.0.1:7233", Host: "127.0.0.1", Port: 7233, Namespace: "default", DBPath: paths.DB, LogPath: paths.Log}); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	started := false
	sup := Supervisor{
		Paths: paths,
		StartBackground: func(context.Context, subprocess.Cmd, subprocess.StartOptions) (subprocess.BackgroundProcess, error) {
			started = true
			return subprocess.BackgroundProcess{}, nil
		},
		Reachable: func(context.Context, string) bool { return true },
		Healthy:   func(context.Context, *cfg.Stack) bool { return true },
	}

	status, err := sup.Start(context.Background(), stack)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if started {
		t.Fatal("StartBackground called for already running managed server")
	}
	if status.State != StateRunning || !status.Managed {
		t.Fatalf("status mismatch: %#v", status)
	}
}

func TestSupervisorStartRefusesUnmanagedReachableServer(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(true), Address: "127.0.0.1:7233", Namespace: "default"}}
	sup := Supervisor{
		Paths:     RuntimePathsFromEnv(),
		Reachable: func(context.Context, string) bool { return true },
		Healthy:   func(context.Context, *cfg.Stack) bool { return true },
	}

	status, err := sup.Start(context.Background(), stack)
	if !errors.Is(err, ErrUnmanagedServer) {
		t.Fatalf("Start err=%v, want ErrUnmanagedServer", err)
	}
	if status.State != StateUnmanaged || status.Managed || status.Running {
		t.Fatalf("status mismatch: %#v", status)
	}
}
