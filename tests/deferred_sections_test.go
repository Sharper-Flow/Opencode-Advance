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
	for _, want := range []string{"providers", "agents", "permissions", "watcher", "lsp", "session", "discord", "skills", "formatters", "commands", "opencode"} {
		if _, ok := stack.DeferredSections[want]; !ok {
			t.Fatalf("DeferredSections missing %q", want)
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
