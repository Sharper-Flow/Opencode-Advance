package advruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultPaths_XDGDataHomeWins(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/xdg/data")

	wantDB := filepath.Join("/xdg", "data", "opencode", "opencode.db")
	if got := DefaultOpenCodeDBPath(); got != wantDB {
		t.Fatalf("DefaultOpenCodeDBPath = %q, want %q", got, wantDB)
	}

	wantWT := filepath.Join("/xdg", "data", "opencode", "worktree")
	if got := DefaultWorktreeRoot(); got != wantWT {
		t.Fatalf("DefaultWorktreeRoot = %q, want %q", got, wantWT)
	}

	wantADV := filepath.Join("/xdg", "data", "opencode", "plugins", "advance")
	if got := DefaultADVStateRoot(); got != wantADV {
		t.Fatalf("DefaultADVStateRoot = %q, want %q", got, wantADV)
	}
}

func TestDefaultPaths_FallbackShape(t *testing.T) {
	// Unset XDG_DATA_HOME to force fallback to home dir.
	t.Setenv("XDG_DATA_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot resolve home dir: %v", err)
	}

	wantDB := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if got := DefaultOpenCodeDBPath(); got != wantDB {
		t.Fatalf("DefaultOpenCodeDBPath = %q, want %q", got, wantDB)
	}

	wantWT := filepath.Join(home, ".local", "share", "opencode", "worktree")
	if got := DefaultWorktreeRoot(); got != wantWT {
		t.Fatalf("DefaultWorktreeRoot = %q, want %q", got, wantWT)
	}

	wantADV := filepath.Join(home, ".local", "share", "opencode", "plugins", "advance")
	if got := DefaultADVStateRoot(); got != wantADV {
		t.Fatalf("DefaultADVStateRoot = %q, want %q", got, wantADV)
	}
}

func TestDefaultPaths_ReturnsEmptyWhenHomeUnavailable(t *testing.T) {
	// Unset XDG_DATA_HOME and HOME to simulate unresolvable home dir.
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "") // Windows fallback

	// os.UserHomeDir will fail when neither HOME nor USERPROFILE is set.
	if got := DefaultOpenCodeDBPath(); got != "" {
		t.Fatalf("DefaultOpenCodeDBPath = %q, want empty string when home unavailable", got)
	}
	if got := DefaultWorktreeRoot(); got != "" {
		t.Fatalf("DefaultWorktreeRoot = %q, want empty string when home unavailable", got)
	}
	if got := DefaultADVStateRoot(); got != "" {
		t.Fatalf("DefaultADVStateRoot = %q, want empty string when home unavailable", got)
	}
}

func TestDefaultPaths_EmptyXDGDataHomeFallsBack(t *testing.T) {
	// Explicitly set XDG_DATA_HOME to empty string (not unset).
	t.Setenv("XDG_DATA_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot resolve home dir: %v", err)
	}

	got := DefaultOpenCodeDBPath()
	if got == "" {
		t.Fatal("DefaultOpenCodeDBPath empty when XDG_DATA_HOME is explicitly empty but home resolvable")
	}
	if !strings.HasPrefix(got, home) {
		t.Fatalf("DefaultOpenCodeDBPath = %q, expected prefix %q", got, home)
	}
}
