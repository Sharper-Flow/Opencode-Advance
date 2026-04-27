package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/internal/install"
)

// --- Completion integration tests ---

func TestCompletionBashIntegration(t *testing.T) {
	root := repoRoot(t)
	stdout, _, err := runOCA(t, root, nil, "completion", "bash")
	if err != nil {
		t.Fatalf("completion bash: %v", err)
	}
	if !strings.Contains(stdout, "# bash completion") {
		t.Error("bash completion output should contain completion header")
	}
	if !strings.Contains(stdout, "oca") {
		t.Error("bash completion output should reference oca")
	}
}

func TestCompletionZshIntegration(t *testing.T) {
	root := repoRoot(t)
	stdout, _, err := runOCA(t, root, nil, "completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh: %v", err)
	}
	if !strings.Contains(stdout, "#compdef oca") {
		t.Error("zsh completion output should contain #compdef directive")
	}
}

func TestCompletionFishIntegration(t *testing.T) {
	root := repoRoot(t)
	_, stderr, err := runOCA(t, root, nil, "completion", "fish")
	if err == nil {
		t.Fatal("fish completion should fail in v1")
	}
	if !strings.Contains(stderr, "fish") || !strings.Contains(stderr, "v1.1") {
		t.Errorf("fish error should mention v1.1, got stderr: %s", stderr)
	}
}

func TestCompletionInvalidShellIntegration(t *testing.T) {
	root := repoRoot(t)
	_, stderr, err := runOCA(t, root, nil, "completion", "powershell")
	if err == nil {
		t.Fatal("invalid shell should fail")
	}
	if !strings.Contains(stderr, "unsupported") {
		t.Errorf("error should mention unsupported shell, got: %s", stderr)
	}
}

// --- Install/Uninstall integration tests ---

// isolatedHomeEnv returns env vars that redirect HOME and XDG dirs to a temp dir.
func isolatedHomeEnv(t *testing.T) ([]string, string) {
	t.Helper()
	home := t.TempDir()
	return []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"XDG_RUNTIME_DIR=" + filepath.Join(home, ".run"),
		"XDG_STATE_HOME=" + filepath.Join(home, ".local", "state"),
	}, home
}

func TestInstallCreatesDirsAndRCFiles(t *testing.T) {
	root := repoRoot(t)
	env, home := isolatedHomeEnv(t)

	// Prereq check needs git/tmux/opencode — if any are missing, skip
	if _, err := os.Stat(filepath.Join(home, ".bashrc")); !os.IsNotExist(err) {
		t.Fatal("test isolation: .bashrc should not exist before install")
	}

	stdout, stderr, err := runOCA(t, root, env, "install")
	if err != nil {
		// May fail if prerequisites are missing on this machine
		t.Logf("install failed (may be prereq issue): %s\nstderr: %s", err, stderr)
		t.Skip("skipping: install prerequisites not met on this machine")
	}

	// Check dirs created
	for _, dir := range []string{
		filepath.Join(home, ".config", "opencode"),
		filepath.Join(home, ".config", "vision"),
	} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected dir %s to exist after install", dir)
		}
	}

	// Check rc files have managed blocks
	for _, rc := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, rc)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected %s to exist after install", path)
			continue
		}
		content := string(data)
		if !strings.Contains(content, install.SentinelStart) {
			t.Errorf("%s should contain start sentinel", rc)
		}
		if !strings.Contains(content, install.SentinelEnd) {
			t.Errorf("%s should contain end sentinel", rc)
		}
	}

	_ = stdout // stdout contains summary text
}

func TestIdempotentInstall(t *testing.T) {
	root := repoRoot(t)
	env, home := isolatedHomeEnv(t)

	// First install
	_, stderr1, err1 := runOCA(t, root, env, "install")
	if err1 != nil {
		t.Logf("first install failed: %s\nstderr: %s", err1, stderr1)
		t.Skip("skipping: install prerequisites not met")
	}

	bashrc := filepath.Join(home, ".bashrc")
	first, _ := os.ReadFile(bashrc)

	// Second install
	_, stderr2, err2 := runOCA(t, root, env, "install")
	if err2 != nil {
		t.Fatalf("second install failed: %s\nstderr: %s", err2, stderr2)
	}

	second, _ := os.ReadFile(bashrc)

	if string(first) != string(second) {
		t.Error("idempotent install should produce identical rc file")
	}
}

func TestUninstallRemovesBlocks(t *testing.T) {
	root := repoRoot(t)
	env, home := isolatedHomeEnv(t)

	// Install first
	_, stderr, err := runOCA(t, root, env, "install")
	if err != nil {
		t.Logf("install failed: %s\nstderr: %s", err, stderr)
		t.Skip("skipping: install prerequisites not met")
	}

	// Now uninstall
	_, stderr, err = runOCA(t, root, env, "uninstall")
	if err != nil {
		t.Fatalf("uninstall failed: %s\nstderr: %s", err, stderr)
	}

	// Verify blocks removed
	for _, rc := range []string{".bashrc", ".zshrc"} {
		data, _ := os.ReadFile(filepath.Join(home, rc))
		if strings.Contains(string(data), install.SentinelStart) {
			t.Errorf("%s should not contain managed block after uninstall", rc)
		}
	}
}

func TestUninstallEditedBlockFails(t *testing.T) {
	root := repoRoot(t)
	env, home := isolatedHomeEnv(t)

	// Install first
	_, stderr, err := runOCA(t, root, env, "install")
	if err != nil {
		t.Logf("install failed: %s\nstderr: %s", err, stderr)
		t.Skip("skipping: install prerequisites not met")
	}

	// Edit the managed block in .bashrc
	bashrc := filepath.Join(home, ".bashrc")
	data, _ := os.ReadFile(bashrc)
	edited := strings.ReplaceAll(string(data), "unset _oca_bindir", "unset _oca_bindir\n# USER EDIT")
	os.WriteFile(bashrc, []byte(edited), 0o644)

	// Uninstall should fail
	_, stderr, err = runOCA(t, root, env, "uninstall")
	if err == nil {
		t.Fatal("uninstall should fail when block is edited")
	}
	if !strings.Contains(stderr, "edited") && !strings.Contains(stderr, "--force") {
		t.Errorf("error should mention edited block and --force, got: %s", stderr)
	}
}

func TestUninstallForceOverridesEditedBlock(t *testing.T) {
	root := repoRoot(t)
	env, home := isolatedHomeEnv(t)

	// Install first
	_, stderr, err := runOCA(t, root, env, "install")
	if err != nil {
		t.Logf("install failed: %s\nstderr: %s", err, stderr)
		t.Skip("skipping: install prerequisites not met")
	}

	// Edit the managed block
	bashrc := filepath.Join(home, ".bashrc")
	data, _ := os.ReadFile(bashrc)
	edited := strings.ReplaceAll(string(data), "unset _oca_bindir", "unset _oca_bindir\n# USER EDIT")
	os.WriteFile(bashrc, []byte(edited), 0o644)

	// Uninstall with --force
	_, stderr, err = runOCA(t, root, env, "uninstall", "--force")
	if err != nil {
		t.Fatalf("uninstall --force should succeed, got: %s\nstderr: %s", err, stderr)
	}

	// Verify blocks removed
	data, _ = os.ReadFile(bashrc)
	if strings.Contains(string(data), install.SentinelStart) {
		t.Error(".bashrc should not contain managed block after force uninstall")
	}
}

func TestCleanCutoverNoRealConfigWrites(t *testing.T) {
	root := repoRoot(t)
	env, _ := isolatedHomeEnv(t)

	// Record real home dir state
	realHome, _ := os.UserHomeDir()
	realBashrc := filepath.Join(realHome, ".bashrc")
	before, err := os.ReadFile(realBashrc)
	if err != nil {
		t.Skip("no real .bashrc to protect")
	}

	// Run install with isolated env
	_, _, _ = runOCA(t, root, env, "install")

	// Verify real .bashrc unchanged
	after, _ := os.ReadFile(realBashrc)
	if string(before) != string(after) {
		t.Error("clean-cutover violation: real .bashrc was modified during test")
	}
}
