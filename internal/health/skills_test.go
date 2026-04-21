package health

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestCheckSkills_HappyPath(t *testing.T) {
	// Setup: create a fake assets root with skill dirs, a target skills dir,
	// and a stack with skills.order matching.
	assetsDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source skill dirs
	os.MkdirAll(filepath.Join(assetsDir, "lgrep", "SKILL.md"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), []byte("lgrep skill"), 0o644)
	os.MkdirAll(filepath.Join(assetsDir, "morph", "SKILL.md"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "morph", "SKILL.md"), []byte("morph skill"), 0o644)

	// Create matching target dirs (already applied)
	os.MkdirAll(filepath.Join(targetDir, "lgrep"), 0o755)
	os.WriteFile(filepath.Join(targetDir, "lgrep", "SKILL.md"), []byte("lgrep skill"), 0o644)
	os.MkdirAll(filepath.Join(targetDir, "morph"), 0o755)
	os.WriteFile(filepath.Join(targetDir, "morph", "SKILL.md"), []byte("morph skill"), 0o644)

	stack := &cfg.Stack{
		Skills: cfg.SkillsSection{
			Order: []string{"lgrep", "morph"},
		},
	}

	opts := Options{
		SkillsAssetsRoot: assetsDir,
		SkillsTargetDir:  targetDir,
	}

	checks, err := CheckSkills(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckSkills returned error: %v", err)
	}

	// All checks should pass
	for _, c := range checks {
		if c.Status != StatusPass {
			t.Errorf("expected pass for %s, got %s: %s", c.Name, c.Status, c.Message)
		}
	}
}

func TestCheckSkills_MissingSkillDir(t *testing.T) {
	assetsDir := t.TempDir()
	targetDir := t.TempDir()

	// Source has lgrep
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), []byte("lgrep skill"), 0o644)

	// Target has nothing

	stack := &cfg.Stack{
		Skills: cfg.SkillsSection{
			Order: []string{"lgrep"},
		},
	}

	opts := Options{
		SkillsAssetsRoot: assetsDir,
		SkillsTargetDir:  targetDir,
	}

	checks, err := CheckSkills(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckSkills returned error: %v", err)
	}

	// Should have at least one warn for missing lgrep
	found := false
	for _, c := range checks {
		if c.Name == "skills.lgrep.present" && c.Status == StatusWarn {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warn for skills.lgrep.present, got checks: %+v", checks)
	}
}

func TestCheckSkills_ReservedAdvNamespace(t *testing.T) {
	assetsDir := t.TempDir()
	targetDir := t.TempDir()

	// Source has adv-something (should never happen in production, but test the guard)
	os.MkdirAll(filepath.Join(assetsDir, "adv-review-methodology"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "adv-review-methodology", "SKILL.md"), []byte("adv skill"), 0o644)

	stack := &cfg.Stack{
		Skills: cfg.SkillsSection{
			Order: []string{"adv-review-methodology"},
		},
	}

	opts := Options{
		SkillsAssetsRoot: assetsDir,
		SkillsTargetDir:  targetDir,
	}

	checks, err := CheckSkills(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckSkills returned error: %v", err)
	}

	// Should fail on reserved adv-*
	found := false
	for _, c := range checks {
		if c.Name == "skills.adv-review-methodology.reserved" && c.Status == StatusFail {
			found = true
		}
	}
	if !found {
		t.Errorf("expected fail for reserved adv-* skill, got: %+v", checks)
	}
}

func TestCheckSkills_ExtraOCASkillInTarget(t *testing.T) {
	assetsDir := t.TempDir()
	targetDir := t.TempDir()

	// Source has lgrep + old-orphan-skill (morph was once OCA-owned but removed from order)
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), []byte("lgrep skill"), 0o644)
	os.MkdirAll(filepath.Join(assetsDir, "old-orphan-skill"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "old-orphan-skill", "SKILL.md"), []byte("old skill"), 0o644)

	// Target has lgrep + old-orphan-skill (no longer in declared order)
	os.MkdirAll(filepath.Join(targetDir, "lgrep"), 0o755)
	os.WriteFile(filepath.Join(targetDir, "lgrep", "SKILL.md"), []byte("lgrep skill"), 0o644)
	os.MkdirAll(filepath.Join(targetDir, "old-orphan-skill"), 0o755)
	os.WriteFile(filepath.Join(targetDir, "old-orphan-skill", "SKILL.md"), []byte("old skill"), 0o644)

	stack := &cfg.Stack{
		Skills: cfg.SkillsSection{
			Order: []string{"lgrep"},
		},
	}

	opts := Options{
		SkillsAssetsRoot: assetsDir,
		SkillsTargetDir:  targetDir,
	}

	checks, err := CheckSkills(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckSkills returned error: %v", err)
	}

	// Should warn about extra OCA-owned dir in target not declared
	found := false
	for _, c := range checks {
		if c.Name == "skills.old-orphan-skill.extra" && c.Status == StatusWarn {
			found = true
		}
	}
	if !found {
		// List all checks for debug
		var names []string
		for _, c := range checks {
			names = append(names, c.Name+"="+string(c.Status))
		}
		sort.Strings(names)
		t.Errorf("expected warn for skills.old-orphan-skill.extra, got: %v", names)
	}
}

func TestCheckSkills_AdvDirInTargetIgnored(t *testing.T) {
	assetsDir := t.TempDir()
	targetDir := t.TempDir()

	// Source has lgrep
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), []byte("lgrep skill"), 0o644)

	// Target has lgrep + adv-something (Advance-owned, should not warn)
	os.MkdirAll(filepath.Join(targetDir, "lgrep"), 0o755)
	os.WriteFile(filepath.Join(targetDir, "lgrep", "SKILL.md"), []byte("lgrep skill"), 0o644)
	os.MkdirAll(filepath.Join(targetDir, "adv-tron"), 0o755)

	stack := &cfg.Stack{
		Skills: cfg.SkillsSection{
			Order: []string{"lgrep"},
		},
	}

	opts := Options{
		SkillsAssetsRoot: assetsDir,
		SkillsTargetDir:  targetDir,
	}

	checks, err := CheckSkills(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckSkills returned error: %v", err)
	}

	// adv-* dirs in target should NOT produce any check (they're Advance-owned)
	for _, c := range checks {
		if c.Status != StatusPass {
			t.Errorf("expected all pass (adv-* ignored), got %s for %s: %s", c.Status, c.Name, c.Message)
		}
	}
}

func TestCheckSkills_DeclaredButNoAssetSource(t *testing.T) {
	assetsDir := t.TempDir()
	targetDir := t.TempDir()

	// Empty assets dir — no lgrep source
	stack := &cfg.Stack{
		Skills: cfg.SkillsSection{
			Order: []string{"lgrep"},
		},
	}

	opts := Options{
		SkillsAssetsRoot: assetsDir,
		SkillsTargetDir:  targetDir,
	}

	checks, err := CheckSkills(context.Background(), stack, opts)
	if err != nil {
		t.Fatalf("CheckSkills returned error: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "skills.lgrep.source" && c.Status == StatusFail {
			found = true
		}
	}
	if !found {
		t.Errorf("expected fail for skills.lgrep.source (no asset dir), got: %+v", checks)
	}
}
