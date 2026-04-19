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
