package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallGuideCoversFirstRunPath(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "INSTALL.md"))
	if err != nil {
		t.Fatalf("INSTALL.md missing: %v", err)
	}
	install := string(data)
	for _, want := range []string{
		"# Install OpenCode Advance",
		"Go 1.22+",
		"tmux",
		"git",
		"oca install",
		"shell profile",
		"oca apply --dry-run",
		"oca apply",
		"oca doctor",
		"OCA_OPENCODE_CONFIG_DIR",
	} {
		if !strings.Contains(install, want) {
			t.Fatalf("INSTALL.md missing %q", want)
		}
	}
}
