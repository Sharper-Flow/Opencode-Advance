package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestUpdateCommand_TrackingRefFetchesLatestAndReapplies(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	remote, work, checkout := initRemoteWorkCheckout(t, tmp)
	initial := gitRevParseHead(t, checkout)
	addCommitAndPush(t, work, "v2")
	latest := gitRevParseHead(t, work)
	if latest == initial {
		t.Fatal("expected remote head to advance")
	}

	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+remote+"\"\nref = \"master\"\ncheckout = \""+checkout+"\"\npath = \""+checkout+"\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"update", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update: %v stderr=%s", err, stderr.String())
	}
	if gitRevParseHead(t, checkout) != latest {
		t.Fatalf("checkout not updated to latest remote head")
	}
	if _, err := os.Stat(filepath.Join(tmp, "opencode", "opencode.json")); err != nil {
		t.Fatalf("expected opencode.json after reapply: %v", err)
	}
	assertShellEnvAndStamp(t, config.ResolvePaths())
}

func TestUpdateCommand_PinnedSHA_SkipsWithExit1(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	remote, _, checkout := initRemoteWorkCheckout(t, tmp)
	sha := gitRevParseHead(t, checkout)
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+remote+"\"\nref = \""+sha+"\"\ncheckout = \""+checkout+"\"\npath = \""+checkout+"\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"update", "--config", stackPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected skip exit")
	}
	ec, ok := err.(interface{ ExitCode() int })
	if !ok || ec.ExitCode() != 1 {
		t.Fatalf("wrong exit code: %v", err)
	}
	if !strings.Contains(stderr.String()+stdout.String(), "skip") && !strings.Contains(err.Error(), "skip") {
		t.Fatalf("expected skip messaging, stdout=%q stderr=%q err=%v", stdout.String(), stderr.String(), err)
	}
}

func TestUpdateCommand_ForceOnPinnedSHADoesNotMutateRef(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(tmp, "opencode"))
	t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(tmp, "vision"))
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))

	remote, _, checkout := initRemoteWorkCheckout(t, tmp)
	sha := gitRevParseHead(t, checkout)
	stackPath := filepath.Join(tmp, "stack.toml")
	writeFile(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+remote+"\"\nref = \""+sha+"\"\ncheckout = \""+checkout+"\"\npath = \""+checkout+"\"\n")

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"update", "--force", "--config", stackPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("force update: %v stderr=%s", err, stderr.String())
	}
	data, _ := os.ReadFile(stackPath)
	if !regexp.MustCompile(`ref = "` + sha + `"`).Match(data) {
		t.Fatalf("force update should not mutate ref:\n%s", string(data))
	}
}

func initRemoteWorkCheckout(t *testing.T, tmp string) (remote, work, checkout string) {
	t.Helper()
	remote = filepath.Join(tmp, "remote.git")
	work = filepath.Join(tmp, "work")
	checkout = filepath.Join(tmp, "checkout")
	runCmd(t, tmp, "git", "init", "--bare", "--initial-branch=master", remote)
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(work, "index.js"), "v1\n")
	runCmd(t, work, "git", "init", "--initial-branch=master")
	runCmd(t, work, "git", "config", "user.email", "test@test.com")
	runCmd(t, work, "git", "config", "user.name", "Test")
	runCmd(t, work, "git", "add", ".")
	runCmd(t, work, "git", "commit", "-m", "init")
	runCmd(t, work, "git", "remote", "add", "origin", remote)
	runCmd(t, work, "git", "push", "-u", "origin", "master")
	runCmd(t, tmp, "git", "clone", remote, checkout)
	return remote, work, checkout
}

func addCommitAndPush(t *testing.T, work, content string) {
	t.Helper()
	writeFile(t, filepath.Join(work, "index.js"), content+"\n")
	runCmd(t, work, "git", "add", ".")
	runCmd(t, work, "git", "commit", "-m", content)
	runCmd(t, work, "git", "push", "origin", "master")
}

func gitRevParseHead(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v\n%s", err, string(out))
	}
	return strings.TrimSpace(string(out))
}
