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
//   1. "no server running on <socket>" (server started but no sessions)
//   2. "no sessions"                    (alternate phrasing)
//   3. "error connecting to <socket> (No such file or directory)"
//      (socket file doesn't exist — this is the common case for a fresh socket)
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
