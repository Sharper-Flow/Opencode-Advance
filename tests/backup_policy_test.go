package tests

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupPolicy_PluginSyncSuppressesBakButPlainPluginKeepsIt(t *testing.T) {
	root := repoRoot(t)

	t.Run("sync plugin suppresses backup", func(t *testing.T) {
		base := t.TempDir()
		opDir := filepath.Join(base, "opencode")
		visionDir := filepath.Join(base, "vision")
		cacheDir := filepath.Join(base, "cache")
		if err := os.MkdirAll(opDir, 0o755); err != nil {
			t.Fatal(err)
		}
		seed := []byte(`{"plugin":["old-plugin"]}`)
		if err := os.WriteFile(filepath.Join(opDir, "opencode.json"), seed, 0o644); err != nil {
			t.Fatal(err)
		}

		advanceRemote := initPluginRemote(t, filepath.Join(base, "advance-src"), "advance")
		stackPath := filepath.Join(base, "stack.toml")
		writeFileAt(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.advance]\nsource = \""+advanceRemote+"\"\nref = \"master\"\ncheckout = \""+filepath.Join(base, "checkouts", "advance")+"\"\nsubdir = \"plugin\"\npath = \"{checkout}/{subdir}\"\nsync = \"{checkout}/scripts/sync-global.sh --fix\"\n")

		env := []string{
			"OCA_OPENCODE_CONFIG_DIR=" + opDir,
			"OCA_VISION_CONFIG_DIR=" + visionDir,
			"OCA_CACHE_DIR=" + cacheDir,
		}
		stdout, stderr, err := runOCA(t, root, env, "apply", "--target", "plugins", "--config", stackPath)
		if err != nil {
			t.Fatalf("apply plugins failed: stdout=%s stderr=%s err=%v", stdout, stderr, err)
		}
		matches, err := filepath.Glob(filepath.Join(opDir, "opencode.json.bak.*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("expected no backups for sync plugin apply, got %v", matches)
		}
	})

	t.Run("plain plugin keeps normal backup", func(t *testing.T) {
		base := t.TempDir()
		opDir := filepath.Join(base, "opencode")
		visionDir := filepath.Join(base, "vision")
		cacheDir := filepath.Join(base, "cache")
		if err := os.MkdirAll(opDir, 0o755); err != nil {
			t.Fatal(err)
		}
		seed := []byte(`{"plugin":["old-plugin"]}`)
		if err := os.WriteFile(filepath.Join(opDir, "opencode.json"), seed, 0o644); err != nil {
			t.Fatal(err)
		}

		morphRemote := initPluginRemote(t, filepath.Join(base, "morph-src"), "root")
		stackPath := filepath.Join(base, "stack.toml")
		writeFileAt(t, stackPath, "[meta]\nversion = \"1.0.0\"\n\n[plugins.morph]\nsource = \""+morphRemote+"\"\nref = \"master\"\ncheckout = \""+filepath.Join(base, "checkouts", "morph")+"\"\npath = \"{checkout}\"\n")

		env := []string{
			"OCA_OPENCODE_CONFIG_DIR=" + opDir,
			"OCA_VISION_CONFIG_DIR=" + visionDir,
			"OCA_CACHE_DIR=" + cacheDir,
		}
		stdout, stderr, err := runOCA(t, root, env, "apply", "--target", "plugins", "--config", stackPath)
		if err != nil {
			t.Fatalf("apply plugins failed: stdout=%s stderr=%s err=%v", stdout, stderr, err)
		}
		matches, err := filepath.Glob(filepath.Join(opDir, "opencode.json.bak.*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) == 0 {
			t.Fatal("expected backup for non-sync plugin apply")
		}
	})
}
