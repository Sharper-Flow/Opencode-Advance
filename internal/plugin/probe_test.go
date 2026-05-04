package plugin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestProbeDetectsUpdateWithoutMutatingRemoteTrackingRef(t *testing.T) {
	remote, work, checkout := initProbeRemoteWorkCheckout(t)
	local := gitProbe(t, checkout, "rev-parse", "HEAD")
	beforeRemoteTracking := gitProbe(t, checkout, "rev-parse", "refs/remotes/origin/master")
	addProbeCommitAndPush(t, work, "v2")
	latest := gitProbe(t, work, "rev-parse", "HEAD")
	if latest == local {
		t.Fatal("expected remote to advance")
	}

	res, err := Probe(context.Background(), "advance", config.Plugin{
		Source:   remote,
		Ref:      "master",
		Checkout: checkout,
	}, ProbeOptions{TimeoutPerPlugin: 5 * time.Second})
	if err != nil {
		t.Fatalf("Probe() = %v", err)
	}

	if res.Status != DriftUpdateAvailable {
		t.Fatalf("Status = %s, want %s (result=%+v)", res.Status, DriftUpdateAvailable, res)
	}
	if res.LocalSHA != local || res.RemoteSHA != latest {
		t.Fatalf("wrong SHAs: %+v local=%s latest=%s", res, local, latest)
	}
	afterRemoteTracking := gitProbe(t, checkout, "rev-parse", "refs/remotes/origin/master")
	if afterRemoteTracking != beforeRemoteTracking {
		t.Fatalf("Probe mutated remote-tracking ref: before=%s after=%s", beforeRemoteTracking, afterRemoteTracking)
	}
}

func TestProbePinnedSHAIsReadOnlySkip(t *testing.T) {
	_, _, checkout := initProbeRemoteWorkCheckout(t)
	sha := gitProbe(t, checkout, "rev-parse", "HEAD")

	res, err := Probe(context.Background(), "advance", config.Plugin{
		Source:   "/unused/remote",
		Ref:      sha,
		Checkout: checkout,
	}, ProbeOptions{TimeoutPerPlugin: 5 * time.Second})
	if err != nil {
		t.Fatalf("Probe() = %v", err)
	}
	if res.Status != DriftPinned {
		t.Fatalf("Status = %s, want %s", res.Status, DriftPinned)
	}
}

func TestProbeAllSkipsDisabledAndNPMPlugins(t *testing.T) {
	remote, _, checkout := initProbeRemoteWorkCheckout(t)
	disabled := false
	results, err := ProbeAll(context.Background(), config.PluginsSection{
		"git": {Source: remote, Ref: "master", Checkout: checkout},
		"npm": {Source: "npm:pkg@1.0.0"},
		"off": {Source: remote, Ref: "master", Checkout: checkout, Enabled: &disabled},
	}, nil, ProbeOptions{TimeoutPerPlugin: 5 * time.Second, TimeoutGlobal: 10 * time.Second, Parallelism: 2})
	if err != nil {
		t.Fatalf("ProbeAll() = %v", err)
	}
	if len(results) != 1 || results[0].Name != "git" {
		t.Fatalf("ProbeAll results = %+v, want only git", results)
	}
}

func initProbeRemoteWorkCheckout(t *testing.T) (remote, work, checkout string) {
	t.Helper()
	tmp := t.TempDir()
	remote = filepath.Join(tmp, "remote.git")
	work = filepath.Join(tmp, "work")
	checkout = filepath.Join(tmp, "checkout")
	gitProbe(t, tmp, "init", "--bare", "--initial-branch=master", remote)
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "index.js"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitProbe(t, work, "init", "--initial-branch=master")
	gitProbe(t, work, "config", "user.email", "test@test.com")
	gitProbe(t, work, "config", "user.name", "Test")
	gitProbe(t, work, "add", ".")
	gitProbe(t, work, "commit", "-m", "init")
	gitProbe(t, work, "remote", "add", "origin", remote)
	gitProbe(t, work, "push", "-u", "origin", "master")
	gitProbe(t, tmp, "clone", remote, checkout)
	return remote, work, checkout
}

func addProbeCommitAndPush(t *testing.T, work, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(work, "index.js"), []byte(content+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitProbe(t, work, "add", ".")
	gitProbe(t, work, "commit", "-m", content)
	gitProbe(t, work, "push", "origin", "master")
}

func gitProbe(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out))
}
