package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

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
