package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	"github.com/Sharper-Flow/Opencode-Advance/internal/session"
)

func TestRootCommandOutsideGitShowsSessionListHint(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	nonGitDir := t.TempDir()
	withWorkingDir(t, nonGitDir)

	cmd := newRootCmd(commandOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Version: VersionInfo{
			Version: "0.1.0-test",
		},
		Environment: brand.Environment{IsTTY: false, Term: "xterm"},
	})

	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String() + stderr.String()
	if !strings.Contains(got, "not in a git project") {
		t.Fatalf("root output missing outside-git hint: %q", got)
	}
	if !strings.Contains(got, "oca session list") {
		t.Fatalf("root output missing session list hint: %q", got)
	}
}

func TestRootCommandProjectModeCreatesAndAttachesProjectSession(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := newGitRepo(t, "opencodeadvance")
	withWorkingDir(t, root)
	configPath := writeStackConfig(t, root, "project")
	tmuxPath, tmuxLog := fakeTmuxForCLI(t, "ERR:no server running on fake")
	prependPATH(t, filepath.Dir(tmuxPath))

	var attached string
	origAttach := attachProjectSession
	attachProjectSession = func(ctx context.Context, mgr *session.Manager, name string) error {
		attached = name
		return nil
	}
	t.Cleanup(func() { attachProjectSession = origAttach })

	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Version: VersionInfo{Version: "0.1.0-test"}, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--config", configPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v (stderr: %s)", err, stderr.String())
	}
	if attached != "opencodeadvance" {
		t.Fatalf("attached session = %q, want opencodeadvance", attached)
	}
	log := readTextFile(t, tmuxLog)
	if !strings.Contains(log, "new-session -d -s opencodeadvance -n trunk -c "+root) {
		t.Fatalf("tmux log missing Pattern B project session create:\n%s", log)
	}
}

func TestRootCommandPerInvocationModeDelegatesToSessionNew(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	workdir := filepath.Join(t.TempDir(), "legacyrepo")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workdir): %v", err)
	}
	withWorkingDir(t, workdir)
	configPath := writeStackConfig(t, workdir, "per-invocation")
	tmuxPath, tmuxLog := fakeTmuxForCLI(t, "ERR:no server running on fake")
	prependPATH(t, filepath.Dir(tmuxPath))

	origAttach := attachProjectSession
	attachProjectSession = func(ctx context.Context, mgr *session.Manager, name string) error {
		t.Fatalf("project attach called in per-invocation mode for %q", name)
		return nil
	}
	t.Cleanup(func() { attachProjectSession = origAttach })

	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Version: VersionInfo{Version: "0.1.0-test"}, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--config", configPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v (stderr: %s)", err, stderr.String())
	}
	log := readTextFile(t, tmuxLog)
	if !strings.Contains(log, "new-session -d -s oca-legacyrepo-0 -c "+workdir) {
		t.Fatalf("tmux log missing Pattern A session create:\n%s", log)
	}
	if !strings.Contains(stdout.String(), "session oca-legacyrepo-0 created") {
		t.Fatalf("stdout missing Pattern A created message: %q", stdout.String())
	}
}

func TestVersionCommandPrintsBrandedBanner(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout: &stdout,
		Stderr: &stdout,
		Version: VersionInfo{
			Version: "0.1.0-test",
		},
		Environment: brand.Environment{IsTTY: false, Term: "xterm"},
	})

	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "░█▀█░█▀█░█▀▀") {
		t.Fatalf("version output missing wordmark: %q", got)
	}

	if !strings.Contains(got, "version: 0.1.0-test") {
		t.Fatalf("version output missing version line: %q", got)
	}

	if strings.Contains(strings.ToLower(got), "scaffold") {
		t.Fatalf("version output still contains scaffold text: %q", got)
	}
}

func TestVersionCommandUsesTruecolorWhenSupported(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout: &stdout,
		Stderr: &stdout,
		Version: VersionInfo{
			Version: "0.1.0-test",
		},
		Environment: brand.Environment{IsTTY: true, Term: "xterm-256color", ColorTerm: "truecolor"},
	})

	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "\x1b[38;2;232;230;227m") {
		t.Fatalf("version output missing truecolor ivory sequence: %q", got)
	}

	if !strings.Contains(got, "\x1b[38;2;108;122;184m") {
		t.Fatalf("version output missing truecolor indigo sequence: %q", got)
	}
}

func TestVersionCommandRespectsNoColor(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout: &stdout,
		Stderr: &stdout,
		Version: VersionInfo{
			Version: "0.1.0-test",
		},
		Environment: brand.Environment{IsTTY: true, NoColor: true, Term: "xterm-256color", ColorTerm: "truecolor"},
	})

	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("version output should not contain ANSI sequences when no-color is active: %q", got)
	}
}

func newGitRepo(t *testing.T, name string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(repo): %v", err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (output: %s)", err, out)
	}
	return root
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd %q: %v", old, err)
		}
	})
}

func writeStackConfig(t *testing.T, dir, mode string) string {
	t.Helper()
	path := filepath.Join(dir, "stack.toml")
	content := fmt.Sprintf("[meta]\nversion = \"1.0.0\"\n\n[session]\nmode = %q\nboot_splash = false\nreaper = false\n", mode)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(stack.toml): %v", err)
	}
	return path
}

func fakeTmuxForCLI(t *testing.T, sessionOutput string) (string, string) {
	return fakeTmuxForCLIWithWindows(t, sessionOutput, "")
}

func fakeTmuxForCLIWithWindows(t *testing.T, sessionOutput, windowOutput string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "tmux.log")
	scriptPath := filepath.Join(dir, "tmux")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %s
cmd=""
for arg in "$@"; do
  case "$arg" in
    list-sessions|new-session|list-windows|new-window|kill-window|setenv)
      cmd="$arg"
      break
      ;;
  esac
done
case "$cmd" in
  list-sessions)
%s
    ;;
  list-windows)
%s
    ;;
  new-session|new-window|kill-window|setenv)
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
`, shellQuoteForTest(logPath), fakeTmuxCLIOutputClause(sessionOutput), fakeTmuxCLIOutputClause(windowOutput))
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(fake tmux): %v", err)
	}
	return scriptPath, logPath
}

func fakeTmuxCLIOutputClause(output string) string {
	exitCode := 0
	stream := "cat <<'EOF'"
	if strings.HasPrefix(output, "ERR:") {
		exitCode = 1
		stream = "cat >&2 <<'EOF'"
		output = strings.TrimPrefix(output, "ERR:")
	}
	return fmt.Sprintf("    %s\n%s\nEOF\n    exit %d", stream, output, exitCode)
}

func prependPATH(t *testing.T, dir string) {
	t.Helper()
	old := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+old)
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return string(b)
}

func shellQuoteForTest(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
