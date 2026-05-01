package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

func TestNewDashboardCmd_Defaults(t *testing.T) {
	opts := commandOptions{
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		Version:     VersionInfo{Version: "test"},
		Environment: brand.Environment{},
	}
	root := newRootCmd(opts)
	cmd, _, err := root.Find([]string{"dashboard"})
	if err != nil {
		t.Fatalf("dashboard command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("dashboard command is nil")
	}

	// Verify default flag values
	bind, _ := cmd.Flags().GetString("bind")
	if bind != "127.0.0.1" {
		t.Errorf("expected default bind 127.0.0.1, got %q", bind)
	}
	port, _ := cmd.Flags().GetInt("port")
	if port != 9191 {
		t.Errorf("expected default port 9191, got %d", port)
	}
	noOpen, _ := cmd.Flags().GetBool("no-open")
	if noOpen != false {
		t.Errorf("expected default no-open false, got %v", noOpen)
	}
}

func TestNewDashboardCmd_HelpRenders(t *testing.T) {
	opts := commandOptions{
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		Version:     VersionInfo{Version: "test"},
		Environment: brand.Environment{},
	}
	root := newRootCmd(opts)
	root.SetArgs([]string{"dashboard", "--help"})
	// Help should not error
	if err := root.Execute(); err != nil {
		t.Fatalf("dashboard --help failed: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "dashboard") {
		t.Errorf("help output missing 'dashboard': %s", stdout)
	}
}

func TestNewDashboardCmd_CustomFlags(t *testing.T) {
	opts := commandOptions{
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		Version:     VersionInfo{Version: "test"},
		Environment: brand.Environment{},
	}
	root := newRootCmd(opts)
	root.SetArgs([]string{"dashboard", "--bind", "0.0.0.0", "--port", "8080", "--no-open"})
	cmd, _, _ := root.Find([]string{"dashboard"})

	// Parse flags manually since Execute would start the server
	if err := cmd.ParseFlags([]string{"--bind", "0.0.0.0", "--port", "8080", "--no-open"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	bind, _ := cmd.Flags().GetString("bind")
	if bind != "0.0.0.0" {
		t.Errorf("expected bind 0.0.0.0, got %q", bind)
	}
	port, _ := cmd.Flags().GetInt("port")
	if port != 8080 {
		t.Errorf("expected port 8080, got %d", port)
	}
	noOpen, _ := cmd.Flags().GetBool("no-open")
	if noOpen != true {
		t.Errorf("expected no-open true, got %v", noOpen)
	}
}
