package advruntime

import (
	"os"
	"path/filepath"
)

// DefaultOpenCodeDBPath returns the default path to OpenCode's SQLite database.
// If XDG_DATA_HOME is set, it wins. If home directory cannot be resolved,
// returns empty string.
func DefaultOpenCodeDBPath() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "opencode", "opencode.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

// DefaultWorktreeRoot returns the default OCA worktree root directory.
// If XDG_DATA_HOME is set, it wins. If home directory cannot be resolved,
// returns empty string.
func DefaultWorktreeRoot() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "opencode", "worktree")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "worktree")
}

// DefaultADVStateRoot returns the default ADV plugin state root directory.
// If XDG_DATA_HOME is set, it wins. If home directory cannot be resolved,
// returns empty string.
func DefaultADVStateRoot() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "opencode", "plugins", "advance")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "plugins", "advance")
}
