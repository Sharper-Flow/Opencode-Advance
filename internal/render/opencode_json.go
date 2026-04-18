package render

import (
	"fmt"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// RenderMCPFragment renders one opencode.json .mcp entry.
func RenderMCPFragment(_ string, s cfg.Server) Fragment {
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
	return 5000
}
