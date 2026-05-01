package render

import (
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestPlanSkills_EnumeratesAssetDirs(t *testing.T) {
	// Create a fake assets/skills/ directory structure
	tmp := t.TempDir()
	assetsDir := filepath.Join(tmp, "assets", "skills")
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	os.MkdirAll(filepath.Join(assetsDir, "morph"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), []byte("lgrep skill"), 0o644)
	os.WriteFile(filepath.Join(assetsDir, "morph", "SKILL.md"), []byte("morph skill"), 0o644)
	os.WriteFile(filepath.Join(assetsDir, "README.md"), []byte("readme"), 0o644) // should be skipped

	stack := &cfg.Stack{
		Meta: cfg.Meta{Version: "1.0.0"},
		MCP:  cfg.MCPSection{Servers: map[string]cfg.Server{"a": {Port: 6276, Command: "echo"}}},
	}
	paths := cfg.Paths{OpencodeConfigDir: filepath.Join(tmp, "config")}

	ops := PlanSkillsOps(stack, paths, assetsDir)
	if len(ops) != 2 {
		t.Fatalf("expected 2 skill ops, got %d: %+v", len(ops), ops)
	}
	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	if !names["skills/lgrep/SKILL.md"] {
		t.Error("missing lgrep/SKILL.md")
	}
	if !names["skills/morph/SKILL.md"] {
		t.Error("missing morph/SKILL.md")
	}
}

func TestPlanSkills_FiltersAdvNamespace(t *testing.T) {
	tmp := t.TempDir()
	assetsDir := filepath.Join(tmp, "assets", "skills")
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	os.MkdirAll(filepath.Join(assetsDir, "adv-apply-methodology"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), []byte("lgrep"), 0o644)
	os.WriteFile(filepath.Join(assetsDir, "adv-apply-methodology", "SKILL.md"), []byte("adv"), 0o644)

	stack := &cfg.Stack{
		Meta: cfg.Meta{Version: "1.0.0"},
		MCP:  cfg.MCPSection{Servers: map[string]cfg.Server{"a": {Port: 6276, Command: "echo"}}},
	}
	paths := cfg.Paths{OpencodeConfigDir: filepath.Join(tmp, "config")}

	ops := PlanSkillsOps(stack, paths, assetsDir)
	if len(ops) != 1 {
		t.Fatalf("expected 1 skill op (adv-* filtered), got %d", len(ops))
	}
	if ops[0].Name != "skills/lgrep/SKILL.md" {
		t.Errorf("expected lgrep, got %s", ops[0].Name)
	}
}

func TestPlanSkills_OrdersFromConfig(t *testing.T) {
	tmp := t.TempDir()
	assetsDir := filepath.Join(tmp, "assets", "skills")
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	os.MkdirAll(filepath.Join(assetsDir, "morph"), 0o755)
	os.MkdirAll(filepath.Join(assetsDir, "prioritizer"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), []byte("lgrep"), 0o644)
	os.WriteFile(filepath.Join(assetsDir, "morph", "SKILL.md"), []byte("morph"), 0o644)
	os.WriteFile(filepath.Join(assetsDir, "prioritizer", "SKILL.md"), []byte("prioritizer"), 0o644)

	// Only request morph and prioritizer
	stack := &cfg.Stack{
		Meta:   cfg.Meta{Version: "1.0.0"},
		MCP:    cfg.MCPSection{Servers: map[string]cfg.Server{"a": {Port: 6276, Command: "echo"}}},
		Skills: cfg.SkillsSection{Order: []string{"morph", "prioritizer"}},
	}
	paths := cfg.Paths{OpencodeConfigDir: filepath.Join(tmp, "config")}

	ops := PlanSkillsOps(stack, paths, assetsDir)
	if len(ops) != 2 {
		t.Fatalf("expected 2 ops (order-restricted), got %d", len(ops))
	}
	for _, op := range ops {
		if filepath.Base(filepath.Dir(op.Path)) == "lgrep" {
			t.Error("lgrep should be excluded when not in order")
		}
	}
}

func TestPlanSkills_NoopWhenMatches(t *testing.T) {
	tmp := t.TempDir()
	assetsDir := filepath.Join(tmp, "assets", "skills")
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	content := []byte("lgrep skill content")
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), content, 0o644)

	// Pre-create matching target
	configDir := filepath.Join(tmp, "config")
	targetDir := filepath.Join(configDir, "skills", "lgrep")
	os.MkdirAll(targetDir, 0o755)
	os.WriteFile(filepath.Join(targetDir, "SKILL.md"), content, 0o644)

	stack := &cfg.Stack{
		Meta: cfg.Meta{Version: "1.0.0"},
		MCP:  cfg.MCPSection{Servers: map[string]cfg.Server{"a": {Port: 6276, Command: "echo"}}},
	}
	paths := cfg.Paths{OpencodeConfigDir: configDir}

	ops := PlanSkillsOps(stack, paths, assetsDir)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Op != "noop" {
		t.Errorf("expected noop for matching content, got %s", ops[0].Op)
	}
}

func TestPlanSkills_EmptyWhenNoAssetsDir(t *testing.T) {
	stack := &cfg.Stack{
		Meta: cfg.Meta{Version: "1.0.0"},
		MCP:  cfg.MCPSection{Servers: map[string]cfg.Server{"a": {Port: 6276, Command: "echo"}}},
	}
	paths := cfg.Paths{OpencodeConfigDir: "/nonexistent"}

	ops := PlanSkillsOps(stack, paths, "/nonexistent/assets/skills")
	if len(ops) != 0 {
		t.Errorf("expected 0 ops for missing assets dir, got %d", len(ops))
	}
}

func TestApply_SkillsCreatesParentDirsAndWrites(t *testing.T) {
	tmp := t.TempDir()
	assetsDir := filepath.Join(tmp, "assets", "skills")
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), []byte("lgrep content"), 0o644)

	configDir := filepath.Join(tmp, "config")
	// Note: skills dir does NOT exist yet — Apply should create it.

	stack := &cfg.Stack{
		Meta: cfg.Meta{Version: "1.0.0"},
		MCP:  cfg.MCPSection{Servers: map[string]cfg.Server{"a": {Port: 6276, Command: "echo"}}},
	}
	paths := cfg.Paths{OpencodeConfigDir: configDir}

	ops := PlanSkillsOps(stack, paths, assetsDir)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}

	plan := &Plan{
		LockPath: filepath.Join(tmp, "test.lock"),
		Targets:  ops,
	}
	result, err := Apply(plan, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !result.Targets[0].Wrote {
		t.Error("expected write")
	}

	// Verify file was written
	got, err := os.ReadFile(filepath.Join(configDir, "skills", "lgrep", "SKILL.md"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(got) != "lgrep content" {
		t.Errorf("content = %q, want %q", got, "lgrep content")
	}
}

func TestApply_SkillsNoopOnMatch(t *testing.T) {
	tmp := t.TempDir()
	assetsDir := filepath.Join(tmp, "assets", "skills")
	os.MkdirAll(filepath.Join(assetsDir, "lgrep"), 0o755)
	content := []byte("lgrep content")
	os.WriteFile(filepath.Join(assetsDir, "lgrep", "SKILL.md"), content, 0o644)

	configDir := filepath.Join(tmp, "config")
	targetDir := filepath.Join(configDir, "skills", "lgrep")
	os.MkdirAll(targetDir, 0o755)
	os.WriteFile(filepath.Join(targetDir, "SKILL.md"), content, 0o644)

	stack := &cfg.Stack{
		Meta: cfg.Meta{Version: "1.0.0"},
		MCP:  cfg.MCPSection{Servers: map[string]cfg.Server{"a": {Port: 6276, Command: "echo"}}},
	}
	paths := cfg.Paths{OpencodeConfigDir: configDir}

	ops := PlanSkillsOps(stack, paths, assetsDir)
	plan := &Plan{LockPath: filepath.Join(tmp, "test.lock"), Targets: ops}
	result, err := Apply(plan, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.Targets[0].Wrote {
		t.Error("expected noop (no write) for matching content")
	}
}
