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
	"syscall"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
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
	Path     string // NEW: working directory of the session
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
	args := []string{"-L", m.socket, "list-sessions", "-F", "#{session_name}\t#{session_attached}\t#{session_path}"}

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
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		if !strings.HasPrefix(name, sessionPrefix) {
			continue
		}
		attached := parts[1] == "1"
		path := ""
		if len(parts) >= 3 {
			path = parts[2]
		}
		sessions = append(sessions, Session{Name: name, Attached: attached, Path: path})
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

// Attach attaches to an existing tmux session, replacing the current process.
// This is a client-side operation that execs tmux directly.
func (m *Manager) Attach(ctx context.Context, name string) error {
	args := []string{"-L", m.socket, "attach", "-t", name}
	// Use syscall.Exec to replace the current process with tmux
	return syscall.Exec(m.tmuxPath, append([]string{"tmux"}, args...), os.Environ())
}

// SwitchClient switches the current tmux client to a different session.
func (m *Manager) SwitchClient(ctx context.Context, name string) error {
	args := []string{"-L", m.socket, "switch-client", "-t", name}
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		return fmt.Errorf("tmux switch-client failed: %w", err)
	}
	return nil
}

// Kill destroys a tmux session.
func (m *Manager) Kill(ctx context.Context, name string) error {
	args := []string{"-L", m.socket, "kill-session", "-t", name}
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		return fmt.Errorf("tmux kill-session failed: %w", err)
	}
	return nil
}

// KillAll destroys all OCA-managed sessions. Returns the count killed.
func (m *Manager) KillAll(ctx context.Context) (int, error) {
	sessions, err := m.List(ctx)
	if err != nil {
		return 0, err
	}

	killed := 0
	for _, s := range sessions {
		if err := m.Kill(ctx, s.Name); err != nil {
			return killed, fmt.Errorf("kill session %q: %w", s.Name, err)
		}
		killed++
	}
	return killed, nil
}

// Restart kills and recreates a session with the same name and working directory.
func (m *Manager) Restart(ctx context.Context, name, workingDir, tmuxConfPath string) error {
	// Get current session info before killing
	var currentPath string
	if s, err := m.GetSessionByName(ctx, name); err == nil && s != nil {
		currentPath = s.Path
	}

	// Use provided working dir, or fall back to current session's path
	if workingDir == "" && currentPath != "" {
		workingDir = currentPath
	}

	// Kill
	if err := m.Kill(ctx, name); err != nil {
		return fmt.Errorf("kill session %q: %w", name, err)
	}

	// Verify dead (poll with 100ms sleep, max 2s timeout)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s, err := m.GetSessionByName(ctx, name)
		if err != nil {
			return fmt.Errorf("verify session dead: %w", err)
		}
		if s == nil {
			break // Session is gone
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Create
	if err := m.Create(ctx, name, workingDir, tmuxConfPath); err != nil {
		return fmt.Errorf("recreate session %q: %w", name, err)
	}
	return nil
}

// ReapCandidate is a stale OCA session that would be reaped at the current
// threshold: the session name and how long it has been idle.
type ReapCandidate struct {
	Name string
	Age  time.Duration
}

// reapMinThreshold is the floor enforced by ReapCandidates / ReapStale to
// prevent fire-and-forget callers from accidentally nuking fresh sessions
// because of a mis-typed config or zero-value duration bug.
const reapMinThreshold = 5 * time.Minute

// ReapCandidates returns the OCA sessions that are stale at the given
// threshold but does not kill them. Used by `oca session reap --dry-run`
// for previews and by ReapStale internally for the actual reap pass.
//
// Behavior:
//   - threshold has a floor of 5 minutes; smaller values are clamped up.
//   - Returns (nil, nil) when the tmux server has no sessions or is not
//     running on the manager's socket.
//   - Sessions without the "oca-" prefix are never returned.
//   - Attached sessions are never returned.
//   - Activity is read from tmux's #{session_activity} (Unix epoch seconds).
func (m *Manager) ReapCandidates(ctx context.Context, threshold time.Duration) ([]ReapCandidate, error) {
	if threshold < reapMinThreshold {
		threshold = reapMinThreshold
	}

	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    []string{"-L", m.socket, "list-sessions", "-F", "#{session_activity}\t#{session_attached}\t#{session_name}"},
		Timeout: sessionTimeout,
	})
	if err != nil {
		output := string(res.Output)
		if strings.Contains(output, "no server running") ||
			strings.Contains(output, "no sessions") ||
			strings.Contains(output, "error connecting to") {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions failed: %w", err)
	}

	now := time.Now().Unix()
	thresholdSec := int64(threshold / time.Second)
	var candidates []ReapCandidate

	for _, line := range strings.Split(string(res.Output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		activityUnix, parseErr := strconv.ParseInt(parts[0], 10, 64)
		if parseErr != nil {
			// Defensive: malformed activity timestamp → skip rather than
			// fail the whole reap pass.
			continue
		}
		attached := parts[1] == "1"
		name := parts[2]

		if !strings.HasPrefix(name, sessionPrefix) {
			continue
		}
		if attached {
			continue
		}
		age := now - activityUnix
		if age <= thresholdSec {
			continue
		}
		candidates = append(candidates, ReapCandidate{
			Name: name,
			Age:  time.Duration(age) * time.Second,
		})
	}
	return candidates, nil
}

// ReapStale kills OCA-managed tmux sessions that have had no activity for
// longer than threshold and are not currently attached. Returns the names
// of reaped sessions. Same filter / floor / "no sessions" semantics as
// ReapCandidates — see that doc for details.
//
// Used by `oca session reap` (without --dry-run) and by the auto-reap
// goroutine triggered from `oca session new` when [session].reaper = true.
func (m *Manager) ReapStale(ctx context.Context, threshold time.Duration) ([]string, error) {
	candidates, err := m.ReapCandidates(ctx, threshold)
	if err != nil {
		return nil, err
	}
	var reaped []string
	for _, c := range candidates {
		if err := m.Kill(ctx, c.Name); err != nil {
			return reaped, fmt.Errorf("kill stale session %q: %w", c.Name, err)
		}
		reaped = append(reaped, c.Name)
	}
	return reaped, nil
}

// SetGlobalEnv sets a global environment variable in the tmux server.
// This is used to inject runtime values (like OCA_REPO_ROOT) that are
// referenced by tmux conf #() format expansions.
func (m *Manager) SetGlobalEnv(ctx context.Context, key, value string) error {
	args := []string{"-L", m.socket, "setenv", "-g", key, value}
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		return fmt.Errorf("tmux setenv -g %s failed: %w", key, err)
	}
	return nil
}

// GetSessionByName finds a single session by exact name.
func (m *Manager) GetSessionByName(ctx context.Context, name string) (*Session, error) {
	sessions, err := m.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range sessions {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, nil
}

// ApplyWatchdogEnv sets tmux global environment variables for the watchdog
// configuration so the OCA plugin can read them at startup. Only sets vars
// when a non-nil WatchdogConfig is provided; otherwise no-ops.
func (m *Manager) ApplyWatchdogEnv(ctx context.Context, wd *cfg.WatchdogConfig) error {
	if wd == nil {
		return nil
	}
	for key, value := range watchdogEnvVars(wd) {
		if err := m.SetGlobalEnv(ctx, key, value); err != nil {
			return fmt.Errorf("watchdog env %s: %w", key, err)
		}
	}
	return nil
}

// watchdogEnvVars constructs the watchdog environment variable map from a
// WatchdogConfig. Returns a map of env var names to string values suitable
// for tmux setenv -g.
func watchdogEnvVars(wd *cfg.WatchdogConfig) map[string]string {
	if wd == nil {
		return nil
	}
	enabled := "0"
	if wd.Enabled {
		enabled = "1"
	}
	timeoutMs := "0"
	if wd.IdleTimeout > 0 {
		timeoutMs = strconv.FormatInt(wd.IdleTimeout.Milliseconds(), 10)
	}
	maxBumps := strconv.Itoa(wd.MaxBumps)
	return map[string]string{
		"OCA_WATCHDOG_ENABLED":         enabled,
		"OCA_WATCHDOG_IDLE_TIMEOUT_MS": timeoutMs,
		"OCA_WATCHDOG_MAX_BUMPS":       maxBumps,
	}
}
