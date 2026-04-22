package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildSessionTestBinary builds the oca binary for integration testing.
func buildSessionTestBinary(t *testing.T) string {
	t.Helper()
	repoRoot := sessionTestRepoRoot(t)
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "oca-test")
	cmd := exec.Command("go", "build", "-o", binPath, filepath.Join(repoRoot, "cmd", "oca"))
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return binPath
}

// cleanupITestSocket kills the tmux server on the given socket.
func cleanupITestSocket(t *testing.T, socket string) {
	t.Helper()
	exec.Command("tmux", "-L", socket, "kill-server").Run()
	// Give tmux a moment to clean up the socket file
	time.Sleep(50 * time.Millisecond)
}

// testSocketFor returns a unique socket name based on the test name.
// This prevents test interference when multiple tests share a tmux server.
func testSocketFor(t *testing.T) string {
	t.Helper()
	name := t.Name()
	// Sanitize: tmux socket names should be simple
	name = strings.ReplaceAll(name, "/", "_")
	return "ocait-" + name
}

func TestIntegration_SessionNew_AutoName(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	socket := testSocketFor(t)
	cleanupITestSocket(t, socket)
	defer cleanupITestSocket(t, socket)

	bin := buildSessionTestBinary(t)
	tmpDir := t.TempDir()

	cmd := exec.Command(bin, "session", "new", "--no-splash")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(),
		"OCA_TMUX_SOCKET="+socket,
		"OCA_OPENCODE_CONFIG_DIR="+filepath.Join(tmpDir, "config"),
		"OCA_CACHE_DIR="+filepath.Join(tmpDir, "cache"),
		"OCA_ASSETS_ROOT="+filepath.Join(sessionTestRepoRoot(t), "assets"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	output := string(out)
	basename := filepath.Base(tmpDir)
	expected := "oca-" + basename + "-0"
	if !strings.Contains(output, expected) {
		t.Errorf("output %q should contain %q", output, expected)
	}
}

func TestIntegration_SessionList_Text(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	socket := testSocketFor(t)
	cleanupITestSocket(t, socket)
	defer cleanupITestSocket(t, socket)

	bin := buildSessionTestBinary(t)
	tmpDir := t.TempDir()
	env := append(os.Environ(),
		"OCA_TMUX_SOCKET="+socket,
		"OCA_OPENCODE_CONFIG_DIR="+filepath.Join(tmpDir, "config"),
		"OCA_CACHE_DIR="+filepath.Join(tmpDir, "cache"),
		"OCA_ASSETS_ROOT="+filepath.Join(sessionTestRepoRoot(t), "assets"),
	)

	// Create a session first
	createCmd := exec.Command(bin, "session", "new", "--name", "oca-testlist-0", "--no-splash")
	createCmd.Dir = tmpDir
	createCmd.Env = env
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	// List sessions
	listCmd := exec.Command(bin, "session", "list", "--output", "text")
	listCmd.Dir = tmpDir
	listCmd.Env = env
	out, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session list failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "oca-testlist-0") {
		t.Errorf("list output should contain 'oca-testlist-0', got: %q", output)
	}
}

func TestIntegration_SessionList_JSON(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	socket := testSocketFor(t)
	cleanupITestSocket(t, socket)
	defer cleanupITestSocket(t, socket)

	bin := buildSessionTestBinary(t)
	tmpDir := t.TempDir()
	env := append(os.Environ(),
		"OCA_TMUX_SOCKET="+socket,
		"OCA_OPENCODE_CONFIG_DIR="+filepath.Join(tmpDir, "config"),
		"OCA_CACHE_DIR="+filepath.Join(tmpDir, "cache"),
		"OCA_ASSETS_ROOT="+filepath.Join(sessionTestRepoRoot(t), "assets"),
	)

	// Create a session
	createCmd := exec.Command(bin, "session", "new", "--name", "oca-jsonlist-0", "--no-splash")
	createCmd.Dir = tmpDir
	createCmd.Env = env
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	// List as JSON
	listCmd := exec.Command(bin, "session", "list", "--output", "json")
	listCmd.Dir = tmpDir
	listCmd.Env = env
	out, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session list failed: %v\n%s", err, out)
	}

	var sessions []struct {
		Name     string `json:"Name"`
		Attached bool   `json:"Attached"`
	}
	if err := json.Unmarshal(out, &sessions); err != nil {
		t.Fatalf("JSON parse error: %v\n%s", err, out)
	}

	found := false
	for _, s := range sessions {
		if s.Name == "oca-jsonlist-0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("JSON output should contain oca-jsonlist-0, got: %+v", sessions)
	}
}

func TestIntegration_SessionNew_CustomName(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	socket := testSocketFor(t)
	cleanupITestSocket(t, socket)
	defer cleanupITestSocket(t, socket)

	bin := buildSessionTestBinary(t)
	tmpDir := t.TempDir()
	env := append(os.Environ(),
		"OCA_TMUX_SOCKET="+socket,
		"OCA_OPENCODE_CONFIG_DIR="+filepath.Join(tmpDir, "config"),
		"OCA_CACHE_DIR="+filepath.Join(tmpDir, "cache"),
		"OCA_ASSETS_ROOT="+filepath.Join(sessionTestRepoRoot(t), "assets"),
	)

	cmd := exec.Command(bin, "session", "new", "--name", "my-custom-session", "--no-splash")
	cmd.Dir = tmpDir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), "my-custom-session") {
		t.Errorf("output should contain 'my-custom-session', got: %q", string(out))
	}
}

func TestIntegration_SessionNew_NoWritesToLiveConfig(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	socket := testSocketFor(t)
	cleanupITestSocket(t, socket)
	defer cleanupITestSocket(t, socket)

	bin := buildSessionTestBinary(t)
	tmpDir := t.TempDir()
	env := append(os.Environ(),
		"OCA_TMUX_SOCKET="+socket,
		"OCA_OPENCODE_CONFIG_DIR="+filepath.Join(tmpDir, "config"),
		"OCA_CACHE_DIR="+filepath.Join(tmpDir, "cache"),
		"OCA_ASSETS_ROOT="+filepath.Join(sessionTestRepoRoot(t), "assets"),
	)

	// Verify no live config dir exists
	liveConfig := filepath.Join(os.Getenv("HOME"), ".config", "opencode")
	beforeStat, beforeErr := os.Stat(liveConfig)

	cmd := exec.Command(bin, "session", "new", "--no-splash")
	cmd.Dir = tmpDir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	// Live config dir should not have been modified
	afterStat, afterErr := os.Stat(liveConfig)
	if beforeErr == nil && afterErr == nil {
		if !afterStat.ModTime().Equal(beforeStat.ModTime()) {
			t.Error("live config dir was modified")
		}
	}
}

func TestIntegration_SessionList_Empty(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	socket := testSocketFor(t)
	cleanupITestSocket(t, socket)
	defer cleanupITestSocket(t, socket)

	bin := buildSessionTestBinary(t)
	tmpDir := t.TempDir()
	env := append(os.Environ(),
		"OCA_TMUX_SOCKET="+socket,
		"OCA_OPENCODE_CONFIG_DIR="+filepath.Join(tmpDir, "config"),
		"OCA_CACHE_DIR="+filepath.Join(tmpDir, "cache"),
		"OCA_ASSETS_ROOT="+filepath.Join(sessionTestRepoRoot(t), "assets"),
	)

	cmd := exec.Command(bin, "session", "list")
	cmd.Dir = tmpDir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session list failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), "no OCA sessions") {
		t.Errorf("expected 'no OCA sessions', got: %q", string(out))
	}
}

// sessionTestRepoRoot returns the repository root directory for the test.
func sessionTestRepoRoot(t *testing.T) string {
	t.Helper()
	// Walk up from test file location to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

// Ensure the integration test binary is not stale.
func waitBrief() {
	// Give tmux servers time to clean up
	time.Sleep(10 * time.Millisecond)
}
