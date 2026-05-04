// Package occupancy provides pane-state parsing, grouping, liveness
// classification, and reconciliation for OCA's worktree occupancy visibility.
package occupancy

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Liveness represents the detected state of a pane record.
type Liveness int

const (
	// Unknown means insufficient data to classify (no paneID, no tmux data).
	Unknown Liveness = iota
	// Active means the pane is confirmed live (tmux pane exists or recent heartbeat).
	Active
	// Stale means the pane state file exists but the tmux pane is gone.
	Stale
)

func (l Liveness) String() string {
	switch l {
	case Active:
		return "active"
	case Stale:
		return "stale"
	default:
		return "unknown"
	}
}

// PaneRecord represents a parsed pane state file (v1 or v2).
// All fields are optional except SessionID and Directory.
type PaneRecord struct {
	// Core fields (v1+)
	SchemaVersion int    `json:"schemaVersion,omitempty"`
	SessionID     string `json:"sessionID"`
	Directory     string `json:"directory"`
	Ts            int64  `json:"ts"`

	// V2 timing
	StartedAt  int64 `json:"startedAt,omitempty"`
	LastSeenAt int64 `json:"lastSeenAt,omitempty"`

	// Tmux metadata
	Socket      string `json:"socket,omitempty"`
	PaneID      string `json:"paneID,omitempty"`
	SessionName string `json:"sessionName,omitempty"`
	WindowID    string `json:"windowID,omitempty"`
	WindowName  string `json:"windowName,omitempty"`

	// Agent
	Agent string `json:"agent,omitempty"`

	// Git metadata
	GitRoot       string `json:"gitRoot,omitempty"`
	GitCommonDir  string `json:"gitCommonDir,omitempty"`
	ProjectID     string `json:"projectId,omitempty"`
	WorktreePath  string `json:"worktreePath,omitempty"`
	WorktreeBranch string `json:"worktreeBranch,omitempty"`

	// Reconciliation-enriched fields (not persisted)
	Liveness       Liveness `json:"-"`
	DirectoryDrift bool     `json:"directoryDrift,omitempty"`
}

// rawPane is the wire format for JSON parsing.
// json.Number is used for ts to handle both number and string forms.
type rawPane struct {
	SchemaVersion  int             `json:"schemaVersion"`
	SessionID      string          `json:"sessionID"`
	Directory      string          `json:"directory"`
	Ts             json.Number     `json:"ts"`
	StartedAt      json.Number     `json:"startedAt"`
	LastSeenAt     json.Number     `json:"lastSeenAt"`
	Socket         string          `json:"socket"`
	PaneID         string          `json:"paneID"`
	SessionName    string          `json:"sessionName"`
	WindowID       string          `json:"windowID"`
	WindowName     string          `json:"windowName"`
	Agent          string          `json:"agent"`
	GitRoot        string          `json:"gitRoot"`
	GitCommonDir   string          `json:"gitCommonDir"`
	ProjectID      string          `json:"projectId"`
	WorktreePath   string          `json:"worktreePath"`
	WorktreeBranch string          `json:"worktreeBranch"`
}

// Parse parses a single pane state JSON blob into PaneRecords.
// Returns successfully parsed records and any malformed input.
// Empty input returns empty slices with no error.
func Parse(data []byte) (records []PaneRecord, malformed []error) {
	if len(data) == 0 {
		return nil, nil
	}
	var raw rawPane
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, []error{fmt.Errorf("parse pane state: %w", err)}
	}

	ts, _ := raw.Ts.Int64()
	startedAt, _ := raw.StartedAt.Int64()
	lastSeenAt, _ := raw.LastSeenAt.Int64()

	sv := raw.SchemaVersion
	if sv == 0 {
		sv = 1 // missing schemaVersion means v1
	}

	r := PaneRecord{
		SchemaVersion:  sv,
		SessionID:      raw.SessionID,
		Directory:      raw.Directory,
		Ts:             ts,
		StartedAt:      startedAt,
		LastSeenAt:     lastSeenAt,
		Socket:         raw.Socket,
		PaneID:         raw.PaneID,
		SessionName:    raw.SessionName,
		WindowID:       raw.WindowID,
		WindowName:     raw.WindowName,
		Agent:          raw.Agent,
		GitRoot:        raw.GitRoot,
		GitCommonDir:   raw.GitCommonDir,
		ProjectID:      raw.ProjectID,
		WorktreePath:   raw.WorktreePath,
		WorktreeBranch: raw.WorktreeBranch,
	}
	return []PaneRecord{r}, nil
}

// DiscoverRecords scans all JSON files under stateDir (recursively through
// socket subdirectories) and returns parsed records, malformed counts, and
// any filesystem error. Missing directory returns empty results (no error).
func DiscoverRecords(stateDir string) (records []PaneRecord, malformed int, err error) {
	_, readErr := os.ReadDir(stateDir)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return nil, 0, nil
		}
		return nil, 0, readErr
	}

	var recs []PaneRecord
	bad := 0

	// Walk recursively to find all .json files under socket subdirs
	filepath.WalkDir(stateDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			bad++
			return nil // continue best-effort discovery while surfacing skipped entries
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			bad++
			return nil
		}
		parsed, malformedBatch := Parse(data)
		if len(malformedBatch) > 0 {
			bad += len(malformedBatch)
			return nil
		}
		recs = append(recs, parsed...)
		return nil
	})

	return recs, bad, nil
}

// GroupByKey determines the grouping dimension.
type GroupByKey int

const (
	// GroupByProject groups by projectId (falls back to gitRoot, then directory).
	GroupByProject GroupByKey = iota
	// GroupByWorktree groups by worktreePath (falls back to directory).
	GroupByWorktree
)

// GroupKey is the composite key for grouping.
type GroupKey struct {
	// ID holds the group identifier (projectId or worktreePath or directory fallback).
	ID string
}

// GroupBy groups records by the specified key dimension.
// Each group key uses the primary field if present, otherwise falls back
// to directory.
func GroupBy(records []PaneRecord, key GroupByKey) map[GroupKey][]PaneRecord {
	groups := make(map[GroupKey][]PaneRecord)
	for _, r := range records {
		var id string
		switch key {
		case GroupByProject:
			id = r.ProjectID
			if id == "" {
				id = r.GitRoot
			}
			if id == "" {
				id = r.Directory
			}
		case GroupByWorktree:
			id = r.WorktreePath
			if id == "" {
				id = r.Directory
			}
		}
		k := GroupKey{ID: id}
		groups[k] = append(groups[k], r)
	}
	return groups
}

// ClassifiedRecord is a PaneRecord with liveness determined.
type ClassifiedRecord struct {
	PaneRecord
	Liveness Liveness
}

// ClassifyLiveness determines liveness for each record based on live tmux panes.
// Records with a PaneID present in livePanes are Active.
// Records with a PaneID not in livePanes are Stale.
// Records without PaneID are Unknown.
func ClassifyLiveness(records []PaneRecord, livePanes map[string]bool, now time.Time) []ClassifiedRecord {
	results := make([]ClassifiedRecord, len(records))
	for i, r := range records {
		cr := ClassifiedRecord{PaneRecord: r}
		if r.PaneID == "" {
			cr.Liveness = Unknown
		} else if livePanes[r.PaneID] {
			cr.Liveness = Active
		} else {
			cr.Liveness = Stale
		}
		results[i] = cr
	}
	return results
}

// OccupancyWarning represents a worktree with multiple active occupants.
type OccupancyWarning struct {
	WorktreePath string
	ActiveCount  int
	Sessions     []string
}

// OccupancyWarnings returns warnings for worktrees with >1 active occupant.
func OccupancyWarnings(classified []ClassifiedRecord) []OccupancyWarning {
	// Count active per worktree
	type acc struct {
		count    int
		sessions []string
	}
	m := make(map[string]*acc)
	for _, c := range classified {
		if c.Liveness != Active {
			continue
		}
		wt := c.WorktreePath
		if wt == "" {
			wt = c.Directory
		}
		if m[wt] == nil {
			m[wt] = &acc{}
		}
		m[wt].count++
		m[wt].sessions = append(m[wt].sessions, c.SessionID)
	}

	var warnings []OccupancyWarning
	for wt, a := range m {
		if a.count > 1 {
			warnings = append(warnings, OccupancyWarning{
				WorktreePath: wt,
				ActiveCount:  a.count,
				Sessions:     a.sessions,
			})
		}
	}
	return warnings
}

// Reconcile combines liveness classification with tmux current-working-directory
// enrichment. When tmux cwd disagrees with pane state directory, the tmux value
// wins and DirectoryDrift is set.
func Reconcile(records []PaneRecord, livePanes map[string]bool, tmuxCwd map[string]string) []ClassifiedRecord {
	classified := ClassifyLiveness(records, livePanes, time.Now())

	for i := range classified {
		c := &classified[i]
		if c.PaneID == "" {
			continue
		}
		tmuxDir, hasTmux := tmuxCwd[c.PaneID]
		if !hasTmux {
			continue
		}
		// Tmux says pane cwd is tmuxDir; if pane state disagrees, prefer tmux
		if c.Directory != "" && c.Directory != tmuxDir {
			c.DirectoryDrift = true
		}
		c.Directory = tmuxDir
	}

	return classified
}
