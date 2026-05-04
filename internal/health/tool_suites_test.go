package health

import (
	"context"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestCheckToolSuites_NoSuites(t *testing.T) {
	stack := &cfg.Stack{}
	opts := Options{}

	checks, err := CheckToolSuites(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckToolSuites: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if checks[0].Name != "tool-suites" {
		t.Errorf("expected tool-suites, got %s", checks[0].Name)
	}
	if checks[0].Status != StatusPass {
		t.Errorf("expected pass, got %s", checks[0].Status)
	}
}

func TestCheckToolSuites_ValidSuite(t *testing.T) {
	enabled := true
	stack := &cfg.Stack{
		MCP: cfg.MCPSection{
			Servers: map[string]cfg.Server{
				"kagi":     {Port: 6279, Command: "uvx", Required: true, Enabled: &enabled},
				"context7": {Port: 6276, Command: "npx", Enabled: &enabled},
				"lgrep":    {Port: 6277, Command: "npx", Enabled: &enabled},
			},
			ToolSuites: map[string]cfg.ToolSuite{
				"research": {
					Servers:     []string{"kagi", "context7"},
					Description: "web search and docs",
				},
				"code-intel": {
					Servers: []string{"lgrep"},
				},
			},
		},
	}
	opts := Options{}

	checks, err := CheckToolSuites(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckToolSuites: %v", err)
	}

	suiteCheckMap := map[string]Check{}
	for _, c := range checks {
		suiteCheckMap[c.Name] = c
	}

	if c, ok := suiteCheckMap["tool-suites.research"]; !ok {
		t.Error("expected tool-suites.research check")
	} else if c.Status != StatusPass {
		t.Errorf("research suite: expected pass, got %s: %s", c.Status, c.Message)
	}

	if c, ok := suiteCheckMap["tool-suites.code-intel"]; !ok {
		t.Error("expected tool-suites.code-intel check")
	} else if c.Status != StatusPass {
		t.Errorf("code-intel suite: expected pass, got %s: %s", c.Status, c.Message)
	}
}

func TestCheckToolSuites_InvalidServerReference(t *testing.T) {
	enabled := true
	stack := &cfg.Stack{
		MCP: cfg.MCPSection{
			Servers: map[string]cfg.Server{
				"kagi": {Port: 6279, Command: "uvx", Enabled: &enabled},
			},
			ToolSuites: map[string]cfg.ToolSuite{
				"research": {
					Servers: []string{"kagi", "nonexistent"},
				},
			},
		},
	}
	opts := Options{}

	checks, err := CheckToolSuites(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckToolSuites: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "tool-suites.research" {
			found = true
			if c.Status != StatusWarn {
				t.Errorf("expected warn for missing server, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !found {
		t.Error("expected tool-suites.research check")
	}
}

func TestCheckToolSuites_UncoveredServers(t *testing.T) {
	enabled := true
	stack := &cfg.Stack{
		MCP: cfg.MCPSection{
			Servers: map[string]cfg.Server{
				"kagi":     {Port: 6279, Command: "uvx", Enabled: &enabled},
				"context7": {Port: 6276, Command: "npx", Enabled: &enabled},
				"lgrep":    {Port: 6277, Command: "npx", Enabled: &enabled},
			},
			ToolSuites: map[string]cfg.ToolSuite{
				"research": {
					Servers: []string{"kagi", "context7"},
				},
			},
		},
	}
	opts := Options{}

	checks, err := CheckToolSuites(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckToolSuites: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "tool-suites.uncovered-servers" {
			found = true
			if c.Status != StatusWarn {
				t.Errorf("expected warn for uncovered, got %s: %s", c.Status, c.Message)
			}
		}
	}
	if !found {
		t.Error("expected tool-suites.uncovered-servers check")
	}
}
