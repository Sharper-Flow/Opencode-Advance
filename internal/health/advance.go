package health

import (
	"context"
	"os"
	"path/filepath"
	"sort"
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

	info, err := os.Stat(buildArtifact)
	if err != nil {
		checks = append(checks, Check{
			Name:    "advance-build-artifact",
			Status:  StatusFail,
			Message: "Advance build artifact not found: " + buildArtifact,
			Hint:    "Run 'oca update advance' to build the plugin",
		})
	} else if info.Size() == 0 {
		checks = append(checks, Check{
			Name:    "advance-build-artifact",
			Status:  StatusFail,
			Message: "Advance build artifact is empty: " + buildArtifact,
			Hint:    "Run 'oca update advance' to rebuild the plugin",
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

	checks = append(checks, checkLocalADVState(opts))

	return checks, nil
}

func isAdvancePlugin(source string) bool {
	return normalizeGitURL(source) == "github.com/Sharper-Flow/Advance"
}

func checkLocalADVState(opts Options) Check {
	root := opts.ProjectRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Check{
				Name:    "adv-local-state",
				Status:  StatusWarn,
				Message: "cannot resolve project root for local .adv scan",
				Hint:    "run oca doctor from a project directory",
			}
		}
		root = wd
	}

	advDir := filepath.Join(root, ".adv")
	entries, err := os.ReadDir(advDir)
	if err != nil {
		if os.IsNotExist(err) {
			return Check{Name: "adv-local-state", Status: StatusPass, Message: "no repo-local .adv directory found"}
		}
		return Check{Name: "adv-local-state", Status: StatusWarn, Message: "cannot read repo-local .adv directory: " + err.Error(), Hint: "verify project permissions"}
	}

	var residue []string
	for _, entry := range entries {
		name := entry.Name()
		switch {
		case name == "specs":
			continue
		case name == "archive" && entry.IsDir():
			residue = append(residue, archiveResidue(filepath.Join(advDir, name))...)
		case name == "changes" || name == "db" || strings.HasPrefix(name, "agenda"):
			residue = append(residue, name)
		}
	}

	if len(residue) == 0 {
		return Check{Name: "adv-local-state", Status: StatusPass, Message: "repo-local .adv state contains only valid specs/archive bundle artifacts"}
	}
	sort.Strings(residue)
	return Check{
		Name:    "adv-local-state",
		Status:  StatusWarn,
		Message: "repo-local .adv contains legacy or non-bundle mutable state: " + strings.Join(residue, ", "),
		Hint:    "from an ADV-capable session, run adv_migrate_cleanup dryRun:true before any approved cleanup; preserve .adv/specs and valid .adv/archive bundles",
	}
}

func archiveResidue(archiveDir string) []string {
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		return []string{"archive (unreadable)"}
	}
	if len(entries) == 0 {
		return []string{"archive (empty)"}
	}

	var residue []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if _, err := os.Stat(filepath.Join(archiveDir, name, "change.json")); err == nil {
				continue
			}
		}
		residue = append(residue, filepath.ToSlash(filepath.Join("archive", name)))
	}
	return residue
}
