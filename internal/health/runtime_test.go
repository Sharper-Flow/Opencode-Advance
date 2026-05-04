package health

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestCheckRuntime_NoConfigDir(t *testing.T) {
	stack := &cfg.Stack{}
	opts := Options{}
	checks, err := CheckRuntime(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckRuntime: %v", err)
	}
	// Should produce at least a warning about missing ConfigDir for ADV prompts
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
}

func TestCheckRuntime_ADVPromptCanary_MissingDir(t *testing.T) {
	dir := t.TempDir()
	stack := &cfg.Stack{}
	opts := Options{ConfigDir: dir}

	checks, err := CheckRuntime(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckRuntime: %v", err)
	}

	// ADV prompts check should skip gracefully
	found := false
	for _, c := range checks {
		if c.Name == "runtime.adv-prompts" {
			found = true
			if c.Status != StatusPass {
				t.Errorf("expected pass for missing agent-parts, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !found {
		t.Error("expected runtime.adv-prompts check")
	}
}

func TestCheckRuntime_ADVPromptCanary_WithCanonical(t *testing.T) {
	dir := t.TempDir()
	agentPartsDir := filepath.Join(dir, "agent-parts", "advance")
	if err := os.MkdirAll(agentPartsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a substantive canonical prompt
	content := make([]byte, 500) // >100 chars
	for i := range content {
		content[i] = 'a'
	}
	if err := os.WriteFile(filepath.Join(agentPartsDir, "adv.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{}
	opts := Options{ConfigDir: dir}

	checks, err := CheckRuntime(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckRuntime: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "runtime.adv-prompts.canonical" {
			found = true
			if c.Status != StatusPass {
				t.Errorf("expected pass for valid canonical, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !found {
		t.Error("expected runtime.adv-prompts.canonical check")
	}
}

func TestCheckRuntime_ADVPromptCanary_StubFile(t *testing.T) {
	dir := t.TempDir()
	agentPartsDir := filepath.Join(dir, "agent-parts", "advance")
	if err := os.MkdirAll(agentPartsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a stub (<100 chars)
	if err := os.WriteFile(filepath.Join(agentPartsDir, "adv.md"), []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{}
	opts := Options{ConfigDir: dir}

	checks, err := CheckRuntime(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckRuntime: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "runtime.adv-prompts.canonical" {
			found = true
			if c.Status != StatusWarn {
				t.Errorf("expected warn for stub file, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !found {
		t.Error("expected runtime.adv-prompts.canonical check")
	}
}

func TestCheckRuntime_ADVPromptCanary_ProviderSpecific(t *testing.T) {
	dir := t.TempDir()
	agentPartsDir := filepath.Join(dir, "agent-parts", "advance")
	providerDir := filepath.Join(agentPartsDir, "providers")
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write canonical
	content := make([]byte, 500)
	for i := range content {
		content[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(agentPartsDir, "adv.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write provider prompt for "gpt"
	if err := os.WriteFile(filepath.Join(providerDir, "gpt.md"), []byte("provider prompt"), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{
		Providers: map[string]cfg.Provider{
			"gpt": {},
		},
	}
	opts := Options{ConfigDir: dir}

	checks, err := CheckRuntime(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckRuntime: %v", err)
	}

	foundCanonical := false
	foundGPT := false
	for _, c := range checks {
		if c.Name == "runtime.adv-prompts.canonical" {
			foundCanonical = true
			if c.Status != StatusPass {
				t.Errorf("canonical: expected pass, got %s", c.Status)
			}
		}
		if c.Name == "runtime.adv-prompts.provider.gpt" {
			foundGPT = true
			if c.Status != StatusPass {
				t.Errorf("gpt provider: expected pass, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !foundCanonical {
		t.Error("expected canonical check")
	}
	if !foundGPT {
		t.Error("expected gpt provider check")
	}
}

func TestCheckRuntime_VisionUnreachable(t *testing.T) {
	stack := &cfg.Stack{}
	opts := Options{
		VisionAdminURL: "http://127.0.0.1:1", // unreachable port
		Timeout:        1,                     // 1ns timeout for fast failure
	}

	checks, err := CheckRuntime(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckRuntime: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "runtime.vision" {
			found = true
			if c.Status != StatusWarn {
				t.Errorf("expected warn for unreachable Vision, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !found {
		t.Error("expected runtime.vision check")
	}
}
