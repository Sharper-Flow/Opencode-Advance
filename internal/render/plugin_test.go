package render

import (
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRenderPluginFragment_GitSourceUsesPath(t *testing.T) {
	p := cfg.Plugin{
		Source: "https://github.com/example/advance.git",
		Path:   "/tmp/advance/plugin/dist/index.js",
	}

	got := RenderPluginFragment(p)
	if got != "/tmp/advance/plugin/dist/index.js" {
		t.Fatalf("got %q want resolved path", got)
	}
}

func TestRenderPluginFragment_NPMSourceUsesLiteral(t *testing.T) {
	p := cfg.Plugin{Source: "npm:@scope/codex-auth@1.2.3"}

	got := RenderPluginFragment(p)
	if got != "@scope/codex-auth@1.2.3" {
		t.Fatalf("got %q want npm literal", got)
	}
}
