package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func testCleanEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", "")
	t.Setenv("XDG_DATA_HOME", "")
}

func TestAdvRecoverCmd_RequiresDryRun(t *testing.T) {
	testCleanEnv(t)
	state := &commandState{output: "text"}
	cmd := newAdvRecoverCmd(state)
	cmd.SetArgs([]string{})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error without --dry-run")
	}
	if !strings.Contains(err.Error(), "dry-run") {
		t.Fatalf("error should mention dry-run: %v", err)
	}
}

func TestAdvRecoverCmd_DryRunShowsReport(t *testing.T) {
	testCleanEnv(t)
	state := &commandState{output: "text"}
	cmd := newAdvRecoverCmd(state)
	cmd.SetArgs([]string{"--dry-run"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Recovery Plan") {
		t.Fatalf("output missing 'Recovery Plan': %q", out)
	}
	// In a clean env, no worktrees or DB exist, so it should show healthy.
	if !strings.Contains(out, "healthy") {
		t.Fatalf("output missing 'healthy' for empty plan in clean env: %q", out)
	}
}

func TestAdvRecoverCmd_DryRunJSON(t *testing.T) {
	testCleanEnv(t)
	state := &commandState{output: "json"}
	cmd := newAdvRecoverCmd(state)
	cmd.SetArgs([]string{"--dry-run"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "dry_run") {
		t.Fatalf("JSON missing dry_run field: %q", out)
	}
	if !strings.Contains(out, "total_steps") {
		t.Fatalf("JSON missing total_steps field: %q", out)
	}
}

func TestAdvRecoverCmd_WithFilters(t *testing.T) {
	testCleanEnv(t)
	state := &commandState{output: "text"}
	cmd := newAdvRecoverCmd(state)
	cmd.SetArgs([]string{"--dry-run", "--project", "test-proj", "--change", "test-change"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "test-proj") {
		t.Fatalf("output missing project filter: %q", out)
	}
	if !strings.Contains(out, "test-change") {
		t.Fatalf("output missing change filter: %q", out)
	}
}

func TestAdvRecoverCmd_WithWorktreeDebt(t *testing.T) {
	// Create a fake worktree to trigger a recovery step.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", "")
	t.Setenv("XDG_DATA_HOME", "")

	home := os.Getenv("HOME")
	wtRoot := filepath.Join(home, ".local", "share", "opencode", "worktree")
	projectDir := filepath.Join(wtRoot, "proj1234", "change", "change-test")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Make it stale by backdating
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(projectDir, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	state := &commandState{output: "text"}
	cmd := newAdvRecoverCmd(state)
	cmd.SetArgs([]string{"--dry-run"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Review stale worktree") {
		t.Fatalf("output missing stale worktree step: %q", out)
	}
}
