package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseWorkflowPublishesVersionTagsWithGoReleaser(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, ".github", "workflows", "release.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("release workflow missing: %v", err)
	}
	workflow := string(data)
	for _, want := range []string{
		"name: Release",
		"tags:",
		"'v*'",
		"permissions:",
		"contents: write",
		"goreleaser/goreleaser-action",
		"GITHUB_TOKEN",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("release workflow missing %q\n%s", want, workflow)
		}
	}
}
