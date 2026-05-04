package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

// TestResolveTmuxConf_KnownTheme verifies that a valid theme name resolves to
// the correct asset path under OCA_ASSETS_ROOT.
func TestResolveTmuxConf_KnownTheme(t *testing.T) {
	tmpRoot := t.TempDir()
	themesDir := filepath.Join(tmpRoot, "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatalf("mkdir themes: %v", err)
	}
	confPath := filepath.Join(themesDir, "obsidian.tmux.conf")
	if err := os.WriteFile(confPath, []byte("# stub"), 0o644); err != nil {
		t.Fatalf("write conf: %v", err)
	}
	t.Setenv("OCA_ASSETS_ROOT", tmpRoot)

	got := resolveTmuxConf("obsidian")
	if got != confPath {
		t.Errorf("resolveTmuxConf(obsidian) = %q, want %q", got, confPath)
	}
}

// TestResolveTmuxConf_EmptyThemeDefaultsToObsidian verifies the empty-string
// theme is treated as "use the default" (obsidian).
func TestResolveTmuxConf_EmptyThemeDefaultsToObsidian(t *testing.T) {
	tmpRoot := t.TempDir()
	themesDir := filepath.Join(tmpRoot, "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatalf("mkdir themes: %v", err)
	}
	confPath := filepath.Join(themesDir, "obsidian.tmux.conf")
	if err := os.WriteFile(confPath, []byte("# stub"), 0o644); err != nil {
		t.Fatalf("write conf: %v", err)
	}
	t.Setenv("OCA_ASSETS_ROOT", tmpRoot)

	got := resolveTmuxConf("")
	if got != confPath {
		t.Errorf("resolveTmuxConf(\"\") = %q, want %q (should default to obsidian)", got, confPath)
	}
}

// TestResolveTmuxConf_UnknownThemeReturnsEmpty verifies graceful degradation
// when an unknown theme name is configured: returns empty string so Manager
// .Create runs without a -f flag rather than failing.
func TestResolveTmuxConf_UnknownThemeReturnsEmpty(t *testing.T) {
	tmpRoot := t.TempDir()
	themesDir := filepath.Join(tmpRoot, "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatalf("mkdir themes: %v", err)
	}
	// Only obsidian exists; "synthwave" does not.
	confPath := filepath.Join(themesDir, "obsidian.tmux.conf")
	if err := os.WriteFile(confPath, []byte("# stub"), 0o644); err != nil {
		t.Fatalf("write conf: %v", err)
	}
	t.Setenv("OCA_ASSETS_ROOT", tmpRoot)

	got := resolveTmuxConf("synthwave")
	if got != "" {
		t.Errorf("resolveTmuxConf(synthwave) = %q, want \"\" (graceful degrade)", got)
	}
}

// TestResolveTmuxConf_TraversalPayloadsBlocked verifies that theme names
// containing path separators, "..", or backslashes are rejected and return
// an empty string even when a matching file exists outside the themes dir.
func TestResolveTmuxConf_TraversalPayloadsBlocked(t *testing.T) {
	tmpRoot := t.TempDir()
	themesDir := filepath.Join(tmpRoot, "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatalf("mkdir themes: %v", err)
	}
	// Create obsidian so the env override is active.
	obsidianConf := filepath.Join(themesDir, "obsidian.tmux.conf")
	if err := os.WriteFile(obsidianConf, []byte("# stub"), 0o644); err != nil {
		t.Fatalf("write obsidian conf: %v", err)
	}
	// Create a file that would match if traversal were allowed.
	evilDir := filepath.Join(tmpRoot, "evil")
	if err := os.MkdirAll(evilDir, 0o755); err != nil {
		t.Fatalf("mkdir evil: %v", err)
	}
	if err := os.WriteFile(filepath.Join(evilDir, "pwned.tmux.conf"), []byte("# evil"), 0o644); err != nil {
		t.Fatalf("write evil conf: %v", err)
	}
	t.Setenv("OCA_ASSETS_ROOT", tmpRoot)

	cases := []struct {
		name  string
		theme string
	}{
		{"dotdot", "../evil"},
		{"double_dotdot", "../../tmp/pwned"},
		{"forward_slash", "bad/name"},
		{"backslash", "bad\\name"},
		{"dotdot_infix", "foo/../bar"},
		{"dotdot_plain", ".."},
		{"dotdot_embedded", "foo..bar"},
		{"null_bytes_not_needed", "foo\x00bar"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveTmuxConf(tc.theme)
			if got != "" {
				t.Errorf("resolveTmuxConf(%q) = %q, want \"\" (traversal blocked)", tc.theme, got)
			}
		})
	}
}

func TestSessionCommandsInHelp(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout:      &stdout,
		Stderr:      &stdout,
		Version:     VersionInfo{Version: "0.1.0-test"},
		Environment: brand.Environment{IsTTY: false},
	})

	cmd.SetArgs([]string{"session", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	expected := []string{"attach", "switch", "kill", "killall", "restart", "reap"}
	for _, sub := range expected {
		if !strings.Contains(got, sub) {
			t.Errorf("session help missing %q subcommand", sub)
		}
	}
}

func TestSessionEnsureWindowCreatesMissingWindow(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	worktree := filepath.Join(t.TempDir(), "change-one")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("MkdirAll(worktree): %v", err)
	}
	tmuxPath, tmuxLog := fakeTmuxForCLIWithWindows(t, "", "")
	prependPATH(t, filepath.Dir(tmuxPath))

	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Version: VersionInfo{Version: "0.1.0-test"}, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"session", "ensure-window", "--session", "opencodeadvance", "--name", "change-one", "--cwd", worktree})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v (stderr: %s)", err, stderr.String())
	}
	log := readTextFile(t, tmuxLog)
	if !strings.Contains(log, "new-window -t opencodeadvance -n change-one -c "+worktree) {
		t.Fatalf("tmux log missing ensure-window create:\n%s", log)
	}
	if !strings.Contains(stdout.String(), "window change-one ensured") {
		t.Fatalf("stdout missing ensure message: %q", stdout.String())
	}
}

func TestSessionEnsureWindowReusesExistingWindow(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	worktree := filepath.Join(t.TempDir(), "change-one")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("MkdirAll(worktree): %v", err)
	}
	windowOutput := "%2\t1\tchange-one\t0\t" + worktree
	tmuxPath, tmuxLog := fakeTmuxForCLIWithWindows(t, "", windowOutput)
	prependPATH(t, filepath.Dir(tmuxPath))

	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Version: VersionInfo{Version: "0.1.0-test"}, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"session", "ensure-window", "--session", "opencodeadvance", "--name", "change-one", "--cwd", worktree})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v (stderr: %s)", err, stderr.String())
	}
	log := readTextFile(t, tmuxLog)
	if strings.Contains(log, "new-window") {
		t.Fatalf("tmux log created duplicate window:\n%s", log)
	}
	if !strings.Contains(stdout.String(), "already existed") {
		t.Fatalf("stdout missing reuse status: %q", stdout.String())
	}
}

func TestSessionEnsureWindowRejectsMissingCwd(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	missing := filepath.Join(t.TempDir(), "missing")
	tmuxPath, _ := fakeTmuxForCLIWithWindows(t, "", "")
	prependPATH(t, filepath.Dir(tmuxPath))

	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Version: VersionInfo{Version: "0.1.0-test"}, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"session", "ensure-window", "--session", "opencodeadvance", "--name", "change-one", "--cwd", missing})
	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() error = nil, want missing cwd error")
	}
}

func TestSessionEnsureWindowRecreatesStaleCwd(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	worktree := filepath.Join(t.TempDir(), "change-one")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("MkdirAll(worktree): %v", err)
	}
	stale := filepath.Join(t.TempDir(), "missing")
	windowOutput := "%2\t1\tchange-one\t0\t" + stale
	tmuxPath, tmuxLog := fakeTmuxForCLIWithWindows(t, "", windowOutput)
	prependPATH(t, filepath.Dir(tmuxPath))

	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Version: VersionInfo{Version: "0.1.0-test"}, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"session", "ensure-window", "--session", "opencodeadvance", "--name", "change-one", "--cwd", worktree})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v (stderr: %s)", err, stderr.String())
	}
	log := readTextFile(t, tmuxLog)
	if !strings.Contains(log, "kill-window -t opencodeadvance:%2") || !strings.Contains(log, "new-window -t opencodeadvance -n change-one -c "+worktree) {
		t.Fatalf("tmux log missing stale recreate commands:\n%s", log)
	}
}

func TestSessionReconcileScansADVWorktreeLayout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := newGitRepo(t, "opencodeadvance")
	withWorkingDir(t, root)
	xdg := t.TempDir()
	changeDir := filepath.Join(xdg, "opencode", "worktree", "project-id", "change", "change-one")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(changeDir): %v", err)
	}
	t.Setenv("XDG_DATA_HOME", xdg)
	tmuxPath, tmuxLog := fakeTmuxForCLIWithWindows(t, "", "")
	prependPATH(t, filepath.Dir(tmuxPath))

	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stderr, Version: VersionInfo{Version: "0.1.0-test"}, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"session", "reconcile"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v (stderr: %s)", err, stderr.String())
	}
	log := readTextFile(t, tmuxLog)
	if !strings.Contains(log, "new-window -t opencodeadvance -n change-one -c "+changeDir) {
		t.Fatalf("tmux log missing reconcile window create:\n%s", log)
	}
	if !strings.Contains(stdout.String(), "reconciled 1 window") {
		t.Fatalf("stdout missing reconcile count: %q", stdout.String())
	}
}
