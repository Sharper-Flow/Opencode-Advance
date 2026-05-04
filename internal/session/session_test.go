package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

const testSocket = "ocatest"

// testTmuxAvailable skips the test if tmux is not installed.
func testTmuxAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
}

// cleanupSocket kills any tmux server on the test socket.
func cleanupSocket(t *testing.T, socket string) {
	t.Helper()
	subprocess.Run(context.Background(), subprocess.Cmd{
		Name: "tmux",
		Args: []string{"-L", socket, "kill-server"},
	})
}

func TestNewManager(t *testing.T) {
	testTmuxAvailable(t)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}
	if m.socket != testSocket {
		t.Errorf("socket = %q, want %q", m.socket, testSocket)
	}
	if m.tmuxPath == "" {
		t.Error("tmuxPath is empty")
	}
}

func TestNewManager_NoTmux(t *testing.T) {
	orig := execLookPath
	execLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	defer func() { execLookPath = orig }()

	_, err := NewManager("test")
	if err == nil {
		t.Fatal("expected error when tmux not found")
	}
	if !strings.Contains(err.Error(), "tmux not found") {
		t.Errorf("error = %q, want mention of 'tmux not found'", err)
	}
}

func TestProjectRoot_FromNestedGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	root := filepath.Join(t.TempDir(), "opencodeadvance")
	nested := filepath.Join(root, "cmd", "oca")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (output: %s)", err, out)
	}

	got, err := ProjectRoot(context.Background(), nested)
	if err != nil {
		t.Fatalf("ProjectRoot: %v", err)
	}
	if got != root {
		t.Errorf("ProjectRoot = %q, want %q", got, root)
	}
}

func TestProjectSessionName(t *testing.T) {
	root := filepath.Join(t.TempDir(), "OpenCode Advance")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, err := ProjectSessionName(root)
	if err != nil {
		t.Fatalf("ProjectSessionName: %v", err)
	}
	if got != "opencode-advance" {
		t.Errorf("ProjectSessionName = %q, want %q", got, "opencode-advance")
	}
}

func TestGetOrCreateProjectSession_CreatesMissingProjectSession(t *testing.T) {
	root := mustProjectRootDir(t)
	tmuxPath, logPath := fakeTmux(t, "ERR:no server running on fake", "")
	m := &Manager{socket: "fake", tmuxPath: tmuxPath}

	s, created, err := m.GetOrCreateProjectSession(context.Background(), root, "")
	if err != nil {
		t.Fatalf("GetOrCreateProjectSession: %v", err)
	}
	if !created {
		t.Fatal("created = false, want true")
	}
	if s.Name != "opencodeadvance" || s.Path != root {
		t.Fatalf("session = %+v, want name opencodeadvance path %q", s, root)
	}
	log := readFakeTmuxLog(t, logPath)
	if !strings.Contains(log, "new-session -d -s opencodeadvance -n trunk -c "+root) {
		t.Fatalf("tmux log missing project new-session command:\n%s", log)
	}
}

func TestGetOrCreateProjectSession_ReusesExistingProjectSession(t *testing.T) {
	root := mustProjectRootDir(t)
	tmuxPath, logPath := fakeTmux(t, fmt.Sprintf("opencodeadvance\t0\t%s", root), "")
	m := &Manager{socket: "fake", tmuxPath: tmuxPath}

	s, created, err := m.GetOrCreateProjectSession(context.Background(), root, "")
	if err != nil {
		t.Fatalf("GetOrCreateProjectSession: %v", err)
	}
	if created {
		t.Fatal("created = true, want false")
	}
	if s.Name != "opencodeadvance" || s.Path != root {
		t.Fatalf("session = %+v, want name opencodeadvance path %q", s, root)
	}
	log := readFakeTmuxLog(t, logPath)
	if strings.Contains(log, "new-session") {
		t.Fatalf("tmux log created duplicate session:\n%s", log)
	}
}

func TestGetOrCreateProjectSession_RecreatesStaleProjectSession(t *testing.T) {
	root := mustProjectRootDir(t)
	stalePath := filepath.Join(t.TempDir(), "missing")
	tmuxPath, logPath := fakeTmux(t, fmt.Sprintf("opencodeadvance\t0\t%s", stalePath), "")
	m := &Manager{socket: "fake", tmuxPath: tmuxPath}

	_, created, err := m.GetOrCreateProjectSession(context.Background(), root, "")
	if err != nil {
		t.Fatalf("GetOrCreateProjectSession: %v", err)
	}
	if !created {
		t.Fatal("created = false, want true after stale session recreation")
	}
	log := readFakeTmuxLog(t, logPath)
	if !strings.Contains(log, "kill-session -t opencodeadvance") {
		t.Fatalf("tmux log missing stale kill-session command:\n%s", log)
	}
	if !strings.Contains(log, "new-session -d -s opencodeadvance -n trunk -c "+root) {
		t.Fatalf("tmux log missing replacement new-session command:\n%s", log)
	}
}

func TestListWindows(t *testing.T) {
	root := mustProjectRootDir(t)
	worktree := filepath.Join(t.TempDir(), "change-one")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("MkdirAll(worktree): %v", err)
	}
	windowOutput := fmt.Sprintf("%%1\t0\ttrunk\t1\t%s\n%%2\t1\tchange-one\t0\t%s", root, worktree)
	tmuxPath, _ := fakeTmux(t, "ERR:no sessions", windowOutput)
	m := &Manager{socket: "fake", tmuxPath: tmuxPath}

	windows, err := m.ListWindows(context.Background(), "opencodeadvance")
	if err != nil {
		t.Fatalf("ListWindows: %v", err)
	}
	if len(windows) != 2 {
		t.Fatalf("len(windows) = %d, want 2: %+v", len(windows), windows)
	}
	if windows[0].ID != "%1" || windows[0].Index != 0 || windows[0].Name != "trunk" || !windows[0].Active || windows[0].Path != root {
		t.Errorf("windows[0] = %+v, want active trunk at %q", windows[0], root)
	}
	if windows[1].ID != "%2" || windows[1].Index != 1 || windows[1].Name != "change-one" || windows[1].Active || windows[1].Path != worktree {
		t.Errorf("windows[1] = %+v, want inactive change-one at %q", windows[1], worktree)
	}
}

func TestEnsureWindow_ReusesExistingWindow(t *testing.T) {
	root := mustProjectRootDir(t)
	windowOutput := fmt.Sprintf("%%1\t0\ttrunk\t1\t%s", root)
	tmuxPath, logPath := fakeTmux(t, "", windowOutput)
	m := &Manager{socket: "fake", tmuxPath: tmuxPath}

	w, created, err := m.EnsureWindow(context.Background(), "opencodeadvance", "trunk", root)
	if err != nil {
		t.Fatalf("EnsureWindow: %v", err)
	}
	if created {
		t.Fatal("created = true, want false")
	}
	if w.Name != "trunk" || w.Path != root {
		t.Fatalf("window = %+v, want trunk at %q", w, root)
	}
	log := readFakeTmuxLog(t, logPath)
	if strings.Contains(log, "new-window") {
		t.Fatalf("tmux log created duplicate window:\n%s", log)
	}
}

func TestEnsureWindow_RecreatesStaleWindowCwd(t *testing.T) {
	root := mustProjectRootDir(t)
	stalePath := filepath.Join(t.TempDir(), "missing")
	windowOutput := fmt.Sprintf("%%2\t1\tchange-one\t0\t%s", stalePath)
	tmuxPath, logPath := fakeTmux(t, "", windowOutput)
	m := &Manager{socket: "fake", tmuxPath: tmuxPath}

	_, created, err := m.EnsureWindow(context.Background(), "opencodeadvance", "change-one", root)
	if err != nil {
		t.Fatalf("EnsureWindow: %v", err)
	}
	if !created {
		t.Fatal("created = false, want true after stale window recreation")
	}
	log := readFakeTmuxLog(t, logPath)
	if !strings.Contains(log, "kill-window -t opencodeadvance:%2") {
		t.Fatalf("tmux log missing stale kill-window command:\n%s", log)
	}
	if !strings.Contains(log, "new-window -t opencodeadvance -n change-one -c "+root) {
		t.Fatalf("tmux log missing replacement new-window command:\n%s", log)
	}
}

func TestCreateAndList(t *testing.T) {
	testTmuxAvailable(t)
	cleanupSocket(t, testSocket)
	defer cleanupSocket(t, testSocket)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sessionName := "oca-test-0"
	err = m.Create(ctx, sessionName, tmpDir, "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	sessions, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	found := false
	for _, s := range sessions {
		if s.Name == sessionName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("session %q not found in list: %+v", sessionName, sessions)
	}
}

func TestCreate_NonexistentDir(t *testing.T) {
	testTmuxAvailable(t)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx := context.Background()
	err = m.Create(ctx, "test-session", "/nonexistent/path/xyz", "")
	if err == nil {
		t.Fatal("expected error for nonexistent working directory")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error = %q, want mention of 'does not exist'", err)
	}
}

func TestNextSessionName(t *testing.T) {
	testTmuxAvailable(t)
	cleanupSocket(t, testSocket)
	defer cleanupSocket(t, testSocket)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// First session should be oca-myrepo-0
	name, err := m.NextSessionName(ctx, "myrepo")
	if err != nil {
		t.Fatalf("NextSessionName() error: %v", err)
	}
	if name != "oca-myrepo-0" {
		t.Errorf("first name = %q, want %q", name, "oca-myrepo-0")
	}

	// Create it
	err = m.Create(ctx, name, tmpDir, "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Next should be oca-myrepo-1
	name, err = m.NextSessionName(ctx, "myrepo")
	if err != nil {
		t.Fatalf("NextSessionName() error: %v", err)
	}
	if name != "oca-myrepo-1" {
		t.Errorf("second name = %q, want %q", name, "oca-myrepo-1")
	}

	// Different repo should start at 0
	name, err = m.NextSessionName(ctx, "otherrepo")
	if err != nil {
		t.Fatalf("NextSessionName() error: %v", err)
	}
	if name != "oca-otherrepo-0" {
		t.Errorf("other repo name = %q, want %q", name, "oca-otherrepo-0")
	}
}

func TestList_Empty(t *testing.T) {
	testTmuxAvailable(t)
	cleanupSocket(t, testSocket)
	defer cleanupSocket(t, testSocket)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx := context.Background()
	sessions, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// TestList_FreshSocket verifies that List() handles all three tmux
// "no sessions" output variants correctly:
//  1. "no server running on <socket>" (server started but no sessions)
//  2. "no sessions"                    (alternate phrasing)
//  3. "error connecting to <socket> (No such file or directory)"
//     (socket file doesn't exist — this is the common case for a fresh socket)
//
// Regression test for a bug where the third variant was not handled.
func TestList_FreshSocket(t *testing.T) {
	testTmuxAvailable(t)

	// Use a unique socket name that's guaranteed to not exist
	freshSocket := "oca-fresh-test-xyz"
	cleanupSocket(t, freshSocket)
	defer cleanupSocket(t, freshSocket)

	m, err := NewManager(freshSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx := context.Background()
	sessions, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() on fresh socket should not error, got: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("fresh socket should have 0 sessions, got %d", len(sessions))
	}
}

func TestNoSessionsOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "no_server_running",
			output: "no server running on /tmp/tmux-1000/oca",
			want:   true,
		},
		{
			name:   "no_sessions",
			output: "no sessions",
			want:   true,
		},
		{
			name:   "error_connecting",
			output: "error connecting to /tmp/tmux-1000/oca (No such file or directory)",
			want:   true,
		},
		{
			name:   "empty_output_exit_one",
			output: "",
			want:   true,
		},
		{
			name:   "real_error",
			output: "permission denied opening /tmp/tmux-1000/oca",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNoSessionsOutput(tt.output); got != tt.want {
				t.Errorf("isNoSessionsOutput(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestCreate_WithConf(t *testing.T) {
	testTmuxAvailable(t)
	cleanupSocket(t, testSocket)
	defer cleanupSocket(t, testSocket)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Write a minimal tmux conf
	confPath := filepath.Join(tmpDir, "test.tmux.conf")
	err = os.WriteFile(confPath, []byte("set -g status off\n"), 0644)
	if err != nil {
		t.Fatalf("writing tmux conf: %v", err)
	}

	sessionName := "oca-conf-test-0"
	err = m.Create(ctx, sessionName, tmpDir, confPath)
	if err != nil {
		t.Fatalf("Create() with conf error: %v", err)
	}

	sessions, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.Name == sessionName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("session %q not found", sessionName)
	}
}

func TestParseSessionList(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []Session
	}{
		{
			name:   "empty",
			input:  "",
			expect: nil,
		},
		{
			name:   "single oca session",
			input:  "oca-repo-0\t1\n",
			expect: []Session{{Name: "oca-repo-0", Attached: true}},
		},
		{
			name:   "mixed sessions filters oca only",
			input:  "main\t1\noca-repo-0\t0\nother\t1\n",
			expect: []Session{{Name: "oca-repo-0", Attached: false}},
		},
		{
			name:   "multiple oca sessions",
			input:  "oca-repo-0\t0\noca-repo-1\t1\n",
			expect: []Session{{Name: "oca-repo-0", Attached: false}, {Name: "oca-repo-1", Attached: true}},
		},
		{
			name:   "with path field",
			input:  "oca-repo-0\t1\t/tmp/repo\n",
			expect: []Session{{Name: "oca-repo-0", Attached: true, Path: "/tmp/repo"}},
		},
		{
			name:   "multiple with path",
			input:  "oca-repo-0\t0\t/home/user/a\noca-repo-1\t1\t/home/user/b\n",
			expect: []Session{{Name: "oca-repo-0", Attached: false, Path: "/home/user/a"}, {Name: "oca-repo-1", Attached: true, Path: "/home/user/b"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSessionList(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("got %d sessions, want %d", len(got), len(tt.expect))
			}
			for i, s := range got {
				if s.Name != tt.expect[i].Name {
					t.Errorf("session[%d].Name = %q, want %q", i, s.Name, tt.expect[i].Name)
				}
				if s.Attached != tt.expect[i].Attached {
					t.Errorf("session[%d].Attached = %v, want %v", i, s.Attached, tt.expect[i].Attached)
				}
				if s.Path != tt.expect[i].Path {
					t.Errorf("session[%d].Path = %q, want %q", i, s.Path, tt.expect[i].Path)
				}
			}
		})
	}
}

func TestGetSessionByName(t *testing.T) {
	testTmuxAvailable(t)
	cleanupSocket(t, testSocket)
	defer cleanupSocket(t, testSocket)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sessionName := "oca-get-test-0"
	if err := m.Create(ctx, sessionName, tmpDir, ""); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Found
	s, err := m.GetSessionByName(ctx, sessionName)
	if err != nil {
		t.Fatalf("GetSessionByName() error: %v", err)
	}
	if s == nil {
		t.Fatal("GetSessionByName() returned nil")
	}
	if s.Name != sessionName {
		t.Errorf("Name = %q, want %q", s.Name, sessionName)
	}

	// Not found
	s, err = m.GetSessionByName(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetSessionByName(nonexistent) error: %v", err)
	}
	if s != nil {
		t.Errorf("GetSessionByName(nonexistent) = %+v, want nil", s)
	}
}

func TestKill(t *testing.T) {
	testTmuxAvailable(t)
	cleanupSocket(t, testSocket)
	defer cleanupSocket(t, testSocket)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sessionName := "oca-kill-test-0"
	if err := m.Create(ctx, sessionName, tmpDir, ""); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Verify it exists
	sessions, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.Name == sessionName {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("session not found before kill")
	}

	// Kill it
	if err := m.Kill(ctx, sessionName); err != nil {
		t.Fatalf("Kill() error: %v", err)
	}

	// Verify it's gone
	sessions, err = m.List(ctx)
	if err != nil {
		t.Fatalf("List() after kill error: %v", err)
	}
	for _, s := range sessions {
		if s.Name == sessionName {
			t.Fatal("session still exists after kill")
		}
	}
}

func TestKillAll(t *testing.T) {
	testTmuxAvailable(t)
	cleanupSocket(t, testSocket)
	defer cleanupSocket(t, testSocket)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create two sessions
	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("oca-killall-test-%d", i)
		if err := m.Create(ctx, name, tmpDir, ""); err != nil {
			t.Fatalf("Create(%s) error: %v", name, err)
		}
	}

	// Verify they exist
	sessions, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Kill all
	count, err := m.KillAll(ctx)
	if err != nil {
		t.Fatalf("KillAll() error: %v", err)
	}
	if count != 2 {
		t.Errorf("KillAll() returned %d, want 2", count)
	}

	// Verify all gone
	sessions, err = m.List(ctx)
	if err != nil {
		t.Fatalf("List() after KillAll error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after KillAll, got %d", len(sessions))
	}
}

func TestRestart(t *testing.T) {
	testTmuxAvailable(t)
	cleanupSocket(t, testSocket)
	defer cleanupSocket(t, testSocket)

	m, err := NewManager(testSocket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sessionName := "oca-restart-test-0"
	if err := m.Create(ctx, sessionName, tmpDir, ""); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Restart
	if err := m.Restart(ctx, sessionName, tmpDir, ""); err != nil {
		t.Fatalf("Restart() error: %v", err)
	}

	// Verify it still exists
	s, err := m.GetSessionByName(ctx, sessionName)
	if err != nil {
		t.Fatalf("GetSessionByName() after restart error: %v", err)
	}
	if s == nil {
		t.Fatal("session not found after restart")
	}
	if s.Name != sessionName {
		t.Errorf("Name = %q, want %q", s.Name, sessionName)
	}
}

func TestRestart_CustomNameRequiresWorkingDir(t *testing.T) {
	testTmuxAvailable(t)
	socket := fmt.Sprintf("ocatest-restart-custom-%d", os.Getpid())
	cleanupSocket(t, socket)
	t.Cleanup(func() { cleanupSocket(t, socket) })

	m, err := NewManager(socket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const customName = "custom-restart-test"
	if err := m.Create(ctx, customName, tmpDir, ""); err != nil {
		t.Fatalf("Create(custom) error: %v", err)
	}

	if err := m.Restart(ctx, customName, "", ""); err == nil {
		t.Fatal("Restart(custom, empty workingDir) error = nil, want error")
	}

	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    []string{"-L", socket, "has-session", "-t", customName},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("custom session should remain after rejected restart: %v (output: %s)", err, res.Output)
	}
}

func TestSetGlobalEnv(t *testing.T) {
	testTmuxAvailable(t)
	socket := fmt.Sprintf("ocatest-setenv-%d", os.Getpid())
	t.Cleanup(func() { cleanupSocket(t, socket) })

	m, err := NewManager(socket)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start a server by creating a session
	tmpDir := t.TempDir()
	if err := m.Create(ctx, "oca-test-env", tmpDir, ""); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Set a global env var
	if err := m.SetGlobalEnv(ctx, "OCA_TEST_VAR", "hello-world"); err != nil {
		t.Fatalf("SetGlobalEnv() error: %v", err)
	}

	// Verify via tmux showenv -g
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    []string{"-L", socket, "showenv", "-g", "OCA_TEST_VAR"},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("showenv -g failed: %v", err)
	}
	got := strings.TrimSpace(string(res.Output))
	if got != "OCA_TEST_VAR=hello-world" {
		t.Errorf("env = %q, want %q", got, "OCA_TEST_VAR=hello-world")
	}
}

// TestReapStale_NoServer verifies ReapStale returns (nil, nil) when no tmux
// server is running (one of the three "no sessions" error variants).
func TestReapStale_NoServer(t *testing.T) {
	testTmuxAvailable(t)
	socket := fmt.Sprintf("ocatest-reap-noserver-%d", os.Getpid())
	t.Cleanup(func() { cleanupSocket(t, socket) })

	m, err := NewManager(socket)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reaped, err := m.ReapStale(ctx, 4*time.Hour)
	if err != nil {
		t.Fatalf("ReapStale on empty socket: unexpected err: %v", err)
	}
	if len(reaped) != 0 {
		t.Errorf("reaped = %v, want empty", reaped)
	}
}

// TestReapStale_FloorEnforced verifies the 5-minute minimum threshold floor:
// a tiny threshold (1ns) does not reap freshly-created sessions because the
// floor pushes the effective threshold to 5m.
func TestReapStale_FloorEnforced(t *testing.T) {
	testTmuxAvailable(t)
	socket := fmt.Sprintf("ocatest-reap-floor-%d", os.Getpid())
	cleanupSocket(t, socket)
	t.Cleanup(func() { cleanupSocket(t, socket) })

	m, err := NewManager(socket)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	// Create an oca- session, leave it detached.
	if err := m.Create(ctx, "oca-floor-0", tmpDir, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// 1ns threshold would reap everything except for the 5m floor.
	reaped, err := m.ReapStale(ctx, 1*time.Nanosecond)
	if err != nil {
		t.Fatalf("ReapStale: %v", err)
	}
	if len(reaped) != 0 {
		t.Errorf("reaped = %v, want empty (5m floor should protect fresh sessions)", reaped)
	}

	// Verify the session is still alive.
	s, err := m.GetSessionByName(ctx, "oca-floor-0")
	if err != nil {
		t.Fatalf("GetSessionByName: %v", err)
	}
	if s == nil {
		t.Error("session was reaped despite 5m floor; floor not enforced")
	}
}

// TestReapStale_NonOcaPrefixSkipped verifies sessions without the oca- prefix
// are never reaped, even if they would otherwise be old enough.
func TestReapStale_NonOcaPrefixSkipped(t *testing.T) {
	testTmuxAvailable(t)
	socket := fmt.Sprintf("ocatest-reap-prefix-%d", os.Getpid())
	cleanupSocket(t, socket)
	t.Cleanup(func() { cleanupSocket(t, socket) })

	m, err := NewManager(socket)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create a non-oca-prefix session directly via tmux.
	tmpDir := t.TempDir()
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    []string{"-L", socket, "new-session", "-d", "-s", "user-other", "-c", tmpDir},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new-session: %v (output: %s)", err, res.Output)
	}

	// Even with no floor concern (4h threshold), the prefix filter must skip it.
	reaped, err := m.ReapStale(ctx, 4*time.Hour)
	if err != nil {
		t.Fatalf("ReapStale: %v", err)
	}
	for _, name := range reaped {
		if name == "user-other" {
			t.Error("ReapStale killed non-oca-prefix session 'user-other' — prefix filter broken")
		}
	}
}

func mustProjectRootDir(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "opencodeadvance")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(root): %v", err)
	}
	return root
}

func fakeTmux(t *testing.T, sessionOutput, windowOutput string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "tmux.log")
	scriptPath := filepath.Join(dir, "tmux")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %s
cmd=""
for arg in "$@"; do
  case "$arg" in
    list-sessions|new-session|kill-session|list-windows|new-window|kill-window|has-session|setenv|showenv)
      cmd="$arg"
      break
      ;;
  esac
done
case "$cmd" in
  list-sessions)
%s
    ;;
  list-windows)
%s
    ;;
  new-session|kill-session|new-window|kill-window|has-session|setenv|showenv)
    exit 0
    ;;
  *)
    echo "unexpected tmux args: $*" >&2
    exit 2
    ;;
esac
`, shellQuote(logPath), fakeTmuxOutputClause(sessionOutput), fakeTmuxOutputClause(windowOutput))
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(fake tmux): %v", err)
	}
	return scriptPath, logPath
}

func fakeTmuxOutputClause(output string) string {
	exitCode := 0
	stream := "cat <<'EOF'"
	if strings.HasPrefix(output, "ERR:") {
		exitCode = 1
		stream = "cat >&2 <<'EOF'"
		output = strings.TrimPrefix(output, "ERR:")
	}
	return fmt.Sprintf("    %s\n%s\nEOF\n    exit %d", stream, output, exitCode)
}

func readFakeTmuxLog(t *testing.T, logPath string) string {
	t.Helper()
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(fake tmux log): %v", err)
	}
	return string(b)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
