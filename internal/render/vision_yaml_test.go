package render

import (
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRenderVisionServers_SkipsDaemonAndRendersTimeout(t *testing.T) {
	stack := &cfg.Stack{MCP: cfg.MCPSection{Servers: map[string]cfg.Server{
		"vision":   {Port: 6275, Type: "daemon"},
		"context7": {Port: 6276, Command: "npx", Timeout: 10000, Source: "https://github.com/upstash/context7", Description: "docs"},
	}}}
	b, err := RenderVisionServers(stack, "stack.toml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "vision:") {
		t.Fatalf("daemon-type server should be skipped from servers.yaml:\n%s", s)
	}
	if !strings.Contains(s, "context7:") || !strings.Contains(s, "request_timeout: 10000ms") {
		t.Fatalf("context7/request_timeout missing:\n%s", s)
	}
	if !strings.Contains(s, "source: https://github.com/upstash/context7") {
		t.Fatalf("source missing:\n%s", s)
	}
}

func TestRenderVisionServers_SlotGroups(t *testing.T) {
	stack := &cfg.Stack{MCP: cfg.MCPSection{
		Servers: map[string]cfg.Server{
			"context7": {Port: 6276, Command: "npx", Source: "https://github.com/upstash/context7"},
		},
		SlotGroups: map[string]cfg.SlotGroup{
			"playwright-headless": {
				Template:  "playwright-headless",
				BasePort:  6301,
				Count:     4,
				GroupPort: 6300,
				Defaults: &cfg.Server{
					Command: "npx",
					Args:    []string{"playwright", "headless"},
				},
			},
		},
	}}
	b, err := RenderVisionServers(stack, "stack.toml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)

	// servers: must still be present
	if !strings.Contains(s, "servers:") {
		t.Fatalf("servers: missing:\n%s", s)
	}
	if !strings.Contains(s, "context7:") {
		t.Fatalf("context7 missing:\n%s", s)
	}

	// slot_groups: must be present
	if !strings.Contains(s, "slot_groups:") {
		t.Fatalf("slot_groups: missing:\n%s", s)
	}
	if !strings.Contains(s, "playwright-headless:") {
		t.Fatalf("playwright-headless group missing:\n%s", s)
	}

	// Core fields
	if !strings.Contains(s, "template: playwright-headless") {
		t.Fatalf("template missing:\n%s", s)
	}
	if !strings.Contains(s, "base_port: 6301") {
		t.Fatalf("base_port missing:\n%s", s)
	}
	if !strings.Contains(s, "count: 4") {
		t.Fatalf("count missing:\n%s", s)
	}
	if !strings.Contains(s, "group_port: 6300") {
		t.Fatalf("group_port missing:\n%s", s)
	}

	// defaults sub-block
	if !strings.Contains(s, "defaults:") {
		t.Fatalf("defaults: missing:\n%s", s)
	}
	if !strings.Contains(s, "command: npx") {
		t.Fatalf("defaults.command missing:\n%s", s)
	}
}

func TestRenderVisionServers_NoSlotGroups(t *testing.T) {
	stack := &cfg.Stack{MCP: cfg.MCPSection{
		Servers: map[string]cfg.Server{
			"context7": {Port: 6276, Command: "npx"},
		},
	}}
	b, err := RenderVisionServers(stack, "stack.toml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "slot_groups:") {
		t.Fatalf("slot_groups should not appear when empty")
	}
}
