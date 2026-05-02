package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangelogCoversV1Phases(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("CHANGELOG.md missing: %v", err)
	}
	changelog := string(data)
	for _, want := range []string{
		"# Changelog",
		"## v1.0.0-rc1",
		"Phase 0 — Foundation + Brand",
		"Phase 1 — stack.toml + MCP Apply",
		"Phase 2 — Plugin + Instruction Management",
		"Phase 3 — Core opencode.json Coverage",
		"Phase 3.5 — Skills + Commands + Formatters",
		"Phase 4 — Session UX + Theme",
		"Phase 5 — Temporal Enablement",
		"Phase 5.5 — Vision Slot Groups",
		"Phase 6 — Installer + Shell",
		"Phase 6.5 — Temporal Dev-Server Supervision",
		"Phase 7 — Migration + Doctor Expansion",
		"Phase 8 — Extras + Polish",
	} {
		if !strings.Contains(changelog, want) {
			t.Fatalf("CHANGELOG missing %q", want)
		}
	}
}
