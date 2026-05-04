package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestApplyCommand_WritesShellEnvAndStamp(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"

[plugins.auth]
source = "npm:opencode-auth@1.0.0"
`)

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--target", "plugins", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply plugins: %v stderr=%s", err, stderr.String())
	}

	paths := config.ResolvePaths()
	assertShellEnvAndStamp(t, paths)
}

func TestApplyCommand_DryRunDoesNotWriteShellEnvOrStamp(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"

[plugins.auth]
source = "npm:opencode-auth@1.0.0"
`)

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--dry-run", "--target", "plugins", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply dry-run: %v stderr=%s", err, stderr.String())
	}

	paths := config.ResolvePaths()
	assertMissing(t, paths.OCAEnvPath())
	assertMissing(t, paths.EnvStampPath())
}

func assertShellEnvAndStamp(t *testing.T, paths config.Paths) {
	t.Helper()
	env, err := os.ReadFile(paths.OCAEnvPath())
	if err != nil {
		t.Fatalf("read shell env: %v", err)
	}
	if !strings.Contains(string(env), "export OCA_CACHE_DIR=") {
		t.Fatalf("shell env missing OCA_CACHE_DIR export:\n%s", string(env))
	}
	if _, err := os.Stat(paths.EnvStampPath()); err != nil {
		t.Fatalf("stat env stamp: %v", err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
		t.Fatalf("%s existence err=%v, want not exist", path, err)
	}
}
