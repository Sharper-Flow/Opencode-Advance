package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// These tests exercise git operations against a real (temporary) git repo.
// They require git to be installed on the system.

func TestClone_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a bare "remote" repo.
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	mustRun(t, "git", "init", "--bare", "--initial-branch=master", remote)

	// Create a local working repo, commit something, push to remote.
	work := filepath.Join(tmp, "work")
	mustRun(t, "git", "init", "--initial-branch=master", work)
	mustRun(t, "git", "-C", work, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", work, "config", "user.name", "Test")
	mustRun(t, "git", "-C", work, "commit", "--allow-empty", "-m", "initial")
	mustRun(t, "git", "-C", work, "remote", "add", "origin", remote)
	mustRun(t, "git", "-C", work, "push", "-u", "origin", "master")

	// Clone into checkout dir.
	checkout := filepath.Join(tmp, "checkout")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := Clone(ctx, remote, checkout, "")
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}
	// Verify checkout exists and has .git.
	if _, err := os.Stat(filepath.Join(checkout, ".git")); err != nil {
		t.Fatalf("checkout/.git missing after clone: %v", err)
	}
}

func TestFetch_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmp := t.TempDir()
	_, work, checkout := setupRemoteAndClone(t, tmp)

	// Add a new commit to the remote via work repo.
	writeFile(t, filepath.Join(work, "new.txt"), "content")
	mustRun(t, "git", "-C", work, "add", "new.txt")
	mustRun(t, "git", "-C", work, "commit", "-m", "second")
	mustRun(t, "git", "-C", work, "push", "origin", "master")

	// Fetch in checkout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := Fetch(ctx, checkout)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
}

func TestCheckout_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmp := t.TempDir()
	_, _, checkout := setupRemoteAndClone(t, tmp)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Checkout HEAD explicitly (should be no-op).
	err := Checkout(ctx, checkout, "master")
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}
}

func TestRevParseHEAD_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmp := t.TempDir()
	_, _, checkout := setupRemoteAndClone(t, tmp)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sha, err := RevParseHEAD(ctx, checkout)
	if err != nil {
		t.Fatalf("RevParseHEAD failed: %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("SHA length = %d, want 40: %q", len(sha), sha)
	}
}

func TestStatusClean_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmp := t.TempDir()
	_, _, checkout := setupRemoteAndClone(t, tmp)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clean, err := StatusClean(ctx, checkout)
	if err != nil {
		t.Fatalf("StatusClean failed: %v", err)
	}
	if !clean {
		t.Error("expected clean checkout after fresh clone")
	}

	// Dirty the checkout.
	writeFile(t, filepath.Join(checkout, "dirty.txt"), "changes")
	clean, err = StatusClean(ctx, checkout)
	if err != nil {
		t.Fatalf("StatusClean after dirty: %v", err)
	}
	if clean {
		t.Error("expected dirty checkout after writing file")
	}
}

// --- helpers ---

func setupRemoteAndClone(t *testing.T, tmp string) (remote, work, checkout string) {
	t.Helper()
	remote = filepath.Join(tmp, "remote.git")
	mustRun(t, "git", "init", "--bare", "--initial-branch=master", remote)

	work = filepath.Join(tmp, "work")
	mustRun(t, "git", "init", "--initial-branch=master", work)
	mustRun(t, "git", "-C", work, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", work, "config", "user.name", "Test")
	mustRun(t, "git", "-C", work, "commit", "--allow-empty", "-m", "initial")
	mustRun(t, "git", "-C", work, "remote", "add", "origin", remote)
	mustRun(t, "git", "-C", work, "push", "-u", "origin", "master")

	checkout = filepath.Join(tmp, "checkout")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := Clone(ctx, remote, checkout, ""); err != nil {
		t.Fatalf("setup clone failed: %v", err)
	}
	return
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := execCommandContext(ctx, name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run %s %v: %v\n%s", name, args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// execLookPath, execCommandContext are var so tests can override them.
var (
	execLookPath       = defaultLookPath
	execCommandContext = defaultCommandContext
)

func defaultLookPath(name string) (string, error) { return execLookPathStd(name) }
func defaultCommandContext(ctx context.Context, name string, args ...string) execCmd {
	return defaultExecCmdContext(ctx, name, args...)
}

// TestValidateGitRef exercises the ref-format allowlist used by Clone and
// Checkout. The cases cover the option-confusion, traversal, and
// metacharacter classes that motivated validation.
func TestValidateGitRef(t *testing.T) {
	good := []string{
		"trunk",
		"main",
		"master",
		"v1.0.0",
		"release/2024-04",
		"feature/foo-bar_baz",
		"abcdef0123456789abcdef0123456789abcdef01",
	}
	for _, ref := range good {
		if err := validateGitRef(ref); err != nil {
			t.Errorf("validateGitRef(%q) = %v, want nil", ref, err)
		}
	}

	bad := []string{
		"",
		"-foo",
		"--upload-pack=evil",
		"..",
		"foo..bar",
		"branch with space",
		"branch;rm -rf /",
		"branch$(whoami)",
		"ref\nwithnewline",
		"ref\x00null",
	}
	for _, ref := range bad {
		if err := validateGitRef(ref); err == nil {
			t.Errorf("validateGitRef(%q) = nil, want error", ref)
		}
	}
}
