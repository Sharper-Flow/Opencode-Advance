package advstatus

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
)

// BranchSafety checks if a path is on a default branch while ADV has
// active worktree records. Returns colored tmux glyph when unsafe.
func BranchSafety(panePath, projectID string) (*BranchSafetyResult, error) {
	// Need git
	branch := gitBranch(panePath)
	if branch == "" || !IsDefaultBranch(branch) {
		return &BranchSafetyResult{Unsafe: false, Branch: branch}, nil
	}

	// Check if ADV has active worktrees for this project
	if !hasActionableWorktrees(projectID) {
		return &BranchSafetyResult{Unsafe: false, Branch: branch}, nil
	}

	return &BranchSafetyResult{
		Unsafe: true,
		Branch: branch,
		Glyph:  "#[fg=#E5A649]⚡#[default]",
	}, nil
}

// WorkspaceState returns the highest-priority workspace state glyph for a project.
// Priority: setup_failed > stale > pending_delete > merged > (active/idle/materializing = empty).
func WorkspaceState(projectID string) (*WorkspaceStateResult, error) {
	projection, err := advruntime.ProjectWorkspaceStates(projectID)
	if err != nil || projection == nil {
		return nil, nil
	}

	// Find highest-priority status across all worktrees.
	// setup_failed wins regardless of order.
	var found string
	for _, wt := range projection.Worktrees {
		switch wt.Status {
		case "setup_failed":
			// Always wins
			return &WorkspaceStateResult{
				Status: "setup_failed",
				Glyph:  StatusToGlyph("setup_failed"),
			}, nil
		case "stale":
			if found == "" || found == "merged" || found == "pending_delete" {
				found = "stale"
			}
		case "pending_delete":
			if found == "" || found == "merged" {
				found = "pending_delete"
			}
		case "merged":
			if found == "" {
				found = "merged"
			}
		}
	}

	if found == "" {
		return nil, nil
	}

	return &WorkspaceStateResult{
		Status: found,
		Glyph:  StatusToGlyph(found),
	}, nil
}

// WorkspaceLookup returns all worktree registry entries as changeId:status pairs.
func WorkspaceLookup(projectID string) ([]WorkspaceLookupEntry, error) {
	projection, err := advruntime.ProjectWorkspaceStates(projectID)
	if err != nil || projection == nil {
		return nil, nil
	}

	var entries []WorkspaceLookupEntry
	for _, wt := range projection.Worktrees {
		changeID := wt.ChangeID
		if changeID == "" {
			// Derive from branch name
			changeID = strings.TrimPrefix(wt.Branch, "change/")
		}
		if changeID != "" {
			entries = append(entries, WorkspaceLookupEntry{
				ChangeID: changeID,
				Status:   wt.Status,
			})
		}
	}
	return entries, nil
}

// hasActionableWorktrees checks if the project has any actionable (non-terminal)
// worktree records.
func hasActionableWorktrees(projectID string) bool {
	projection, err := advruntime.ProjectWorkspaceStates(projectID)
	if err != nil || projection == nil {
		return false
	}
	for _, wt := range projection.Worktrees {
		if IsActionableWorktree(wt.Status) {
			return true
		}
	}
	return false
}

// gitBranch returns the current git branch name for a directory path.
// Returns empty string on any error (not a git repo, detached HEAD, etc).
func gitBranch(panePath string) string {
	if panePath == "" {
		return ""
	}
	out, err := exec.Command("git", "-C", panePath, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// FormatBranchSafetyText returns the text output for branch safety query.
func FormatBranchSafetyText(r *BranchSafetyResult) string {
	if r == nil || !r.Unsafe {
		return ""
	}
	return r.Glyph
}

// FormatWorkspaceStateText returns the text output for workspace state query.
func FormatWorkspaceStateText(r *WorkspaceStateResult) string {
	if r == nil {
		return ""
	}
	return r.Glyph
}

// FormatTemporalHealthTextOrDefault returns the temporal health glyph or empty.
func FormatTemporalHealthTextOrDefault(h *TemporalHealth) string {
	if h == nil {
		return ""
	}
	return h.Glyph
}

// FormatActiveChangeTextOrDefault returns the change summary or empty.
func FormatActiveChangeTextOrDefault(s *ChangeSummary) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s", s.ShortID, s.CurrentGate)
}
