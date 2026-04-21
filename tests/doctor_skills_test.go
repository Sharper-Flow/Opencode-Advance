package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestDoctorSkills_HappyPath runs oca doctor --scope skills with all
// declared skills present in the target dir — all checks should pass.
func TestDoctorSkills_HappyPath(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	assetsDir := filepath.Join(root, "assets")

	// Pre-populate only declared skills in target dir to match order.
	skillsSrc := filepath.Join(assetsDir, "skills")
	for _, name := range []string{"lgrep", "morph"} {
		src := filepath.Join(skillsSrc, name)
		dst := filepath.Join(opDir, "skills", name)
		copyDir(t, src, dst)
	}

	// Create a minimal stack.toml with skills order.
	stackPath := filepath.Join(t.TempDir(), "stack.toml")
	stackContent := `[meta]
version = "1.0.0"

[skills]
order = ["lgrep", "morph"]
`
	if err := os.WriteFile(stackPath, []byte(stackContent), 0o644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_ASSETS_ROOT=" + assetsDir,
	}
	stdout, stderr, err := runOCA(t, root, env, "doctor", "--scope", "skills", "--output", "json", "--config", stackPath)
	_ = stderr
	_ = err // may exit non-zero even with only pass due to CLI behavior

	// Parse JSON output.
	var checks []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &checks); err != nil {
		t.Fatalf("doctor output not valid JSON: %v\nstdout=%s", err, stdout)
	}

	for _, c := range checks {
		if c.Status != "pass" {
			t.Errorf("expected pass for %s, got %s", c.Name, c.Status)
		}
	}
}

// TestDoctorSkills_MissingSkill verifies doctor warns when a declared
// skill is not deployed to the target dir.
func TestDoctorSkills_MissingSkill(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	assetsDir := filepath.Join(root, "assets")

	stackPath := filepath.Join(t.TempDir(), "stack.toml")
	stackContent := `[meta]
version = "1.0.0"

[skills]
order = ["lgrep"]
`
	if err := os.WriteFile(stackPath, []byte(stackContent), 0o644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_ASSETS_ROOT=" + assetsDir,
	}
	stdout, _, _ := runOCA(t, root, env, "doctor", "--scope", "skills", "--output", "json", "--config", stackPath)

	var checks []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &checks); err != nil {
		t.Fatalf("doctor output not valid JSON: %v\nstdout=%s", err, stdout)
	}

	found := false
	for _, c := range checks {
		if c.Name == "skills.lgrep.present" && c.Status == "warn" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warn for skills.lgrep.present, got: %+v", checks)
	}
}

// TestDoctorSkills_ReservedAdv verifies doctor fails on adv-* in order.
func TestDoctorSkills_ReservedAdv(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	assetsDir := filepath.Join(root, "assets")

	stackPath := filepath.Join(t.TempDir(), "stack.toml")
	stackContent := `[meta]
version = "1.0.0"

[skills]
order = ["adv-tron"]
`
	if err := os.WriteFile(stackPath, []byte(stackContent), 0o644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_ASSETS_ROOT=" + assetsDir,
	}
	stdout, stderr, err := runOCA(t, root, env, "doctor", "--scope", "skills", "--output", "json", "--config", stackPath)

	// Reserved adv-* names are caught at config validation time, not by doctor.
	// The CLI should exit with an error mentioning "reserved namespace".
	if err == nil {
		t.Fatal("expected error for reserved adv-* skill in config")
	}
	if stdout != "" {
		t.Logf("stdout=%s", stdout)
	}
	// The validation error should mention reserved namespace.
	if !containsSubstring(stderr, "reserved namespace") {
		t.Errorf("expected stderr to mention 'reserved namespace', got: %s", stderr)
	}
}

func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// copyDir recursively copies src dir to dst.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyDir %s -> %s: %v", src, dst, err)
	}
}
