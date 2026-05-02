package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	syncpkg "github.com/Sharper-Flow/Opencode-Advance/internal/sync"
)

func TestOrderingContractRenderFailure_SyncNeverCalled(t *testing.T) {
	tmp := t.TempDir()
	setCLIEnv(t, tmp)
	remote := initOrderingPluginRemote(t, filepath.Join(tmp, "advance-src"), "advance")
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+remote+"\"\nref = \"master\"\ncheckout = \""+filepath.Join(tmp, "checkouts", "advance")+"\"\nsubdir = \"plugin\"\npath = \"{checkout}/{subdir}\"\nsync = \"{checkout}/scripts/sync-global.sh --fix\"\n")

	var syncCalls int32
	prevInvoke := invokeAdvance
	invokeAdvance = func(ctx context.Context, plugin cfg.Plugin) (syncpkg.SyncResult, error) {
		atomic.AddInt32(&syncCalls, 1)
		return syncpkg.SyncResult{}, nil
	}
	defer func() { invokeAdvance = prevInvoke }()

	restoreRename := render.SetRenameForTesting(func(oldPath, newPath string) error {
		if strings.HasSuffix(newPath, filepath.Join("opencode", "opencode.json")) {
			return errors.New("injected rename failure")
		}
		return os.Rename(oldPath, newPath)
	})
	defer restoreRename()

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--target", "plugins", "--config", stackPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected render/apply failure")
	}
	if atomic.LoadInt32(&syncCalls) != 0 {
		t.Fatalf("sync should not be called when render fails; calls=%d", syncCalls)
	}
}

func TestOrderingContractSyncFailure_RenderNotRolledBack(t *testing.T) {
	tmp := t.TempDir()
	setCLIEnv(t, tmp)
	remote := initOrderingPluginRemote(t, filepath.Join(tmp, "advance-src"), "advance")
	stackPath := filepath.Join(tmp, "stack.toml")
	checkout := filepath.Join(tmp, "checkouts", "advance")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+remote+"\"\nref = \"master\"\ncheckout = \""+checkout+"\"\nsubdir = \"plugin\"\npath = \"{checkout}/{subdir}\"\nsync = \"{checkout}/scripts/sync-global.sh --fix\"\n")

	prevInvoke := invokeAdvance
	invokeAdvance = func(ctx context.Context, plugin cfg.Plugin) (syncpkg.SyncResult, error) {
		return syncpkg.SyncResult{ExitCode: 7}, errors.New("injected sync failure")
	}
	defer func() { invokeAdvance = prevInvoke }()

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--target", "plugins", "--config", stackPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected sync failure")
	}
	root := readJSONFile(t, filepath.Join(tmp, "opencode", "opencode.json"))
	plugins := jsonStringArray(t, root, "plugin")
	if !sliceContainsSubstring(plugins, filepath.Join(checkout, "plugin")) {
		t.Fatalf("rendered plugin entry should remain after sync failure: %#v", plugins)
	}
}

func TestSyncFailureDiagnosticIncludesDetails(t *testing.T) {
	tmp := t.TempDir()
	setCLIEnv(t, tmp)
	remote := initOrderingPluginRemote(t, filepath.Join(tmp, "advance-src"), "advance")
	stackPath := filepath.Join(tmp, "stack.toml")
	checkout := filepath.Join(tmp, "checkouts", "advance")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+remote+"\"\nref = \"master\"\ncheckout = \""+checkout+"\"\nsubdir = \"plugin\"\npath = \"{checkout}/{subdir}\"\nsync = \"{checkout}/scripts/sync-global.sh --fix\"\n")

	prevInvoke := invokeAdvance
	invokeAdvance = func(ctx context.Context, plugin cfg.Plugin) (syncpkg.SyncResult, error) {
		return syncpkg.SyncResult{
			ExitCode:  7,
			ExitClass: "non-zero-exit",
			Output:    []byte("fatal: could not resolve host github.com"),
		}, errors.New("injected sync failure")
	}
	defer func() { invokeAdvance = prevInvoke }()

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--target", "plugins", "--config", stackPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected sync failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "advance") {
		t.Errorf("diagnostic missing plugin name: %s", msg)
	}
	if !strings.Contains(msg, "exit code 7") {
		t.Errorf("diagnostic missing exit code: %s", msg)
	}
	if !strings.Contains(msg, "non-zero-exit") {
		t.Errorf("diagnostic missing exit class: %s", msg)
	}
	if !strings.Contains(msg, "fatal: could not resolve host github.com") {
		t.Errorf("diagnostic missing output excerpt: %s", msg)
	}
	if !strings.Contains(msg, "oca doctor") {
		t.Errorf("diagnostic missing actionable doctor command: %s", msg)
	}
}

func TestConcurrentApply_SerializesAndBothProceed(t *testing.T) {
	tmp := t.TempDir()
	setCLIEnv(t, tmp)
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[mcp.servers.context7]\nport = 6276\ncommand = \"npx\"\n")

	restoreRename := render.SetRenameForTesting(func(oldPath, newPath string) error {
		time.Sleep(150 * time.Millisecond)
		return os.Rename(oldPath, newPath)
	})
	defer restoreRename()

	var wg sync.WaitGroup
	errs := make([]error, 2)
	start := time.Now()
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var stdout, stderr bytes.Buffer
			cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
			cmd.SetArgs([]string{"apply", "--target", "mcp", "--config", stackPath})
			errs[idx] = cmd.Execute()
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent apply %d failed: %v", i, err)
		}
	}
	if elapsed < 250*time.Millisecond {
		t.Fatalf("expected serialized applies to take longer, elapsed=%v", elapsed)
	}
}

func setCLIEnv(t *testing.T, tmp string) {
	t.Helper()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))
}

func initOrderingPluginRemote(t *testing.T, path string, layout string) string {
	t.Helper()
	repo := filepath.Join(path, "repo")
	remote := filepath.Join(path, "remote.git")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if layout == "advance" {
		if err := os.MkdirAll(filepath.Join(repo, "plugin"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(repo, "scripts"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(repo, "plugin", "index.js"), "export default {}\n")
		writeFile(t, filepath.Join(repo, "scripts", "sync-global.sh"), "#!/bin/sh\npwd > sync-ran.txt\n")
		runCmd(t, repo, "chmod", "+x", filepath.Join(repo, "scripts", "sync-global.sh"))
	}
	runCmd(t, repo, "git", "init", "--initial-branch=master")
	runCmd(t, repo, "git", "config", "user.email", "test@test.com")
	runCmd(t, repo, "git", "config", "user.name", "Test")
	runCmd(t, repo, "git", "add", ".")
	runCmd(t, repo, "git", "commit", "-m", "init")
	runCmd(t, path, "git", "init", "--bare", "--initial-branch=master", remote)
	runCmd(t, repo, "git", "remote", "add", "origin", remote)
	runCmd(t, repo, "git", "push", "-u", "origin", "master")
	return remote
}

func sliceContainsSubstring(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}
