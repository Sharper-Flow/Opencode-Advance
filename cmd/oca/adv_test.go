package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdvRecoverCmd_RequiresDryRun(t *testing.T) {
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

func TestAdvRecoverCmd_DryRunEmptyPlan(t *testing.T) {
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
	if !strings.Contains(out, "healthy") {
		t.Fatalf("output missing 'healthy' for empty plan: %q", out)
	}
}

func TestAdvRecoverCmd_DryRunJSON(t *testing.T) {
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
