package config

import (
	"os"
	"path/filepath"
	"testing"
)

const minimalTOML = `
[meta]
version = "1.0.0"
name = "test"

[mcp.servers.context7]
port = 6276
command = "npx"
args = ["-y", "@upstash/context7-mcp@latest"]
timeout = 10000
autostart = true
source = "https://github.com/upstash/context7"
description = "Library docs"
`

func TestParse_MinimalStackParsesTypedSections(t *testing.T) {
	stack, err := Parse([]byte(minimalTOML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if stack.Meta.Version != "1.0.0" {
		t.Errorf("Meta.Version = %q, want 1.0.0", stack.Meta.Version)
	}
	srv, ok := stack.MCP.Servers["context7"]
	if !ok {
		t.Fatalf("context7 missing from servers")
	}
	if srv.Port != 6276 {
		t.Errorf("port = %d, want 6276", srv.Port)
	}
	if srv.Command != "npx" {
		t.Errorf("command = %q, want npx", srv.Command)
	}
	if len(srv.Args) != 2 {
		t.Errorf("args len = %d, want 2", len(srv.Args))
	}
	if srv.Source != "https://github.com/upstash/context7" {
		t.Errorf("source not preserved: %q", srv.Source)
	}
}

func TestParse_DeferredSectionsAreCollected(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[plugins.advance]
source = "https://github.com/Sharper-Flow/Advance.git"
ref = "trunk"

[providers.openai]

[providers.openai.models."gpt-5.2"]
name = "GPT 5.2"
context = 272000

[agents]
build = "anthropic/claude-opus-4-6"
`
	stack, err := Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed on stack with deferred sections: %v", err)
	}
	for _, expected := range []string{"providers", "agents"} {
		if _, ok := stack.DeferredSections[expected]; !ok {
			t.Errorf("DeferredSections missing %q; got keys: %v", expected, deferredKeys(stack))
		}
	}
	// Typed sections still populated.
	if stack.Meta.Version != "1.0.0" {
		t.Errorf("Meta.Version lost")
	}
	if _, ok := stack.MCP.Servers["a"]; !ok {
		t.Errorf("mcp.servers.a lost")
	}
	// plugins is now a typed section.
	if _, ok := stack.Plugins["advance"]; !ok {
		t.Errorf("plugins.advance not typed")
	}
	if stack.Plugins["advance"].Source != "https://github.com/Sharper-Flow/Advance.git" {
		t.Errorf("plugins.advance source not decoded")
	}
}

func TestParse_MalformedTOMLReturnsParseError(t *testing.T) {
	_, err := Parse([]byte("this is not valid toml [[[\n"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *ParseError
	// errors.As via unwrap
	if pe = asParseError(err); pe == nil {
		t.Errorf("err = %T %v, want *ParseError", err, err)
	}
}

func TestParseFile_ReadsDiskFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stack.toml")
	if err := os.WriteFile(path, []byte(minimalTOML), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	stack, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if stack.Meta.Version != "1.0.0" {
		t.Errorf("Meta.Version mismatched after ParseFile")
	}
}

func TestParseFile_MissingFileReturnsParseError(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if asParseError(err) == nil {
		t.Errorf("err = %T, want *ParseError", err)
	}
}

// asParseError returns the inner *ParseError if present, else nil.
func asParseError(err error) *ParseError {
	for {
		if err == nil {
			return nil
		}
		if pe, ok := err.(*ParseError); ok {
			return pe
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return nil
		}
		err = u.Unwrap()
	}
}

func deferredKeys(s *Stack) []string {
	out := make([]string, 0, len(s.DeferredSections))
	for k := range s.DeferredSections {
		out = append(out, k)
	}
	return out
}
