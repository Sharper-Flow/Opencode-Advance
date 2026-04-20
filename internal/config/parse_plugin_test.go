package config

import (
	"testing"
)

const pluginTOML = `
[meta]
version = "1.0.0"
name = "test"

[mcp.servers.vision]
port = 6275
command = "vision"
type = "daemon"

[plugins.advance]
source      = "https://github.com/Sharper-Flow/Advance.git"
ref         = "trunk"
checkout    = "~/dev/oc-plugins/advance"
subdir      = "plugin"
build       = ["pnpm install --frozen-lockfile", "pnpm build"]
path        = "{checkout}/{subdir}"
sync        = "{checkout}/scripts/sync-global.sh --fix"
provides    = ["adv-commands", "adv-agents", "adv-skills", "adv-overlays"]
instructions = ["{checkout}/ADV_INSTRUCTIONS.md"]

[plugins.npm-fmt]
source = "npm:@franlol/opencode-md-table-formatter@latest"

[temporal]
# reserved for Phase 6.5
`

func TestParse_PluginsSectionTypedFields(t *testing.T) {
	stack, err := Parse([]byte(pluginTOML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify typed [plugins.advance] decode
	plugins, ok := stack.Plugins["advance"]
	if !ok {
		t.Fatal("plugins[advance] not parsed as typed field")
	}
	if plugins.Source != "https://github.com/Sharper-Flow/Advance.git" {
		t.Errorf("Source = %q, want git URL", plugins.Source)
	}
	if plugins.Ref != "trunk" {
		t.Errorf("Ref = %q, want trunk", plugins.Ref)
	}
	if plugins.Checkout != "~/dev/oc-plugins/advance" {
		t.Errorf("Checkout = %q, want ~/dev/oc-plugins/advance", plugins.Checkout)
	}
	if plugins.Subdir != "plugin" {
		t.Errorf("Subdir = %q, want plugin", plugins.Subdir)
	}
	if len(plugins.Build) != 2 {
		t.Errorf("Build len = %d, want 2", len(plugins.Build))
	}
	if plugins.Path != "{checkout}/{subdir}" {
		t.Errorf("Path = %q, want {checkout}/{subdir}", plugins.Path)
	}
	if plugins.Sync != "{checkout}/scripts/sync-global.sh --fix" {
		t.Errorf("Sync = %q, want {checkout}/scripts/sync-global.sh --fix", plugins.Sync)
	}
	if len(plugins.Provides) != 4 {
		t.Errorf("Provides len = %d, want 4", len(plugins.Provides))
	}
	if plugins.Instructions[0] != "{checkout}/ADV_INSTRUCTIONS.md" {
		t.Errorf("Instructions[0] = %q, want {checkout}/ADV_INSTRUCTIONS.md", plugins.Instructions[0])
	}
}

func TestParse_NPMSourcePlugin(t *testing.T) {
	stack, err := Parse([]byte(pluginTOML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	plugins, ok := stack.Plugins["npm-fmt"]
	if !ok {
		t.Fatal("plugins[npm-fmt] not parsed as typed field")
	}
	if plugins.Source != "npm:@franlol/opencode-md-table-formatter@latest" {
		t.Errorf("Source = %q, want npm source string", plugins.Source)
	}
	if plugins.Checkout != "" {
		t.Errorf("Checkout = %q, want empty for npm source", plugins.Checkout)
	}
	if plugins.Ref != "" {
		t.Errorf("Ref = %q, want empty for npm source", plugins.Ref)
	}
}

func TestParse_InstructionsSectionTyped(t *testing.T) {
	instrTOML := `
[meta]
version = "1.0.0"

[instructions]
order = ["identity.md", "rules.yaml", "{checkout}/ADV_INSTRUCTIONS.md"]
`
	stack, err := Parse([]byte(instrTOML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(stack.Instructions.Order) != 3 {
		t.Errorf("Order len = %d, want 3", len(stack.Instructions.Order))
	}
	if stack.Instructions.Order[0] != "identity.md" {
		t.Errorf("Order[0] = %q, want identity.md", stack.Instructions.Order[0])
	}
}

func TestParse_TemporalSectionAdvisoryTolerance(t *testing.T) {
	// [temporal] is reserved for Phase 6.5; parser must tolerate it without error.
	temporalTOML := `
[meta]
version = "1.0.0"

[temporal]
# reserved for Phase 6.5
enabled = false
`
	stack, err := Parse([]byte(temporalTOML))
	if err != nil {
		t.Fatalf("Parse failed on [temporal]: %v", err)
	}
	if stack.Temporal == nil {
		t.Fatal("Temporal not parsed (advisory tolerance failed)")
	}
}

func TestParse_PluginsNotDeferred(t *testing.T) {
	stack, err := Parse([]byte(pluginTOML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	// plugins and instructions must NOT appear in DeferredSections
	if _, ok := stack.DeferredSections["plugins"]; ok {
		t.Error("plugins should be typed, not deferred")
	}
	if _, ok := stack.DeferredSections["instructions"]; ok {
		t.Error("instructions should be typed, not deferred")
	}
	if _, ok := stack.DeferredSections["temporal"]; ok {
		t.Error("temporal should be typed (advisory), not deferred")
	}
}

func TestProvidesCategory_ValidValues(t *testing.T) {
	valid := []ProvidesCategory{
		ProvidesCommands,
		ProvidesAgents,
		ProvidesSkills,
		ProvidesOverlays,
		ProvidesInstructions,
		ProvidesTemporal,
	}
	for _, p := range valid {
		if !p.IsValid() {
			t.Errorf("ProvidesCategory %q failed IsValid()", p)
		}
	}
}

func TestProvidesCategory_InvalidValue(t *testing.T) {
	p := ProvidesCategory("invalid-category")
	if p.IsValid() {
		t.Error("invalid category should return false on IsValid()")
	}
}

func TestPlugin_IsGitSource(t *testing.T) {
	gitPlugin := Plugin{Source: "https://github.com/foo/bar.git"}
	if !gitPlugin.IsGitSource() {
		t.Error("git URL should be IsGitSource() = true")
	}

	npmPlugin := Plugin{Source: "npm:some-pkg@latest"}
	if npmPlugin.IsGitSource() {
		t.Error("npm source should be IsGitSource() = false")
	}
}

func TestPlugin_IsNPMSource(t *testing.T) {
	npmPlugin := Plugin{Source: "npm:some-pkg@latest"}
	if !npmPlugin.IsNPMSource() {
		t.Error("npm: prefix should be IsNPMSource() = true")
	}

	gitPlugin := Plugin{Source: "https://github.com/foo/bar.git"}
	if gitPlugin.IsNPMSource() {
		t.Error("git URL should be IsNPMSource() = false")
	}
}

func TestPlugin_IsEnabled(t *testing.T) {
	// nil Enabled = true (default)
	p1 := Plugin{}
	if !p1.IsEnabled() {
		t.Error("nil Enabled should default to true")
	}

	falseVal := false
	p2 := Plugin{Enabled: &falseVal}
	if p2.IsEnabled() {
		t.Error("Enabled=false should return false")
	}
}

func TestKnownSections_IncludesTemporal(t *testing.T) {
	if !knownSections["temporal"] {
		t.Error("knownSections should include 'temporal'")
	}
}
