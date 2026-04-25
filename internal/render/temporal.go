package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// Env var names for the Temporal environment file.
const (
	EnvTemporalAddress    = "ADV_TEMPORAL_ADDRESS"
	EnvTemporalNamespace  = "ADV_TEMPORAL_NAMESPACE"
	EnvTemporalAllowRemote = "ADV_TEMPORAL_ALLOW_REMOTE"
	EnvNodePath           = "ADV_NODE_PATH"
)

// CacheDir returns the OCA cache directory using a 4-tier fallback chain:
// 1. $OCA_CACHE_DIR (explicit override)
// 2. $XDG_RUNTIME_DIR/opencode-advance
// 3. $TMPDIR/opencode-advance
// 4. /tmp/opencode-advance
func CacheDir() string {
	if d := os.Getenv("OCA_CACHE_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "opencode-advance")
	}
	if d := os.Getenv("TMPDIR"); d != "" {
		return filepath.Join(d, "opencode-advance")
	}
	return "/tmp/opencode-advance"
}

// RenderTemporalEnv renders the Temporal environment file content from a
// validated stack config. Only non-empty values are emitted as key=value
// lines. Returns empty string with no error when Temporal is nil or disabled.
func RenderTemporalEnv(stack *config.Stack) (string, error) {
	if stack.Temporal == nil || !stack.Temporal.IsEnabled() {
		return "", nil
	}

	t := stack.Temporal
	var lines []string

	// Address is always present after validation (defaults applied).
	if t.Address != "" {
		lines = append(lines, fmt.Sprintf("%s=%s", EnvTemporalAddress, t.Address))
	}
	if t.Namespace != "" {
		lines = append(lines, fmt.Sprintf("%s=%s", EnvTemporalNamespace, t.Namespace))
	}
	if t.AllowRemote != nil {
		lines = append(lines, fmt.Sprintf("%s=%v", EnvTemporalAllowRemote, *t.AllowRemote))
	}
	if t.NodePath != "" {
		lines = append(lines, fmt.Sprintf("%s=%s", EnvNodePath, t.NodePath))
	}

	return strings.Join(lines, "\n") + "\n", nil
}
