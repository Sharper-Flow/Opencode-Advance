package health

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestCheckContextBudget_NoConfigDir(t *testing.T) {
	stack := &cfg.Stack{}
	opts := Options{}
	checks, err := CheckContextBudget(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckContextBudget: %v", err)
	}
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	if checks[0].Status != StatusWarn {
		t.Errorf("expected warn for missing ConfigDir, got %s", checks[0].Status)
	}
}

func TestCheckContextBudget_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	stack := &cfg.Stack{}
	opts := Options{ConfigDir: dir}

	checks, err := CheckContextBudget(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckContextBudget: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "context-budget" {
			found = true
			if c.Status != StatusPass {
				t.Errorf("expected pass for empty dir, got %s", c.Status)
			}
		}
	}
	if !found {
		t.Error("expected context-budget check")
	}
}

func TestCheckContextBudget_WithFiles(t *testing.T) {
	dir := t.TempDir()

	// Create instruction files
	instrDir := filepath.Join(dir, "instructions")
	if err := os.MkdirAll(instrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instrDir, "rules.yaml"), make([]byte, 5000), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instrDir, "shell.md"), make([]byte, 3000), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create agent files
	agentDir := filepath.Join(dir, "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "explore.md"), make([]byte, 8000), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{}
	opts := Options{ConfigDir: dir}

	checks, err := CheckContextBudget(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckContextBudget: %v", err)
	}

	// Should have per-category, total, and largest-file checks
	foundInstructions := false
	foundAgents := false
	foundTotal := false
	for _, c := range checks {
		if c.Name == "context-budget.instructions" {
			foundInstructions = true
		}
		if c.Name == "context-budget.agents" {
			foundAgents = true
		}
		if c.Name == "context-budget.total" {
			foundTotal = true
		}
	}
	if !foundInstructions {
		t.Error("expected context-budget.instructions check")
	}
	if !foundAgents {
		t.Error("expected context-budget.agents check")
	}
	if !foundTotal {
		t.Error("expected context-budget.total check")
	}
}

func TestCheckContextBudget_LargeFileWarning(t *testing.T) {
	dir := t.TempDir()

	instrDir := filepath.Join(dir, "instructions")
	if err := os.MkdirAll(instrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a file >50KB
	if err := os.WriteFile(filepath.Join(instrDir, "huge.md"), make([]byte, 60*1024), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{}
	opts := Options{ConfigDir: dir}

	checks, err := CheckContextBudget(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckContextBudget: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "context-budget.largest.1" {
			found = true
			if c.Status != StatusWarn {
				t.Errorf("expected warn for >50KB file, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !found {
		t.Error("expected context-budget.largest.1 check")
	}
}

func TestCheckContextBudget_StackInstructionOrder(t *testing.T) {
	dir := t.TempDir()

	// Create instruction files outside the standard dir
	rulesPath := filepath.Join(dir, "custom-rules.yaml")
	if err := os.WriteFile(rulesPath, make([]byte, 2000), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{
		Instructions: cfg.InstructionsSection{
			Order: []string{rulesPath},
		},
	}
	opts := Options{ConfigDir: dir}

	checks, err := CheckContextBudget(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckContextBudget: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "context-budget.instructions" {
			found = true
			if c.Status != StatusPass {
				t.Errorf("expected pass, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !found {
		t.Error("expected context-budget.instructions check")
	}
}
