package advstatus

import (
	"os"
	"path/filepath"
	"strings"
)

// FindActiveChanges scans a changes directory for non-archived, non-closed
// change IDs. Returns them sorted by directory modification time (most recent
// first). Returns empty slice (not error) when dir is missing.
func FindActiveChanges(changesDir string) ([]string, error) {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil, nil // graceful: missing dir is normal
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		changeDir := filepath.Join(changesDir, entry.Name())
		cj, _ := readChangeJSON(changeDir)
		if cj == nil {
			continue
		}
		if cj.Status != "archived" && cj.Status != "closed" {
			result = append(result, entry.Name())
		}
	}
	return result, nil
}

// SummarizeChange reads change.json from the given directory and returns
// a ChangeSummary with the current gate progress. Returns nil (no error)
// when change.json is missing or unreadable (graceful degradation).
func SummarizeChange(changeDir string) (*ChangeSummary, error) {
	cj, _ := readChangeJSON(changeDir)
	if cj == nil {
		return nil, nil
	}

	currentGate := "✓"
	if cj.Gates != nil {
		currentGate = findFirstPendingGate(cj.Gates)
	}

	return &ChangeSummary{
		ID:          cj.ID,
		ShortID:     shortenChangeID(cj.ID),
		CurrentGate: currentGate,
		Status:      cj.Status,
	}, nil
}

// ActiveChangeSummary scans all ADV projects for the first active change
// and returns its summary. Returns nil when none found.
// Uses advruntime.DefaultADVStateRoot() to locate the ADV state directory.
func ActiveChangeSummary() (*ChangeSummary, error) {
	advRoot := advStateRoot()
	if advRoot == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(advRoot)
	if err != nil {
		return nil, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		changesDir := filepath.Join(advRoot, entry.Name(), "changes")
		ids, err := FindActiveChanges(changesDir)
		if err != nil || len(ids) == 0 {
			continue
		}
		// Return the first active change found
		changeDir := filepath.Join(changesDir, ids[0])
		return SummarizeChange(changeDir)
	}
	return nil, nil
}

// ActiveChangeSummaryForProject scans a specific project's changes directory.
// Returns nil when none found.
func ActiveChangeSummaryForProject(changesDir string) (*ChangeSummary, error) {
	ids, err := FindActiveChanges(changesDir)
	if err != nil || len(ids) == 0 {
		return nil, nil
	}
	changeDir := filepath.Join(changesDir, ids[0])
	return SummarizeChange(changeDir)
}

// advStateRoot returns the ADV plugin state root directory.
// Extracted as a function for testability.
func advStateRoot() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode", "plugins", "advance")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "plugins", "advance")
}

// FindActiveChangesAllProjects scans all ADV project changes directories.
// Returns change IDs across all projects.
func FindActiveChangesAllProjects() ([]string, error) {
	advRoot := advStateRoot()
	if advRoot == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(advRoot)
	if err != nil {
		return nil, nil
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		changesDir := filepath.Join(advRoot, entry.Name(), "changes")
		ids, err := FindActiveChanges(changesDir)
		if err != nil {
			continue
		}
		result = append(result, ids...)
	}
	return result, nil
}

// readTemporalAddress reads ADV_TEMPORAL_ADDRESS from temporal.env in the
// given cache directory. Returns empty string when file missing or key absent.
func readTemporalAddress(cacheDir string) string {
	envFile := filepath.Join(cacheDir, "temporal.env")
	data, err := os.ReadFile(envFile)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ADV_TEMPORAL_ADDRESS=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ADV_TEMPORAL_ADDRESS="))
		}
	}
	return ""
}

// cacheDir returns the OCA cache directory, matching render.CacheDir().
func cacheDir() string {
	if d := os.Getenv("OCA_CACHE_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "opencode-advance")
	}
	return filepath.Join(os.TempDir(), "opencode-advance")
}
