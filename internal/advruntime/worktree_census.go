package advruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WorktreeCensus scans OCA worktree roots and ADV project state roots,
// classifying worktrees as active or stale. It never deletes anything.
type WorktreeCensus struct {
	worktreeRoot string
	advRoot      string
	config       Config
	nowFunc      func() time.Time
}

// NewWorktreeCensus creates a census scanner for the given root directories.
func NewWorktreeCensus(worktreeRoot, advRoot string, config Config) *WorktreeCensus {
	config = config.WithDefaults()
	return &WorktreeCensus{
		worktreeRoot: worktreeRoot,
		advRoot:      advRoot,
		config:       config,
		nowFunc:      time.Now,
	}
}

// Scan discovers worktrees across all project roots and returns findings.
// Returns an empty slice (not nil) if no worktrees are found.
func (c *WorktreeCensus) Scan(_ context.Context) ([]WorktreeFinding, error) {
	var findings []WorktreeFinding

	// Phase 1: Scan OCA worktree roots.
	// Layout: {worktreeRoot}/{projectID}/change/{changeID}/
	worktrees, err := c.scanWorktreeRoot()
	if err != nil {
		return nil, fmt.Errorf("scan worktree root: %w", err)
	}

	now := c.nowFunc()
	for _, wt := range worktrees {
		finding := WorktreeFinding{
			Path:   wt.path,
			Branch: wt.changeID,
			Status: StatusPass,
		}

		// Classify staleness by directory modification time.
		if now.Sub(wt.modTime) >= c.config.WorktreeStaleAfter {
			finding.Status = StatusWarn
			finding.Age = now.Sub(wt.modTime).Round(time.Second)
			finding.Message = fmt.Sprintf("worktree %s not modified for %s (threshold %s)",
				wt.changeID, finding.Age, c.config.WorktreeStaleAfter.Round(time.Hour))
			finding.Hint = "if change is archived and merged, run 'oca adv recover' or manually remove the worktree"
		}

		findings = append(findings, finding)
	}

	// Phase 2: Identify ADV state roots with no corresponding worktrees.
	// These are informational — orphan state isn't necessarily wrong, just worth noting.
	orphanHints := c.findOrphanADVState(findings)
	for _, hint := range orphanHints {
		findings = append(findings, WorktreeFinding{
			Path:    hint,
			Status:  StatusPass,
			Message: "ADV project state exists but has no active worktrees",
			Hint:    "this is informational; state is retained for future changes in this project",
		})
	}

	return findings, nil
}

type worktreeEntry struct {
	path      string
	projectID string
	changeID  string
	modTime   time.Time
}

func (c *WorktreeCensus) scanWorktreeRoot() ([]worktreeEntry, error) {
	if c.worktreeRoot == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(c.worktreeRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var worktrees []worktreeEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectDir := filepath.Join(c.worktreeRoot, entry.Name())
		changeDir := filepath.Join(projectDir, "change")

		changeEntries, err := os.ReadDir(changeDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			// Non-fatal: skip unreadable directories.
			continue
		}

		for _, ce := range changeEntries {
			if !ce.IsDir() {
				continue
			}
			wtPath := filepath.Join(changeDir, ce.Name())
			info, err := os.Stat(wtPath)
			if err != nil {
				continue
			}

			worktrees = append(worktrees, worktreeEntry{
				path:      wtPath,
				projectID: entry.Name(),
				changeID:  ce.Name(),
				modTime:   info.ModTime(),
			})
		}
	}

	return worktrees, nil
}

func (c *WorktreeCensus) findOrphanADVState(existingFindings []WorktreeFinding) []string {
	if c.advRoot == "" {
		return nil
	}

	entries, err := os.ReadDir(c.advRoot)
	if err != nil {
		return nil
	}

	// Build set of project IDs that have worktree findings.
	projectsInWorktrees := map[string]bool{}
	for _, f := range existingFindings {
		// Extract project ID from path: {advRoot}/{projectID}/...
		rel, err := filepath.Rel(c.worktreeRoot, f.Path)
		if err == nil {
			parts := strings.Split(filepath.ToSlash(rel), "/")
			if len(parts) >= 1 {
				projectsInWorktrees[parts[0]] = true
			}
		}
	}

	var orphans []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectID := entry.Name()
		if !projectsInWorktrees[projectID] {
			orphans = append(orphans, filepath.Join(c.advRoot, projectID))
		}
	}

	return orphans
}
