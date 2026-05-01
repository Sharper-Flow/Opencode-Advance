package migrate

import (
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestEmitInit(t *testing.T) {
	output, err := EmitInit()
	if err != nil {
		t.Fatalf("EmitInit failed: %v", err)
	}

	// Should contain essential sections
	sections := []string{
		"[mcp.servers.vision]",
		"[mcp.servers.context7]",
		"[plugins.advance]",
		"[instructions]",
		"[providers.google]",
		"[opencode]",
	}
	for _, section := range sections {
		if !strings.Contains(output, section) {
			t.Errorf("missing section: %s", section)
		}
	}

	// Should have sensible defaults
	if !strings.Contains(output, `theme = "obsidian"`) {
		t.Error("missing default theme")
	}
	if !strings.Contains(output, "6275") {
		t.Error("missing vision port")
	}
	if !strings.Contains(output, "6276") {
		t.Error("missing context7 port")
	}
}

func TestEmitInit_ValidTOML(t *testing.T) {
	output, err := EmitInit()
	if err != nil {
		t.Fatalf("EmitInit failed: %v", err)
	}

	// Write to temp file and parse
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/stack.toml"
	if err := writeFile(tmpFile, []byte(output)); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	stack, err := cfg.Load(tmpFile)
	if err != nil {
		t.Fatalf("parse init TOML: %v", err)
	}

	// Verify parsed structure
	if stack.Meta.Name != "default" {
		t.Errorf("meta.name = %q, want default", stack.Meta.Name)
	}

	if len(stack.MCP.Servers) != 2 {
		t.Errorf("mcp servers = %d, want 2", len(stack.MCP.Servers))
	}

	vision, ok := stack.MCP.Servers["vision"]
	if !ok {
		t.Fatal("vision server not found")
	}
	if vision.Port != 6275 {
		t.Errorf("vision port = %d, want 6275", vision.Port)
	}

	if len(stack.Plugins) != 1 {
		t.Errorf("plugins = %d, want 1", len(stack.Plugins))
	}

	adv, ok := stack.Plugins["advance"]
	if !ok {
		t.Fatal("advance plugin not found")
	}
	if adv.Source != "https://github.com/Sharper-Flow/Advance.git" {
		t.Errorf("advance source = %q", adv.Source)
	}

	if stack.OpenCode.Theme != "obsidian" {
		t.Errorf("theme = %q, want obsidian", stack.OpenCode.Theme)
	}
}
