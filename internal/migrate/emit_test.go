package migrate

import (
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestEmitTOML(t *testing.T) {
	state := &OpenChadState{
		MCPServers: map[string]MCPServerState{
			"context7": {
				Port:      6276,
				Command:   "npx",
				Args:      []string{"-y", "@upstash/context7-mcp@latest"},
				Autostart: true,
				Source:    "https://github.com/upstash/context7",
			},
		},
		Plugins: []PluginState{
			{Name: "advance", Source: "https://github.com/Sharper-Flow/Advance.git", Ref: "abc123"},
		},
		Instructions: []string{"~/.config/opencode/instructions/identity.md"},
		Providers: map[string]ProviderState{
			"google": {
				Name: "google",
				Models: []ModelState{
					{Name: "gemini-2.5-flash"},
				},
			},
		},
		OpenCode: &OpenCodeState{Theme: "obsidian"},
	}

	output, err := EmitTOML(state)
	if err != nil {
		t.Fatalf("EmitTOML failed: %v", err)
	}

	// Should contain meta header
	if !strings.Contains(output, "migrated from open-chad") {
		t.Error("missing migration header")
	}

	// Should contain MCP server
	if !strings.Contains(output, "[mcp.servers.context7]") {
		t.Error("missing context7 MCP server section")
	}
	if !strings.Contains(output, "port = 6276") {
		t.Error("missing port")
	}

	// Should contain plugin
	if !strings.Contains(output, "[plugins.advance]") {
		t.Error("missing plugins section")
	}
	if !strings.Contains(output, `source = "https://github.com/Sharper-Flow/Advance.git"`) {
		t.Error("missing advance plugin source")
	}

	// Should contain instructions
	if !strings.Contains(output, "[instructions]") {
		t.Error("missing instructions section")
	}

	// Should contain provider
	if !strings.Contains(output, "[providers.google]") {
		t.Error("missing google provider")
	}

	// Should be valid TOML (basic check)
	if !strings.Contains(output, "[meta]") {
		t.Error("missing meta section")
	}
}

// TestSanitizePluginKey exercises the defense-in-depth emit-side guard for
// TOML-unsafe plugin names. Canonical fix lives in classifyPluginEntry; this
// helper catches anything that leaks through.
func TestSanitizePluginKey(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "plain", input: "morph-fast-apply", want: "morph-fast-apply"},
		{name: "dotted", input: "some.dotted.name", want: "some.dotted.name"},
		{name: "underscore", input: "some_under_score", want: "some_under_score"},
		{name: "alphanumeric", input: "Pkg2024", want: "Pkg2024"},
		{name: "rejects npm version spec", input: "pkg@latest", wantErr: true},
		{name: "rejects space", input: "bad name", wantErr: true},
		{name: "rejects bracket", input: "bad[name]", wantErr: true},
		{name: "rejects slash", input: "bad/name", wantErr: true},
		{name: "rejects empty", input: "", wantErr: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizePluginKey(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("sanitizePluginKey(%q) = %q, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("sanitizePluginKey(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("sanitizePluginKey(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestEmitTOML_SkipsTOMLUnsafePluginNames verifies the emit-side defensive
// guard: if classification leaks a TOML-unsafe name into PluginState, EmitTOML
// must skip the plugin with a warning instead of emitting invalid TOML.
func TestEmitTOML_SkipsTOMLUnsafePluginNames(t *testing.T) {
	state := &OpenChadState{
		Plugins: []PluginState{
			{Name: "good-plugin", Checkout: "/abs/path/good"},
			{Name: "bad@plugin", Checkout: "ignored"}, // simulates classification failure leak
		},
	}
	output, err := EmitTOML(state)
	if err != nil {
		t.Fatalf("EmitTOML failed: %v", err)
	}
	if strings.Contains(output, "[plugins.bad@plugin]") {
		t.Errorf("emitted TOML contains skipped bad plugin:\n%s", output)
	}
	if !strings.Contains(output, "[plugins.good-plugin]") {
		t.Errorf("emitted TOML missing good plugin:\n%s", output)
	}
	// Warning must be appended to state.
	foundWarning := false
	for _, w := range state.Warnings {
		if strings.Contains(w, "bad@plugin") && strings.Contains(w, "TOML-unsafe") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected warning about bad@plugin in state.Warnings; got %v", state.Warnings)
	}
}

func TestEmitTOML_EmptyState(t *testing.T) {
	state := &OpenChadState{}

	output, err := EmitTOML(state)
	if err != nil {
		t.Fatalf("EmitTOML failed: %v", err)
	}

	// Should still produce valid output with just meta
	if !strings.Contains(output, "[meta]") {
		t.Error("missing meta section")
	}
}

// TestEmitTOML_NPMSpecPluginRoundTrips locks the regression for the migration
// npm-spec bug (M5 of finalizeMustQueueTriage3).
//
// BEFORE the fix:
//   readOpenCodeJSON emits PluginState{Name: "opencode-openai-codex-auth@latest",
//   Checkout: "opencode-openai-codex-auth@latest"} for an npm-style plugin entry.
//   EmitTOML then writes [plugins.opencode-openai-codex-auth@latest] which is
//   invalid TOML — `@` is not legal in unquoted keys. cfg.Load fails with:
//   `expected '.' or ']' to end table name, but got '@' instead`.
//
// AFTER the fix:
//   1. classifyPluginEntry (read-side) reshapes the entry to
//      PluginState{Name: "opencode-openai-codex-auth",
//      Source: "npm:opencode-openai-codex-auth@latest", Checkout: ""}.
//   2. sanitizePluginKey (defensive emit-side) catches any leaked unsafe name
//      and skips the plugin with a warning.
//
// This test exercises BOTH the buggy current shape (caught by defensive sanitize)
// AND the post-fix canonical shape (round-trips cleanly).
func TestEmitTOML_NPMSpecPluginRoundTrips(t *testing.T) {
	t.Run("buggy_current_shape_must_not_break_load", func(t *testing.T) {
		// Replicates exactly what readOpenCodeJSON produces today for an npm
		// entry. With sanitizePluginKey defensive guard, EmitTOML must NOT
		// emit invalid TOML; it should either skip the plugin with a warning
		// or sanitize the key. Either way, cfg.Load must succeed.
		state := &OpenChadState{
			Plugins: []PluginState{
				{
					Name:     "opencode-openai-codex-auth@latest",
					Checkout: "opencode-openai-codex-auth@latest",
				},
			},
		}

		output, err := EmitTOML(state)
		if err != nil {
			t.Fatalf("EmitTOML failed: %v", err)
		}
		if strings.Contains(output, "[plugins.opencode-openai-codex-auth@") {
			t.Fatalf("emitted TOML contains @ in table key (invalid TOML):\n%s", output)
		}

		tmpDir := t.TempDir()
		tmpFile := tmpDir + "/stack.toml"
		if err := writeFile(tmpFile, []byte(output)); err != nil {
			t.Fatalf("write temp file: %v", err)
		}
		if _, err := cfg.Load(tmpFile); err != nil {
			t.Fatalf("cfg.Load failed on output from buggy shape: %v\n--- emitted TOML ---\n%s", err, output)
		}
	})

	t.Run("canonical_post_fix_shape_round_trips_npm_source", func(t *testing.T) {
		// Post-fix canonical shape: classifyPluginEntry has reshaped the
		// PluginState. EmitTOML must emit a valid TOML plugin entry that
		// preserves source = "npm:<full-spec>" and round-trips through
		// cfg.Load with IsNPMSource() == true.
		state := &OpenChadState{
			Plugins: []PluginState{
				{
					Name:     "opencode-openai-codex-auth",
					Source:   "npm:opencode-openai-codex-auth@latest",
					Checkout: "",
				},
			},
		}

		output, err := EmitTOML(state)
		if err != nil {
			t.Fatalf("EmitTOML failed: %v", err)
		}

		tmpDir := t.TempDir()
		tmpFile := tmpDir + "/stack.toml"
		if err := writeFile(tmpFile, []byte(output)); err != nil {
			t.Fatalf("write temp file: %v", err)
		}

		stack, err := cfg.Load(tmpFile)
		if err != nil {
			t.Fatalf("cfg.Load failed on canonical shape: %v\n--- emitted TOML ---\n%s", err, output)
		}

		plugin, ok := stack.Plugins["opencode-openai-codex-auth"]
		if !ok {
			t.Fatalf("plugin %q not found in loaded stack; got keys: %v", "opencode-openai-codex-auth", stack.Plugins)
		}
		if plugin.Source != "npm:opencode-openai-codex-auth@latest" {
			t.Errorf("plugin source = %q, want %q", plugin.Source, "npm:opencode-openai-codex-auth@latest")
		}
		if !plugin.IsNPMSource() {
			t.Errorf("IsNPMSource() = false, want true for source %q", plugin.Source)
		}
		if plugin.Checkout != "" {
			t.Errorf("plugin checkout = %q, want empty for npm source", plugin.Checkout)
		}
	})
}
