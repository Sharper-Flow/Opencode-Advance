package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func init() {
	registerBuiltin("adv-assets", CheckAdvAssets)
}

// CheckAdvAssets audits deployed config files against declared plugin
// ownership. It detects three classes of issues:
//  1. ORPHANED — files in a provides-covered directory with no claiming plugin
//  2. DUPLICATE-OWNER — two or more plugins claiming the same category
//  3. STALE — provider-variant instruction files for absent providers
func CheckAdvAssets(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	configDir := opts.ConfigDir
	if configDir == "" {
		return []Check{{
			Name:    "adv-assets.config",
			Status:  StatusWarn,
			Message: "adv-assets health requires ConfigDir option",
		}}, nil
	}

	// Check if config dir exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return []Check{{
			Name:    "adv-assets.config_dir",
			Status:  StatusPass,
			Message: fmt.Sprintf("config directory does not exist: %s (no deployed assets to audit)", configDir),
		}}, nil
	}

	var checks []Check

	// Build ownership map: category → list of plugin names claiming it
	ownershipMap := make(map[cfg.ProvidesCategory][]string)
	for name, plugin := range stack.Plugins {
		for _, cat := range plugin.Provides {
			ownershipMap[cat] = append(ownershipMap[cat], name)
		}
	}

	// Stable iteration order
	var categories []cfg.ProvidesCategory
	for cat := range ownershipMap {
		categories = append(categories, cat)
	}
	sort.Slice(categories, func(i, j int) bool {
		return string(categories[i]) < string(categories[j])
	})

	for _, cat := range categories {
		owners := whichProvides(stack.Plugins, cat)
		dir, hasDir := categoryTargetDir(cat, configDir)
		if !hasDir {
			continue
		}

		// DUPLICATE-OWNER: 2+ plugins claim the same category
		if len(owners) >= 2 {
			sort.Strings(owners)
			checks = append(checks, Check{
				Name:    fmt.Sprintf("adv-assets.%s.duplicate-owner", cat),
				Status:  StatusFail,
				Message: fmt.Sprintf("plugins %s all declare provides = [\"%s\"]", strings.Join(owners, ", "), cat),
				Hint:    "only one plugin should own each category; remove the duplicate provides declaration",
			})
		}

		// ORPHANED: scan directory for files not owned by any plugin
		// Only scan if exactly 1 owner (valid ownership) or 2+ (still scan to list orphans)
		entries, err := os.ReadDir(dir)
		if err != nil {
			// Directory doesn't exist — no orphans to find
			continue
		}

		for _, entry := range entries {
			name := entry.Name()
			// Skip adv-* files — these are plugin-managed
			if strings.HasPrefix(name, "adv-") {
				continue
			}
			// This file is NOT plugin-managed → orphaned
			checks = append(checks, Check{
				Name:    fmt.Sprintf("adv-assets.%s.orphaned.%s", cat, name),
				Status:  StatusWarn,
				Message: fmt.Sprintf("file %q in %s/ has no owner (no plugin manages non-adv-* files in this category)", name, filepath.Base(dir)),
				Hint:    "this may be a user-owned file or a leftover from an uninstalled plugin",
			})
		}
	}

	// STALE: scan instructions dir for provider-variant files
	// Pattern: adv-{provider}.md where {provider} is not in [providers]
	checks = append(checks, checkStaleProviders(configDir, stack)...)

	return checks, nil
}

// staleProviderPattern matches adv-{name}.md files in the instructions directory.
var staleProviderPattern = regexp.MustCompile(`^adv-([a-zA-Z0-9_-]+)\.md$`)

// checkStaleProviders scans the instructions directory for adv-{provider}.md
// files and checks if the provider exists in the stack config.
func checkStaleProviders(configDir string, stack *cfg.Stack) []Check {
	instrDir := filepath.Join(configDir, "instructions")
	entries, err := os.ReadDir(instrDir)
	if err != nil {
		return nil
	}

	var checks []Check
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := staleProviderPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		providerName := matches[1]
		if _, exists := stack.Providers[providerName]; !exists {
			checks = append(checks, Check{
				Name:    fmt.Sprintf("adv-assets.stale.%s", providerName),
				Status:  StatusWarn,
				Message: fmt.Sprintf("stale provider instruction file %q but no [%s] provider in stack.toml", entry.Name(), providerName),
				Hint:    fmt.Sprintf("remove %s or add a [providers.%s] section to stack.toml", entry.Name(), providerName),
			})
		}
	}
	return checks
}

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
	sort.Strings(names)
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
