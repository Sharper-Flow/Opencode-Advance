package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestEmitTOML_GoldenFullState(t *testing.T) {
	state := &OpenChadState{
		MCPServers: map[string]MCPServerState{
			"context7": {
				Port:      6276,
				Command:   "npx",
				Args:      []string{"-y", "@upstash/context7-mcp@latest"},
				Autostart: true,
				Source:    "https://github.com/upstash/context7",
				Timeout:   10000,
			},
			"firecrawl": {
				Port:      6281,
				Command:   "npx",
				Args:      []string{"-y", "@mendableai/firecrawl-mcp@latest"},
				Autostart: true,
				Source:    "https://github.com/mendableai/firecrawl",
				Timeout:   15000,
			},
		},
		Plugins: []PluginState{
			{
				Name:     "advance",
				Source:   "https://github.com/Sharper-Flow/Advance.git",
				Ref:      "abc123def456",
				Checkout: "~/dev/oc-plugins/advance",
				Build:    []string{"pnpm install", "pnpm build"},
				Provides: []string{"adv-commands", "adv-agents", "adv-skills"},
			},
		},
		Instructions: []string{
			"~/.config/opencode/instructions/identity.md",
			"~/.config/opencode/instructions/rules.yaml",
		},
		Providers: map[string]ProviderState{
			"google": {
				Name: "google",
				Models: []ModelState{
					{Name: "gemini-2.5-flash", Limit: &LimitState{Tokens: intPtr(32000)}},
				},
			},
		},
		OpenCode: &OpenCodeState{Theme: "obsidian"},
	}

	output, err := EmitTOML(state)
	if err != nil {
		t.Fatalf("EmitTOML failed: %v", err)
	}

	// Verify all major sections are present
	sections := []string{
		"[mcp.servers.context7]",
		"[mcp.servers.firecrawl]",
		"[plugins.advance]",
		"[instructions]",
		"[providers.google]",
		"[providers.google.models.gemini-2.5-flash]",
		"[opencode]",
	}
	for _, section := range sections {
		if !strings.Contains(output, section) {
			t.Errorf("missing section: %s", section)
		}
	}

	// Verify specific values
	if !strings.Contains(output, `port = 6276`) {
		t.Error("missing context7 port")
	}
	if !strings.Contains(output, `command = "npx"`) {
		t.Error("missing command")
	}
	if !strings.Contains(output, `ref = "abc123def456"`) {
		t.Error("missing plugin ref")
	}
	if !strings.Contains(output, `context = 32000`) {
		t.Error("missing model context limit")
	}
}

func TestEmitTOML_RoundTrip(t *testing.T) {
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
			{Name: "advance", Source: "https://github.com/Sharper-Flow/Advance.git", Checkout: "~/dev/oc-plugins/advance"},
		},
		Instructions: []string{"~/.config/opencode/instructions/identity.md"},
		OpenCode:     &OpenCodeState{Theme: "obsidian"},
	}

	output, err := EmitTOML(state)
	if err != nil {
		t.Fatalf("EmitTOML failed: %v", err)
	}

	// Write to temp file and parse back
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "stack.toml")
	if err := os.WriteFile(tmpFile, []byte(output), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	// Parse the emitted TOML
	stack, err := cfg.Load(tmpFile)
	if err != nil {
		t.Fatalf("parse emitted TOML: %v", err)
	}

	// Verify parsed values
	if stack.Meta.Name != "migrated" {
		t.Errorf("meta.name = %q, want migrated", stack.Meta.Name)
	}

	if len(stack.MCP.Servers) != 1 {
		t.Errorf("mcp servers = %d, want 1", len(stack.MCP.Servers))
	}

	ctx7, ok := stack.MCP.Servers["context7"]
	if !ok {
		t.Fatal("context7 not found in parsed MCP servers")
	}
	if ctx7.Port != 6276 {
		t.Errorf("context7 port = %d, want 6276", ctx7.Port)
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

	if len(stack.Instructions.Order) != 1 {
		t.Errorf("instructions = %d, want 1", len(stack.Instructions.Order))
	}

	if stack.OpenCode.Theme != "obsidian" {
		t.Errorf("theme = %q, want obsidian", stack.OpenCode.Theme)
	}
}

func intPtr(i int) *int {
	return &i
}
