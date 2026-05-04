package health

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func testAdvanceStack(t *testing.T) (*cfg.Stack, string) {
	t.Helper()
	tmpDir := t.TempDir()
	checkoutDir := filepath.Join(tmpDir, "advance")
	if err := os.MkdirAll(filepath.Join(checkoutDir, "dist"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkoutDir, "dist", "index.js"), []byte("// built"), 0644); err != nil {
		t.Fatal(err)
	}
	return &cfg.Stack{Plugins: cfg.PluginsSection{"advance": {Source: "https://github.com/Sharper-Flow/Advance.git", Checkout: checkoutDir}}}, tmpDir
}

func findCheck(checks []Check, name string) (Check, bool) {
	for _, check := range checks {
		if check.Name == name {
			return check, true
		}
	}
	return Check{}, false
}

func TestCheckADVPlugin_LocalADVStateSpecsOnlyPasses(t *testing.T) {
	stack, root := testAdvanceStack(t)
	if err := os.MkdirAll(filepath.Join(root, ".adv", "specs"), 0755); err != nil {
		t.Fatal(err)
	}

	checks, err := CheckADVPlugin(context.Background(), stack, Options{ProjectRoot: root})
	if err != nil {
		t.Fatalf("CheckADVPlugin: %v", err)
	}
	check, ok := findCheck(checks, "adv-local-state")
	if !ok {
		t.Fatalf("missing adv-local-state check: %#v", checks)
	}
	if check.Status != StatusPass {
		t.Fatalf("status = %s, want pass: %s", check.Status, check.Message)
	}
}

func TestCheckADVPlugin_LocalADVArchiveBundlePasses(t *testing.T) {
	stack, root := testAdvanceStack(t)
	if err := os.MkdirAll(filepath.Join(root, ".adv", "archive", "2026-05-04-example"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".adv", "archive", "2026-05-04-example", "change.json"), []byte(`{"id":"example"}`), 0644); err != nil {
		t.Fatal(err)
	}

	checks, err := CheckADVPlugin(context.Background(), stack, Options{ProjectRoot: root})
	if err != nil {
		t.Fatalf("CheckADVPlugin: %v", err)
	}
	check, ok := findCheck(checks, "adv-local-state")
	if !ok {
		t.Fatalf("missing adv-local-state check: %#v", checks)
	}
	if check.Status != StatusPass {
		t.Fatalf("status = %s, want pass: %s", check.Status, check.Message)
	}
}

func TestCheckADVPlugin_LocalADVArchiveResidueWarns(t *testing.T) {
	stack, root := testAdvanceStack(t)
	if err := os.MkdirAll(filepath.Join(root, ".adv", "archive"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".adv", "archive", ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	checks, err := CheckADVPlugin(context.Background(), stack, Options{ProjectRoot: root})
	if err != nil {
		t.Fatalf("CheckADVPlugin: %v", err)
	}
	check, ok := findCheck(checks, "adv-local-state")
	if !ok {
		t.Fatalf("missing adv-local-state check: %#v", checks)
	}
	if check.Status != StatusWarn {
		t.Fatalf("status = %s, want warn", check.Status)
	}
	if !strings.Contains(check.Message, "archive") || !strings.Contains(check.Hint, "adv_migrate_cleanup") {
		t.Fatalf("unexpected message/hint: %q / %q", check.Message, check.Hint)
	}
}

func TestCheckADVPlugin_LocalADVLegacyDirsWarn(t *testing.T) {
	stack, root := testAdvanceStack(t)
	for _, dir := range []string{"changes", "db"} {
		if err := os.MkdirAll(filepath.Join(root, ".adv", dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".adv", "agenda.jsonl"), []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	checks, err := CheckADVPlugin(context.Background(), stack, Options{ProjectRoot: root})
	if err != nil {
		t.Fatalf("CheckADVPlugin: %v", err)
	}
	check, ok := findCheck(checks, "adv-local-state")
	if !ok {
		t.Fatalf("missing adv-local-state check: %#v", checks)
	}
	if check.Status != StatusWarn {
		t.Fatalf("status = %s, want warn", check.Status)
	}
	for _, want := range []string{"changes", "db", "agenda.jsonl"} {
		if !strings.Contains(check.Message, want) {
			t.Fatalf("message missing %q: %q", want, check.Message)
		}
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

func TestCheckADVPlugin_EmptyArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	checkoutDir := filepath.Join(tmpDir, "advance")
	os.MkdirAll(checkoutDir, 0755)
	os.MkdirAll(filepath.Join(checkoutDir, "dist"), 0755)
	os.WriteFile(filepath.Join(checkoutDir, "dist", "index.js"), []byte(""), 0644)

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

	// Should have: declared (pass), checkout (pass), build (fail/warn), state (warn or pass)
	if len(checks) < 3 {
		t.Fatalf("expected at least 3 checks, got %d", len(checks))
	}

	if checks[0].Status != StatusPass {
		t.Errorf("declared: expected pass, got %s", checks[0].Status)
	}
	if checks[1].Status != StatusPass {
		t.Errorf("checkout: expected pass, got %s", checks[1].Status)
	}
	if checks[2].Status != StatusFail {
		t.Errorf("build: expected fail for empty artifact, got %s", checks[2].Status)
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
