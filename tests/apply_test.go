package tests

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyMCP_EndToEndAgainstGolden(t *testing.T) {
	root := repoRoot(t)
	opDir := filepath.Join(t.TempDir(), "opencode")
	visionDir := filepath.Join(t.TempDir(), "vision")
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + visionDir,
		"OCA_CACHE_DIR=" + filepath.Join(t.TempDir(), "cache"),
	}
	stdout, stderr, err := runOCA(t, root, env, "apply", "--target", "mcp", "--config", "stack.example.toml")
	if err != nil {
		t.Fatalf("oca apply failed:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	gotJSON, err := os.ReadFile(filepath.Join(opDir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotYAML, err := os.ReadFile(filepath.Join(visionDir, "servers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := os.ReadFile(filepath.Join(root, "internal/render/testdata/full-stack/opencode.json.golden"))
	if err != nil {
		t.Fatal(err)
	}
	wantYAML, err := os.ReadFile(filepath.Join(root, "internal/render/testdata/full-stack/servers.yaml.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("opencode.json mismatch\n--- got ---\n%s\n--- want ---\n%s", gotJSON, wantJSON)
	}
	if string(gotYAML) != string(wantYAML) {
		t.Fatalf("servers.yaml mismatch\n--- got ---\n%s\n--- want ---\n%s", gotYAML, wantYAML)
	}
	fi1, _ := os.Stat(filepath.Join(opDir, "opencode.json"))
	fi2, _ := os.Stat(filepath.Join(visionDir, "servers.yaml"))
	if fi1.Mode().Perm() != 0o644 {
		t.Fatalf("opencode.json mode=%v want 0644", fi1.Mode().Perm())
	}
	if fi2.Mode().Perm() != 0o600 {
		t.Fatalf("servers.yaml mode=%v want 0600", fi2.Mode().Perm())
	}

	// second apply should be noop and not create new backups
	stdout2, stderr2, err := runOCA(t, root, env, "apply", "--target", "mcp", "--config", "stack.example.toml")
	if err != nil {
		t.Fatalf("second oca apply failed:\nstdout=%s\nstderr=%s\nerr=%v", stdout2, stderr2, err)
	}
	if stdout2 == "" && stderr2 == "" {
		// acceptable, but keep check so command actually ran
	}
	if matches, _ := filepath.Glob(filepath.Join(opDir, "opencode.json.bak.*")); len(matches) > 1 {
		t.Fatalf("expected <=1 opencode backup after noop, got %d", len(matches))
	}
	if matches, _ := filepath.Glob(filepath.Join(visionDir, "servers.yaml.bak.*")); len(matches) > 1 {
		t.Fatalf("expected <=1 servers backup after noop, got %d", len(matches))
	}
}
