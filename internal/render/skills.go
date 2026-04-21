package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanSkills builds the render plan for copying OCA-owned skills from
// assets/skills/ to the target skills directory. Skills are filesystem
// writes (not JSON merges), so they return TargetOps without mutating
// currentDoc.
func PlanSkills(stack *cfg.Stack, paths cfg.Paths, source string, assetsRoot string) (*Plan, error) {
	ops := PlanSkillsOps(stack, paths, assetsRoot)
	return &Plan{
		Source:   source,
		LockPath: paths.ApplyLockPath(),
		Targets:  ops,
	}, nil
}

// PlanSkillsOps enumerates OCA-owned skill directories from assetsRoot,
// filters reserved adv-* namespaces, resolves the desired set from
// [skills].order (or all OCA-owned if omitted), and emits per-file
// TargetOps under <OpencodeSkillsDir>/<skill>/... . When existing file
// bytes match source bytes, the op is "noop".
func PlanSkillsOps(stack *cfg.Stack, paths cfg.Paths, assetsRoot string) []TargetOp {
	entries, err := os.ReadDir(assetsRoot)
	if err != nil {
		return nil
	}

	// Build the set of desired skill names.
	desired := make(map[string]bool)
	if len(stack.Skills.Order) > 0 {
		for _, name := range stack.Skills.Order {
			if !strings.HasPrefix(name, "adv-") {
				desired[name] = true
			}
		}
	} else {
		// No order specified: all OCA-owned, non-reserved skills.
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), "adv-") {
				desired[e.Name()] = true
			}
		}
	}

	targetDir := paths.OpencodeSkillsDir()
	var ops []TargetOp

	// Stable iteration order.
	names := make([]string, 0, len(desired))
	for n := range desired {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, skillName := range names {
		skillDir := filepath.Join(assetsRoot, skillName)
		fileOps := planSkillDirOps(skillDir, filepath.Join(targetDir, skillName), skillName)
		ops = append(ops, fileOps...)
	}

	return ops
}

// planSkillDirOps walks one skill directory and emits TargetOps for
// each file, setting noop when existing bytes match source bytes.
func planSkillDirOps(srcDir, dstDir, skillName string) []TargetOp {
	var ops []TargetOp
	filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		name := fmt.Sprintf("skills/%s/%s", skillName, rel)
		dstPath := filepath.Join(dstDir, rel)

		srcBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		dstBytes, _ := os.ReadFile(dstPath)
		op := "write"
		if bytes.Equal(srcBytes, dstBytes) {
			op = "noop"
		}

		info, _ := d.Info()
		mode := os.FileMode(0o644)
		if info != nil {
			mode = info.Mode().Perm()
		}

		ops = append(ops, TargetOp{
			Name:   name,
			Path:   dstPath,
			Op:     op,
			Before: dstBytes,
			After:  srcBytes,
			Mode:   mode,
			Reason: fmt.Sprintf("copy OCA-owned skill file %s", name),
		})
		return nil
	})
	return ops
}
