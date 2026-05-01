package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestStackExample_ParsesWithDeferredSections(t *testing.T) {
	root := repoRoot(t)
	stack, err := cfg.Load(filepath.Join(root, "stack.example.toml"))
	if err != nil {
		t.Fatalf("Load(stack.example.toml) failed: %v", err)
	}
	// Phase 3 graduated sections must be on typed fields, not deferred.
	for _, wantTyped := range []string{"providers", "permissions", "watcher", "lsp", "skills", "formatters", "commands", "opencode", "session"} {
		if _, ok := stack.DeferredSections[wantTyped]; ok {
			t.Errorf("%s should be on typed field, not DeferredSections", wantTyped)
		}
	}
	// Deferred sections that still appear in stack.example.toml remain deferred.
	for _, wantDeferred := range []string{"discord"} {
		if _, ok := stack.DeferredSections[wantDeferred]; !ok {
			t.Errorf("%s should remain in DeferredSections", wantDeferred)
		}
	}
	if _, ok := stack.MCP.Servers["vision"]; !ok {
		t.Fatal("vision server missing")
	}
	if _, ok := stack.Plugins["advance"]; !ok {
		t.Fatal("advance plugin missing from typed Plugins section")
	}
	if len(stack.Instructions.Order) == 0 {
		t.Fatal("instructions order missing from typed Instructions section")
	}
}

func TestUnknownSection_FailsWithFieldPath(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "stack.toml")
	content := "[meta]\nversion=\"1.0.0\"\n\n[mcp.servers.a]\nport=6276\ncommand=\"echo\"\n\n[foobar]\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := cfg.Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "foobar") {
		t.Fatalf("expected foobar in error: %v", err)
	}
}
