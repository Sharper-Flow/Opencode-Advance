package health

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func init() {
	registerBuiltin("adv-plugin", CheckADVPlugin)
}

// CheckADVPlugin verifies the Advance plugin checkout state and ADV state
// directory health.
func CheckADVPlugin(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	var checks []Check

	// 1. Check that Advance plugin is declared
	var advancePlugin *cfg.Plugin
	var advanceName string
	for name, p := range stack.Plugins {
		if isAdvancePlugin(p.Source) {
			advancePlugin = &p
			advanceName = name
			break
		}
	}

	if advancePlugin == nil {
		checks = append(checks, Check{
			Name:    "advance-plugin-declared",
			Status:  StatusFail,
			Message: "No Advance plugin declared in stack.toml",
			Hint:    "Add [plugins.advance] with source = \"https://github.com/Sharper-Flow/Advance.git\"",
		})
		return checks, nil
	}

	checks = append(checks, Check{
		Name:    "advance-plugin-declared",
		Status:  StatusPass,
		Message: "Advance plugin declared as " + advanceName,
	})

	// 2. Check checkout directory exists
	checkout := advancePlugin.Checkout
	if checkout == "" {
		// Try to resolve from path token
		checkout = advancePlugin.Path
	}
	if checkout == "" {
		checks = append(checks, Check{
			Name:    "advance-checkout-exists",
			Status:  StatusFail,
			Message: "Advance plugin has no checkout or path configured",
			Hint:    "Set checkout = \"{checkout}/advance\" in [plugins.advance]",
		})
		return checks, nil
	}

	if _, err := os.Stat(checkout); err != nil {
		checks = append(checks, Check{
			Name:    "advance-checkout-exists",
			Status:  StatusFail,
			Message: "Advance plugin checkout not found: " + checkout,
			Hint:    "Run 'oca apply --target plugins' to clone the Advance plugin",
		})
		return checks, nil
	}

	checks = append(checks, Check{
		Name:    "advance-checkout-exists",
		Status:  StatusPass,
		Message: "Advance plugin checkout found: " + checkout,
	})

	// 3. Check build artifact exists
	buildArtifact := filepath.Join(checkout, "dist", "index.js")
	if advancePlugin.Subdir != "" {
		buildArtifact = filepath.Join(checkout, advancePlugin.Subdir, "dist", "index.js")
	}

	if _, err := os.Stat(buildArtifact); err != nil {
		checks = append(checks, Check{
			Name:    "advance-build-artifact",
			Status:  StatusFail,
			Message: "Advance build artifact not found: " + buildArtifact,
			Hint:    "Run 'oca update advance' to build the plugin",
		})
	} else {
		checks = append(checks, Check{
			Name:    "advance-build-artifact",
			Status:  StatusPass,
			Message: "Advance build artifact found: " + buildArtifact,
		})
	}

	// 4. Check ADV state directory readable
	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData == "" {
		home, _ := os.UserHomeDir()
		xdgData = filepath.Join(home, ".local", "share")
	}
	advStateDir := filepath.Join(xdgData, "opencode", "plugins", "advance")

	if _, err := os.Stat(advStateDir); err != nil {
		checks = append(checks, Check{
			Name:    "adv-state-directory",
			Status:  StatusWarn,
			Message: "ADV state directory not found: " + advStateDir,
			Hint:    "ADV state is created after the first ADV change is created in any project",
		})
	} else {
		checks = append(checks, Check{
			Name:    "adv-state-directory",
			Status:  StatusPass,
			Message: "ADV state directory readable: " + advStateDir,
		})
	}

	return checks, nil
}

func isAdvancePlugin(source string) bool {
	return source == "https://github.com/Sharper-Flow/Advance.git" ||
		source == "github.com/Sharper-Flow/Advance" ||
		strings.Contains(source, "Sharper-Flow/Advance")
}
