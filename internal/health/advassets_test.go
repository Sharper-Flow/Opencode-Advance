package health

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestWhichProvides(t *testing.T) {
	plugins := cfg.PluginsSection{
		"advance": {
			Source:   "https://github.com/example/advance.git",
			Provides: []cfg.ProvidesCategory{cfg.ProvidesCommands, cfg.ProvidesAgents, cfg.ProvidesOverlays},
		},
		"other": {
			Source:   "https://github.com/example/other.git",
			Provides: []cfg.ProvidesCategory{cfg.ProvidesCommands},
		},
	}

	tests := []struct {
		category cfg.ProvidesCategory
		want     []string
	}{
		{cfg.ProvidesCommands, []string{"advance", "other"}},
		{cfg.ProvidesAgents, []string{"advance"}},
		{cfg.ProvidesOverlays, []string{"advance"}},
		{cfg.ProvidesInstructions, nil},
		{cfg.ProvidesSkills, nil},
		{cfg.ProvidesTemporal, nil},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			got := whichProvides(plugins, tt.category)
			if len(got) != len(tt.want) {
				t.Fatalf("whichProvides(%+v) = %v, want %v", tt.category, got, tt.want)
			}
			for i, name := range got {
				if name != tt.want[i] {
					t.Errorf("whichProvides[%d] = %q, want %q", i, name, tt.want[i])
				}
			}
		})
	}
}

func TestWhichProvidesEmpty(t *testing.T) {
	got := whichProvides(nil, cfg.ProvidesCommands)
	if got != nil {
		t.Fatalf("whichProvides(nil) = %v, want nil", got)
	}
}

func TestCategoryTargetDir(t *testing.T) {
	configDir := "/home/user/.config/opencode"

	tests := []struct {
		category cfg.ProvidesCategory
		wantDir  string
		wantOK   bool
	}{
		{cfg.ProvidesCommands, configDir + "/commands", true},
		{cfg.ProvidesAgents, configDir + "/agents", true},
		{cfg.ProvidesSkills, configDir + "/skills", true},
		{cfg.ProvidesOverlays, configDir + "/overlays", true},
		{cfg.ProvidesInstructions, configDir + "/instructions", true},
		{cfg.ProvidesTemporal, "", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			gotDir, gotOK := categoryTargetDir(tt.category, configDir)
			if gotDir != tt.wantDir || gotOK != tt.wantOK {
				t.Errorf("categoryTargetDir(%v, %q) = (%q, %v), want (%q, %v)",
					tt.category, configDir, gotDir, gotOK, tt.wantDir, tt.wantOK)
			}
		})
	}
}

func TestCheckAdvAssets_DuplicateOwner(t *testing.T) {
	tmp := t.TempDir()
	commandsDir := filepath.Join(tmp, "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"plugin-a": {Source: "https://example.com/a.git", Provides: []cfg.ProvidesCategory{cfg.ProvidesCommands}},
			"plugin-b": {Source: "https://example.com/b.git", Provides: []cfg.ProvidesCategory{cfg.ProvidesCommands}},
		},
	}

	opts := Options{ConfigDir: tmp}
	checks, err := CheckAdvAssets(context.Background(), stack, opts)
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, c := range checks {
		if c.Status == StatusFail && c.Name == "adv-assets.adv-commands.duplicate-owner" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate-owner fail check, got: %+v", checks)
	}
}

func TestCheckAdvAssets_Orphaned(t *testing.T) {
	tmp := t.TempDir()
	commandsDir := filepath.Join(tmp, "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(commandsDir, "orphan.md"), []byte("# orphan"), 0644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {Source: "https://example.com/adv.git", Provides: []cfg.ProvidesCategory{cfg.ProvidesCommands}},
		},
	}

	opts := Options{ConfigDir: tmp}
	checks, err := CheckAdvAssets(context.Background(), stack, opts)
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, c := range checks {
		if c.Status == StatusWarn && c.Name == "adv-assets.adv-commands.orphaned" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphaned warn check, got: %+v", checks)
	}
}

func TestCheckAdvAssets_Stale(t *testing.T) {
	tmp := t.TempDir()
	instrDir := filepath.Join(tmp, "instructions")
	if err := os.MkdirAll(instrDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instrDir, "adv-phantom.md"), []byte("# phantom"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instrDir, "adv-real.md"), []byte("# real"), 0644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {Source: "https://example.com/adv.git", Provides: []cfg.ProvidesCategory{cfg.ProvidesInstructions}},
		},
		Providers: cfg.ProvidersSection{
			"real": {},
		},
	}

	opts := Options{ConfigDir: tmp}
	checks, err := CheckAdvAssets(context.Background(), stack, opts)
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, c := range checks {
		if c.Status == StatusWarn && c.Name == "adv-assets.stale.phantom" {
			found = true
		}
	}
	if !found {
		var names []string
		for _, c := range checks {
			names = append(names, c.Name)
		}
		sort.Strings(names)
		t.Errorf("expected stale warn check for phantom, got checks: %v", names)
	}
}

func TestCheckAdvAssets_Clean(t *testing.T) {
	tmp := t.TempDir()
	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {Source: "https://example.com/adv.git", Provides: []cfg.ProvidesCategory{cfg.ProvidesCommands}},
		},
	}

	opts := Options{ConfigDir: tmp}
	checks, err := CheckAdvAssets(context.Background(), stack, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range checks {
		if c.Status == StatusFail || c.Status == StatusWarn {
			t.Errorf("unexpected non-pass check: %+v", c)
		}
	}
}

func TestCheckAdvAssets_ConfigDirMissing(t *testing.T) {
	stack := &cfg.Stack{
		Plugins: cfg.PluginsSection{
			"advance": {Source: "https://example.com/adv.git", Provides: []cfg.ProvidesCategory{cfg.ProvidesCommands}},
		},
	}

	opts := Options{ConfigDir: "/nonexistent/path"}
	checks, err := CheckAdvAssets(context.Background(), stack, opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	for _, c := range checks {
		if c.Status == StatusFail {
			t.Errorf("missing config dir should not fail, got: %+v", c)
		}
	}
}
