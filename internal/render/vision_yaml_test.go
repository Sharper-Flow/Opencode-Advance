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
