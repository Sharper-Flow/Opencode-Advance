package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

func TestApplyCommand_TargetPluginsRendersNPMPlugin(t *testing.T) {
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

	root := readJSONFile(t, filepath.Join(tmp, "opencode", "opencode.json"))
	plugins := jsonStringArray(t, root, "plugin")
	if len(plugins) != 1 || plugins[0] != "opencode-auth@1.0.0" {
		t.Fatalf("plugin entries=%#v", plugins)
	}
}

func TestApplyCommand_TargetInstructionsRendersInstructions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"

[plugins.advance]
source = "npm:advance@1.0.0"
instructions = ["/plugins/advance/ADV_INSTRUCTIONS.md"]

[instructions]
order = ["identity.md", "rules.yaml"]
`)

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--target", "instructions", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply instructions: %v stderr=%s", err, stderr.String())
	}

	root := readJSONFile(t, filepath.Join(tmp, "opencode", "opencode.json"))
	instructions := jsonStringArray(t, root, "instructions")
	if len(instructions) != 3 {
		t.Fatalf("instructions=%#v", instructions)
	}
}

func TestApplyCommand_MultipleTargetsChainsPluginsThenInstructions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"

[plugins.auth]
source = "npm:opencode-auth@1.0.0"

[instructions]
order = ["identity.md"]
`)

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--target", "plugins", "--target", "instructions", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply multiple targets: %v stderr=%s", err, stderr.String())
	}

	root := readJSONFile(t, filepath.Join(tmp, "opencode", "opencode.json"))
	if got := jsonStringArray(t, root, "plugin"); len(got) != 1 || got[0] != "opencode-auth@1.0.0" {
		t.Fatalf("plugin=%#v", got)
	}
	if got := jsonStringArray(t, root, "instructions"); len(got) != 1 || got[0] != "identity.md" {
		t.Fatalf("instructions=%#v", got)
	}
}

func TestApplyCommand_TargetPluginsRunsSyncAfterRender(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))
	t.Setenv("OCA_PLUGIN_CHECKOUT_ROOT", filepath.Join(tmp, "checkouts"))

	remote := initPluginRemote(t)
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+remote+"\"\nref = \"master\"\ncheckout = \""+filepath.Join(tmp, "checkouts", "advance")+"\"\npath = \"{checkout}\"\nsync = \"{checkout}/sync.sh\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--target", "plugins", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply plugins sync: %v stderr=%s", err, stderr.String())
	}

	marker := filepath.Join(tmp, "checkouts", "advance", "sync-ran.txt")
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("sync marker missing: %v", err)
	}
	root := readJSONFile(t, filepath.Join(tmp, "opencode", "opencode.json"))
	plugins := jsonStringArray(t, root, "plugin")
	if len(plugins) != 1 || !strings.Contains(plugins[0], filepath.Join(tmp, "checkouts", "advance")) {
		t.Fatalf("plugin entries=%#v", plugins)
	}
}

func TestApplyCommand_TemporalReserved(t *testing.T) {
	tmp := t.TempDir()
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, `[meta]
version = "1.0.0"
`)

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"apply", "--target", "temporal", "--config", stackPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected reserved temporal error")
	}
	ec, ok := err.(interface{ ExitCode() int })
	if !ok || ec.ExitCode() != 2 {
		t.Fatalf("exit code wrong: %v", err)
	}
	if !strings.Contains(err.Error(), "reserved for Phase 6.5") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func initPluginRemote(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	remote := filepath.Join(root, "remote.git")
	if err := os.MkdirAll(repo, 0o755); err != nil { t.Fatal(err) }
	writeFile(t, filepath.Join(repo, "sync.sh"), "#!/bin/sh\npwd > sync-ran.txt\n")
	runCmd(t, repo, "chmod", "+x", "sync.sh")
	runCmd(t, repo, "git", "init", "--initial-branch=master")
	runCmd(t, repo, "git", "config", "user.email", "test@test.com")
	runCmd(t, repo, "git", "config", "user.name", "Test")
	runCmd(t, repo, "git", "add", ".")
	runCmd(t, repo, "git", "commit", "-m", "init")
	runCmd(t, root, "git", "init", "--bare", "--initial-branch=master", remote)
	runCmd(t, repo, "git", "remote", "add", "origin", remote)
	runCmd(t, repo, "git", "push", "-u", "origin", "master")
	return remote
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { t.Fatal(err) }
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil { t.Fatal(err) }
	return root
}

func jsonStringArray(t *testing.T, root map[string]any, key string) []string {
	t.Helper()
	raw, ok := root[key].([]any)
	if !ok { t.Fatalf("%s missing or not array: %#v", key, root[key]) }
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok { t.Fatalf("non-string item in %s: %#v", key, item) }
		out = append(out, s)
	}
	return out
}
