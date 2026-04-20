package health

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestCheckPlugins_HappyPathAndUserAddedDrift(t *testing.T) {
	opencodeDir := filepath.Join(t.TempDir(), "opencode")
	if err := os.MkdirAll(opencodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", opencodeDir)

	checkout, pluginPath := initPluginRepo(t)
	if err := os.WriteFile(filepath.Join(opencodeDir, "opencode.json"), []byte(`{"plugin":["`+pluginPath+`","user/extra@1.0.0"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{Plugins: cfg.PluginsSection{
		"advance": {Source: checkout, Checkout: checkout, Ref: "master", Path: pluginPath},
	}}

	checks, err := CheckPlugins(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckPlugins: %v", err)
	}
	assertCheckStatus(t, checks, "plugins.advance.checkout_exists", StatusPass)
	assertCheckStatus(t, checks, "plugins.advance.built_artifact", StatusPass)
	assertCheckStatus(t, checks, "plugins.advance.git_ref", StatusPass)
	assertCheckStatus(t, checks, "plugins.advance.path_in_opencode_json", StatusPass)
	assertCheckStatus(t, checks, "plugins.drift.declared_and_present.advance", StatusPass)
	assertCheckStatus(t, checks, "plugins.drift.user_added.user/extra@1.0.0", StatusPass)
}

func TestCheckPlugins_ProvidesPatchedInstructionsInfo(t *testing.T) {
	opencodeDir := filepath.Join(t.TempDir(), "opencode")
	if err := os.MkdirAll(opencodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", opencodeDir)
	if err := os.WriteFile(filepath.Join(opencodeDir, "opencode.json"), []byte(`{"instructions":["identity.md"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	stack := &cfg.Stack{Plugins: cfg.PluginsSection{
		"advance": {
			Provides:     []cfg.ProvidesCategory{cfg.ProvidesInstructions},
			Instructions: []string{"/plugins/advance/ADV_INSTRUCTIONS.md"},
		},
	}}

	checks, err := CheckPlugins(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckPlugins: %v", err)
	}
	assertCheckStatus(t, checks, "plugins.drift.provides_patched.advance", StatusPass)
}

func TestCheckPlugins_BuiltinRegistryPresentAfterReset(t *testing.T) {
	ResetForTesting()
	t.Cleanup(ResetForTesting)
	if !containsScope(KnownScopes(), "plugins") {
		t.Fatalf("expected builtin plugins scope, got %#v", KnownScopes())
	}
}

func initPluginRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	pluginPath := filepath.Join(repo, "dist", "index.js")
	if err := os.MkdirAll(filepath.Dir(pluginPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pluginPath, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCmd(t, repo, "git", "init", "--initial-branch=master")
	runCmd(t, repo, "git", "config", "user.email", "test@test.com")
	runCmd(t, repo, "git", "config", "user.name", "Test")
	runCmd(t, repo, "git", "add", ".")
	runCmd(t, repo, "git", "commit", "-m", "init")
	return repo, pluginPath
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(out))
	}
}

func assertCheckStatus(t *testing.T, checks []Check, name string, want Status) {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			if check.Status != want {
				t.Fatalf("check %s status=%s want %s (%s)", name, check.Status, want, check.Message)
			}
			return
		}
	}
	t.Fatalf("missing check %s in %#v", name, checks)
}
