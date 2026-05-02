package health

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestCheckCross_PortCollision(t *testing.T) {
	stack := &cfg.Stack{
		MCP: cfg.MCPSection{
			Servers: map[string]cfg.Server{
				"context7":  {Port: 6276},
				"firecrawl": {Port: 6276}, // collision
			},
		},
	}

	checks, err := CheckCross(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckCross failed: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "mcp-port-collision" && c.Status == StatusFail {
			found = true
		}
	}
	if !found {
		t.Error("expected port collision failure")
	}
}

func TestCheckCross_PluginDrift(t *testing.T) {
	tmpDir := t.TempDir()
	checkoutDir := filepath.Join(tmpDir, "advance")
	os.MkdirAll(checkoutDir, 0755)
	gitDir := filepath.Join(checkoutDir, ".git")
	os.MkdirAll(gitDir, 0755)
	configContent := `[remote "origin"]
	url = https://github.com/other/Advance.git
`
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644)

	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {
				Source:   "https://github.com/Sharper-Flow/Advance.git",
				Checkout: checkoutDir,
			},
		},
	}

	checks, err := CheckCross(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckCross failed: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "plugin-drift-advance" && c.Status == StatusFail {
			found = true
		}
	}
	if !found {
		t.Error("expected plugin drift failure")
	}
}

func TestCheckCross_PluginMatch(t *testing.T) {
	tmpDir := t.TempDir()
	checkoutDir := filepath.Join(tmpDir, "advance")
	os.MkdirAll(checkoutDir, 0755)
	gitDir := filepath.Join(checkoutDir, ".git")
	os.MkdirAll(gitDir, 0755)
	configContent := `[remote "origin"]
	url = https://github.com/Sharper-Flow/Advance.git
`
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644)

	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {
				Source:   "https://github.com/Sharper-Flow/Advance.git",
				Checkout: checkoutDir,
			},
		},
	}

	checks, err := CheckCross(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckCross failed: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "plugin-drift-advance" && c.Status == StatusPass {
			found = true
		}
	}
	if !found {
		t.Error("expected plugin match pass")
	}
}

func TestCheckCross_InstructionMissing(t *testing.T) {
	stack := &cfg.Stack{
		Instructions: cfg.InstructionsSection{
			Order: []string{"/nonexistent/path/identity.md"},
		},
	}

	checks, err := CheckCross(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckCross failed: %v", err)
	}

	found := false
	for _, c := range checks {
		if strings.Contains(c.Name, "instruction-exists-") && c.Status == StatusWarn {
			found = true
		}
	}
	if !found {
		t.Error("expected instruction missing warning")
	}
}

func TestUrlsMatch(t *testing.T) {
	tests := []struct {
		a, b   string
		expect bool
	}{
		{"https://github.com/foo/bar.git", "https://github.com/foo/bar", true},
		{"https://github.com/foo/bar", "git@github.com:foo/bar.git", true},
		{"https://github.com/foo/bar", "https://github.com/other/bar", false},
	}
	for _, tt := range tests {
		got := urlsMatch(tt.a, tt.b)
		if got != tt.expect {
			t.Errorf("urlsMatch(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expect)
		}
	}
}

func TestResolvePathAssets(t *testing.T) {
	assetsRoot := filepath.Join(t.TempDir(), "assets")
	got := resolvePath("{assets}/instructions/identity.md", Options{
		SkillsAssetsRoot: filepath.Join(assetsRoot, "skills"),
	})
	want := filepath.Join(assetsRoot, "instructions", "identity.md")
	if got != want {
		t.Fatalf("resolvePath assets = %q, want %q", got, want)
	}
}
