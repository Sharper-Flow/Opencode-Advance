package tests

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestApplyConcurrency_NoCorruption(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + filepath.Join(tmp, "opencode"),
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(tmp, "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(tmp, "cache"),
	}
	var wg sync.WaitGroup
	results := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _, err := runOCA(t, root, env, "apply", "--target", "mcp", "--config", "stack.example.toml")
			results[idx] = err
		}(i)
	}
	wg.Wait()
	if results[0] != nil && results[1] != nil {
		t.Fatalf("both concurrent apply runs failed: %v / %v", results[0], results[1])
	}
}
