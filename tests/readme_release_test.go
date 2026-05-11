package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadmeReflectsV1ReleaseState(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme := string(data)
	for _, want := range []string{
		"_Status: v1.0 release candidate._",
		"## Install quickstart",
		"## Command reference",
		"## stack.toml overview",
		"oca temporal start",
		"oca migrate from-open-chad",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"Phase 6 (installer + shell profile) is in progress",
		"Phase 8:",
		"Not started",
		"open-chad stays the daily driver",
	} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("README contains stale release text %q", forbidden)
		}
	}
}
