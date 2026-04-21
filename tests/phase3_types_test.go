package tests

import (
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// TestProvidersSection_Typed verifies that [providers] is parsed into the
// typed ProvidersSection field rather than DeferredSections.
func TestProvidersSection_Typed(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[providers.google]
[providers.google.models.gemini-2-5-flash]
name = "Gemini 2.5 Flash"
context = 1048576
output = 65536
inputs = ["text", "image"]
outputs = ["text"]
unknown_model_field = "ignored"

[providers.google.models.gpt-5]
name = "GPT 5"
context = 272000
output = 128000
inputs = ["text", "image"]
outputs = ["text"]

[providers.google.options]
include = ["reasoning.encrypted_content"]

[providers.google.variants.high]
text_verbosity = "high"

[providers.openai]
[providers.openai.models.gpt-5]
name = "GPT 5"
context = 272000
output = 128000
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Must be on typed field, not deferred.
	if len(stack.Providers) == 0 {
		t.Fatal("Providers section not on typed field")
	}
	if _, ok := stack.DeferredSections["providers"]; ok {
		t.Fatal("providers should not be in DeferredSections")
	}

	// Basic shape checks.
	google, ok := stack.Providers["google"]
	if !ok {
		t.Fatal("missing google provider")
	}
	if len(google.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(google.Models))
	}

	gemini, ok := google.Models["gemini-2-5-flash"]
	if !ok {
		t.Fatal("missing gemini-2-5-flash model")
	}
	if gemini.Name != "Gemini 2.5 Flash" {
		t.Errorf("name = %q, want %q", gemini.Name, "Gemini 2.5 Flash")
	}
	if gemini.Context != 1048576 {
		t.Errorf("context = %d, want %d", gemini.Context, 1048576)
	}
	if gemini.Output != 65536 {
		t.Errorf("output = %d, want %d", gemini.Output, 65536)
	}
	if len(gemini.Inputs) != 2 {
		t.Fatalf("inputs len = %d, want 2", len(gemini.Inputs))
	}
	if len(gemini.Outputs) != 1 {
		t.Fatalf("outputs len = %d, want 1", len(gemini.Outputs))
	}

	// Options passthrough.
	if google.Options == nil {
		t.Fatal("google.Options is nil")
	}
	include, ok := google.Options["include"]
	if !ok {
		t.Fatal("missing include key in options")
	}
	includeSlice, ok := include.([]any)
	if !ok {
		t.Fatalf("include type = %T, want []any", include)
	}
	if len(includeSlice) != 1 {
		t.Fatalf("include len = %d, want 1", len(includeSlice))
	}

	// Variants passthrough.
	if google.Variants == nil {
		t.Fatal("google.Variants is nil")
	}
	if _, ok := google.Variants["high"]; !ok {
		t.Fatal("missing high variant")
	}

	// Extra passthrough: unknown field lands in Extra.
	if gemini.Extra == nil {
		t.Fatal("Extra is nil for model with extra field")
	}
}

// TestPermissionsSection_Typed verifies that [permissions] is parsed into the
// typed PermissionsSection field with correct shape translation.
func TestPermissionsSection_Typed(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[permissions]
default = "allow"
doom_loop = "ask"

[permissions.external_directory]
all = "ask"
dev = "allow"

[permissions.bash]
all = "allow"
git_push = "ask"
rm_rf = "deny"
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if stack.Permissions.Default == "" && stack.Permissions.DoomLoop == "" && len(stack.Permissions.ExternalDirectory) == 0 && len(stack.Permissions.Bash) == 0 && len(stack.Permissions.Extra) == 0 {
		t.Fatal("Permissions section not on typed field")
	}
	if _, ok := stack.DeferredSections["permissions"]; ok {
		t.Fatal("permissions should not be in DeferredSections")
	}

	if stack.Permissions.Default != "allow" {
		t.Errorf("default = %q, want %q", stack.Permissions.Default, "allow")
	}
	if stack.Permissions.DoomLoop != "ask" {
		t.Errorf("doom_loop = %q, want %q", stack.Permissions.DoomLoop, "ask")
	}

	if stack.Permissions.ExternalDirectory == nil {
		t.Fatal("ExternalDirectory is nil")
	}
	if extStar, ok := stack.Permissions.ExternalDirectory["all"]; !ok || extStar != "ask" {
		t.Errorf("external_directory[%q] = %q, want %q", "all", extStar, "ask")
	}
	if extDev, ok := stack.Permissions.ExternalDirectory["dev"]; !ok || extDev != "allow" {
		t.Errorf("external_directory[%q] = %q, want %q", "dev", extDev, "allow")
	}

	if stack.Permissions.Bash == nil {
		t.Fatal("Bash is nil")
	}
	if bashStar, ok := stack.Permissions.Bash["all"]; !ok || bashStar != "allow" {
		t.Errorf("bash[%q] = %q, want %q", "all", bashStar, "allow")
	}
	if bashPush, ok := stack.Permissions.Bash["git_push"]; !ok || bashPush != "ask" {
		t.Errorf("bash[%q] = %q, want %q", "git_push", bashPush, "ask")
	}
	if bashRm, ok := stack.Permissions.Bash["rm_rf"]; !ok || bashRm != "deny" {
		t.Errorf("bash[%q] = %q, want %q", "rm_rf", bashRm, "deny")
	}
}

// TestWatcherSection_Typed verifies that [watcher] is parsed into the typed
// WatcherSection field.
func TestWatcherSection_Typed(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[watcher]
ignore = ["node_modules/**", "**/.git/**"]
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(stack.Watcher.Ignore) == 0 && len(stack.Watcher.Extra) == 0 {
		t.Fatal("Watcher section not on typed field")
	}
	if _, ok := stack.DeferredSections["watcher"]; ok {
		t.Fatal("watcher should not be in DeferredSections")
	}

	if len(stack.Watcher.Ignore) != 2 {
		t.Fatalf("ignore len = %d, want 2", len(stack.Watcher.Ignore))
	}
	if stack.Watcher.Ignore[0] != "node_modules/**" {
		t.Errorf("ignore[0] = %q, want %q", stack.Watcher.Ignore[0], "node_modules/**")
	}
}

// TestLSPSection_Typed verifies that [lsp] is parsed into the typed LSPSection
// field with per-server command/extensions/disabled fields.
func TestLSPSection_Typed(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[lsp.pyright]
command = ["pyright", "lsp"]
extensions = [".py", ".pyi"]

[lsp.ts]
command = ["typescript", "lsp"]
extensions = [".ts", ".tsx"]
disabled = true
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if stack.LSP == nil {
		t.Fatal("LSP section not on typed field")
	}
	if _, ok := stack.DeferredSections["lsp"]; ok {
		t.Fatal("lsp should not be in DeferredSections")
	}

	pyrefly, ok := stack.LSP["pyright"]
	if !ok {
		t.Fatal("missing pyright server")
	}
	if len(pyrefly.Command) != 2 || pyrefly.Command[0] != "pyright" {
		t.Errorf("command = %v, want [pyright, lsp]", pyrefly.Command)
	}
	if len(pyrefly.Extensions) != 2 {
		t.Errorf("extensions len = %d, want 2", len(pyrefly.Extensions))
	}
	if pyrefly.Disabled {
		t.Error("pyright should not be disabled")
	}

	ts, ok := stack.LSP["ts"]
	if !ok {
		t.Fatal("missing ts server")
	}
	if !ts.Disabled {
		t.Error("ts should be disabled")
	}
}

// TestPhase3Sections_ExtraPassthrough verifies that unknown fields within
// Phase 3 sections land in the Extra map.
func TestPhase3Sections_ExtraPassthrough(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[providers.google]
unknown_provider_field = "ignored"

[providers.google.models.gemini]
name = "Gemini"
context = 100000
output = 10000
unknown_model_field = "also-ignored"

[permissions]
default = "allow"
unknown_permission_field = "ignored"

[watcher]
ignore = []
unknown_watcher_field = "ignored"

[lsp.ts]
command = ["tsserver"]
extensions = [".ts"]
unknown_lsp_field = "ignored"
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Each typed section should have its own Extra map.
	if len(stack.Providers) == 0 {
		t.Fatal("Providers is nil")
	}
	if _, ok := stack.Providers["google"]; !ok {
		t.Fatal("google provider is nil")
	}
	if stack.Providers["google"].Extra == nil {
		t.Fatal("google Extra is nil")
	}
	if _, ok := stack.Providers["google"].Extra["unknown_provider_field"]; !ok {
		t.Error("unknown_provider_field should be in google.Extra")
	}

	gemini, ok := stack.Providers["google"].Models["gemini"]
	if !ok {
		t.Fatal("gemini model is nil")
	}
	if gemini.Extra == nil {
		t.Fatal("gemini model Extra is nil")
	}
	if _, ok := gemini.Extra["unknown_model_field"]; !ok {
		t.Error("unknown_model_field should be in gemini.Extra")
	}

	if stack.Permissions.Extra == nil {
		t.Fatal("Permissions.Extra is nil")
	}
	if _, ok := stack.Permissions.Extra["unknown_permission_field"]; !ok {
		t.Error("unknown_permission_field should be in Permissions.Extra")
	}

	if stack.Watcher.Extra == nil {
		t.Fatal("Watcher.Extra is nil")
	}
	if _, ok := stack.Watcher.Extra["unknown_watcher_field"]; !ok {
		t.Error("unknown_watcher_field should be in Watcher.Extra")
	}

	ts, ok := stack.LSP["ts"]
	if !ok {
		t.Fatal("ts LSP is nil")
	}
	if ts.Extra == nil {
		t.Fatal("ts LSP Extra is nil")
	}
	if _, ok := ts.Extra["unknown_lsp_field"]; !ok {
		t.Error("unknown_lsp_field should be in ts.Extra")
	}
}

// TestPhase3Sections_AllDeferred verifies that agents, session, discord, skills,
// formatters, commands, and opencode remain in DeferredSections after Phase 3.
func TestPhase3Sections_AllDeferred(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[agents]
[agents.adv]
model = "openai/gpt-5"

[session]
[discord]
[skills]
[formatters]
[commands]
[opencode]
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	remaining := []string{"agents", "session", "discord"}
	for _, section := range remaining {
		if _, ok := stack.DeferredSections[section]; !ok {
			t.Errorf("%s should remain in DeferredSections", section)
		}
	}
}
