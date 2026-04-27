package render

import (
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRenderMCPFragment_UsesTimeoutPrecedence(t *testing.T) {
	tests := []struct {
		name string
		srv  cfg.Server
		want int
	}{
		{
			name: "explicit timeout wins",
			srv:  cfg.Server{Port: 6276, Timeout: 12000, RequestTimeout: "3s"},
			want: 12000,
		},
		{
			name: "request timeout duration fallback",
			srv:  cfg.Server{Port: 6276, RequestTimeout: "3s"},
			want: 3000,
		},
		{
			name: "default timeout fallback",
			srv:  cfg.Server{Port: 6276},
			want: defaultTimeoutMS,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fragment := RenderMCPFragment(tc.srv)
			if got := fragment["timeout"]; got != tc.want {
				t.Fatalf("timeout = %#v, want %d", got, tc.want)
			}
		})
	}
}

func TestRenderSlotGroupFragment(t *testing.T) {
	g := cfg.SlotGroup{
		Template:  "playwright-headless",
		BasePort:  6301,
		Count:     4,
		GroupPort: 6300,
		Defaults:  &cfg.Server{Timeout: 12000},
	}
	frag := RenderSlotGroupFragment(g)
	if frag["type"] != "remote" {
		t.Fatalf("type = %#v, want remote", frag["type"])
	}
	wantURL := "http://localhost:6300/mcp"
	if frag["url"] != wantURL {
		t.Fatalf("url = %#v, want %s", frag["url"], wantURL)
	}
	if frag["enabled"] != true {
		t.Fatalf("enabled = %#v, want true", frag["enabled"])
	}
	if frag["oauth"] != false {
		t.Fatalf("oauth = %#v, want false", frag["oauth"])
	}
	if frag["timeout"] != 12000 {
		t.Fatalf("timeout = %#v, want 12000", frag["timeout"])
	}
}
