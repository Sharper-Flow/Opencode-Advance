package maintain

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestDetectPluginDrift_MissingMarkerReportsStale(t *testing.T) {
	checkout := initRuntimePluginRepo(t)
	plugin := cfg.Plugin{Checkout: checkout, Subdir: "plugin", Build: []string{"pnpm build"}}

	drift, err := DetectPluginDrift(context.Background(), "advance", plugin)
	if err != nil {
		t.Fatalf("DetectPluginDrift: %v", err)
	}
	if drift.Status != DriftStatusMissingMarker {
		t.Fatalf("status=%s want %s", drift.Status, DriftStatusMissingMarker)
	}
	if drift.SourceRoot != checkout {
		t.Fatalf("source_root=%q want %q", drift.SourceRoot, checkout)
	}
}

func TestWriteBuildMarkerAndDetectFresh(t *testing.T) {
	checkout := initRuntimePluginRepo(t)
	plugin := cfg.Plugin{Checkout: checkout, Subdir: "plugin", Build: []string{"pnpm build"}}

	marker, err := WriteBuildMarker(context.Background(), "advance", plugin)
	if err != nil {
		t.Fatalf("WriteBuildMarker: %v", err)
	}
	if marker.GitSHA == "" {
		t.Fatal("marker GitSHA empty")
	}
	if _, err := os.Stat(filepath.Join(checkout, "plugin", "dist", "oca-build.json")); err != nil {
		t.Fatalf("marker file missing: %v", err)
	}

	drift, err := DetectPluginDrift(context.Background(), "advance", plugin)
	if err != nil {
		t.Fatalf("DetectPluginDrift after marker: %v", err)
	}
	if drift.Status != DriftStatusFresh {
		t.Fatalf("status=%s want %s reason=%s", drift.Status, DriftStatusFresh, drift.Reason)
	}
}

func initRuntimePluginRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeRuntimeFile(t, filepath.Join(repo, "plugin", "src", "tools", "task.ts"), "export const task = 1\n")
	writeRuntimeFile(t, filepath.Join(repo, "plugin", "dist", "index.js"), "export default {}\n")
	runCmd(t, repo, "git", "init", "--initial-branch=trunk")
	runCmd(t, repo, "git", "config", "user.email", "test@test.com")
	runCmd(t, repo, "git", "config", "user.name", "Test")
	runCmd(t, repo, "git", "add", ".")
	runCmd(t, repo, "git", "commit", "-m", "init")
	return repo
}

func writeRuntimeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(out))
	}
}
