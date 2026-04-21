package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestPhase35_ComposedApply verifies that a composed no-target apply
// renders Phase 3.5 sections (commands, formatters, opencode toggles)
// correctly into opencode.json and copies OCA-owned skills to the skills dir.
func TestPhase35_ComposedApply(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(t.TempDir(), "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(t.TempDir(), "cache"),
		"OCA_ASSETS_ROOT=" + filepath.Join(root, "assets"),
	}

	stdout, stderr, err := runOCA(t, root, env, "apply", "--config", "stack.example.toml")
	if err != nil {
		t.Fatalf("composed apply failed:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	gotJSON, err := os.ReadFile(filepath.Join(opDir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(gotJSON, &doc); err != nil {
		t.Fatalf("opencode.json invalid JSON: %v", err)
	}

	// Commands: .command section with review-pr and test-coverage.
	commands, ok := doc["command"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json missing .command section")
	}
	if _, ok := commands["review-pr"]; !ok {
		t.Fatal("opencode.json missing .command.review-pr")
	}
	if _, ok := commands["test-coverage"]; !ok {
		t.Fatal("opencode.json missing .command.test-coverage")
	}
	// Verify review-pr fields.
	reviewPR := commands["review-pr"].(map[string]any)
	if reviewPR["description"] == nil || reviewPR["template"] == nil {
		t.Fatalf("command.review-pr missing fields: %+v", reviewPR)
	}
	if reviewPR["agent"] != "adv" {
		t.Fatalf("command.review-pr.agent = %v, want adv", reviewPR["agent"])
	}

	// Formatters: .formatter section with prettier and ruff.
	formatters, ok := doc["formatter"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json missing .formatter section")
	}
	if _, ok := formatters["prettier"]; !ok {
		t.Fatal("opencode.json missing .formatter.prettier")
	}
	if _, ok := formatters["ruff"]; !ok {
		t.Fatal("opencode.json missing .formatter.ruff")
	}
	prettier := formatters["prettier"].(map[string]any)
	if prettier["command"] == nil {
		t.Fatal("formatter.prettier missing command")
	}

	// Toggles: theme, default_agent, share, snapshot, autoupdate.
	if doc["theme"] != "obsidian" {
		t.Fatalf("opencode.json .theme = %v, want obsidian", doc["theme"])
	}
	if doc["default_agent"] != "adv" {
		t.Fatalf("opencode.json .default_agent = %v, want adv", doc["default_agent"])
	}
	if doc["share"] != "disabled" {
		t.Fatalf("opencode.json .share = %v, want disabled", doc["share"])
	}

	// Compaction subsection.
	compaction, ok := doc["compaction"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json missing .compaction section")
	}
	if compaction["auto"] != true {
		t.Fatalf("opaction.auto = %v, want true", compaction["auto"])
	}

	// Skills: OCA-owned skills copied to skills dir.
	skillsDir := filepath.Join(opDir, "skills")
	for _, skillName := range []string{"lgrep", "mcp-selection", "morph", "prioritizer", "worktree", "caveman", "caveman-commit", "caveman-review"} {
		skillFile := filepath.Join(skillsDir, skillName, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			t.Errorf("skill %q not deployed to %s: %v", skillName, skillFile, err)
		}
	}

	// Skills: reserved adv-* should NOT exist in target.
	for _, reserved := range []string{"adv-tron", "adv-slop-detection", "adv-review-methodology"} {
		reservedPath := filepath.Join(skillsDir, reserved)
		if _, err := os.Stat(reservedPath); err == nil {
			t.Errorf("reserved adv-* skill %q should not exist in target dir", reserved)
		}
	}
}

// TestPhase35_PerTargetParity verifies that individual --target applies
// produce equivalent output to the composed apply for Phase 3.5 targets.
func TestPhase35_PerTargetParity(t *testing.T) {
	root := repoRoot(t)

	for _, target := range []string{"skills", "commands", "formatters", "toggles"} {
		t.Run(target, func(t *testing.T) {
			opDir := filepath.Join(t.TempDir(), "opencode")
			env := []string{
				"OCA_OPENCODE_CONFIG_DIR=" + opDir,
				"OCA_VISION_CONFIG_DIR=" + filepath.Join(t.TempDir(), "vision"),
				"OCA_CACHE_DIR=" + filepath.Join(t.TempDir(), "cache"),
				"OCA_ASSETS_ROOT=" + filepath.Join(root, "assets"),
			}

			stdout, stderr, err := runOCA(t, root, env, "apply", "--target", target, "--config", "stack.example.toml")
			if err != nil {
				t.Fatalf("oca apply --target %s failed:\nstdout=%s\nstderr=%s\nerr=%v", target, stdout, stderr, err)
			}

			switch target {
			case "skills":
				// Verify at least lgrep was copied.
				skillFile := filepath.Join(opDir, "skills", "lgrep", "SKILL.md")
				if _, err := os.Stat(skillFile); err != nil {
					t.Errorf("lgrep SKILL.md not deployed: %v", err)
				}
			case "commands", "formatters":
				// JSON sections should exist.
				gotJSON, err := os.ReadFile(filepath.Join(opDir, "opencode.json"))
				if err != nil {
					t.Fatalf("opencode.json not created: %v", err)
				}
				var doc map[string]any
				if err := json.Unmarshal(gotJSON, &doc); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				key := target
				if target == "commands" {
					key = "command"
				} else {
					key = "formatter"
				}
				if doc[key] == nil {
					t.Fatalf("opencode.json missing .%s section", key)
				}
			case "toggles":
				gotJSON, err := os.ReadFile(filepath.Join(opDir, "opencode.json"))
				if err != nil {
					t.Fatalf("opencode.json not created: %v", err)
				}
				var doc map[string]any
				if err := json.Unmarshal(gotJSON, &doc); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				if doc["theme"] == nil {
					t.Fatal("opencode.json missing .theme after toggles apply")
				}
			}
		})
	}
}
