package tests

import (
	"os"
	"path/filepath"
	"strings"
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

func TestApplyMCP_WithSlotGroups(t *testing.T) {
	root := repoRoot(t)
	tmpDir := t.TempDir()
	stackPath := filepath.Join(tmpDir, "stack.toml")
	stackTOML := `[meta]
version = "1.0.0"
name = "slot-group-test"

[mcp.servers.context7]
port = 6276
command = "npx"
source = "https://github.com/upstash/context7"

[mcp.slot_groups.playwright-headless]
template = "playwright-headless"
base_port = 6301
count = 4
group_port = 6300

[mcp.slot_groups.playwright-headless.defaults]
command = "npx"
args = ["playwright", "headless"]
`
	if err := os.WriteFile(stackPath, []byte(stackTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	opDir := filepath.Join(tmpDir, "opencode")
	visionDir := filepath.Join(tmpDir, "vision")
	if err := os.MkdirAll(opDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(visionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-seed opencode.json with a user-added .mcp entry to verify preservation
	userMCP := `{"mcp":{"my-custom-server":{"type":"remote","url":"http://localhost:9999/mcp"}}}`
	if err := os.WriteFile(filepath.Join(opDir, "opencode.json"), []byte(userMCP), 0o644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + visionDir,
		"OCA_CACHE_DIR=" + filepath.Join(tmpDir, "cache"),
	}
	stdout, stderr, err := runOCA(t, root, env, "apply", "--target", "mcp", "--config", stackPath)
	if err != nil {
		t.Fatalf("oca apply failed:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}

	gotYAML, err := os.ReadFile(filepath.Join(visionDir, "servers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(gotYAML)
	if !strings.Contains(yamlStr, "servers:") {
		t.Fatalf("servers: missing from vision/servers.yaml")
	}
	if !strings.Contains(yamlStr, "slot_groups:") {
		t.Fatalf("slot_groups: missing from vision/servers.yaml")
	}
	if !strings.Contains(yamlStr, "playwright-headless:") {
		t.Fatalf("playwright-headless group missing from vision/servers.yaml")
	}
	if !strings.Contains(yamlStr, "base_port: 6301") {
		t.Fatalf("base_port missing from vision/servers.yaml")
	}

	gotJSON, err := os.ReadFile(filepath.Join(opDir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	jsonStr := string(gotJSON)
	if !strings.Contains(jsonStr, `"playwright-headless"`) {
		t.Fatalf("playwright-headless missing from opencode.json .mcp")
	}
	if !strings.Contains(jsonStr, `"url":"http://localhost:6300/mcp"`) && !strings.Contains(jsonStr, `"url": "http://localhost:6300/mcp"`) {
		t.Fatalf("group_port URL missing from opencode.json .mcp: %s", jsonStr)
	}
	// User-added key preserved
	if !strings.Contains(jsonStr, `"my-custom-server"`) {
		t.Fatalf("user-added .mcp key my-custom-server was not preserved: %s", jsonStr)
	}
}
