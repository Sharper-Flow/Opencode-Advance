package render

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMergeMCP_PreservesUserAddedAndOverwritesDeclared(t *testing.T) {
	existing := []byte(`{"mcp":{"user_added":{"type":"remote","url":"https://example.com/mcp"},"context7":{"type":"remote","url":"http://old:1/mcp"}},"theme":"obsidian"}`)
	declared := map[string]Fragment{
		"context7": {"type": "remote", "url": "http://localhost:6276/mcp", "enabled": true, "oauth": false, "timeout": 10000},
	}
	out, err := MergeMCP(existing, declared)
	if err != nil {
		t.Fatalf("MergeMCP: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	mcp := root["mcp"].(map[string]any)
	if _, ok := mcp["user_added"]; !ok {
		t.Fatal("user_added key lost")
	}
	ctx := mcp["context7"].(map[string]any)
	if ctx["url"] != "http://localhost:6276/mcp" {
		t.Fatalf("declared url not overwritten: %#v", ctx)
	}
	if root["theme"] != "obsidian" {
		t.Fatalf("top-level key lost: %#v", root)
	}
	if !strings.HasSuffix(string(out), "\n") {
		t.Fatal("expected trailing newline")
	}
}
