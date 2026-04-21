package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func init() {
	registerBuiltin("skills", CheckSkills)
}

// CheckSkills verifies OCA-owned skill deployment health:
//  1. Declared skills in [skills].order must have source asset directories
//  2. Reserved adv-* names are rejected
//  3. Declared skills must be present in the target dir
//  4. Extra OCA-owned skill dirs in target (not declared) produce warnings
//  5. Advance-owned adv-* dirs in target are ignored
func CheckSkills(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	assetsRoot := opts.SkillsAssetsRoot
	targetDir := opts.SkillsTargetDir

	if assetsRoot == "" || targetDir == "" {
		return []Check{{
			Name:    "skills.config",
			Status:  StatusWarn,
			Message: "skills health requires SkillsAssetsRoot and SkillsTargetDir",
		}}, nil
	}

	var checks []Check

	// Determine desired set from stack config.
	desired := stack.Skills.Order
	if len(desired) == 0 {
		// No order specified: enumerate all non-reserved asset dirs.
		entries, err := os.ReadDir(assetsRoot)
		if err != nil {
			return []Check{{
				Name:    "skills.assets_root",
				Status:  StatusFail,
				Message: fmt.Sprintf("cannot read assets root: %v", err),
			}}, nil
		}
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), "adv-") {
				desired = append(desired, e.Name())
			}
		}
	}

	// Stable order.
	sort.Strings(desired)

	// Track which skills are OCA-deployed for extra-dir detection.
	ocaDeployed := make(map[string]bool)

	for _, name := range desired {
		// Reserved namespace check.
		if strings.HasPrefix(name, "adv-") {
			checks = append(checks, Check{
				Name:    fmt.Sprintf("skills.%s.reserved", name),
				Status:  StatusFail,
				Message: fmt.Sprintf("skill %q uses reserved adv-* namespace owned by Advance", name),
				Hint:    "remove adv-* entries from [skills].order; Advance manages its own skills",
			})
			continue
		}

		ocaDeployed[name] = true

		// Check source asset dir exists.
		srcDir := filepath.Join(assetsRoot, name)
		if _, err := os.Stat(srcDir); err != nil {
			checks = append(checks, Check{
				Name:    fmt.Sprintf("skills.%s.source", name),
				Status:  StatusFail,
				Message: fmt.Sprintf("declared skill %q has no source directory in assets", name),
				Hint:    fmt.Sprintf("expected directory: %s", srcDir),
			})
			continue
		}
		checks = append(checks, Check{
			Name:   fmt.Sprintf("skills.%s.source", name),
			Status: StatusPass,
			Message: fmt.Sprintf("source asset directory exists: %s", name),
		})

		// Check target dir exists.
		dstDir := filepath.Join(targetDir, name)
		if _, err := os.Stat(dstDir); err != nil {
			checks = append(checks, Check{
				Name:    fmt.Sprintf("skills.%s.present", name),
				Status:  StatusWarn,
				Message: fmt.Sprintf("skill %q not deployed to target dir", name),
				Hint:    "run oca apply --target skills",
			})
			continue
		}
		checks = append(checks, Check{
			Name:   fmt.Sprintf("skills.%s.present", name),
			Status: StatusPass,
			Message: fmt.Sprintf("skill %q present in target dir", name),
		})
	}

	// Detect extra OCA-owned skill dirs in target not in the declared set.
	// Only warn for non-adv-* dirs that aren't in ocaDeployed.
	targetEntries, err := os.ReadDir(targetDir)
	if err != nil {
		// Target dir doesn't exist yet — no extras to check.
		return checks, nil
	}
	for _, e := range targetEntries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip Advance-owned directories.
		if strings.HasPrefix(name, "adv-") {
			continue
		}
		// Skip already-declared/checked skills.
		if ocaDeployed[name] {
			continue
		}
		// Check if this dir exists in assets (indicating it was once an OCA skill).
		srcDir := filepath.Join(assetsRoot, name)
		if _, err := os.Stat(srcDir); err == nil {
			checks = append(checks, Check{
				Name:    fmt.Sprintf("skills.%s.extra", name),
				Status:  StatusWarn,
				Message: fmt.Sprintf("extra OCA-owned skill directory %q in target but not declared", name),
				Hint:    fmt.Sprintf("add %q to [skills].order or run oca apply --target skills to prune", name),
			})
		}
	}

	return checks, nil
}
