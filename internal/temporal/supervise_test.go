package temporal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRuntimePathsUseOCACacheDir(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())

	paths := RuntimePathsFromEnv()

	wantRoot := filepath.Join(os.Getenv("OCA_CACHE_DIR"), "temporal")
	if paths.Root != wantRoot {
		t.Fatalf("Root = %q, want %q", paths.Root, wantRoot)
	}
	if paths.Metadata != filepath.Join(wantRoot, "temporal.pid.json") {
		t.Fatalf("Metadata = %q", paths.Metadata)
	}
	if paths.Log != filepath.Join(wantRoot, "temporal.log") {
		t.Fatalf("Log = %q", paths.Log)
	}
	if paths.DB != filepath.Join(wantRoot, "temporal.db") {
		t.Fatalf("DB = %q", paths.DB)
	}
}

func TestMetadataRoundTripAndStaleRemoval(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	paths := RuntimePathsFromEnv()
	now := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	meta := Metadata{
		PID:       os.Getpid(),
		StartedAt: now,
		Address:   "127.0.0.1:7233",
		Host:      "127.0.0.1",
		Port:      7233,
		Namespace: "default",
		DBPath:    paths.DB,
		LogPath:   paths.Log,
		CLIPath:   "/usr/bin/temporal",
		Args:      []string{"server", "start-dev"},
		Version:   "temporal version test",
	}

	if err := WriteMetadata(paths, meta); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}
	got, ok, err := ReadMetadata(paths)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if !ok {
		t.Fatal("ReadMetadata ok=false")
	}
	if got.PID != meta.PID || got.Address != meta.Address || got.DBPath != paths.DB || got.LogPath != paths.Log {
		t.Fatalf("metadata round trip mismatch: %#v", got)
	}

	stale := meta
	stale.PID = 99999999
	if err := WriteMetadata(paths, stale); err != nil {
		t.Fatalf("WriteMetadata stale: %v", err)
	}
	removed, err := CleanupStaleMetadata(paths)
	if err != nil {
		t.Fatalf("CleanupStaleMetadata: %v", err)
	}
	if !removed {
		t.Fatal("CleanupStaleMetadata removed=false")
	}
	if _, err := os.Stat(paths.Metadata); !os.IsNotExist(err) {
		t.Fatalf("metadata should be removed, stat err=%v", err)
	}
}

func TestStatusDisabledStoppedAndUnmanaged(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", t.TempDir())
	paths := RuntimePathsFromEnv()
	ctx := context.Background()

	disabled := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(false)}}
	got, err := EvaluateStatus(ctx, disabled, paths, ProbeFuncs{})
	if err != nil {
		t.Fatalf("EvaluateStatus disabled: %v", err)
	}
	if got.State != StateDisabled || !got.Configured || got.Enabled || got.Managed || got.Running {
		t.Fatalf("disabled status mismatch: %#v", got)
	}

	enabled := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: boolPtr(true), Address: "127.0.0.1:7233", Namespace: "default"}}
	got, err = EvaluateStatus(ctx, enabled, paths, ProbeFuncs{
		Reachable: func(context.Context, string) bool { return false },
	})
	if err != nil {
		t.Fatalf("EvaluateStatus stopped: %v", err)
	}
	if got.State != StateStopped || !got.Configured || !got.Enabled || got.Managed || got.Running || got.Reachable {
		t.Fatalf("stopped status mismatch: %#v", got)
	}

	got, err = EvaluateStatus(ctx, enabled, paths, ProbeFuncs{
		Reachable: func(context.Context, string) bool { return true },
		Healthy:   func(context.Context, *cfg.Stack) bool { return true },
	})
	if err != nil {
		t.Fatalf("EvaluateStatus unmanaged: %v", err)
	}
	if got.State != StateUnmanaged || got.Managed || got.Running || !got.Reachable || !got.Healthy {
		t.Fatalf("unmanaged status mismatch: %#v", got)
	}
}

func boolPtr(v bool) *bool { return &v }
