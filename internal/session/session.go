// Package session provides tmux session lifecycle operations for OpenCode
// Advance. All external commands go through internal/subprocess.Run.
package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

const (
	// sessionTimeout is the default wall-clock budget for tmux operations.
	sessionTimeout = 10 * time.Second

	// sessionPrefix is the namespace prefix for all OCA-managed sessions.
	sessionPrefix = "oca-"
)

// sessionNamePattern matches OCA-managed session names: oca-<slug>-<n>
var sessionNamePattern = regexp.MustCompile(`^oca-([a-zA-Z0-9_-]+)-(\d+)$`)

// Session represents a tmux session managed by OCA.
type Session struct {
	Name     string
	Attached bool
}

// Manager manages OCA tmux sessions on a specific socket.
type Manager struct {
	socket   string
	tmuxPath string
}

// NewManager creates a session manager for the given tmux socket name.
// Validates that tmux is available on PATH.
func NewManager(socket string) (*Manager, error) {
	tmuxPath, err := findTmux()
	if err != nil {
		return nil, err
	}
	return &Manager{socket: socket, tmuxPath: tmuxPath}, nil
}

// Create creates a new detached tmux session.
// The -f flag is a top-level tmux flag (before the subcommand).
// Pre-validates that workingDir exists.
func (m *Manager) Create(ctx context.Context, name, workingDir, tmuxConfPath string) error {
	// Validate working directory exists
	info, err := os.Stat(workingDir)
	if err != nil {
		return fmt.Errorf("working directory %q does not exist: %w", workingDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("working directory %q is not a directory", workingDir)
	}

	// Build args: tmux -L <socket> [-f <conf>] new-session -d -s <name> -c <dir>
	args := []string{"-L", m.socket}
	if tmuxConfPath != "" {
		args = append(args, "-f", tmuxConfPath)
	}
	args = append(args, "new-session", "-d", "-s", name, "-c", workingDir)

	_, err = subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		return fmt.Errorf("tmux new-session failed: %w", err)
	}
	return nil
}

// List returns all OCA-managed sessions on the manager's socket.
// Filters by the "oca-" prefix.
func (m *Manager) List(ctx context.Context) ([]Session, error) {
	args := []string{"-L", m.socket, "list-sessions", "-F", "#{session_name}\t#{session_attached}"}

	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		// tmux returns exit code 1 in three "no sessions" cases:
		//   1. "no server running on <socket>"         — server running but no sessions
		//   2. "no sessions"                           — alternate phrasing
		//   3. "error connecting to <socket> (No such file or directory)"
		//       — socket file doesn't exist yet (fresh socket, never started)
		output := string(res.Output)
		if strings.Contains(output, "no server running") ||
			strings.Contains(output, "no sessions") ||
			strings.Contains(output, "error connecting to") {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions failed: %w", err)
	}

	return parseSessionList(string(res.Output)), nil
}

// NextSessionName returns the next sequential session name for the given repo slug.
// Pattern: oca-<slug>-<n> where n is the next available integer starting from 0.
func (m *Manager) NextSessionName(ctx context.Context, repoSlug string) (string, error) {
	sessions, err := m.List(ctx)
	if err != nil {
		return "", err
	}

	maxN := -1
	prefix := sessionPrefix + repoSlug + "-"
	for _, s := range sessions {
		if !strings.HasPrefix(s.Name, prefix) {
			continue
		}
		numStr := strings.TrimPrefix(s.Name, prefix)
		n, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		if n > maxN {
			maxN = n
		}
	}

	return fmt.Sprintf("%s%d", prefix, maxN+1), nil
}

// parseSessionList parses tmux list-sessions output into Session structs.
func parseSessionList(output string) []Session {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil
	}

	var sessions []Session
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		if !strings.HasPrefix(name, sessionPrefix) {
			continue
		}
		attached := parts[1] == "1"
		sessions = append(sessions, Session{Name: name, Attached: attached})
	}
	return sessions
}

// findTmux locates the tmux binary on PATH.
func findTmux() (string, error) {
	path, err := execLookPath("tmux")
	if err != nil {
		return "", fmt.Errorf("tmux not found on PATH: %w", err)
	}
	return path, nil
}

// execLookPath is extracted for testability.
var execLookPath = defaultLookPath

func defaultLookPath(file string) (string, error) {
	return exec.LookPath(file)
}
