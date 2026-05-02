package health

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestCheckADVPlugin_NoAdvancePlugin(t *testing.T) {
	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"other": {Source: "https://github.com/other/plugin.git", Checkout: "/tmp/other"},
		},
	}

	checks, err := CheckADVPlugin(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckADVPlugin failed: %v", err)
	}

	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if checks[0].Status != StatusFail {
		t.Errorf("expected fail, got %s", checks[0].Status)
	}
	if checks[0].Name != "advance-plugin-declared" {
		t.Errorf("unexpected check name: %s", checks[0].Name)
	}
}

func TestCheckADVPlugin_MissingCheckout(t *testing.T) {
	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {Source: "https://github.com/Sharper-Flow/Advance.git"},
		},
	}

	checks, err := CheckADVPlugin(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckADVPlugin failed: %v", err)
	}

	// Should have declared (pass) and checkout-exists (fail)
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
	if checks[0].Status != StatusPass {
		t.Errorf("check 0: expected pass, got %s", checks[0].Status)
	}
	if checks[1].Status != StatusFail {
		t.Errorf("check 1: expected fail, got %s", checks[1].Status)
	}
}

func TestCheckADVPlugin_Full(t *testing.T) {
	tmpDir := t.TempDir()
	checkoutDir := filepath.Join(tmpDir, "advance")
	os.MkdirAll(checkoutDir, 0755)
	os.MkdirAll(filepath.Join(checkoutDir, "dist"), 0755)
	os.WriteFile(filepath.Join(checkoutDir, "dist", "index.js"), []byte("// built"), 0644)

	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {
				Source:   "https://github.com/Sharper-Flow/Advance.git",
				Checkout: checkoutDir,
			},
		},
	}

	checks, err := CheckADVPlugin(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckADVPlugin failed: %v", err)
	}

	// Should have: declared (pass), checkout (pass), build (pass), state (warn or pass)
	if len(checks) < 3 {
		t.Fatalf("expected at least 3 checks, got %d", len(checks))
	}

	if checks[0].Status != StatusPass {
		t.Errorf("declared: expected pass, got %s", checks[0].Status)
	}
	if checks[1].Status != StatusPass {
		t.Errorf("checkout: expected pass, got %s", checks[1].Status)
	}
	if checks[2].Status != StatusPass {
		t.Errorf("build: expected pass, got %s", checks[2].Status)
	}
}

func TestIsAdvancePlugin(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{source: "https://github.com/Sharper-Flow/Advance.git", want: true},
		{source: "git@github.com:Sharper-Flow/Advance.git", want: true},
		{source: "github.com/Sharper-Flow/Advance", want: true},
		{source: "https://github.com/example/Sharper-Flow/Advance-proxy.git", want: false},
		{source: "https://github.com/Sharper-Flow/Advance-fork.git", want: false},
	}

	for _, tt := range tests {
		got := isAdvancePlugin(tt.source)
		if got != tt.want {
			t.Errorf("isAdvancePlugin(%q) = %v, want %v", tt.source, got, tt.want)
		}
	}
}
