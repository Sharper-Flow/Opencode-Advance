package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

func TestWatchdogCommandsInHelp(t *testing.T) {
	var stdout bytes.Buffer
	cmd := newRootCmd(commandOptions{
		Stdout:      &stdout,
		Stderr:      &stdout,
		Version:     VersionInfo{Version: "0.1.0-test"},
		Environment: brand.Environment{IsTTY: false},
	})

	cmd.SetArgs([]string{"watchdog", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	for _, sub := range []string{"status", "reset"} {
		if !strings.Contains(got, sub) {
			t.Errorf("watchdog help missing %q subcommand", sub)
		}
	}
}
