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

func TestGoReleaserConfigBuildsOcaBinaries(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, ".goreleaser.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("goreleaser config missing: %v", err)
	}
	config := string(data)
	for _, want := range []string{
		"project_name: oca",
		"main: ./cmd/oca",
		"binary: oca",
		"CGO_ENABLED=0",
		"-trimpath",
		"linux",
		"darwin",
		"amd64",
		"arm64",
		"-X main.version={{ .Version }}",
		"mod_timestamp: '{{ .CommitTimestamp }}'",
		"algorithm: sha256",
		"SHA256SUMS.txt",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("goreleaser config missing %q\n%s", want, config)
		}
	}
}
