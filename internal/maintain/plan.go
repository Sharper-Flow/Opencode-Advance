package maintain

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

const SchemaVersion = 1

type GateStatus string

const (
	GateStatusPass    GateStatus = "pass"
	GateStatusBlocked GateStatus = "blocked"
	GateStatusWarn    GateStatus = "warn"
)

type Options struct {
	ProjectRoot    string
	ConfigPath     string
	DryRun         bool
	Execute        bool
	IncludeMerge   bool
	IncludeRebuild bool
	IncludeCleanup bool
}

type Plan struct {
	SchemaVersion     int                `json:"schema_version"`
	GeneratedAt       time.Time          `json:"generated_at"`
	ProjectRoot       string             `json:"project_root"`
	DefaultBranch     string             `json:"default_branch"`
	SessionGate       GateReport         `json:"session_gate"`
	PluginDrift       []PluginDrift      `json:"plugin_drift,omitempty"`
	MergeCandidates   []MergeCandidate   `json:"merge_candidates,omitempty"`
	Rebuilds          []RebuildAction    `json:"rebuilds,omitempty"`
	CleanupCandidates []CleanupCandidate `json:"cleanup_candidates,omitempty"`
	TemporalHealth    []HealthFinding    `json:"temporal_health,omitempty"`
	Actions           []Action           `json:"actions,omitempty"`
	Blockers          []Blocker          `json:"blockers,omitempty"`
}

type GateReport struct {
	Status   GateStatus `json:"status"`
	Blockers []Blocker  `json:"blockers,omitempty"`
}

type Blocker struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type Action struct {
	ID             string    `json:"id"`
	Kind           string    `json:"kind"`
	WouldRun       bool      `json:"would_run"`
	ExecuteAllowed bool      `json:"execute_allowed"`
	BlockedBy      []Blocker `json:"blocked_by,omitempty"`
	Evidence       []string  `json:"evidence,omitempty"`
}

type PluginDrift struct {
	Plugin string `json:"plugin"`
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
}

type MergeCandidate struct {
	ChangeID string `json:"change_id"`
	Branch   string `json:"branch"`
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

type RebuildAction struct {
	Plugin string `json:"plugin"`
	Needed bool   `json:"needed"`
	Reason string `json:"reason,omitempty"`
}

type CleanupCandidate struct {
	Branch   string `json:"branch"`
	Path     string `json:"path"`
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

type HealthFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type Planner struct {
	Now       func() time.Time
	GateCheck func(context.Context, Options) GateReport
}

func NewPlanner() Planner {
	return Planner{
		Now:       time.Now,
		GateCheck: CheckSessionGate,
	}
}

func BuildPlan(ctx context.Context, opts Options) (Plan, error) {
	return NewPlanner().Build(ctx, opts)
}

func (p Planner) Build(ctx context.Context, opts Options) (Plan, error) {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	gateCheck := p.GateCheck
	if gateCheck == nil {
		gateCheck = CheckSessionGate
	}
	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		if wd, err := os.Getwd(); err == nil {
			projectRoot = wd
		}
	}
	projectRoot = filepath.Clean(projectRoot)
	gate := gateCheck(ctx, opts)
	plan := Plan{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   now().UTC(),
		ProjectRoot:   projectRoot,
		DefaultBranch: "trunk",
		SessionGate:   gate,
	}
	if gate.Status == GateStatusBlocked {
		plan.Blockers = append(plan.Blockers, gate.Blockers...)
	}
	return plan, nil
}
