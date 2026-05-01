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
