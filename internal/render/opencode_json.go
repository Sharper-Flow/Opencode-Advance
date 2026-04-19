package render

import (
	"fmt"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// defaultTimeoutMS is the fallback per-request timeout for an MCP server
// when neither Server.Timeout nor Server.RequestTimeout yields a positive
// value. Emitted into opencode.json as the .timeout field.
const defaultTimeoutMS = 5000

// RenderMCPFragment renders one opencode.json .mcp entry for a server.
func RenderMCPFragment(s cfg.Server) Fragment {
	return Fragment{
		"type":    "remote",
		"url":     fmt.Sprintf("http://localhost:%d/mcp", s.Port),
		"enabled": s.IsEnabled(),
		"oauth":   false,
		"timeout": effectiveTimeoutMS(s),
	}
}

func effectiveTimeoutMS(s cfg.Server) int {
	if s.Timeout > 0 {
		return s.Timeout
	}
	if s.RequestTimeout != "" {
		if d, err := time.ParseDuration(s.RequestTimeout); err == nil {
			return int(d.Milliseconds())
		}
	}
	return defaultTimeoutMS
}
