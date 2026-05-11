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
			// Gap 4: paths ending in "/plugin" derive name from parent dir
			// to avoid collisions on the OpenCode plugin convention
			// <repo>/<name>/plugin. See classifyPluginEntry path branch.
			name:         "absolute path ending in /plugin",
			entry:        "/abs/path/to/plugin",
			wantName:     "to",
			wantSource:   "",
			wantCheckout: "/abs/path/to/plugin",
		},
		{
			// Gap 4: tilde-prefixed path ending /plugin → parent dir name.
			name:         "tilde path ending in /plugin",
			entry:        "~/dev/oc-plugins/advance/plugin",
			wantName:     "advance",
			wantSource:   "",
			wantCheckout: "~/dev/oc-plugins/advance/plugin",
		},
		{
			name:         "absolute path without /plugin suffix",
			entry:        "/abs/path/to/foo",
			wantName:     "foo",
			wantSource:   "",
			wantCheckout: "/abs/path/to/foo",
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

// TestClassifyPluginEntry_PluginSuffixNameDerivation locks Gap 4 of
// close4PreExistingDataCoverage: paths ending in `/plugin` derive their Name
// from the parent dir, not the literal "plugin" basename. Prevents collisions
// when multiple checkouts use the `<repo>/<name>/plugin` convention.
func TestClassifyPluginEntry_PluginSuffixNameDerivation(t *testing.T) {
	cases := []struct {
		name     string
		entry    string
		wantName string
	}{
		{"plugin-suffix path → parent dir name", "/x/foo/plugin", "foo"},
		{"plain path → basename", "/x/foo", "foo"},
		{"path with non-plugin basename", "/x/foo/bar.js", "bar.js"},
		{"deeply nested plugin path", "/home/user/dev/oc-plugins/advance/plugin", "advance"},
		{"tilde plugin path", "~/dev/oc-plugins/morph-fast-apply/plugin", "morph-fast-apply"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := classifyPluginEntry(tc.entry)
			if err != nil {
				t.Fatalf("classifyPluginEntry(%q): %v", tc.entry, err)
			}
			if got.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tc.wantName)
			}
		})
	}
}

// TestClassifyPluginEntry_NoCollisionAcrossPluginPaths is the integration-level
// assertion: two distinct `/x/<name>/plugin` paths produce two distinct
// PluginState entries with non-colliding Names.
func TestClassifyPluginEntry_NoCollisionAcrossPluginPaths(t *testing.T) {
	entries := []string{
		"/dev/oc-plugins/foo/plugin",
		"/dev/oc-plugins/bar/plugin",
	}
	names := make(map[string]bool)
	for _, e := range entries {
		ps, err := classifyPluginEntry(e)
		if err != nil {
			t.Fatalf("classify %q: %v", e, err)
		}
		if names[ps.Name] {
			t.Fatalf("collision: name %q produced twice for entries %v", ps.Name, entries)
		}
		names[ps.Name] = true
	}
	if !names["foo"] || !names["bar"] {
		t.Errorf("expected names {foo, bar}, got %v", names)
	}
}

// TestExpandTilde exercises Gap 2's tilde helper.
func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available")
	}
	cases := []struct {
		in, want string
	}{
		{"~", home},
		{"~/foo", filepath.Join(home, "foo")},
		{"~/foo/bar", filepath.Join(home, "foo", "bar")},
		{"/abs/path", "/abs/path"},
		{"relative/path", "relative/path"},
		{"", ""},
		{"~user/foo", "~user/foo"}, // ~user form unsupported; passthrough
	}
	for _, tc := range cases {
		got := expandTilde(tc.in)
		if got != tc.want {
			t.Errorf("expandTilde(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestFallbackResolvePluginSources_LocalFallback verifies Gap 2 fallback when
// the plugin path is not a git checkout: source should be set to
// "local:<absolute-path>" and Checkout cleared.
func TestFallbackResolvePluginSources_LocalFallback(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "plugins", "oca")
	os.MkdirAll(pluginDir, 0755)
	// Deliberately no .git/ → triggers local: fallback.

	state := &OpenChadState{
		Plugins: []PluginState{
			{Name: "oca", Checkout: pluginDir},
		},
	}
	fallbackResolvePluginSources(state)

	if got := state.Plugins[0].Source; got != "local:"+pluginDir {
		t.Errorf("Source = %q, want %q", got, "local:"+pluginDir)
	}
	if got := state.Plugins[0].Checkout; got != "" {
		t.Errorf("Checkout = %q, want empty for local source", got)
	}
}

// TestFallbackResolvePluginSources_GitDiscovery verifies that when a
// non-canonical plugin path has a .git/config with a remote URL, fallback
// resolves Source to that URL (not local:).
func TestFallbackResolvePluginSources_GitDiscovery(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "out-of-tree", "morph")
	os.MkdirAll(filepath.Join(pluginDir, ".git"), 0755)
	os.WriteFile(filepath.Join(pluginDir, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0644)
	os.WriteFile(filepath.Join(pluginDir, ".git", "config"), []byte(`[remote "origin"]
	url = https://github.com/example/morph.git
`), 0644)

	state := &OpenChadState{
		Plugins: []PluginState{
			{Name: "morph", Checkout: pluginDir},
		},
	}
	fallbackResolvePluginSources(state)

	if got := state.Plugins[0].Source; got != "https://github.com/example/morph.git" {
		t.Errorf("Source = %q, want git remote URL", got)
	}
	// Checkout should be preserved (only cleared for local: sources).
	if got := state.Plugins[0].Checkout; got == "" {
		t.Errorf("Checkout cleared for git source; want preserved")
	}
}

// TestFallbackResolvePluginSources_LeavesEnriched verifies that plugins
// already enriched by discoverPlugins (or classifyPluginEntry for npm) are
// left untouched.
func TestFallbackResolvePluginSources_LeavesEnriched(t *testing.T) {
	state := &OpenChadState{
		Plugins: []PluginState{
			{Name: "advance", Source: "https://github.com/Sharper-Flow/Advance.git", Checkout: "/x/advance"},
			{Name: "npm-pkg", Source: "npm:foo@latest", Checkout: ""},
		},
	}
	fallbackResolvePluginSources(state)
	if state.Plugins[0].Source != "https://github.com/Sharper-Flow/Advance.git" {
		t.Errorf("git plugin Source mutated: %q", state.Plugins[0].Source)
	}
	if state.Plugins[0].Checkout != "/x/advance" {
		t.Errorf("git plugin Checkout cleared: %q", state.Plugins[0].Checkout)
	}
	if state.Plugins[1].Source != "npm:foo@latest" {
		t.Errorf("npm plugin Source mutated: %q", state.Plugins[1].Source)
	}
}

// TestTranslateMCPType exercises Gap 1 of close4PreExistingDataCoverage:
// the URL-suffix-aware translation of legacy `type` values into OCA's schema
// enum, mirroring inferTransport (validate.go:474-491).
func TestTranslateMCPType(t *testing.T) {
	cases := []struct {
		name, in, url, want string
		wantWarn            bool
	}{
		{"remote with /mcp url → http", "remote", "http://localhost:6276/mcp", "http", true},
		{"remote with https /mcp url → http", "remote", "https://example.com/mcp", "http", true},
		{"remote with non-/mcp url → sse", "remote", "https://mcp.grep.app", "sse", true},
		{"remote with non-/mcp localhost → sse", "remote", "http://localhost:9000/api", "sse", true},
		{"remote with empty url → empty + warning", "remote", "", "", true},
		{"stdio passthrough", "stdio", "", "stdio", false},
		{"http passthrough", "http", "http://x/mcp", "http", false},
		{"sse passthrough", "sse", "http://x", "sse", false},
		{"daemon passthrough", "daemon", "", "daemon", false},
		{"empty type passes through", "", "http://x", "", false},
		{"unknown type passthrough with warning", "websocket", "ws://x", "websocket", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, warn := translateMCPType(tc.in, tc.url)
			if got != tc.want {
				t.Errorf("translateMCPType(%q, %q) = %q, want %q", tc.in, tc.url, got, tc.want)
			}
			if tc.wantWarn && warn == "" {
				t.Errorf("expected warning for (%q, %q), got empty", tc.in, tc.url)
			}
			if !tc.wantWarn && warn != "" {
				t.Errorf("unexpected warning %q for (%q, %q)", warn, tc.in, tc.url)
			}
		})
	}
}

// TestPortFromURL exercises URL → port extraction for the MCP type fix-up.
func TestPortFromURL(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"http://localhost:6276/mcp", 6276},
		{"https://localhost:8443/mcp", 8443},
		{"http://localhost/mcp", 0},  // no explicit port
		{"https://mcp.grep.app", 0},  // no explicit port (bare HTTPS)
		{"https://mcp.grep.app:443", 443},
		{"", 0},
		{"::not-a-url::", 0}, // unparseable
	}
	for _, tc := range cases {
		got := portFromURL(tc.in)
		if got != tc.want {
			t.Errorf("portFromURL(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestReadOpenChadState_MCPTypeTranslation exercises the integration of
// translateMCPType + portFromURL against an operator-shape opencode.json with
// 6 mixed-suffix `type=remote` servers.
func TestReadOpenChadState_MCPTypeTranslation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ReaderConfig{
		OpenChadRepo:      filepath.Join(tmpDir, "open-chad"),
		OpenCodeConfigDir: filepath.Join(tmpDir, "opencode"),
		VisionConfigDir:   filepath.Join(tmpDir, "vision"),
		OcPluginsDir:      filepath.Join(tmpDir, "oc-plugins"),
	}
	os.MkdirAll(cfg.OpenCodeConfigDir, 0755)
	// Operator-shape opencode.json: 5 /mcp servers + 1 portless HTTPS.
	json := `{
		"mcp": {
			"vision":     {"type": "remote", "url": "http://localhost:6275/mcp"},
			"context7":   {"type": "remote", "url": "http://localhost:6276/mcp"},
			"kagi":       {"type": "remote", "url": "http://localhost:6279/mcp"},
			"firecrawl":  {"type": "remote", "url": "http://localhost:6281/mcp"},
			"lgrep":      {"type": "remote", "url": "http://localhost:6285/mcp"},
			"gh_grep":    {"type": "remote", "url": "https://mcp.grep.app"}
		},
		"plugin": []
	}`
	os.WriteFile(filepath.Join(cfg.OpenCodeConfigDir, "opencode.json"), []byte(json), 0644)

	state, err := ReadOpenChadState(cfg)
	if err != nil {
		t.Fatalf("ReadOpenChadState: %v", err)
	}

	expected := map[string]struct {
		typ  string
		port int
	}{
		"vision":    {"http", 6275},
		"context7":  {"http", 6276},
		"kagi":      {"http", 6279},
		"firecrawl": {"http", 6281},
		"lgrep":     {"http", 6285},
		"gh_grep":   {"sse", 0}, // portless HTTPS is OK for sse transport
	}
	for name, want := range expected {
		got, ok := state.MCPServers[name]
		if !ok {
			t.Errorf("server %q missing", name)
			continue
		}
		if got.Type != want.typ {
			t.Errorf("%s.Type = %q, want %q", name, got.Type, want.typ)
		}
		if got.Port != want.port {
			t.Errorf("%s.Port = %d, want %d", name, got.Port, want.port)
		}
	}
}

// TestReadVisionServers_SlotGroupSchemaAlignment locks Gap 3 of
// close4PreExistingDataCoverage: the Vision YAML shape uses base_port + count,
// not min_slots + max_slots. Before the fix the migrator's yaml struct tags
// read fields that don't exist in the real Vision YAML, leaving BasePort/Count
// at zero — which then fail validate.go:362,372 (base_port/count required).
func TestReadVisionServers_SlotGroupSchemaAlignment(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ReaderConfig{
		OpenChadRepo:      filepath.Join(tmpDir, "open-chad"),
		OpenCodeConfigDir: filepath.Join(tmpDir, "opencode"),
		VisionConfigDir:   filepath.Join(tmpDir, "vision"),
		OcPluginsDir:      filepath.Join(tmpDir, "oc-plugins"),
	}
	os.MkdirAll(cfg.OpenCodeConfigDir, 0755)
	os.WriteFile(filepath.Join(cfg.OpenCodeConfigDir, "opencode.json"), []byte(`{"mcp":{},"plugin":[],"instructions":[]}`), 0644)
	os.MkdirAll(cfg.VisionConfigDir, 0755)

	// Real Vision YAML shape verified against ~/.config/vision/servers.yaml.
	yaml := `servers: {}
slot_groups:
  playwright-headless:
    template: playwright-headless-slot
    base_port: 6301
    count: 4
    group_port: 6300
  playwright-headed:
    template: playwright-headed-slot
    base_port: 6306
    count: 2
    group_port: 6305
`
	os.WriteFile(filepath.Join(cfg.VisionConfigDir, "servers.yaml"), []byte(yaml), 0644)

	state, err := ReadOpenChadState(cfg)
	if err != nil {
		t.Fatalf("ReadOpenChadState: %v", err)
	}

	if len(state.SlotGroups) != 2 {
		t.Fatalf("expected 2 slot groups, got %d (%+v)", len(state.SlotGroups), state.SlotGroups)
	}

	cases := map[string]struct {
		basePort, count, groupPort int
		template                   string
	}{
		"playwright-headless": {6301, 4, 6300, "playwright-headless-slot"},
		"playwright-headed":   {6306, 2, 6305, "playwright-headed-slot"},
	}
	for name, want := range cases {
		got, ok := state.SlotGroups[name]
		if !ok {
			t.Errorf("slot group %q missing", name)
			continue
		}
		if got.BasePort != want.basePort {
			t.Errorf("%s BasePort = %d, want %d", name, got.BasePort, want.basePort)
		}
		if got.Count != want.count {
			t.Errorf("%s Count = %d, want %d", name, got.Count, want.count)
		}
		if got.GroupPort != want.groupPort {
			t.Errorf("%s GroupPort = %d, want %d", name, got.GroupPort, want.groupPort)
		}
		if got.Template != want.template {
			t.Errorf("%s Template = %q, want %q", name, got.Template, want.template)
		}
	}
}
