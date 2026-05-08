package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolishDocsRemoveDeferredDiscordReferences(t *testing.T) {
	root := repoRoot(t)
	checks := map[string][]string{
		"docs/design/architecture.md": {
			"does **not** render agents, discord",
			"DeferredSections` (`agents`, `session`, `discord`)",
		},
		"lib/README.md": {
			"discord/setup.sh",
			"discord/update.sh",
			"Discord scripts are Phase 8",
		},
		"docs/proposals/phases.md": {
			"| 8: Extras + polish                           | 3-5 days    | Not started |",
			"Total shipped: Phases 0–7",
		},
		"docs/proposals/v1-implementation.md": {
			"Current recommended next phase",
		},
	}
	for rel, forbidden := range checks {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		for _, phrase := range forbidden {
			if strings.Contains(content, phrase) {
				t.Fatalf("%s contains stale phrase %q", rel, phrase)
			}
		}
	}
}

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
		if !strings.Contains(string(data), "phase8ExtrasPolishDiscord") {
			t.Fatalf("%s should mention phase8ExtrasPolishDiscord", rel)
		}
	}
}
