package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

func TestThemeList(t *testing.T) {
	t.Setenv("OCA_ASSETS_ROOT", "../../assets")
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout:      &stdout,
		Stderr:      &stdout,
		Version:     VersionInfo{Version: "0.1.0-test"},
		Environment: brand.Environment{IsTTY: false},
	})

	cmd.SetArgs([]string{"theme", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "obsidian") {
		t.Errorf("theme list missing obsidian: %q", got)
	}
}

func TestThemeApply(t *testing.T) {
	t.Setenv("OCA_ASSETS_ROOT", "../../assets")
	var stdout bytes.Buffer

	cmd := newRootCmd(commandOptions{
		Stdout:      &stdout,
		Stderr:      &stdout,
		Version:     VersionInfo{Version: "0.1.0-test"},
		Environment: brand.Environment{IsTTY: false},
	})

	cmd.SetArgs([]string{"theme", "apply", "obsidian"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "theme obsidian applied") {
		t.Errorf("theme apply missing confirmation: %q", got)
	}
	if !strings.Contains(got, "colors:") {
		t.Errorf("theme apply missing colors: %q", got)
	}
}
