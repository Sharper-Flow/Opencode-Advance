package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestApplyConcurrency_NoCorruption launches two concurrent `apply` runs
// sharing the same target config dir and asserts:
//  1. Both runs do not fail — the lock serializes them, so at least one
//     must succeed.
//  2. The resulting opencode.json on disk parses as valid JSON, proving
//     that interleaved writes did not corrupt the file.
func TestApplyConcurrency_NoCorruption(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	opencodeDir := filepath.Join(tmp, "opencode")
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + opencodeDir,
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(tmp, "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(tmp, "cache"),
	}
	var wg sync.WaitGroup
	results := make([]error, 2)
	stderrs := make([]string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, stderr, err := runOCA(t, root, env, "apply", "--target", "mcp", "--config", "stack.example.toml")
			results[idx] = err
			stderrs[idx] = stderr
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		}
	}
	if successes == 0 {
		t.Fatalf("both concurrent apply runs failed; stderr[0]=%q stderr[1]=%q", stderrs[0], stderrs[1])
	}

	// Verify the resulting opencode.json parses as valid JSON — if atomic
	// writes or locking were broken, interleaved writes would produce a
	// truncated or corrupted file that fails to parse.
	outPath := filepath.Join(opencodeDir, "opencode.json")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read opencode.json: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("opencode.json is not valid JSON (corruption): %v\ncontent=%s", err, string(data))
	}
}
