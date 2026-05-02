package temporal

import (
	"context"
	"errors"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

func TestSupervisorStopRemovesStaleMetadata(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	paths := RuntimePathsFromEnv()
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(true), Address: "127.0.0.1:7233", Namespace: "default"}}
	if err := WriteMetadata(paths, Metadata{PID: 99999999, Address: "127.0.0.1:7233", DBPath: paths.DB, LogPath: paths.Log}); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	status, err := (Supervisor{Paths: paths}).Stop(context.Background(), stack)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if status.State != StateStopped || status.Managed || status.Running {
		t.Fatalf("status mismatch: %#v", status)
	}
	if _, err := os.Stat(paths.Metadata); !os.IsNotExist(err) {
		t.Fatalf("metadata should be removed, stat err=%v", err)
	}
}

func TestSupervisorStopRefusesUnmanagedReachableServer(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(true), Address: "127.0.0.1:7233", Namespace: "default"}}
	sup := Supervisor{
		Paths:     RuntimePathsFromEnv(),
		Reachable: func(context.Context, string) bool { return true },
		Healthy:   func(context.Context, *cfg.Stack) bool { return true },
	}

	status, err := sup.Stop(context.Background(), stack)
	if !errors.Is(err, ErrUnmanagedServer) {
		t.Fatalf("Stop err=%v, want ErrUnmanagedServer", err)
	}
	if status.State != StateUnmanaged || status.Managed || status.Running {
		t.Fatalf("status mismatch: %#v", status)
	}
}

func TestSupervisorStopSignalsManagedProcessGroup(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	paths := RuntimePathsFromEnv()
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(true), Address: "127.0.0.1:7233", Namespace: "default"}}
	if err := WriteMetadata(paths, Metadata{PID: 1234, Address: "127.0.0.1:7233", DBPath: paths.DB, LogPath: paths.Log}); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}
	var signals []syscall.Signal
	aliveCalls := 0
	sup := Supervisor{
		Paths: paths,
		PIDAlive: func(int) bool {
			aliveCalls++
			return aliveCalls <= 1
		},
		SignalProcessGroup: func(pgid int, sig syscall.Signal) error {
			if pgid != 1234 {
				t.Fatalf("pgid=%d, want 1234", pgid)
			}
			signals = append(signals, sig)
			return nil
		},
		StopTick: time.Nanosecond,
	}

	status, err := sup.Stop(context.Background(), stack)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !reflect.DeepEqual(signals, []syscall.Signal{syscall.SIGTERM}) {
		t.Fatalf("signals=%v", signals)
	}
	if status.State != StateStopped || status.Managed || status.Running {
		t.Fatalf("status mismatch: %#v", status)
	}
}

func TestSupervisorRestartPreservesDBPath(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	paths := RuntimePathsFromEnv()
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(true), Address: "127.0.0.1:7233", Namespace: "default"}}
	if err := WriteMetadata(paths, Metadata{PID: 99999999, Address: "127.0.0.1:7233", DBPath: paths.DB, LogPath: paths.Log}); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}
	started := false
	sup := Supervisor{
		Paths: paths,
		DetectCLI: func(context.Context) (DetectResult, error) {
			return DetectResult{Path: "/bin/temporal", Version: "temporal version test"}, nil
		},
		StartBackground: func(context.Context, subprocess.Cmd, subprocess.StartOptions) (subprocess.BackgroundProcess, error) {
			started = true
			return subprocess.BackgroundProcess{PID: os.Getpid(), ProcessGroupID: os.Getpid()}, nil
		},
		Reachable: func(context.Context, string) bool { return started },
		Healthy:   func(context.Context, *cfg.Stack) bool { return true },
	}

	status, err := sup.Restart(context.Background(), stack)
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if status.State != StateRunning || status.DBPath != paths.DB {
		t.Fatalf("status mismatch: %#v", status)
	}
}
