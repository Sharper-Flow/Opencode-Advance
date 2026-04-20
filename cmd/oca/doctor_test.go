package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

func TestDoctorCommand_PluginsScope(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	checkout := filepath.Join(tmp, "checkout")
	if err := os.MkdirAll(checkout, 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(checkout, "index.js"), []byte("x"), 0o644); err != nil { t.Fatal(err) }
	runCmd(t, checkout, "git", "init", "--initial-branch=master")
	runCmd(t, checkout, "git", "config", "user.email", "test@test.com")
	runCmd(t, checkout, "git", "config", "user.name", "Test")
	runCmd(t, checkout, "git", "add", ".")
	runCmd(t, checkout, "git", "commit", "-m", "init")
	if err := os.MkdirAll(filepath.Join(tmp, "opencode"), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(tmp, "opencode", "opencode.json"), []byte(`{"plugin":["`+checkout+`/index.js"]}`), 0o644); err != nil { t.Fatal(err) }

	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+checkout+"\"\nref = \"master\"\ncheckout = \""+checkout+"\"\npath = \""+checkout+"/index.js\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"doctor", "--scope", "plugins", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor plugins: %v stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "plugins.advance.path_in_opencode_json") {
		t.Fatalf("expected plugin doctor output, got %q", stdout.String())
	}
}

func TestDoctorCommand_TemporalReserved(t *testing.T) {
	tmp := t.TempDir()
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"doctor", "--scope", "temporal", "--config", stackPath})
	err := cmd.Execute()
	if err == nil { t.Fatal("expected temporal reserved error") }
	ec, ok := err.(interface{ ExitCode() int })
	if !ok || ec.ExitCode() != 2 { t.Fatalf("wrong exit code: %v", err) }
	if !strings.Contains(err.Error(), "reserved for Phase 6.5") { t.Fatalf("unexpected error: %v", err) }
}

func TestDoctorCommand_UnknownScope(t *testing.T) {
	tmp := t.TempDir()
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"doctor", "--scope", "wat", "--config", stackPath})
	err := cmd.Execute()
	if err == nil { t.Fatal("expected unknown scope error") }
	ec, ok := err.(interface{ ExitCode() int })
	if !ok || ec.ExitCode() != 2 { t.Fatalf("wrong exit code: %v", err) }
	if !strings.Contains(err.Error(), "unknown scope") { t.Fatalf("unexpected error: %v", err) }
}
