package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsNoStaleAgentNames(t *testing.T) {
	root := repoRoot(t)
	forbidden := []string{"scout.md", "refine.md"}
	for _, rel := range []string{"AGENTS.md", "assets/agents/README.md"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		for _, name := range forbidden {
			if strings.Contains(content, name) {
				t.Fatalf("%s must not reference stale agent name %q — these were consolidated by Advance and are not current shipped agents", rel, name)
			}
		}
	}
}

func TestPhase8RoadmapDocsShowReleaseCandidate(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{"docs/proposals/phases.md", "docs/proposals/v1-implementation.md"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "phase8ExtrasPolish") {
			t.Fatalf("%s should mention phase8ExtrasPolish", rel)
		}
	}
}
