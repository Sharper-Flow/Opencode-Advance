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

func TestIntegration_SessionKill(t *testing.T) {
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
	createCmd := exec.Command(bin, "session", "new", "--name", "oca-testkill-0", "--no-splash")
	createCmd.Dir = tmpDir
	createCmd.Env = env
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	// Kill the session
	killCmd := exec.Command(bin, "session", "kill", "oca-testkill-0")
	killCmd.Dir = tmpDir
	killCmd.Env = env
	if out, err := killCmd.CombinedOutput(); err != nil {
		t.Fatalf("session kill failed: %v\n%s", err, out)
	}

	// Verify it's gone
	listCmd := exec.Command(bin, "session", "list")
	listCmd.Dir = tmpDir
	listCmd.Env = env
	out, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session list failed: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "oca-testkill-0") {
		t.Errorf("killed session should not appear in list, got: %q", string(out))
	}
}

func TestIntegration_SessionKill_CustomName(t *testing.T) {
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

	const customName = "my-custom-kill-session"
	createCmd := exec.Command(bin, "session", "new", "--name", customName, "--no-splash")
	createCmd.Dir = tmpDir
	createCmd.Env = env
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("session new custom failed: %v\n%s", err, out)
	}

	// Custom names do not appear in `oca session list`, but exact-name
	// commands still target them on the dedicated OCA socket.
	hasCmd := exec.Command("tmux", "-L", socket, "has-session", "-t", customName)
	if out, err := hasCmd.CombinedOutput(); err != nil {
		t.Fatalf("custom tmux session should exist: %v\n%s", err, out)
	}

	killCmd := exec.Command(bin, "session", "kill", customName)
	killCmd.Dir = tmpDir
	killCmd.Env = env
	if out, err := killCmd.CombinedOutput(); err != nil {
		t.Fatalf("session kill custom failed: %v\n%s", err, out)
	}

	hasCmd = exec.Command("tmux", "-L", socket, "has-session", "-t", customName)
	if err := hasCmd.Run(); err == nil {
		t.Fatal("custom tmux session still exists after exact-name kill")
	}
}

func TestIntegration_SessionKillall(t *testing.T) {
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

	// Create two sessions
	for _, name := range []string{"oca-testkillall-0", "oca-testkillall-1"} {
		createCmd := exec.Command(bin, "session", "new", "--name", name, "--no-splash")
		createCmd.Dir = tmpDir
		createCmd.Env = env
		if out, err := createCmd.CombinedOutput(); err != nil {
			t.Fatalf("session new %s failed: %v\n%s", name, err, out)
		}
	}

	// Kill all sessions
	killallCmd := exec.Command(bin, "session", "killall")
	killallCmd.Dir = tmpDir
	killallCmd.Env = env
	if out, err := killallCmd.CombinedOutput(); err != nil {
		t.Fatalf("session killall failed: %v\n%s", err, out)
	}

	// Verify list is empty
	listCmd := exec.Command(bin, "session", "list")
	listCmd.Dir = tmpDir
	listCmd.Env = env
	out, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session list failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "no OCA sessions") {
		t.Errorf("expected 'no OCA sessions' after killall, got: %q", string(out))
	}
}

func TestIntegration_SessionRestart(t *testing.T) {
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
	createCmd := exec.Command(bin, "session", "new", "--name", "oca-testrestart-0", "--no-splash")
	createCmd.Dir = tmpDir
	createCmd.Env = env
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	// Restart it
	restartCmd := exec.Command(bin, "session", "restart", "oca-testrestart-0")
	restartCmd.Dir = tmpDir
	restartCmd.Env = env
	if out, err := restartCmd.CombinedOutput(); err != nil {
		t.Fatalf("session restart failed: %v\n%s", err, out)
	}

	// Verify it still exists
	listCmd := exec.Command(bin, "session", "list")
	listCmd.Dir = tmpDir
	listCmd.Env = env
	out, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session list failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "oca-testrestart-0") {
		t.Errorf("restarted session should appear in list, got: %q", string(out))
	}
}

func TestIntegration_SessionReap(t *testing.T) {
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
	createCmd := exec.Command(bin, "session", "new", "--name", "oca-testreap-0", "--no-splash")
	createCmd.Dir = tmpDir
	createCmd.Env = env
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	// Dry-run reap on fresh session should not error (fresh sessions are below min age)
	reapCmd := exec.Command(bin, "session", "reap", "--dry-run")
	reapCmd.Dir = tmpDir
	reapCmd.Env = env
	out, err := reapCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session reap --dry-run failed: %v\n%s", err, out)
	}

	// Fresh sessions won't be reaped due to 5m minimum age; just verify no crash.
	_ = string(out)

	// Actual reap on fresh session should also not error
	reapRealCmd := exec.Command(bin, "session", "reap")
	reapRealCmd.Dir = tmpDir
	reapRealCmd.Env = env
	if out, err := reapRealCmd.CombinedOutput(); err != nil {
		t.Fatalf("session reap failed: %v\n%s", err, out)
	}

	// Session should still exist since it's fresh
	listCmd := exec.Command(bin, "session", "list")
	listCmd.Dir = tmpDir
	listCmd.Env = env
	out, err = listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("session list failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "oca-testreap-0") {
		t.Errorf("fresh session should not be reaped, got: %q", string(out))
	}
}

func TestIntegration_SessionNew_SetsRepoRoot(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	socket := testSocketFor(t)
	cleanupITestSocket(t, socket)
	defer cleanupITestSocket(t, socket)

	bin := buildSessionTestBinary(t)
	tmpDir := t.TempDir()
	repoRoot := sessionTestRepoRoot(t)
	env := append(os.Environ(),
		"OCA_TMUX_SOCKET="+socket,
		"OCA_OPENCODE_CONFIG_DIR="+filepath.Join(tmpDir, "config"),
		"OCA_CACHE_DIR="+filepath.Join(tmpDir, "cache"),
		"OCA_ASSETS_ROOT="+filepath.Join(repoRoot, "assets"),
	)

	// Create session from repo root so mustGetRepoRoot() finds lib/boot_splash.sh
	createCmd := exec.Command(bin, "session", "new", "--name", "oca-testroot-0", "--no-splash")
	createCmd.Dir = repoRoot
	createCmd.Env = env
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("session new failed: %v\n%s", err, out)
	}

	// Read OCA_REPO_ROOT from tmux global environment
	showenvCmd := exec.Command("tmux", "-L", socket, "showenv", "-g", "OCA_REPO_ROOT")
	showenvCmd.Dir = repoRoot
	out, err := showenvCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tmux showenv failed: %v\n%s", err, out)
	}

	output := string(out)
	// tmux showenv outputs: OCA_REPO_ROOT=/path/to/repo
	if !strings.Contains(output, repoRoot) {
		t.Errorf("OCA_REPO_ROOT should contain %q, got: %q", repoRoot, output)
	}
}
