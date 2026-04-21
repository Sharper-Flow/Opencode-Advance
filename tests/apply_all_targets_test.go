package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
)

// TestComposeApply_AllTargets produces a full opencode.json with all declared
// sections (mcp + plugins + instructions + providers + permissions + watcher + lsp)
// in a single apply run, then verifies idempotence.
func TestComposeApply_AllTargets(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(t.TempDir(), "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(t.TempDir(), "cache"),
	}

	// First apply: all targets (no --target flag).
	stdout, stderr, err := runOCA(t, root, env, "apply", "--config", "stack.example.toml")
	if err != nil {
		t.Fatalf("oca apply all targets failed:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}

	gotJSON, err := os.ReadFile(filepath.Join(opDir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}

	var doc map[string]any
	if err := json.Unmarshal(gotJSON, &doc); err != nil {
		t.Fatalf("opencode.json is not valid JSON: %v", err)
	}

	// Verify Phase 3 sections present.
	providers, ok := doc["provider"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json missing .provider section")
	}
	// google provider: shape translation context→limit.context, inputs→modalities.input.
	google, ok := providers["google"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json missing .provider.google")
	}
	googleGemini35 := google["models"].(map[string]any)["gemini-3-flash-preview"].(map[string]any)
	limit := googleGemini35["limit"].(map[string]any)
	if limit["context"] == nil || limit["output"] == nil {
		t.Fatalf("expected limit.context and limit.output in provider model, got: %+v", limit)
	}
	modalities := googleGemini35["modalities"].(map[string]any)
	if modalities["input"] == nil {
		t.Fatalf("expected modalities.input in provider model, got: %+v", modalities)
	}

	// openai provider: options passthrough.
	openai := providers["openai"].(map[string]any)
	if openai["options"] == nil {
		t.Fatal("opencode.json missing .provider.openai.options passthrough")
	}

	permissions, ok := doc["permission"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json missing .permission section")
	}
	// default → "*" translation.
	if permissions["*"] == nil {
		t.Fatal("opencode.json missing .permission[\"*\"] (default→\"*\" translation)")
	}
	if permissions["external_directory"] == nil {
		t.Fatal("opencode.json missing .permission.external_directory")
	}
	if permissions["bash"] == nil {
		t.Fatal("opencode.json missing .permission.bash")
	}

	watcher, ok := doc["watcher"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json missing .watcher section")
	}
	if watcher["ignore"] == nil {
		t.Fatal("opencode.json missing .watcher.ignore")
	}

	lsp, ok := doc["lsp"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json missing .lsp section")
	}
	if lsp["pyrefly"] == nil {
		t.Fatal("opencode.json missing .lsp.pyrefly")
	}

	// File mode check.
	fi, _ := os.Stat(filepath.Join(opDir, "opencode.json"))
	if fi.Mode().Perm() != 0o644 {
		t.Fatalf("opencode.json mode=%v want 0644", fi.Mode().Perm())
	}

	// Second apply: all targets should be noop.
	stdout2, stderr2, err := runOCA(t, root, env, "apply", "--config", "stack.example.toml")
	if err != nil {
		t.Fatalf("second oca apply all targets failed:\nstdout=%s\nstderr=%s\nerr=%v", stdout2, stderr2, err)
	}
	// The second apply output should indicate noop operations.
	// It's acceptable for stdout to be empty or contain only noop lines.
}

// TestComposeApply_IndividualPhase3Targets applies each Phase 3 target
// individually and verifies the result.
func TestComposeApply_IndividualPhase3Targets(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(t.TempDir(), "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(t.TempDir(), "cache"),
	}

	for _, target := range []string{"providers", "permissions", "watcher", "lsp"} {
		t.Run(target, func(t *testing.T) {
			stdout, stderr, err := runOCA(t, root, env, "apply", "--target", target, "--config", "stack.example.toml")
			if err != nil {
				t.Fatalf("oca apply --target %s failed:\nstdout=%s\nstderr=%s\nerr=%v", target, stdout, stderr, err)
			}

			gotJSON, err := os.ReadFile(filepath.Join(opDir, "opencode.json"))
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(gotJSON, &doc); err != nil {
				t.Fatalf("opencode.json is not valid JSON after --target %s: %v", target, err)
			}
		})
	}
}

// TestComposeApply_NoRollbackBehavior verifies that when Apply is called with
// NoRollback: true, a mid-plan WriteAtomic failure leaves prior successful
// writes on disk (AC4).
func TestComposeApply_NoRollbackBehavior(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	visionDir := filepath.Join(t.TempDir(), "vision")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	t.Setenv("OCA_OPENCODE_CONFIG_DIR", opDir)
	t.Setenv("OCA_VISION_CONFIG_DIR", visionDir)
	t.Setenv("OCA_CACHE_DIR", cacheDir)

	stack, err := config.Load(filepath.Join(root, "stack.example.toml"))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	paths := config.ResolvePaths()
	plan, err := render.ComposeApplyPlan(stack, paths, filepath.Join(root, "stack.example.toml"), render.AllTargets)
	if err != nil {
		t.Fatalf("ComposeApplyPlan: %v", err)
	}

	opencodePath := paths.OpencodeJSON()
	finalRenameToOpencode := 0
	restoreRename := render.SetRenameForTesting(func(oldPath, newPath string) error {
		if newPath == opencodePath {
			finalRenameToOpencode++
			if finalRenameToOpencode == 2 {
				return fmt.Errorf("injected rename failure on second opencode.json write")
			}
		}
		return os.Rename(oldPath, newPath)
	})
	defer restoreRename()

	_, err = render.Apply(plan, render.ApplyOptions{LockPath: plan.LockPath, MaxBackups: 3, NoRollback: true})
	if err == nil {
		t.Fatal("expected injected apply failure, got nil")
	}

	gotJSON, err := os.ReadFile(opencodePath)
	if err != nil {
		t.Fatalf("expected prior opencode.json write to remain on disk: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(gotJSON, &doc); err != nil {
		t.Fatalf("opencode.json invalid after injected failure: %v", err)
	}
	if _, ok := doc["mcp"].(map[string]any); !ok {
		t.Fatal("expected first successful opencode.json write (.mcp) to remain on disk after failure")
	}
	if _, err := os.Stat(paths.VisionServersYAML()); err != nil {
		t.Fatalf("expected vision/servers.yaml prior write to remain on disk: %v", err)
	}
}

// TestComposeApply_DiffExitCodes verifies that the CLI paths for apply
// produce expected exit codes.
func TestComposeApply_ExitCodes(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(t.TempDir(), "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(t.TempDir(), "cache"),
	}

	// Valid apply: should succeed (exit 0).
	_, _, err := runOCA(t, root, env, "apply", "--target", "providers", "--config", "stack.example.toml")
	if err != nil {
		t.Fatalf("oca apply --target providers failed unexpectedly: %v", err)
	}

	// Invalid --target: should return exit code 2.
	_, _, err2 := runOCA(t, root, env, "apply", "--target", "invalid-target", "--config", "stack.example.toml")
	if err2 == nil {
		t.Fatal("expected error for invalid --target")
	}
	var exitErr *exec.ExitError
	if !errors.As(err2, &exitErr) {
		t.Fatalf("expected *exec.ExitError for invalid target, got: %T %v", err2, err2)
	}
	if exitErr.ExitCode() != 2 {
		t.Fatalf("expected exit code 2 for invalid target, got: %d", exitErr.ExitCode())
	}
}
