package render

import (
	"bytes"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanWatcher builds the render plan for opencode.json .watcher section.
// Merges declared watcher.ignore entries into .watcher.ignore while preserving
// user-added ignore entries and other watcher keys.
func PlanWatcher(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	// Extract declared ignore entries from stack.
	declared := stack.Watcher.Ignore

	merged, err := MergeWatcherIgnore(opBefore, declared)
	if err != nil {
		return nil, err
	}
	opOp := "merge"
	if bytes.Equal(opBefore, merged) {
		opOp = "noop"
	}

	return &Plan{
		Source:   source,
		LockPath: paths.ApplyLockPath(),
		Targets: []TargetOp{{
			Name:   "opencode.json",
			Path:   opPath,
			Op:     opOp,
			Before: opBefore,
			After:  merged,
			Mode:   0o644,
			Reason: "merge declared watcher.ignore entries into .watcher preserving user-added entries",
		}},
	}, nil
}
