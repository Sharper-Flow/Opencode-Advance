package render

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestInstructionAssetsReadmeParity verifies that the README inventory in
// assets/instructions/ matches the actual contents of that directory.
//
// The README declares its inventory between HTML markers:
//
//	<!-- INVENTORY:START -->
//	| File | Purpose |
//	| ---- | ------- |
//	| `name.md` | ... |
//	...
//	<!-- INVENTORY:END -->
//
// Drift (file present in dir but missing from README, or vice versa) fails the
// test with an explicit diff. Locks the M4 parity gap that previously allowed
// the README to drift silently from real contents.
func TestInstructionAssetsReadmeParity(t *testing.T) {
	root := instructionsAssetsRoot(t)

	// 1. Read README and extract files declared between INVENTORY markers.
	readmePath := filepath.Join(root, "README.md")
	readmeBytes, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	declared, err := parseInventoryMarkers(string(readmeBytes))
	if err != nil {
		t.Fatalf("parse inventory: %v", err)
	}
	if len(declared) == 0 {
		t.Fatal("INVENTORY markers found no files; check marker placement in README")
	}

	// 2. List actual files in dir, excluding README.md itself and any
	//    non-markdown / non-yaml asset files.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read assets/instructions dir: %v", err)
	}
	actualSet := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "README.md" {
			continue
		}
		actualSet[name] = true
	}

	declaredSet := make(map[string]bool)
	for _, f := range declared {
		declaredSet[f] = true
	}

	// 3. Compute symmetric difference.
	var missingFromReadme []string // in dir but not README
	for name := range actualSet {
		if !declaredSet[name] {
			missingFromReadme = append(missingFromReadme, name)
		}
	}
	var missingFromDir []string // in README but not dir
	for name := range declaredSet {
		if !actualSet[name] {
			missingFromDir = append(missingFromDir, name)
		}
	}
	sort.Strings(missingFromReadme)
	sort.Strings(missingFromDir)

	if len(missingFromReadme) == 0 && len(missingFromDir) == 0 {
		return
	}

	var b strings.Builder
	b.WriteString("assets/instructions/ ↔ README.md INVENTORY drift detected:\n")
	if len(missingFromReadme) > 0 {
		b.WriteString("\n  Files present in dir but missing from README INVENTORY:\n")
		for _, f := range missingFromReadme {
			b.WriteString("    - ")
			b.WriteString(f)
			b.WriteByte('\n')
		}
	}
	if len(missingFromDir) > 0 {
		b.WriteString("\n  Files listed in README INVENTORY but missing from dir:\n")
		for _, f := range missingFromDir {
			b.WriteString("    - ")
			b.WriteString(f)
			b.WriteByte('\n')
		}
	}
	b.WriteString("\n  Fix: edit assets/instructions/README.md between the\n")
	b.WriteString("  <!-- INVENTORY:START --> and <!-- INVENTORY:END --> markers\n")
	b.WriteString("  to match the actual directory contents.")
	t.Fatal(b.String())
}

// instructionsAssetsRoot returns the path to assets/instructions/. Honors
// OCA_ASSETS_ROOT for test environments running from compiled binaries
// outside the repo tree; otherwise uses the standard relative location.
func instructionsAssetsRoot(t *testing.T) string {
	t.Helper()
	if env := os.Getenv("OCA_ASSETS_ROOT"); env != "" {
		return filepath.Join(env, "instructions")
	}
	// internal/render/ → ../../assets/instructions/
	return filepath.Join("..", "..", "assets", "instructions")
}

// parseInventoryMarkers extracts file names from markdown table rows enclosed
// between <!-- INVENTORY:START --> and <!-- INVENTORY:END -->. File names are
// the first column, surrounded by backticks: `name.md`.
//
// Returns an error if markers are missing or mis-ordered.
func parseInventoryMarkers(content string) ([]string, error) {
	const startMarker = "<!-- INVENTORY:START -->"
	const endMarker = "<!-- INVENTORY:END -->"

	startIdx := strings.Index(content, startMarker)
	if startIdx < 0 {
		return nil, &parseInventoryError{"missing INVENTORY:START marker"}
	}
	endIdx := strings.Index(content, endMarker)
	if endIdx < 0 {
		return nil, &parseInventoryError{"missing INVENTORY:END marker"}
	}
	if endIdx < startIdx {
		return nil, &parseInventoryError{"INVENTORY:END appears before INVENTORY:START"}
	}

	section := content[startIdx+len(startMarker) : endIdx]
	// Match backtick-quoted filename in the first column of a markdown table row:
	//   | `name.ext` | ... |
	rowFile := regexp.MustCompile("^\\s*\\|\\s*`([^`]+)`\\s*\\|")
	var files []string
	for _, line := range strings.Split(section, "\n") {
		m := rowFile.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		files = append(files, m[1])
	}
	return files, nil
}

type parseInventoryError struct {
	msg string
}

func (e *parseInventoryError) Error() string {
	return e.msg
}
