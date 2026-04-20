package render

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MergeArrayOptions controls MergeArray behavior.
type MergeArrayOptions struct {
	// StripStaleWorktree enables pruning of worktree-sourced entries
	// whose path no longer exists on disk. Per jc-strip1, the base
	// directory is resolved at runtime from $XDG_DATA_HOME (or
	// $HOME/.local/share).
	StripStaleWorktree bool
}

// MergeArrayResult summarizes the merge outcome.
type MergeArrayResult struct {
	// Bytes is the merged JSON document.
	Bytes []byte
	// PrunedCount is the number of stale worktree entries removed.
	PrunedCount int
	// AddedCount is the number of declared entries that were new.
	AddedCount int
	// PreservedCount is the number of existing user entries kept.
	PreservedCount int
}

func buildWorktreePattern() *regexp.Regexp {
	base := xdgDataHome()
	// Escape for regex: handle dots and slashes.
	escaped := regexp.QuoteMeta(base)
	// Pattern: <base>/opencode/worktree/<sha>/change/<name>/...
	// SHA is hex, name is alphanumeric+dash.
	pattern := "^" + escaped + `/opencode/worktree/[0-9a-f]+/change/[A-Za-z0-9_-]+(?:/.*)?$`
	return regexp.MustCompile(pattern)
}

// xdgDataHome resolves $XDG_DATA_HOME per the XDG Base Directory spec.
func xdgDataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/.local/share"
	}
	return home + "/.local/share"
}

// MergeArray merges declared string entries into a JSON array at the given
// path within root. Declared entries come first, then user entries that
// aren't duplicates. Deduplication uses canonical paths (filepath.Clean +
// home expansion).
//
// When StripStaleWorktree is true, entries matching the worktree pattern
// whose path no longer exists on disk are pruned.
func MergeArray(root []byte, arrayPath string, declared []string, opts MergeArrayOptions) (*MergeArrayResult, error) {
	var doc map[string]any
	if len(root) > 0 {
		if err := json.Unmarshal(root, &doc); err != nil {
			return nil, fmt.Errorf("parse existing JSON: %w", err)
		}
	}
	if doc == nil {
		doc = map[string]any{}
	}

	// Read existing entries.
	var existing []string
	if raw, ok := doc[arrayPath]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					existing = append(existing, s)
				}
			}
		case []string:
			existing = v
		default:
			return nil, fmt.Errorf("existing .%s is not an array", arrayPath)
		}
	}

	// Build canonical set of declared entries for dedup.
	declaredSet := make(map[string]bool, len(declared))
	for _, d := range declared {
		declaredSet[canonicalPath(d)] = true
	}

	// Filter existing entries: keep if not a duplicate of declared
	// and (if StripStaleWorktree) not a stale worktree entry.
	var preserved []string
	var prunedCount int
	for _, e := range existing {
		// Dedup against declared entries.
		if declaredSet[canonicalPath(e)] {
			continue // will be replaced by declared version
		}

		// Worktree stripping.
		if opts.StripStaleWorktree && isStaleWorktree(e) {
			prunedCount++
			continue
		}

		preserved = append(preserved, e)
	}

	// Count added = declared entries not already in existing.
	existingSet := make(map[string]bool, len(existing))
	for _, e := range existing {
		existingSet[canonicalPath(e)] = true
	}
	var addedCount int
	for _, d := range declared {
		if !existingSet[canonicalPath(d)] {
			addedCount++
		}
	}

	// Build merged array: declared first, then preserved.
	merged := make([]any, 0, len(declared)+len(preserved))
	for _, d := range declared {
		merged = append(merged, d)
	}
	for _, p := range preserved {
		merged = append(merged, p)
	}

	doc[arrayPath] = merged

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	out = append(out, '\n')

	return &MergeArrayResult{
		Bytes:          out,
		PrunedCount:    prunedCount,
		AddedCount:     addedCount,
		PreservedCount: len(preserved),
	}, nil
}

// canonicalPath normalizes a path for dedup: filepath.Clean + ~ → home expansion.
func canonicalPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			p = home + p[1:]
		}
	}
	return filepath.Clean(p)
}

// isStaleWorktree checks if a path matches the worktree pattern AND
// the path no longer exists on disk.
func isStaleWorktree(p string) bool {
	if !buildWorktreePattern().MatchString(p) {
		return false
	}
	// Only prune if the path doesn't exist.
	_, err := os.Stat(p)
	return err != nil
}
