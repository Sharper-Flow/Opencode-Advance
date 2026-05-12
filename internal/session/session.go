// Package session provides tmux session lifecycle operations for OpenCode
// Advance. All external commands go through internal/subprocess.Run.
package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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

var projectSlugInvalidChars = regexp.MustCompile(`[^a-z0-9_-]+`)

// Session represents a tmux session managed by OCA.
type Session struct {
	Name     string
	Attached bool
	Path     string // working directory of the session
}

// Window represents a tmux window inside an OCA project session.
type Window struct {
	ID     string
	Index  int
	Name   string
	Active bool
	Path   string // current pane working directory for the window
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

// ProjectRoot resolves the git project root for workingDir.
func ProjectRoot(ctx context.Context, workingDir string) (string, error) {
	if workingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get current directory: %w", err)
		}
		workingDir = wd
	}
	info, err := os.Stat(workingDir)
	if err != nil {
		return "", fmt.Errorf("working directory %q does not exist: %w", workingDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working directory %q is not a directory", workingDir)
	}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git not found on PATH: %w", err)
	}
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    gitPath,
		Args:    []string{"-C", workingDir, "rev-parse", "--show-toplevel"},
		Timeout: sessionTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("resolve git project root for %q: %w", workingDir, err)
	}
	root := strings.TrimSpace(string(res.Output))
	if root == "" {
		return "", fmt.Errorf("resolve git project root for %q: empty git output", workingDir)
	}
	return filepath.Clean(root), nil
}

// ProjectSessionName returns the Pattern B project tmux session name for root.
// Pattern B uses the git repo slug directly: no "oca-" prefix and no numeric
// per-invocation suffix.
func ProjectSessionName(projectRoot string) (string, error) {
	if projectRoot == "" {
		return "", fmt.Errorf("project root is required")
	}
	slug := strings.ToLower(filepath.Base(filepath.Clean(projectRoot)))
	slug = projectSlugInvalidChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "", fmt.Errorf("project root %q does not produce a valid session name", projectRoot)
	}
	return slug, nil
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

// GetOrCreateProjectSession returns the Pattern B session for projectRoot,
// creating it when missing. Existing sessions with a stale/nonexistent cwd are
// killed and recreated at projectRoot with a "trunk" window.
func (m *Manager) GetOrCreateProjectSession(ctx context.Context, projectRoot, tmuxConfPath string) (*Session, bool, error) {
	if err := validateDir(projectRoot, "project root"); err != nil {
		return nil, false, err
	}
	name, err := ProjectSessionName(projectRoot)
	if err != nil {
		return nil, false, err
	}

	existing, err := m.getRawSessionByName(ctx, name)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		if dirExists(existing.Path) {
			return existing, false, nil
		}
		if err := m.Kill(ctx, name); err != nil {
			return nil, false, fmt.Errorf("kill stale project session %q: %w", name, err)
		}
	}

	if err := m.createProjectSession(ctx, name, projectRoot, tmuxConfPath); err != nil {
		return nil, false, err
	}
	return &Session{Name: name, Attached: false, Path: projectRoot}, true, nil
}

func (m *Manager) createProjectSession(ctx context.Context, name, projectRoot, tmuxConfPath string) error {
	args := []string{"-L", m.socket}
	if tmuxConfPath != "" {
		args = append(args, "-f", tmuxConfPath)
	}
	args = append(args, "new-session", "-d", "-s", name, "-n", "trunk", "-c", projectRoot)

	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		return fmt.Errorf("tmux new project session failed: %w", err)
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
		if isNoSessionsOutput(output) {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions failed: %w", err)
	}

	return parseSessionList(string(res.Output)), nil
}

func (m *Manager) getRawSessionByName(ctx context.Context, name string) (*Session, error) {
	args := []string{"-L", m.socket, "list-sessions", "-F", "#{session_name}\t#{session_attached}\t#{session_path}"}

	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		output := string(res.Output)
		if isNoSessionsOutput(output) {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions failed: %w", err)
	}

	for _, s := range parseAllSessionList(string(res.Output)) {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, nil
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
	all := parseAllSessionList(output)
	var sessions []Session
	for _, s := range all {
		if strings.HasPrefix(s.Name, sessionPrefix) {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

func parseAllSessionList(output string) []Session {
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
		attached := parts[1] == "1"
		path := ""
		if len(parts) >= 3 {
			path = parts[2]
		}
		sessions = append(sessions, Session{Name: name, Attached: attached, Path: path})
	}
	return sessions
}

func validateDir(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s %q does not exist: %w", label, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %q is not a directory", label, path)
	}
	return nil
}

func dirExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isNoSessionsOutput(output string) bool {
	output = strings.TrimSpace(output)
	return output == "" ||
		strings.Contains(output, "no server running") ||
		strings.Contains(output, "no sessions") ||
		strings.Contains(output, "error connecting to")
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

// Attach attaches to an existing tmux session by exact name, replacing the
// current process. Exact-name operations intentionally do not apply the
// sessionPrefix filter so custom names created via `oca session new --name`
// remain addressable on the manager's configured socket.
func (m *Manager) Attach(ctx context.Context, name string) error {
	args := []string{"-L", m.socket, "attach", "-t", name}
	// Use syscall.Exec to replace the current process with tmux
	return syscall.Exec(m.tmuxPath, append([]string{"tmux"}, args...), os.Environ())
}

// AttachAndWait attaches to an existing tmux session using exec.Command with fd
// passthrough (instead of syscall.Exec), allowing post-exit code to run for
// resume-hint emission. The method:
//  1. Spawns tmux attach as a subprocess with Stdin/Stdout/Stderr passed through
//  2. Forwards signals (SIGINT, SIGTERM, SIGWINCH, SIGTSTP) to the tmux subprocess
//  3. After tmux exits, checks session liveness and kill sentinel
//  4. On clean exit with session destroyed, queries opencode for the session ID
//     and emits a resume hint to stdout and cache file
//
// cacheDir is the OCA cache directory for sentinel files and hint output.
// sessionWorkdir is the working directory of the session (used to match opencode sessions).
func (m *Manager) AttachAndWait(ctx context.Context, name, sessionWorkdir, cacheDir string) error {
	args := []string{"-L", m.socket, "attach", "-t", name}
	cmd := exec.Command(m.tmuxPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = FilterTMUX(os.Environ())

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("tmux attach start: %w", err)
	}

	// Forward signals to the tmux subprocess while it runs.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH, syscall.SIGTSTP)
	go func() {
		for sig := range sigCh {
			_ = cmd.Process.Signal(sig)
		}
	}()

	waitErr := cmd.Wait()
	signal.Stop(sigCh)
	close(sigCh)

	// If tmux failed, return the error (no hint).
	if waitErr != nil {
		return fmt.Errorf("tmux attach: %w", waitErr)
	}

	// tmux returned 0 — determine if hint should be emitted.
	m.emitResumeHintIfNeeded(ctx, name, sessionWorkdir, cacheDir)
	return nil
}

// emitResumeHintIfNeeded checks post-exit conditions and emits a resume hint
// to stdout and file if appropriate. Conditions for emit:
//   - Session no longer exists on the socket (tmux has-session fails)
//   - No kill sentinel present (not an intentional kill)
//   - opencode binary is on PATH and returns a matching session
//   - stdout is a TTY (or skip silently)
//
// This method never returns an error — hint emission is best-effort and must
// never fail the attach flow.
func (m *Manager) emitResumeHintIfNeeded(ctx context.Context, name, sessionWorkdir, cacheDir string) {
	// Check if session still exists (user detached via Ctrl-B d, not exited).
	if m.sessionExists(ctx, name) {
		return
	}

	// Session is gone — check for kill sentinel.
	if CheckKillSentinel(name, cacheDir) {
		CleanupKillSentinel(name, cacheDir)
		return
	}

	// Clean up any stale sentinel (no-op if absent).
	CleanupKillSentinel(name, cacheDir)

	// Check TTY — hint is for interactive use only.
	if !IsTerminal(os.Stdout.Fd()) {
		return
	}

	// Query opencode for the session ID.
	sessionID, err := QueryOpenCodeSessions(ctx, sessionWorkdir)
	if err != nil {
		// Silent skip — opencode not on PATH, no matching session, parse error.
		return
	}

	noColor := NOColor()
	hint := FormatHint(sessionID, sessionWorkdir, noColor)
	fileHint := FormatHintFile(sessionID, sessionWorkdir)

	// Emit to stdout.
	fmt.Fprintln(os.Stdout, hint)

	// Emit to file (best effort).
	_ = EmitHintToFile(fileHint, cacheDir)
}

// sessionExists returns true if the named session exists on the manager's socket.
func (m *Manager) sessionExists(ctx context.Context, name string) bool {
	args := []string{"-L", m.socket, "has-session", "-t", name}
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: 5 * time.Second,
	})
	return err == nil
}

// SwitchClient switches the current tmux client to a different session by
// exact name on the manager's configured socket. It preserves custom-name
// support; bulk/safety operations carry the sessionPrefix filter instead.
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

// ListWindows returns windows for an exact tmux session name. Pattern B project
// sessions intentionally do not use the legacy oca- prefix, so this helper is
// exact-name scoped rather than prefix-scoped.
func (m *Manager) ListWindows(ctx context.Context, sessionName string) ([]Window, error) {
	args := []string{"-L", m.socket, "list-windows", "-t", sessionName, "-F", "#{window_id}\t#{window_index}\t#{window_name}\t#{window_active}\t#{pane_current_path}"}
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		output := string(res.Output)
		if isNoSessionsOutput(output) {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-windows failed: %w", err)
	}
	return parseWindowList(string(res.Output)), nil
}

// EnsureWindow returns the named project-session window, creating it at
// workingDir when missing. If a matching window has a stale cwd, it is replaced
// so the next attach lands in the expected worktree.
func (m *Manager) EnsureWindow(ctx context.Context, sessionName, windowName, workingDir string) (*Window, bool, error) {
	if sessionName == "" {
		return nil, false, fmt.Errorf("session name is required")
	}
	if windowName == "" {
		return nil, false, fmt.Errorf("window name is required")
	}
	if err := validateDir(workingDir, "working directory"); err != nil {
		return nil, false, err
	}

	windows, err := m.ListWindows(ctx, sessionName)
	if err != nil {
		return nil, false, err
	}
	for _, w := range windows {
		if w.Name != windowName {
			continue
		}
		if dirExists(w.Path) {
			return &w, false, nil
		}
		if err := m.killWindow(ctx, sessionName, w.ID); err != nil {
			return nil, false, fmt.Errorf("kill stale window %q: %w", windowName, err)
		}
		break
	}

	if err := m.createWindow(ctx, sessionName, windowName, workingDir); err != nil {
		return nil, false, err
	}
	return &Window{Index: -1, Name: windowName, Path: workingDir}, true, nil
}

func (m *Manager) createWindow(ctx context.Context, sessionName, windowName, workingDir string) error {
	args := []string{"-L", m.socket, "new-window", "-t", sessionName, "-n", windowName, "-c", workingDir}
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		return fmt.Errorf("tmux new-window failed: %w", err)
	}
	return nil
}

func (m *Manager) killWindow(ctx context.Context, sessionName, windowID string) error {
	target := sessionName
	if windowID != "" {
		target = sessionName + ":" + windowID
	}
	args := []string{"-L", m.socket, "kill-window", "-t", target}
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: sessionTimeout,
	})
	if err != nil {
		return fmt.Errorf("tmux kill-window failed: %w", err)
	}
	return nil
}

func parseWindowList(output string) []Window {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil
	}
	var windows []Window
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 4 {
			continue
		}
		index, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		path := ""
		if len(parts) >= 5 {
			path = parts[4]
		}
		windows = append(windows, Window{
			ID:     parts[0],
			Index:  index,
			Name:   parts[2],
			Active: parts[3] == "1",
			Path:   path,
		})
	}
	return windows
}

// Kill destroys a tmux session by exact name on the manager's configured
// socket. It preserves custom-name support; callers that need prefix-scoped
// bulk deletion should use KillAll or ReapStale.
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

// Restart kills and recreates a session by exact name. Custom-named sessions
// are allowed, but their current path cannot be inferred through
// GetSessionByName because that helper is prefix-scoped; pass workingDir when
// restarting custom names.
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
	if workingDir == "" {
		return fmt.Errorf("working directory is required to restart custom or unlisted session %q", name)
	}

	// Kill
	if err := m.Kill(ctx, name); err != nil {
		return fmt.Errorf("kill session %q: %w", name, err)
	}

	// Verify dead (poll with 100ms sleep, max 2s timeout)
	deadline := time.Now().Add(2 * time.Second)
	dead := false
	for time.Now().Before(deadline) {
		s, err := m.GetSessionByName(ctx, name)
		if err != nil {
			return fmt.Errorf("verify session dead: %w", err)
		}
		if s == nil {
			dead = true
			break // Session is gone
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !dead {
		return fmt.Errorf("session %q did not stop within 2s", name)
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
		if isNoSessionsOutput(output) {
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

// GetSessionByName finds a single prefix-scoped OCA session by exact name.
// It delegates through List(), so custom names without sessionPrefix are not
// visible here even though exact-name operations can still target them.
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
