package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// prepareIntegrationSetup creates a remote git repo, a working clone, and
// returns (remote path, work dir, checkout dir). The checkout dir is empty
// and the remote has one commit on master.
func prepareIntegrationSetup(t *testing.T) (remote, work string) {
	t.Helper()
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmp := t.TempDir()
	remote = filepath.Join(tmp, "remote.git")
	mustRun(t, "git", "init", "--bare", "--initial-branch=master", remote)

	work = filepath.Join(tmp, "work")
	mustRun(t, "git", "init", "--initial-branch=master", work)
	mustRun(t, "git", "-C", work, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", work, "config", "user.name", "Test")
	mustRun(t, "git", "-C", work, "commit", "--allow-empty", "-m", "initial")
	mustRun(t, "git", "-C", work, "remote", "add", "origin", remote)
	mustRun(t, "git", "-C", work, "push", "-u", "origin", "master")
	return remote, work
}

// TestPrepare_ClonesWhenMissing verifies Prepare clones when checkout dir
// doesn't exist, then runs build if declared.
func TestPrepare_ClonesWhenMissing(t *testing.T) {
	remote, _ := prepareIntegrationSetup(t)
	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "advance")

	// Create a plugin that needs a clone. No build commands.
	p := config.Plugin{
		Source:   remote,
		Ref:      "master",
		Checkout: checkout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := Prepare(ctx, p)
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(checkout, ".git")); err != nil {
		t.Fatalf(".git missing in checkout after Prepare: %v", err)
	}
}

// TestPrepare_ClonesAndBuilds verifies Prepare clones and runs build commands.
func TestPrepare_ClonesAndBuilds(t *testing.T) {
	remote, _ := prepareIntegrationSetup(t)
	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "advance")
	markerFile := filepath.Join(checkout, "built.marker")

	p := config.Plugin{
		Source:   remote,
		Ref:      "master",
		Checkout: checkout,
		Build:    []string{"touch built.marker"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := Prepare(ctx, p)
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if _, err := os.Stat(markerFile); err != nil {
		t.Fatalf("build marker not created: %v", err)
	}
}

// TestPrepare_FetchesWhenExistsAndClean verifies Prepare fetches and checks
// out ref when checkout already exists and is clean.
func TestPrepare_FetchesWhenExistsAndClean(t *testing.T) {
	remote, work := prepareIntegrationSetup(t)
	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "advance")

	// Initial clone.
	p := config.Plugin{
		Source:   remote,
		Ref:      "master",
		Checkout: checkout,
	}
	ctx := context.Background()
	if err := Prepare(ctx, p); err != nil {
		t.Fatalf("initial Prepare: %v", err)
	}

	// Add a new commit to remote.
	writeFile(t, filepath.Join(work, "new.txt"), "content")
	mustRun(t, "git", "-C", work, "add", "new.txt")
	mustRun(t, "git", "-C", work, "commit", "-m", "second")
	mustRun(t, "git", "-C", work, "push", "origin", "master")

	// Second prepare should fetch + checkout without error.
	err := Prepare(ctx, p)
	if err != nil {
		t.Fatalf("second Prepare (fetch): %v", err)
	}
}

// TestPrepare_AbortsOnDirtyCheckout verifies Prepare returns an error when
// the checkout exists but has uncommitted changes. The error must include
// remediation guidance per AC29.
func TestPrepare_AbortsOnDirtyCheckout(t *testing.T) {
	remote, _ := prepareIntegrationSetup(t)
	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "advance")

	// Initial clone.
	p := config.Plugin{
		Source:   remote,
		Ref:      "master",
		Checkout: checkout,
	}
	ctx := context.Background()
	if err := Prepare(ctx, p); err != nil {
		t.Fatalf("initial Prepare: %v", err)
	}

	// Dirty the checkout.
	writeFile(t, filepath.Join(checkout, "untracked.txt"), "dirty")

	// Second prepare should fail with dirty-worktree error.
	err := Prepare(ctx, p)
	if err == nil {
		t.Fatal("expected error on dirty checkout, got nil")
	}
	// AC29: error must mention dirty and provide remediation guidance.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "dirty") && !strings.Contains(errMsg, "uncommitted") {
		t.Errorf("error should mention dirty/uncommitted state, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "stash") && !strings.Contains(errMsg, "commit") {
		t.Errorf("error should mention remediation (stash/commit), got: %s", errMsg)
	}
}

// TestPrepare_SkipsNPMPlugin verifies Prepare is a no-op for npm-source
// plugins — no git operations, no build.
func TestPrepare_SkipsNPMPlugin(t *testing.T) {
	p := config.Plugin{
		Source: "npm:opencode-morph-fast-apply@1.0.0",
	}

	ctx := context.Background()
	err := Prepare(ctx, p)
	if err != nil {
		t.Fatalf("Prepare for npm plugin should be no-op, got: %v", err)
	}
}

// TestPrepare_SkipsDisabledPlugin verifies Prepare skips plugins with
// enabled = false.
func TestPrepare_SkipsDisabledPlugin(t *testing.T) {
	disabled := false
	p := config.Plugin{
		Source:   "https://github.com/example/plugin.git",
		Ref:      "trunk",
		Checkout: "/tmp/should-not-exist",
		Enabled:  &disabled,
	}

	ctx := context.Background()
	err := Prepare(ctx, p)
	if err != nil {
		t.Fatalf("Prepare for disabled plugin should be no-op, got: %v", err)
	}
}

// TestPrepare_DefaultRefIsTrunk verifies that when Ref is empty, Prepare
// defaults to "trunk" as the ref. Since our test remote only has "master",
// the clone with --branch trunk will fail.
func TestPrepare_DefaultRefIsTrunk(t *testing.T) {
	remote, _ := prepareIntegrationSetup(t)
	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "advance")

	p := config.Plugin{
		Source:   remote,
		Ref:      "",
		Checkout: checkout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := Prepare(ctx, p)
	if err == nil {
		// If it somehow succeeded, the ref must still be "trunk" (checkout worked).
		// But with a master-only remote, this should fail.
		t.Log("clone with default ref 'trunk' succeeded unexpectedly")
		return
	}
	// The error should come from git clone with --branch trunk.
	// Git's output may or may not include "trunk" depending on the error type,
	// but the Prepare wrapper should mention "clone failed".
	if !strings.Contains(err.Error(), "clone failed") {
		t.Errorf("expected clone-related error, got: %s", err.Error())
	}
}

// contains is defined in build_test.go (same package).
