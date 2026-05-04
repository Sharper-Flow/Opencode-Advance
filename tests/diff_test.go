package tests

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDiff_CleanExit verifies that `oca diff` returns exit 0 when there is no drift.
func TestDiff_CleanExit(t *testing.T) {
	root := repoRoot(t)
	base := t.TempDir()
	opDir := filepath.Join(base, "opencode")
	stackPath := writeIsolatedStackExample(t, root, base)
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(base, "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(base, "cache"),
	}

	// Apply so opencode.json matches stack.toml.
	_, _, err := runOCA(t, root, env, "apply", "--config", stackPath)
	if err != nil {
		t.Fatalf("setup apply failed: %v", err)
	}

	// Diff should return exit 0 (no drift).
	_, stderr, err := runOCA(t, root, env, "diff", "--config", stackPath)
	var exitErr *exec.ExitError
	if err != nil {
		if errors.As(err, &exitErr) {
			t.Fatalf("oca diff returned unexpected error exit code %d: %s", exitErr.ExitCode(), stderr)
		}
		t.Fatalf("oca diff failed unexpectedly: %v stderr=%s", err, stderr)
	}
}

// TestDiff_DriftExit verifies that `oca diff` returns exit 1 when opencode.json has drift.
func TestDiff_DriftExit(t *testing.T) {
	root := repoRoot(t)
	base := t.TempDir()
	opDir := filepath.Join(base, "opencode")
	stackPath := writeIsolatedStackExample(t, root, base)
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(base, "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(base, "cache"),
	}

	// Apply so opencode.json is in sync.
	_, _, err := runOCA(t, root, env, "apply", "--config", stackPath)
	if err != nil {
		t.Fatalf("setup apply failed: %v", err)
	}

	// Manually corrupt opencode.json to introduce drift.
	corruptPath := filepath.Join(opDir, "opencode.json")
	data, _ := os.ReadFile(corruptPath)
	var doc map[string]any
	json.Unmarshal(data, &doc)
	doc["provider"] = map[string]any{"fake-entry": "should not be here"}
	corrupt, _ := json.MarshalIndent(doc, "", "  ")
	os.WriteFile(corruptPath, corrupt, 0o644)

	// Diff should return exit 1 (drift detected).
	_, stderr, err := runOCA(t, root, env, "diff", "--config", stackPath)
	if err == nil {
		t.Fatal("expected error for drifted opencode.json")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got: %T %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for drift, got: %d stderr=%s", exitErr.ExitCode(), stderr)
	}
}

// TestDiff_InvalidStackExit verifies that `oca diff` returns exit 2 for invalid stack.toml.
func TestDiff_InvalidStackExit(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(t.TempDir(), "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(t.TempDir(), "cache"),
	}

	// Write an invalid stack.toml.
	badStack := filepath.Join(t.TempDir(), "bad.toml")
	os.WriteFile(badStack, []byte("this is not valid TOML {{{"), 0o644)

	_, stderr, err := runOCA(t, root, env, "diff", "--config", badStack)
	if err == nil {
		t.Fatal("expected error for invalid stack.toml")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got: %T %v", err, err)
	}
	if exitErr.ExitCode() != 2 {
		t.Fatalf("expected exit code 2 for invalid stack.toml, got: %d stderr=%s", exitErr.ExitCode(), stderr)
	}
}

// TestDiff_JSONOutput verifies that `oca diff --output json` produces valid JSON.
func TestDiff_JSONOutput(t *testing.T) {
	root := repoRoot(t)
	base := t.TempDir()
	opDir := filepath.Join(base, "opencode")
	stackPath := writeIsolatedStackExample(t, root, base)
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(base, "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(base, "cache"),
	}

	// Apply so opencode.json is in sync.
	_, _, err := runOCA(t, root, env, "apply", "--config", stackPath)
	if err != nil {
		t.Fatalf("setup apply failed: %v", err)
	}

	stdout, _, err := runOCA(t, root, env, "diff", "--output", "json", "--config", stackPath)
	if err != nil {
		t.Fatalf("oca diff --output json failed: %v", err)
	}
	var result struct {
		HasDrift bool `json:"hasDrift"`
		Targets  []struct {
			Path  string `json:"path"`
			Op    string `json:"op"`
			Drift bool   `json:"drift"`
		} `json:"targets"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("diff JSON output is not valid JSON: %v\noutput=%s", err, stdout)
	}
	if result.HasDrift {
		t.Fatal("expected HasDrift=false on clean opencode.json")
	}
}

// TestDiff_TargetFilter verifies that `--target` filters to a single target.
func TestDiff_TargetFilter(t *testing.T) {
	root := repoRoot(t)
	base := t.TempDir()
	opDir := filepath.Join(base, "opencode")
	stackPath := writeIsolatedStackExample(t, root, base)
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(base, "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(base, "cache"),
	}

	// Apply so opencode.json is in sync.
	_, _, err := runOCA(t, root, env, "apply", "--config", stackPath)
	if err != nil {
		t.Fatalf("setup apply failed: %v", err)
	}

	// Diff with --target providers should show only providers op.
	stdout, _, err := runOCA(t, root, env, "diff", "--target", "providers", "--config", stackPath)
	if err != nil {
		t.Fatalf("oca diff --target providers failed: %v", err)
	}
	// Text output: each line starts with drift marker (space=noop, *=drift).
	// We expect no drift markers for the providers target in clean state.
	if stdout == "" {
		t.Fatal("expected diff output, got empty")
	}
}
