package discord

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTaglinesFromTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taglines.toml")
	content := "taglines = [\"Spec-driven shipping\", \"Workflow state synced\"]\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	taglines, err := LoadTaglines(path)
	if err != nil {
		t.Fatalf("LoadTaglines returned error: %v", err)
	}
	if len(taglines) != 2 {
		t.Fatalf("len(taglines) = %d, want 2", len(taglines))
	}
	if taglines[0] != "Spec-driven shipping" {
		t.Fatalf("first tagline = %q", taglines[0])
	}
}

func TestLoadTaglinesFallsBackForMissingFile(t *testing.T) {
	taglines, err := LoadTaglines(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("missing file should use fallback without error: %v", err)
	}
	if len(taglines) == 0 || taglines[0] != DefaultTagline {
		t.Fatalf("fallback taglines = %#v, want first %q", taglines, DefaultTagline)
	}
}

func TestChooseTaglineAvoidsLastIndex(t *testing.T) {
	taglines := []string{"first", "second"}
	picks := []int{0, 1}
	idx, tagline := chooseTagline(taglines, 0, func(n int) int {
		pick := picks[0]
		picks = picks[1:]
		return pick % n
	})
	if idx != 1 || tagline != "second" {
		t.Fatalf("choice = (%d, %q), want (1, second)", idx, tagline)
	}
}

func TestSelectTaglinePersistsLastIndex(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "last")
	choice, err := SelectTagline([]string{"only"}, statePath)
	if err != nil {
		t.Fatalf("SelectTagline returned error: %v", err)
	}
	if choice != "only" {
		t.Fatalf("choice = %q, want only", choice)
	}
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "0" {
		t.Fatalf("state file = %q, want 0", data)
	}
}
