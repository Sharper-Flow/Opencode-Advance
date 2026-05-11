package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

// TestClassifyPluginEntry exercises the full classification matrix from
// design.md (M5 of finalizeMustQueueTriage3).
func TestClassifyPluginEntry(t *testing.T) {
	cases := []struct {
		name         string
		entry        string
		wantName     string
		wantSource   string
		wantCheckout string
		wantErr      bool
	}{
		{
			name:         "plain bare name",
			entry:        "morph-fast-apply",
			wantName:     "morph-fast-apply",
			wantSource:   "",
			wantCheckout: "morph-fast-apply",
		},
		{
			name:         "absolute path",
			entry:        "/abs/path/to/plugin",
			wantName:     "plugin",
			wantSource:   "",
			wantCheckout: "/abs/path/to/plugin",
		},
		{
			name:         "tilde path",
			entry:        "~/dev/oc-plugins/advance/plugin",
			wantName:     "plugin",
			wantSource:   "",
			wantCheckout: "~/dev/oc-plugins/advance/plugin",
		},
		{
			name:         "npm unscoped with @latest",
			entry:        "opencode-openai-codex-auth@latest",
			wantName:     "opencode-openai-codex-auth",
			wantSource:   "npm:opencode-openai-codex-auth@latest",
			wantCheckout: "",
		},
		{
			name:         "npm unscoped with semver",
			entry:        "some-pkg@1.2.3",
			wantName:     "some-pkg",
			wantSource:   "npm:some-pkg@1.2.3",
			wantCheckout: "",
		},
		{
			name:         "npm unscoped with caret range",
			entry:        "some-pkg@^1.0.0",
			wantName:     "some-pkg",
			wantSource:   "npm:some-pkg@^1.0.0",
			wantCheckout: "",
		},
		{
			name:         "npm unscoped with tilde range",
			entry:        "some-pkg@~1.0",
			wantName:     "some-pkg",
			wantSource:   "npm:some-pkg@~1.0",
			wantCheckout: "",
		},
		{
			name:         "npm unscoped with sha-like spec",
			entry:        "some-pkg@abc1234",
			wantName:     "some-pkg",
			wantSource:   "npm:some-pkg@abc1234",
			wantCheckout: "",
		},
		{
			name:         "npm scoped with @latest",
			entry:        "@franlol/opencode-md-table-formatter@latest",
			wantName:     "opencode-md-table-formatter",
			wantSource:   "npm:@franlol/opencode-md-table-formatter@latest",
			wantCheckout: "",
		},
		{
			name:         "npm scoped without version",
			entry:        "@franlol/foo",
			wantName:     "foo",
			wantSource:   "npm:@franlol/foo",
			wantCheckout: "",
		},
		{
			name:         "path with trailing version-like suffix in basename",
			entry:        "/abs/path/pkg@latest",
			wantName:     "pkg",
			wantSource:   "",
			wantCheckout: "/abs/path/pkg@latest",
		},
		{
			name:    "rejects TOML-unsafe spaces",
			entry:   "bad name with spaces",
			wantErr: true,
		},
		{
			name:    "rejects TOML-unsafe brackets",
			entry:   "bad[bracket]name",
			wantErr: true,
		},
		{
			name:    "rejects empty entry",
			entry:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := classifyPluginEntry(tc.entry)
			if tc.wantErr {
				if err == nil {
					t.Errorf("classifyPluginEntry(%q) = %+v, want error", tc.entry, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("classifyPluginEntry(%q) unexpected error: %v", tc.entry, err)
			}
			if got.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tc.wantName)
			}
			if got.Source != tc.wantSource {
				t.Errorf("Source = %q, want %q", got.Source, tc.wantSource)
			}
			if got.Checkout != tc.wantCheckout {
				t.Errorf("Checkout = %q, want %q", got.Checkout, tc.wantCheckout)
			}
		})
	}
}

// setupOpenChadFixture creates a temporary directory tree that mimics
// open-chad's state layout for testing.
func setupOpenChadFixture(t *testing.T) (ReaderConfig, func()) {
	t.Helper()
	tmpDir := t.TempDir()

	cfg := ReaderConfig{
		OpenChadRepo:      filepath.Join(tmpDir, "open-chad"),
		OpenCodeConfigDir: filepath.Join(tmpDir, "opencode"),
		VisionConfigDir:   filepath.Join(tmpDir, "vision"),
		OcPluginsDir:      filepath.Join(tmpDir, "oc-plugins"),
	}

	// Create opencode.json
	openCodeJSON := `{
		"mcp": {
			"context7": {"type": "remote", "url": "http://localhost:6276/mcp", "enabled": true},
			"firecrawl": {"type": "remote", "url": "http://localhost:6281/mcp", "enabled": true}
		},
		"plugin": ["~/dev/oc-plugins/advance/plugin", "~/dev/oc-plugins/morph-fast-apply/plugin", "@franlol/opencode-md-table-formatter@latest"],
		"instructions": ["~/.config/opencode/instructions/identity.md", "~/.config/opencode/instructions/rules.yaml"],
		"theme": "ayu-dark",
		"provider": [
			{"name": "google", "models": [{"name": "gemini-2.5-flash", "limit": {"tokens": 32000}}]}
		]
	}`
	os.MkdirAll(cfg.OpenCodeConfigDir, 0755)
	os.WriteFile(filepath.Join(cfg.OpenCodeConfigDir, "opencode.json"), []byte(openCodeJSON), 0644)

	// Create vision/servers.yaml
	serversYAML := `servers:
  context7:
    port: 6276
    command: npx
    args: ["-y", "@upstash/context7-mcp@latest"]
    autostart: true
    source: https://github.com/upstash/context7
  firecrawl:
    port: 6281
    command: npx
    args: ["-y", "@mendableai/firecrawl-mcp@latest"]
    autostart: true
    source: https://github.com/mendableai/firecrawl
`
	os.MkdirAll(cfg.VisionConfigDir, 0755)
	os.WriteFile(filepath.Join(cfg.VisionConfigDir, "servers.yaml"), []byte(serversYAML), 0644)

	// Create plugin checkouts with .git stubs
	for _, name := range []string{"advance", "morph-fast-apply"} {
		dir := filepath.Join(cfg.OcPluginsDir, name)
		os.MkdirAll(dir, 0755)
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)
		os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("abc123def456"), 0644)
		configContent := `[remote "origin"]
	url = https://github.com/example/` + name + `.git
`
		os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644)
	}

	// Create bundled instructions
	instrDir := filepath.Join(cfg.OpenChadRepo, "config", "opencode", "instructions")
	os.MkdirAll(instrDir, 0755)
	for _, name := range []string{"identity.md", "rules.yaml", "shell_strategy.md"} {
		os.WriteFile(filepath.Join(instrDir, name), []byte("# "+name), 0644)
	}
	// Add a stale instruction
	os.WriteFile(filepath.Join(instrDir, "post_install_verification.md"), []byte("# old"), 0644)

	// Create bundled skills
	skillsDir := filepath.Join(cfg.OpenChadRepo, "config", "opencode", "skills")
	os.MkdirAll(filepath.Join(skillsDir, "lgrep"), 0755)
	os.MkdirAll(filepath.Join(skillsDir, "worktree"), 0755)
	os.WriteFile(filepath.Join(skillsDir, "lgrep", "SKILL.md"), []byte("# lgrep"), 0644)
	os.WriteFile(filepath.Join(skillsDir, "worktree", "SKILL.md"), []byte("# worktree"), 0644)

	return cfg, func() {}
}

func TestReadOpenChadState_Full(t *testing.T) {
	cfg, cleanup := setupOpenChadFixture(t)
	defer cleanup()

	state, err := ReadOpenChadState(cfg)
	if err != nil {
		t.Fatalf("ReadOpenChadState failed: %v", err)
	}

	// MCP servers merged from opencode.json + vision/servers.yaml
	if len(state.MCPServers) != 2 {
		t.Errorf("expected 2 MCP servers, got %d", len(state.MCPServers))
	}

	ctx7, ok := state.MCPServers["context7"]
	if !ok {
		t.Fatal("context7 MCP server not found")
	}
	if ctx7.Port != 6276 {
		t.Errorf("context7 port = %d, want 6276", ctx7.Port)
	}
	if ctx7.Command != "npx" {
		t.Errorf("context7 command = %q, want npx", ctx7.Command)
	}
	if ctx7.URL != "http://localhost:6276/mcp" {
		t.Errorf("context7 url = %q", ctx7.URL)
	}

	// Plugins: 2 path-based from oc-plugins + 1 npm-scoped from opencode.json.
	// Scoped npm (`@franlol/opencode-md-table-formatter@latest`) was previously
	// dropped; classifyPluginEntry now captures it as an npm source.
	if len(state.Plugins) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(state.Plugins))
	}
	// Verify the scoped npm entry is captured with the canonical npm source form.
	var foundNPM bool
	for _, p := range state.Plugins {
		if p.Name == "opencode-md-table-formatter" {
			foundNPM = true
			if p.Source != "npm:@franlol/opencode-md-table-formatter@latest" {
				t.Errorf("npm plugin source = %q, want canonical npm form", p.Source)
			}
			if p.Checkout != "" {
				t.Errorf("npm plugin checkout = %q, want empty", p.Checkout)
			}
		}
	}
	if !foundNPM {
		t.Errorf("expected npm-scoped plugin %q not found in state", "opencode-md-table-formatter")
	}

	// Instructions: 2 from opencode.json + 3 from bundled dir (stale excluded)
	if len(state.Instructions) != 5 {
		t.Errorf("expected 5 instructions, got %d", len(state.Instructions))
	}

	// Skills
	if len(state.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(state.Skills))
	}

	// Theme
	if state.OpenCode == nil || state.OpenCode.Theme != "ayu-dark" {
		t.Errorf("theme = %q, want ayu-dark", state.OpenCode.Theme)
	}

	// Providers
	if len(state.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(state.Providers))
	}

	// Warnings: stale instruction should be logged
	foundStale := false
	for _, w := range state.Warnings {
		if contains(w, "post_install_verification") {
			foundStale = true
			break
		}
	}
	if !foundStale {
		t.Error("expected warning about stale instruction")
	}
}

func TestPluginCheckoutMatches(t *testing.T) {
	tests := []struct {
		checkout string
		name     string
		want     bool
	}{
		{checkout: "/tmp/oc-plugins/advance", name: "advance", want: true},
		{checkout: "/tmp/oc-plugins/advance/plugin", name: "advance", want: true},
		{checkout: "/tmp/oc-plugins/advance-proxy", name: "advance", want: false},
		{checkout: "/tmp/oc-plugins/foo/plugin", name: "bar", want: false},
	}

	for _, tt := range tests {
		got := pluginCheckoutMatches(tt.checkout, tt.name)
		if got != tt.want {
			t.Errorf("pluginCheckoutMatches(%q, %q) = %v, want %v", tt.checkout, tt.name, got, tt.want)
		}
	}
}

func TestReadOpenChadState_MissingVisionConfig(t *testing.T) {
	cfg, cleanup := setupOpenChadFixture(t)
	defer cleanup()

	// Remove vision config
	os.RemoveAll(cfg.VisionConfigDir)

	state, err := ReadOpenChadState(cfg)
	if err != nil {
		t.Fatalf("ReadOpenChadState failed: %v", err)
	}

	// Should still succeed but with skipped source
	foundSkipped := false
	for _, s := range state.SkippedSources {
		if contains(s, "vision") {
			foundSkipped = true
			break
		}
	}
	if !foundSkipped {
		t.Error("expected vision config to be in skipped sources")
	}

	// MCP servers should still have data from opencode.json (urls only, no ports)
	if len(state.MCPServers) != 2 {
		t.Errorf("expected 2 MCP servers from opencode.json, got %d", len(state.MCPServers))
	}
}

func TestReadOpenChadState_EmptyPlugins(t *testing.T) {
	cfg, cleanup := setupOpenChadFixture(t)
	defer cleanup()

	// Remove plugins
	os.RemoveAll(cfg.OcPluginsDir)

	state, err := ReadOpenChadState(cfg)
	if err != nil {
		t.Fatalf("ReadOpenChadState failed: %v", err)
	}

	// Plugins from opencode.json: 2 path-based + 1 npm-scoped (now captured by
	// classifyPluginEntry rather than dropped). Path-based entries still have
	// no on-disk checkout when OcPluginsDir is removed, but the entries
	// themselves remain.
	if len(state.Plugins) != 3 {
		t.Errorf("expected 3 plugins from opencode.json, got %d", len(state.Plugins))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
