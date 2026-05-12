// Package advstatus reads ADV external state and formats it for the tmux
// status bar and CLI output. It replaces lib/adv_status.sh with typed,
// testable Go code.
package advstatus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// QueryMode enumerates the --query values.
type QueryMode string

const (
	QueryActiveChange    QueryMode = "active-change"
	QueryTemporalHealth  QueryMode = "temporal-health"
	QueryBranchSafety    QueryMode = "branch-safety"
	QueryWorktrees       QueryMode = "worktrees"
	QueryWorkspaceLookup QueryMode = "workspace-lookup"
)

// DefaultBranches are branch names considered "default" for trunk guard.
// Must match adv_status.sh:332 (main|master|trunk|develop).
var DefaultBranches = map[string]bool{
	"main":    true,
	"master":  true,
	"trunk":   true,
	"develop": true,
}

// ActionableWorktreeStatuses are worktree states that indicate active work.
// Terminal states (merged, stale, deleted) must NOT trigger branch safety warning.
// Must match adv_status.sh:317.
var ActionableWorktreeStatuses = map[string]bool{
	"active":         true,
	"idle":           true,
	"materializing":  true,
	"setup_failed":   true,
	"pending_delete": true,
	"unmaterialized": true,
}

// ChangeSummary is the parsed summary of a single change.
type ChangeSummary struct {
	ID          string `json:"id"`
	ShortID     string `json:"shortId"`
	CurrentGate string `json:"currentGate"`
	Status      string `json:"status"`
}

// TemporalHealth is the result of a Temporal TCP probe.
type TemporalHealth struct {
	Reachable bool   `json:"reachable"`
	Address   string `json:"address,omitempty"`
	Glyph     string `json:"glyph,omitempty"`
}

// BranchSafetyResult is the trunk guard assessment.
type BranchSafetyResult struct {
	Unsafe bool   `json:"unsafe"`
	Branch string `json:"branch"`
	Glyph  string `json:"glyph,omitempty"`
}

// WorkspaceStateResult is the highest-priority workspace state glyph.
type WorkspaceStateResult struct {
	Status string `json:"status,omitempty"`
	Glyph  string `json:"glyph,omitempty"`
}

// WorkspaceLookupEntry is a single changeId:status pair.
type WorkspaceLookupEntry struct {
	ChangeID string `json:"changeId"`
	Status   string `json:"status"`
}

// IsDefaultBranch returns true if the branch name is a recognized default branch.
func IsDefaultBranch(branch string) bool {
	return DefaultBranches[branch]
}

// IsActionableWorktree returns true if the worktree status indicates active work
// (not a terminal state like merged/stale/deleted).
func IsActionableWorktree(status string) bool {
	return ActionableWorktreeStatuses[status]
}

// StatusToGlyph returns a tmux-colored glyph for a workspace status, or empty
// string for normal/unknown statuses.
func StatusToGlyph(status string) string {
	switch status {
	case "setup_failed":
		return "#[fg=#E55353]✗#[default]"
	case "stale":
		return "#[fg=#A8A6A3]ѻ#[default]"
	case "merged":
		return "#[fg=#57AB5A]✓#[default]"
	case "pending_delete":
		return "#[fg=#A8A6A3]␡#[default]"
	default:
		return ""
	}
}

// FormatChangeSummaryText returns the text format for a change summary:
// {shortId}:{currentGate} or {shortId}:✓
func FormatChangeSummaryText(s *ChangeSummary) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s", s.ShortID, s.CurrentGate)
}

// FormatTemporalHealthText returns the text glyph for temporal health.
func FormatTemporalHealthText(h *TemporalHealth) string {
	if h == nil {
		return ""
	}
	return h.Glyph
}

// FormatWorkspaceLookupTSV returns changeId:status lines for shell consumption.
func FormatWorkspaceLookupTSV(entries []WorkspaceLookupEntry) string {
	var lines []string
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("%s:%s", e.ChangeID, e.Status))
	}
	return strings.Join(lines, "\n")
}

// FormatJSON returns a JSON envelope {query, result} for any query output.
func FormatJSON(query string, result any) (string, error) {
	envelope := map[string]any{
		"query":  query,
		"result": result,
	}
	b, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("marshal JSON: %w", err)
	}
	return string(b), nil
}

// shortenChangeID truncates a change ID to ~12 chars if longer than 15.
func shortenChangeID(id string) string {
	if len(id) > 15 {
		return id[:12] + "..."
	}
	return id
}

// changeJSON is the minimal shape we parse from change.json.
type changeJSON struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Gates  map[string]struct {
		Status string `json:"status"`
	} `json:"gates"`
}

// readChangeJSON reads and parses a change.json file.
// Returns nil with no error when file is missing or unreadable (graceful degradation).
func readChangeJSON(changeDir string) (*changeJSON, error) {
	path := filepath.Join(changeDir, "change.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil // graceful: missing file is normal
	}
	var cj changeJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return nil, nil // graceful: malformed JSON
	}
	return &cj, nil
}

// findFirstPendingGate returns the first gate with status != "done", or "✓" if all done.
func findFirstPendingGate(gates map[string]struct {
	Status string `json:"status"`
}) string {
	// Gate order matches ADV 7-gate lifecycle.
	order := []string{"proposal", "discovery", "design", "planning", "execution", "acceptance", "release"}
	for _, gate := range order {
		if g, ok := gates[gate]; ok {
			if g.Status != "done" {
				return gate
			}
		}
	}
	return "✓"
}
