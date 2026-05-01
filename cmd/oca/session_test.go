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
