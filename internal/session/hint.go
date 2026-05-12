package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// openCodeSession represents a single session from `opencode session list --format json`.
// Only fields used by the hint system are parsed; unknown fields are ignored.
type openCodeSession struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Updated   int64  `json:"updated"`
	Created   int64  `json:"created"`
	ProjectID string `json:"projectId"`
	Directory string `json:"directory"`
}

// WriteKillSentinel creates a sentinel file indicating an intentional session kill.
// The file is written atomically (temp + rename) under cacheDir/kill-sentinel/<sessionName>.
func WriteKillSentinel(sessionName, cacheDir string) error {
	sentinelDir := filepath.Join(cacheDir, "kill-sentinel")
	if err := os.MkdirAll(sentinelDir, 0o755); err != nil {
		return fmt.Errorf("create sentinel dir: %w", err)
	}

	target := filepath.Join(sentinelDir, sessionName)
	content := []byte(fmt.Sprintf("%d", time.Now().UnixMilli()))

	// Atomic write: temp file in same dir, then rename
	tmp, err := os.CreateTemp(sentinelDir, sessionName+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp sentinel: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write sentinel: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close sentinel temp: %w", err)
	}

	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename sentinel: %w", err)
	}

	return nil
}

// CheckKillSentinel returns true if a kill sentinel exists for the given session.
func CheckKillSentinel(sessionName, cacheDir string) bool {
	path := filepath.Join(cacheDir, "kill-sentinel", sessionName)
	_, err := os.Stat(path)
	return err == nil
}

// CleanupKillSentinel removes the kill sentinel for the given session.
// No error if the sentinel does not exist.
func CleanupKillSentinel(sessionName, cacheDir string) {
	path := filepath.Join(cacheDir, "kill-sentinel", sessionName)
	os.Remove(path)
}

// ParseOpenCodeSessions parses the JSON output of `opencode session list --format json`
// and returns the ID of the most-recently-updated session matching the given workdir.
// Returns an error if no matching session is found or the JSON is malformed.
func ParseOpenCodeSessions(data []byte, workdir string) (string, error) {
	var sessions []openCodeSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return "", fmt.Errorf("parse opencode sessions: %w", err)
	}

	// Filter by directory
	var matches []openCodeSession
	for _, s := range sessions {
		if s.Directory == workdir {
			matches = append(matches, s)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no opencode session found for workdir %q", workdir)
	}

	// Sort by updated descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Updated > matches[j].Updated
	})

	return matches[0].ID, nil
}

// QueryOpenCodeSessions runs `opencode session list --format json` and returns
// the most-recently-updated session ID for the given workdir.
// Returns an error if opencode is not on PATH, JSON parse fails, or no match.
// Callers should treat errors as "skip hint" — never fail the attach flow.
func QueryOpenCodeSessions(ctx context.Context, workdir string) (string, error) {
	opencodePath, err := exec.LookPath("opencode")
	if err != nil {
		return "", fmt.Errorf("opencode not found on PATH: %w", err)
	}

	cmd := exec.CommandContext(ctx, opencodePath, "session", "list", "--format", "json")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = nil // discard stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("opencode session list failed: %w", err)
	}

	return ParseOpenCodeSessions(buf.Bytes(), workdir)
}

// FormatHint returns the resume hint string for terminal output.
// When noColor is true, the output is plain ASCII without emoji.
func FormatHint(sessionID, workdir string, noColor bool) string {
	cmd := fmt.Sprintf("opencode --session %s", sessionID)
	if noColor {
		return fmt.Sprintf("Resume: %s  (project: %s)", cmd, workdir)
	}
	return fmt.Sprintf("💤 Resume: %s  (project: %s)", cmd, workdir)
}

// FormatHintFile returns the resume hint string for file output.
// Uses --project flag for cwd independence.
func FormatHintFile(sessionID, workdir string) string {
	return fmt.Sprintf("opencode --session %s --project %s", sessionID, workdir)
}

// EmitHintToFile writes the hint to the cache directory file atomically.
func EmitHintToFile(hint, cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	target := filepath.Join(cacheDir, "last-session-hint")

	tmp, err := os.CreateTemp(cacheDir, "last-session-hint.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp hint: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(hint); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write hint: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close hint temp: %w", err)
	}

	// Set permissions before rename (rename preserves perms on Linux)
	os.Chmod(tmpName, 0o600)

	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename hint: %w", err)
	}

	return nil
}

// FilterTMUX removes TMUX and TMUX_PANE variables from the environment slice.
// This prevents tmux nesting issues when attaching from inside an existing tmux session.
func FilterTMUX(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		key := strings.SplitN(e, "=", 2)[0]
		if key == "TMUX" || key == "TMUX_PANE" {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

// ResolveCacheDir determines the OCA cache directory from environment variables.
// Priority: OCA_CACHE_DIR > $XDG_RUNTIME_DIR/opencode-advance.
func ResolveCacheDir() string {
	if dir := os.Getenv("OCA_CACHE_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "opencode-advance")
	}
	// Fallback for non-XDG systems (macOS, containers)
	return filepath.Join(os.TempDir(), fmt.Sprintf("opencode-advance-%d", os.Getuid()))
}

// IsTerminal returns true if fd is connected to a terminal.
func IsTerminal(fd uintptr) bool {
	file := os.NewFile(fd, "")
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// NOColor returns true if the NO_COLOR environment variable is set.
func NOColor() bool {
	return os.Getenv("NO_COLOR") != ""
}
