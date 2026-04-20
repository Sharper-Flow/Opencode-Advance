package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestPlanPlugins_MergesDeclaredPluginEntries(t *testing.T) {
	paths := testPaths(t)
	existing := `{"plugin":["user/plugin"],"theme":"obsidian"}`
	if err := os.WriteFile(paths.OpencodeJSON(), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {Path: "/plugins/advance/plugin"},
			"auth":    {Source: "npm:opencode-auth@1.0.0"},
		},
	}

	plan, err := PlanPlugins(stack, paths, "stack.toml")
	if err != nil {
		t.Fatalf("PlanPlugins: %v", err)
	}
	if len(plan.Targets) != 1 {
		t.Fatalf("targets=%d want 1", len(plan.Targets))
	}
	target := plan.Targets[0]
	if target.Path != paths.OpencodeJSON() {
		t.Fatalf("target path=%q want %q", target.Path, paths.OpencodeJSON())
	}
	root := decodeJSONBytes(t, target.After)
	plugins := stringArrayAt(t, root, "plugin")
	if len(plugins) != 3 {
		t.Fatalf("plugins=%#v", plugins)
	}
	if plugins[0] != "/plugins/advance/plugin" || plugins[1] != "opencode-auth@1.0.0" || plugins[2] != "user/plugin" {
		t.Fatalf("unexpected plugin list: %#v", plugins)
	}
	if root["theme"] != "obsidian" {
		t.Fatalf("top-level key lost: %#v", root)
	}
}

func TestPlanPlugins_SetsSuppressBackupWhenAnyPluginHasSync(t *testing.T) {
	paths := testPaths(t)
	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {Path: "/plugins/advance/plugin", Sync: "/plugins/advance/sync-global.sh --fix"},
		},
	}

	plan, err := PlanPlugins(stack, paths, "stack.toml")
	if err != nil {
		t.Fatalf("PlanPlugins: %v", err)
	}
	if !plan.Targets[0].SuppressBackup {
		t.Fatal("SuppressBackup=false want true when sync plugin exists")
	}
}

func TestPlanInstructions_MergesDeclaredInstructions(t *testing.T) {
	paths := testPaths(t)
	existing := `{"instructions":["user-extra.md"]}`
	if err := os.WriteFile(paths.OpencodeJSON(), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{
		Instructions: cfg.InstructionsSection{Order: []string{"identity.md", "rules.yaml"}},
		Plugins: cfg.PluginsSection{
			"advance": {Instructions: []string{"/plugins/advance/ADV_INSTRUCTIONS.md"}},
		},
	}

	plan, err := PlanInstructions(stack, paths, "stack.toml")
	if err != nil {
		t.Fatalf("PlanInstructions: %v", err)
	}
	root := decodeJSONBytes(t, plan.Targets[0].After)
	instructions := stringArrayAt(t, root, "instructions")
	if len(instructions) != 4 {
		t.Fatalf("instructions=%#v", instructions)
	}
	if instructions[0] != "identity.md" || instructions[1] != "rules.yaml" || instructions[2] != "/plugins/advance/ADV_INSTRUCTIONS.md" || instructions[3] != "user-extra.md" {
		t.Fatalf("unexpected instructions list: %#v", instructions)
	}
	if plan.Targets[0].SuppressBackup {
		t.Fatal("instructions plan should not suppress backups")
	}
}

func testPaths(t *testing.T) cfg.Paths {
	t.Helper()
	root := t.TempDir()
	opencodeDir := filepath.Join(root, "opencode")
	if err := os.MkdirAll(opencodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return cfg.Paths{
		OpencodeConfigDir: opencodeDir,
		VisionConfigDir:   filepath.Join(root, "vision"),
		CacheDir:          filepath.Join(root, "cache"),
	}
}

func decodeJSONBytes(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, string(b))
	}
	return root
}
