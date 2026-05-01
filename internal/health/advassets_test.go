package health

import (
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestWhichProvides(t *testing.T) {
	plugins := cfg.PluginsSection{
		"advance": cfg.Plugin{
			Source:   "https://github.com/example/advance.git",
			Provides: []cfg.ProvidesCategory{cfg.ProvidesCommands, cfg.ProvidesAgents, cfg.ProvidesOverlays},
		},
		"other": cfg.Plugin{
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
