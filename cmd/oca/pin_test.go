package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

func TestPinCommand_PinsAllGitPluginsAndSkipsNPM(t *testing.T) {
	tmp := t.TempDir()
	checkout := initLocalPinnedRepo(t, filepath.Join(tmp, "advance"))
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+checkout+"\"\nref = \"master\"\ncheckout = \""+checkout+"\"\npath = \""+checkout+"\"\n\n[plugins.auth]\nsource = \"npm:opencode-auth@1.0.0\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"pin", "--config", stackPath})
	if err := cmd.Execute(); err != nil { t.Fatalf("pin: %v stderr=%s", err, stderr.String()) }

	data, err := os.ReadFile(stackPath)
	if err != nil { t.Fatal(err) }
	if !regexp.MustCompile(`ref = "[0-9a-f]{40}"`).Match(data) {
		t.Fatalf("expected pinned SHA in stack.toml:\n%s", string(data))
	}
	if !regexp.MustCompile(`source = "npm:opencode-auth@1.0.0"`).Match(data) {
		t.Fatalf("npm plugin should remain unchanged:\n%s", string(data))
	}
}

func TestPinCommand_TargetedPluginOnly(t *testing.T) {
	tmp := t.TempDir()
	advance := initLocalPinnedRepo(t, filepath.Join(tmp, "advance"))
	morph := initLocalPinnedRepo(t, filepath.Join(tmp, "morph"))
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+advance+"\"\nref = \"master\"\ncheckout = \""+advance+"\"\npath = \""+advance+"\"\n\n[plugins.morph]\nsource = \""+morph+"\"\nref = \"master\"\ncheckout = \""+morph+"\"\npath = \""+morph+"\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"pin", "advance", "--config", stackPath})
	if err := cmd.Execute(); err != nil { t.Fatalf("pin targeted: %v stderr=%s", err, stderr.String()) }

	data, _ := os.ReadFile(stackPath)
	matches := regexp.MustCompile(`ref = "[0-9a-f]{40}"`).FindAll(data, -1)
	if len(matches) != 1 {
		t.Fatalf("expected exactly one pinned SHA, got %d:\n%s", len(matches), string(data))
	}
}

func TestPinCommand_UnreadableCheckoutExit2(t *testing.T) {
	tmp := t.TempDir()
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"

[plugins.advance]
source = "/missing/repo"
ref = "master"
checkout = "/missing/repo"
path = "/missing/repo"
`)

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"pin", "--config", stackPath})
	err := cmd.Execute()
	if err == nil { t.Fatal("expected error") }
	ec, ok := err.(interface{ ExitCode() int })
	if !ok || ec.ExitCode() != 2 { t.Fatalf("wrong exit code: %v", err) }
}

func initLocalPinnedRepo(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil { t.Fatal(err) }
	writeFile(t, filepath.Join(path, "index.js"), "x\n")
	runCmd(t, path, "git", "init", "--initial-branch=master")
	runCmd(t, path, "git", "config", "user.email", "test@test.com")
	runCmd(t, path, "git", "config", "user.name", "Test")
	runCmd(t, path, "git", "add", ".")
	runCmd(t, path, "git", "commit", "-m", "init")
	return path
}
