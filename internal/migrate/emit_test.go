package migrate

import (
	"strings"
	"testing"
)

func TestEmitTOML(t *testing.T) {
	state := &OpenChadState{
		MCPServers: map[string]MCPServerState{
			"context7": {
				Port:      6276,
				Command:   "npx",
				Args:      []string{"-y", "@upstash/context7-mcp@latest"},
				Autostart: true,
				Source:    "https://github.com/upstash/context7",
			},
		},
		Plugins: []PluginState{
			{Name: "advance", Source: "https://github.com/Sharper-Flow/Advance.git", Ref: "abc123"},
		},
		Instructions: []string{"~/.config/opencode/instructions/identity.md"},
		Providers: map[string]ProviderState{
			"google": {
				Name: "google",
				Models: []ModelState{
					{Name: "gemini-2.5-flash"},
				},
			},
		},
		OpenCode: &OpenCodeState{Theme: "obsidian"},
	}

	output, err := EmitTOML(state)
	if err != nil {
		t.Fatalf("EmitTOML failed: %v", err)
	}

	// Should contain meta header
	if !strings.Contains(output, "migrated from open-chad") {
		t.Error("missing migration header")
	}

	// Should contain MCP server
	if !strings.Contains(output, "[mcp.servers.context7]") {
		t.Error("missing context7 MCP server section")
	}
	if !strings.Contains(output, "port = 6276") {
		t.Error("missing port")
	}

	// Should contain plugin
	if !strings.Contains(output, "[plugins.advance]") {
		t.Error("missing plugins section")
	}
	if !strings.Contains(output, `source = "https://github.com/Sharper-Flow/Advance.git"`) {
		t.Error("missing advance plugin source")
	}

	// Should contain instructions
	if !strings.Contains(output, "[instructions]") {
		t.Error("missing instructions section")
	}

	// Should contain provider
	if !strings.Contains(output, "[providers.google]") {
		t.Error("missing google provider")
	}

	// Should be valid TOML (basic check)
	if !strings.Contains(output, "[meta]") {
		t.Error("missing meta section")
	}
}

func TestEmitTOML_EmptyState(t *testing.T) {
	state := &OpenChadState{}

	output, err := EmitTOML(state)
	if err != nil {
		t.Fatalf("EmitTOML failed: %v", err)
	}

	// Should still produce valid output with just meta
	if !strings.Contains(output, "[meta]") {
		t.Error("missing meta section")
	}
}
