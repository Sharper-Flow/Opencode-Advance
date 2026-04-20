package config

import (
	"os"
	"testing"
)

func TestExpandPluginTokens_BasicSubstitution(t *testing.T) {
	p := Plugin{
		Checkout: "/home/user/dev/plugins/advance",
		Subdir:   "plugin",
		Path:     "{checkout}/{subdir}",
		Sync:     "{checkout}/scripts/sync-global.sh --fix",
	}
	err := expandPluginTokens(&p)
	if err != nil {
		t.Fatalf("expandPluginTokens failed: %v", err)
	}
	if p.Path != "/home/user/dev/plugins/advance/plugin" {
		t.Errorf("Path = %q, want /home/user/dev/plugins/advance/plugin", p.Path)
	}
	if p.Sync != "/home/user/dev/plugins/advance/scripts/sync-global.sh --fix" {
		t.Errorf("Sync = %q, want /home/user/dev/plugins/advance/scripts/sync-global.sh --fix", p.Sync)
	}
}

func TestExpandPluginTokens_EmptyCheckoutSkipsSubstitution(t *testing.T) {
	// When Checkout is empty, {checkout} token should be left unchanged
	// (the empty string would create a leading slash, which is not useful).
	p := Plugin{
		Checkout: "",
		Subdir:   "plugin",
		Path:     "{checkout}/{subdir}",
	}
	err := expandPluginTokens(&p)
	if err != nil {
		t.Fatalf("expandPluginTokens failed on empty checkout: %v", err)
	}
	// expandAll on "" returns ""; we only substitute {checkout} when
	// the resolved checkout value is non-empty.
	if p.Path != "{checkout}/plugin" {
		t.Errorf("Path with empty checkout = %q, want {checkout}/plugin", p.Path)
	}
}

func TestExpandPluginTokens_UnknownTokenErrors(t *testing.T) {
	p := Plugin{
		Checkout: "/home/user/dev",
		Path:     "{checkout}/{unknown}",
	}
	err := expandPluginTokens(&p)
	if err == nil {
		t.Fatal("expandPluginTokens should error on unknown token {unknown}")
	}
}

func TestExpandPluginTokens_InstructionsExpanded(t *testing.T) {
	p := Plugin{
		Checkout:     "/home/user/dev/advance",
		Instructions: []string{"{checkout}/ADV_INSTRUCTIONS.md", "identity.md"},
	}
	err := expandPluginTokens(&p)
	if err != nil {
		t.Fatalf("expandPluginTokens failed: %v", err)
	}
	if p.Instructions[0] != "/home/user/dev/advance/ADV_INSTRUCTIONS.md" {
		t.Errorf("Instructions[0] = %q, want expanded path", p.Instructions[0])
	}
	if p.Instructions[1] != "identity.md" {
		t.Errorf("Instructions[1] = %q, want unchanged bare filename", p.Instructions[1])
	}
}

func TestResolvePlugins_CallsExpandPluginTokens(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[plugins.advance]
source   = "https://github.com/Sharper-Flow/Advance.git"
checkout = "~/dev/oc-plugins/advance"
subdir   = "plugin"
path     = "{checkout}/{subdir}"
sync     = "{checkout}/scripts/sync-global.sh"
`
	stack, err := Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	stack.Resolve()

	adv := stack.Plugins["advance"]
	// Resolve() expands ~/... first via expandAll (→ /home/user/dev/...),
	// then expandPluginTokens substitutes {checkout}/{subdir}.
	// The final Path should be the fully resolved absolute path.
	wantPrefix := "/dev/oc-plugins/advance/plugin"
	if len(adv.Path) < len(wantPrefix) || adv.Path[len(adv.Path)-len(wantPrefix):] != wantPrefix {
		t.Errorf("Resolve() did not call expandPluginTokens: Path = %q (expected to end with %q)", adv.Path, wantPrefix)
	}
}

func TestPaths_PluginCheckoutRoot(t *testing.T) {
	// Default: falls back to ~/dev/oc-plugins
	t.Setenv("OCA_PLUGIN_CHECKOUT_ROOT", "")

	// Also clear XDG override if set
	orig := os.Getenv("OCA_PLUGIN_CHECKOUT_ROOT")
	os.Unsetenv("OCA_PLUGIN_CHECKOUT_ROOT")
	defer func() {
		if orig != "" {
			os.Setenv("OCA_PLUGIN_CHECKOUT_ROOT", orig)
		}
	}()

	paths := ResolvePaths()
	root := paths.PluginCheckoutRoot()
	if root == "" {
		t.Error("PluginCheckoutRoot returned empty with no override")
	}

	// With override set.
	tmp := t.TempDir()
	os.Setenv("OCA_PLUGIN_CHECKOUT_ROOT", tmp)
	defer os.Setenv("OCA_PLUGIN_CHECKOUT_ROOT", "")

	paths = ResolvePaths()
	root = paths.PluginCheckoutRoot()
	if root != tmp {
		t.Errorf("PluginCheckoutRoot = %q, want %q", root, tmp)
	}
}
