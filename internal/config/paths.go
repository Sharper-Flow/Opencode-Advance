package config

import (
	"os"
	"path/filepath"
)

// Paths are the resolved target directories OCA writes to. Every field
// honors the matching environment override; unset envs fall back to the
// user's XDG/home defaults.
//
// The cache directory is where the apply-lock (flock) lives.
type Paths struct {
	OpencodeConfigDir string // $OCA_OPENCODE_CONFIG_DIR or ~/.config/opencode
	VisionConfigDir   string // $OCA_VISION_CONFIG_DIR or ~/.config/vision
	CacheDir          string // $OCA_CACHE_DIR or $XDG_RUNTIME_DIR/opencode-advance or /tmp/opencode-advance-$UID
}

// ResolvePaths returns the effective target paths for this process.
// Safe to call during tests — pass env overrides via os.Setenv to
// redirect writes to t.TempDir().
func ResolvePaths() Paths {
	return Paths{
		OpencodeConfigDir: resolvePath("OCA_OPENCODE_CONFIG_DIR", filepath.Join(home(), ".config", "opencode")),
		VisionConfigDir:   resolvePath("OCA_VISION_CONFIG_DIR", filepath.Join(home(), ".config", "vision")),
		CacheDir:          resolveCacheDir(),
	}
}

// OpencodeJSON is the full path to the OpenCode JSON config file.
func (p Paths) OpencodeJSON() string {
	return filepath.Join(p.OpencodeConfigDir, "opencode.json")
}

// VisionServersYAML is the full path to Vision's servers.yaml.
func (p Paths) VisionServersYAML() string {
	return filepath.Join(p.VisionConfigDir, "servers.yaml")
}

// ApplyLockPath is the file used by WriteAtomic flock to serialize
// concurrent oca apply runs.
func (p Paths) ApplyLockPath() string {
	return filepath.Join(p.CacheDir, "apply.lock")
}

// resolvePath returns the env-var value if set, else the fallback.
func resolvePath(envName, fallback string) string {
	if v := os.Getenv(envName); v != "" {
		return v
	}
	return fallback
}

// resolveCacheDir prefers OCA_CACHE_DIR, then $XDG_RUNTIME_DIR/opencode-advance,
// then /tmp/opencode-advance-$USER as a last resort (macOS / containers).
func resolveCacheDir() string {
	if v := os.Getenv("OCA_CACHE_DIR"); v != "" {
		return v
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, "opencode-advance")
	}
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}
	return filepath.Join(os.TempDir(), "opencode-advance-"+user)
}

// home returns the user's home dir, falling back to /tmp for environments
// where UserHomeDir fails.
func home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "/tmp"
}
