package advruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WorktreeWorkspaceState represents the projected workspace state of a single
// ADV worktree as read from the ADV Temporal-backed worktree registry.
// OCA reads this as a non-authoritative, rebuildable projection — OCA never
// writes to the registry.
type WorktreeWorkspaceState struct {
	Branch             string `json:"branch"`
	Path               string `json:"path,omitempty"`
	Materialized      bool   `json:"materialized"`
	ChangeID          string `json:"changeId,omitempty"`
	Status            string `json:"status"`                       // active, idle, setup_failed, materializing, pending_delete, merged, stale, deleted
	SetupReady        bool   `json:"setupReady"`
	SetupFailureReason string `json:"setupFailureReason,omitempty"`
	BaseRef           string `json:"baseRef,omitempty"`
	HeadSha           string `json:"headSha,omitempty"`
	Source            string `json:"source,omitempty"`
}

// WorkspaceProjection holds all projected workspace states for a project.
type WorkspaceProjection struct {
	ProjectID string                   `json:"projectId"`
	Worktrees []WorktreeWorkspaceState `json:"worktrees"`
}

// ProjectWorkspaceStates reads the ADV worktree registry projection for the
// given projectID. Returns an empty projection (never nil) when the registry
// is unavailable or empty — OCA degrades gracefully.
//
// The registry lives in the ADV Temporal state store at:
//
//	$XDG_DATA_HOME/opencode/plugins/advance/{projectID}/
//
// OCA reads the cached project state snapshot if available, or returns empty.
// This is intentionally non-authoritative: ADV Temporal is the source of truth.
func ProjectWorkspaceStates(projectID string) (*WorkspaceProjection, error) {
	result := &WorkspaceProjection{
		ProjectID: projectID,
		Worktrees: []WorktreeWorkspaceState{},
	}

	if projectID == "" {
		return result, nil
	}

	advRoot := DefaultADVStateRoot()
	if advRoot == "" {
		return result, nil
	}

	// Try reading the cached project state snapshot.
	// The snapshot is written by the ADV plugin's ticker and contains the
	// full ProjectWorkflowState including worktree_registry.
	snapshotPath := filepath.Join(advRoot, projectID, "snapshot.json")
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		// Snapshot not available is a normal state — Temporal may be offline
		// or the project may not have any ADV state yet.
		return result, nil
	}

	var state struct {
		WorktreeRegistry map[string]struct {
			Branch             string `json:"branch"`
			Path               string `json:"path"`
			Materialized       bool   `json:"materialized"`
			ChangeID           string `json:"changeId"`
			Status             string `json:"status"`
			SetupReady         bool   `json:"setupReady"`
			SetupFailureReason string `json:"setupFailureReason"`
			BaseRef            string `json:"baseRef"`
			HeadSha            string `json:"headSha"`
			Source             string `json:"source"`
		} `json:"worktree_registry"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return result, fmt.Errorf("parse ADV snapshot for project %s: %w", projectID, err)
	}

	if state.WorktreeRegistry == nil {
		return result, nil
	}

	for _, rec := range state.WorktreeRegistry {
		result.Worktrees = append(result.Worktrees, WorktreeWorkspaceState{
			Branch:             rec.Branch,
			Path:               rec.Path,
			Materialized:       rec.Materialized,
			ChangeID:           rec.ChangeID,
			Status:             rec.Status,
			SetupReady:         rec.SetupReady,
			SetupFailureReason: rec.SetupFailureReason,
			BaseRef:            rec.BaseRef,
			HeadSha:            rec.HeadSha,
			Source:             rec.Source,
		})
	}

	return result, nil
}

// SessionView is a minimal session interface that OCA session structs satisfy.
// Defined here to avoid importing the session package (which requires tmux).
type SessionView struct {
	Name     string
	Attached bool
	Path     string
}

// EnrichedSession enriches an OCA session with projected ADV workspace state
// when available. Fields are informational only — OCA never writes them.
type EnrichedSession struct {
	Name     string `json:"name"`
	Attached bool   `json:"attached"`
	Path     string `json:"path"`

	// ADV workspace projection (empty when ADV unavailable)
	WorkspaceStatus    string `json:"workspaceStatus,omitempty"`
	WorkspaceChangeID  string `json:"workspaceChangeId,omitempty"`
	WorkspaceBranch    string `json:"workspaceBranch,omitempty"`
	WorkspaceSetupReady bool  `json:"workspaceSetupReady,omitempty"`
	WorkspaceFailure   string `json:"workspaceFailure,omitempty"`
}

// EnrichSessionsWithWorkspaceState enriches OCA session data with ADV
// workspace projection states. Returns enriched sessions; when the ADV
// projection is unavailable, sessions are returned with empty workspace
// fields (graceful degradation).
func EnrichSessionsWithWorkspaceState(
	sessions []SessionView,
	projection *WorkspaceProjection,
) []EnrichedSession {
	result := make([]EnrichedSession, len(sessions))
	for i, s := range sessions {
		result[i] = EnrichedSession{
			Name:     s.Name,
			Attached: s.Attached,
			Path:     s.Path,
		}

		if projection == nil {
			continue
		}

		// Match session to workspace by path or by change-derived name.
		for _, wt := range projection.Worktrees {
			if matchSessionToWorkspace(s.Name, s.Path, wt) {
				result[i].WorkspaceStatus = wt.Status
				result[i].WorkspaceChangeID = wt.ChangeID
				result[i].WorkspaceBranch = wt.Branch
				result[i].WorkspaceSetupReady = wt.SetupReady
				result[i].WorkspaceFailure = wt.SetupFailureReason
				break
			}
		}
	}
	return result
}

// matchSessionToWorkspace attempts to match an OCA session to an ADV workspace
// record by comparing paths, branch-derived window names, and change IDs.
func matchSessionToWorkspace(sessionName, sessionPath string, wt WorktreeWorkspaceState) bool {
	// Direct path match.
	if sessionPath != "" && wt.Path != "" && sessionPath == wt.Path {
		return true
	}

	// Session/window name derived from change ID.
	// Pattern B sessions use repo slug; windows within use change/<id> basename.
	if wt.ChangeID != "" && sessionName == wt.ChangeID {
		return true
	}
	if wt.Branch != "" {
		// Branch like "change/myChange" → window name "myChange"
		shortBranch := wt.Branch
		if prefix := "change/"; len(shortBranch) > len(prefix) && shortBranch[:len(prefix)] == prefix {
			shortBranch = shortBranch[len(prefix):]
		}
		if sessionName == shortBranch {
			return true
		}
	}

	return false
}
