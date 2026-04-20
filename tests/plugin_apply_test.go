package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPluginsAndInstructions_EndToEndFromStackExampleFixture(t *testing.T) {
	root := repoRoot(t)
	base := t.TempDir()
	opDir := filepath.Join(base, "opencode")
	visionDir := filepath.Join(base, "vision")
	cacheDir := filepath.Join(base, "cache")
	if err := os.MkdirAll(opDir, 0o755); err != nil {
		t.Fatal(err)
	}

	advanceRemote := initPluginRemote(t, filepath.Join(base, "advance-src"), "advance")
	morphRemote := initPluginRemote(t, filepath.Join(base, "morph-src"), "root")
	visionRemote := initPluginRemote(t, filepath.Join(base, "vision-src"), "vision-subdir")

	fixtureBytes, err := os.ReadFile(filepath.Join(root, "stack.example.toml"))
	if err != nil {
		t.Fatal(err)
	}
	fixture := string(fixtureBytes)
	fixture = strings.ReplaceAll(fixture, "https://github.com/Sharper-Flow/Advance.git", advanceRemote)
	fixture = strings.ReplaceAll(fixture, "~/dev/oc-plugins/advance", filepath.Join(base, "checkouts", "advance"))
	fixture = strings.ReplaceAll(fixture, "build        = [\"pnpm install\", \"pnpm build\"]", "build        = []")
	fixture = strings.ReplaceAll(fixture, "build    = [\"pnpm install\", \"pnpm build\"]", "build    = []")
	fixture = strings.ReplaceAll(fixture, "provides     = [\"adv-commands\", \"adv-agents\", \"adv-skills\", \"adv-overlays\"]", "provides     = [\"adv-commands\", \"adv-agents\", \"adv-skills\", \"adv-overlays\", \"adv-instructions\"]")
	fixture = strings.ReplaceAll(fixture, "https://github.com/JRedeker/opencode-morph-fast-apply.git", morphRemote)
	fixture = strings.ReplaceAll(fixture, "~/dev/oc-plugins/morph-fast-apply", filepath.Join(base, "checkouts", "morph-fast-apply"))
	fixture = strings.ReplaceAll(fixture, "https://github.com/Sharper-Flow/vision.git", visionRemote)
	fixture = strings.ReplaceAll(fixture, "~/dev/vision", filepath.Join(base, "checkouts", "vision"))
	fixture = strings.ReplaceAll(fixture, `ref          = "trunk"`, `ref          = "master"`)
	fixture = strings.ReplaceAll(fixture, `ref      = "trunk"`, `ref      = "master"`)

	stackPath := filepath.Join(base, "stack.example.toml")
	if err := os.WriteFile(stackPath, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}

	stale := filepath.Join(base, "xdg-data", "opencode", "worktree", "abc123", "change", "old", "plugin.js")
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "xdg-data"))
	seed := map[string]any{
		"plugin":       []any{stale, "user/custom-plugin"},
		"instructions": []any{"user-extra.md"},
	}
	seedBytes, _ := json.Marshal(seed)
	if err := os.WriteFile(filepath.Join(opDir, "opencode.json"), seedBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opDir,
		"OCA_VISION_CONFIG_DIR=" + visionDir,
		"OCA_CACHE_DIR=" + cacheDir,
	}
	stdout, stderr, err := runOCA(t, root, env, "apply", "--target", "plugins", "--target", "instructions", "--config", stackPath)
	if err != nil {
		t.Fatalf("oca apply plugins+instructions failed:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}

	gotJSON, err := os.ReadFile(filepath.Join(opDir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rootJSON map[string]any
	if err := json.Unmarshal(gotJSON, &rootJSON); err != nil {
		t.Fatal(err)
	}

	plugins := mustStringArray(t, rootJSON["plugin"])
	if containsString(plugins, stale) {
		t.Fatalf("stale worktree plugin entry not pruned: %#v", plugins)
	}
	if !containsString(plugins, "user/custom-plugin") {
		t.Fatalf("user-added plugin not preserved: %#v", plugins)
	}
	if !containsAnyContaining(plugins, filepath.Join(base, "checkouts", "advance", "plugin")) {
		t.Fatalf("advance plugin path missing: %#v", plugins)
	}

	instructions := mustStringArray(t, rootJSON["instructions"])
	if !containsString(instructions, "user-extra.md") {
		t.Fatalf("user-added instruction not preserved: %#v", instructions)
	}
	for _, instruction := range instructions {
		if strings.HasSuffix(instruction, "ADV_INSTRUCTIONS.md") {
			t.Fatalf("ADV instructions should be absent from OCA render when provides-patched: %#v", instructions)
		}
	}
	if !strings.Contains(stdout, "prun") {
		t.Fatalf("expected prune summary in stdout, got: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(base, "checkouts", "advance", "sync-ran.txt")); err != nil {
		t.Fatalf("advance sync hook did not run: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
}

func initPluginRemote(t *testing.T, path string, layout string) string {
	t.Helper()
	repo := filepath.Join(path, "repo")
	remote := filepath.Join(path, "remote.git")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	switch layout {
	case "advance":
		if err := os.MkdirAll(filepath.Join(repo, "plugin"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(repo, "scripts"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFileAt(t, filepath.Join(repo, "plugin", "index.js"), "export default {}\n")
		writeFileAt(t, filepath.Join(repo, "ADV_INSTRUCTIONS.md"), "adv\n")
		writeFileAt(t, filepath.Join(repo, "scripts", "sync-global.sh"), "#!/bin/sh\npwd > sync-ran.txt\n")
		runTestCmd(t, repo, "chmod", "+x", filepath.Join(repo, "scripts", "sync-global.sh"))
	case "vision-subdir":
		if err := os.MkdirAll(filepath.Join(repo, "opencode-plugin"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFileAt(t, filepath.Join(repo, "opencode-plugin", "index.js"), "export default {}\n")
	default:
		writeFileAt(t, filepath.Join(repo, "index.js"), "export default {}\n")
	}
	runTestCmd(t, repo, "git", "init", "--initial-branch=master")
	runTestCmd(t, repo, "git", "config", "user.email", "test@test.com")
	runTestCmd(t, repo, "git", "config", "user.name", "Test")
	runTestCmd(t, repo, "git", "add", ".")
	runTestCmd(t, repo, "git", "commit", "-m", "init")
	runTestCmd(t, path, "git", "init", "--bare", "--initial-branch=master", remote)
	runTestCmd(t, repo, "git", "remote", "add", "origin", remote)
	runTestCmd(t, repo, "git", "push", "-u", "origin", "master")
	return remote
}

func runTestCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(out))
	}
}

func writeFileAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustStringArray(t *testing.T, raw any) []string {
	t.Helper()
	arr, ok := raw.([]any)
	if !ok {
		t.Fatalf("not array: %#v", raw)
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("non-string: %#v", item)
		}
		out = append(out, s)
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsAnyContaining(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}
