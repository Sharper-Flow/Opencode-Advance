package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemporalSupervisionDocsShippedSurface(t *testing.T) {
	root := repoRoot(t)
	cli := readDoc(t, root, "docs/design/cli-surface.md")
	for _, want := range []string{
		"oca temporal status",
		"oca temporal start",
		"oca temporal stop",
		"oca temporal restart",
		"oca temporal logs",
		"--follow",
		"bounded recent log output",
	} {
		if !strings.Contains(cli, want) {
			t.Fatalf("cli-surface missing %q", want)
		}
	}

	architecture := readDoc(t, root, "docs/design/architecture.md")
	if !strings.Contains(architecture, "oca temporal status/start/stop/restart/logs") {
		t.Fatalf("architecture missing shipped temporal supervision summary")
	}

	phasePlan := readDoc(t, root, "docs/proposals/phases.md")
	if !strings.Contains(phasePlan, "**Status:** Shipped") {
		t.Fatalf("phase plan should mark Phase 6.5 shipped")
	}
}

func readDoc(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}
