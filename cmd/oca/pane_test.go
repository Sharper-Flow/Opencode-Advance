package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTmuxSocket(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/tmp/tmux-1000/oca,12345", "oca"},
		{"/tmp/tmux-1000/default,12345", "default"},
		{"", "oca"},
		{",12345", "oca"},
	}
	for _, tt := range tests {
		got := parseTmuxSocket(tt.input)
		if got != tt.expected {
			t.Errorf("parseTmuxSocket(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSanitizePaneID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"%42", "42"},
		{"%0", "0"},
		{"42", "42"},
		{"0", "0"},
	}
	for _, tt := range tests {
		got := sanitizePaneID(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizePaneID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestXdgStateHome(t *testing.T) {
	// With XDG_STATE_HOME set
	t.Run("env set", func(t *testing.T) {
		original := os.Getenv("XDG_STATE_HOME")
		os.Setenv("XDG_STATE_HOME", "/custom/state")
		defer os.Setenv("XDG_STATE_HOME", original)

		got := xdgStateHome()
		if got != "/custom/state" {
			t.Errorf("xdgStateHome() = %q, want %q", got, "/custom/state")
		}
	})

	// With XDG_STATE_HOME unset
	t.Run("fallback", func(t *testing.T) {
		original := os.Getenv("XDG_STATE_HOME")
		os.Unsetenv("XDG_STATE_HOME")
		defer os.Setenv("XDG_STATE_HOME", original)

		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".local", "state")
		got := xdgStateHome()
		if got != want {
			t.Errorf("xdgStateHome() = %q, want %q", got, want)
		}
	})
}

func TestReadPaneState(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		tmp := t.TempDir()
		file := filepath.Join(tmp, "state.json")
		os.WriteFile(file, []byte(`{"sessionID":"ses_abc","directory":"/tmp","ts":12345}`), 0o644)

		ps, err := readPaneState(file)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ps.SessionID != "ses_abc" || ps.Directory != "/tmp" || ps.Ts != 12345 {
			t.Errorf("unexpected state: %+v", ps)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readPaneState("/nonexistent/path/state.json")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		tmp := t.TempDir()
		file := filepath.Join(tmp, "bad.json")
		os.WriteFile(file, []byte("not json"), 0o644)

		_, err := readPaneState(file)
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})
}

func TestPaneStateFilePath(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	originalXdg := os.Getenv("XDG_STATE_HOME")
	defer func() {
		os.Setenv("TMUX", originalTmux)
		os.Setenv("XDG_STATE_HOME", originalXdg)
	}()

	os.Setenv("XDG_STATE_HOME", "/test/state")
	os.Setenv("TMUX", "/tmp/tmux-1000/oca,12345")

	got := paneStateFilePath("%42")
	want := filepath.Join("/test/state", "oca", "panes", "oca", "42.json")
	if got != want {
		t.Errorf("paneStateFilePath(%%42) = %q, want %q", got, want)
	}
}
