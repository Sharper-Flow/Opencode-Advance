package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

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

	// Create open-chad.json
	openchadJSON := `{"discordPresence": {"enabled": true, "clientId": "123456789"}}`
	os.WriteFile(filepath.Join(cfg.OpenCodeConfigDir, "open-chad.json"), []byte(openchadJSON), 0644)

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

	// Plugins discovered from oc-plugins dir
	if len(state.Plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(state.Plugins))
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

	// Discord
	if state.Discord == nil || !state.Discord.Enabled {
		t.Error("discord should be enabled")
	}
	if state.Discord.ClientID != "123456789" {
		t.Errorf("discord clientId = %q", state.Discord.ClientID)
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

	// Plugins from opencode.json should still be present (just no checkout info)
	if len(state.Plugins) != 2 {
		t.Errorf("expected 2 plugins from opencode.json, got %d", len(state.Plugins))
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
