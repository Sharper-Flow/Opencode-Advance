package health

import (
	"path/filepath"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// whichProvides returns the names of plugins that declare the given category
// in their provides list. Returns nil if no plugin claims the category.
func whichProvides(plugins cfg.PluginsSection, category cfg.ProvidesCategory) []string {
	var names []string
	for name, plugin := range plugins {
		for _, provided := range plugin.Provides {
			if provided == category {
				names = append(names, name)
				break
			}
		}
	}
	return names
}

// categoryTargetDir maps a ProvidesCategory to its deployed directory path
// under the given configDir. Returns ("", false) for categories that don't
// map to a file directory (e.g. adv-temporal).
func categoryTargetDir(category cfg.ProvidesCategory, configDir string) (string, bool) {
	switch category {
	case cfg.ProvidesCommands:
		return filepath.Join(configDir, "commands"), true
	case cfg.ProvidesAgents:
		return filepath.Join(configDir, "agents"), true
	case cfg.ProvidesSkills:
		return filepath.Join(configDir, "skills"), true
	case cfg.ProvidesOverlays:
		return filepath.Join(configDir, "overlays"), true
	case cfg.ProvidesInstructions:
		return filepath.Join(configDir, "instructions"), true
	default:
		// ProvidesTemporal and any future categories that don't map to dirs
		return "", false
	}
}
