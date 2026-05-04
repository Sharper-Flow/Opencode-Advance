package maintain

import (
	"context"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
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
	Target         string    `json:"target,omitempty"`
	Branch         string    `json:"branch,omitempty"`
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
	Now            func() time.Time
	GateCheck      func(context.Context, Options) GateReport
	LoadStack      func(string) (*cfg.Stack, error)
	InspectAdvance func(context.Context, cfg.Plugin) (AdvanceInspectReport, error)
	DriftCheck     func(context.Context, string, cfg.Plugin) (PluginDriftReport, error)
}

func NewPlanner() Planner {
	return Planner{
		Now:            time.Now,
		GateCheck:      CheckSessionGate,
		LoadStack:      cfg.Load,
		InspectAdvance: InspectAdvancePlugin,
		DriftCheck:     DetectPluginDrift,
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
	if opts.ConfigPath != "" && (opts.IncludeMerge || opts.IncludeRebuild) {
		loadStack := p.LoadStack
		if loadStack == nil {
			loadStack = cfg.Load
		}
		stack, err := loadStack(opts.ConfigPath)
		if err != nil {
			return plan, err
		}
		if opts.IncludeMerge {
			if err := p.addMergeCandidates(ctx, &plan, stack, gate); err != nil {
				return plan, err
			}
		}
		if opts.IncludeRebuild {
			if err := p.addRebuildActions(ctx, &plan, stack, gate); err != nil {
				return plan, err
			}
		}
	}
	return plan, nil
}

func (p Planner) addMergeCandidates(ctx context.Context, plan *Plan, stack *cfg.Stack, gate GateReport) error {
	inspectAdvance := p.InspectAdvance
	if inspectAdvance == nil {
		inspectAdvance = InspectAdvancePlugin
	}
	advancePlugin, ok := stack.Plugins["advance"]
	if !ok || !advancePlugin.IsEnabled() || advancePlugin.Checkout == "" {
		return nil
	}
	report, err := inspectAdvance(ctx, advancePlugin)
	if err != nil {
		return err
	}
	for _, candidate := range report.Candidates() {
		branch := candidate.Branch
		if branch == "" {
			branch = "change/" + candidate.ChangeID
		}
		mergeCandidate := MergeCandidate{
			ChangeID: candidate.ChangeID,
			Branch:   branch,
			Eligible: candidate.Eligible && candidate.ReleaseGate == "done",
		}
		if !mergeCandidate.Eligible {
			mergeCandidate.Reason = candidate.IneligibleReason()
		}
		plan.MergeCandidates = append(plan.MergeCandidates, mergeCandidate)
		if mergeCandidate.Eligible {
			blocked := gate.Status == GateStatusBlocked
			plan.Actions = append(plan.Actions, Action{
				ID:             "merge:" + candidate.ChangeID,
				Kind:           "merge",
				Target:         "advance",
				Branch:         branch,
				WouldRun:       true,
				ExecuteAllowed: !blocked,
				BlockedBy:      gate.Blockers,
				Evidence:       []string{"archive status verified", "release gate done"},
			})
		}
	}
	return nil
}

func (p Planner) addRebuildActions(ctx context.Context, plan *Plan, stack *cfg.Stack, gate GateReport) error {
	driftCheck := p.DriftCheck
	if driftCheck == nil {
		driftCheck = DetectPluginDrift
	}
	for name, plugin := range stack.Plugins {
		if !plugin.IsEnabled() || plugin.Checkout == "" {
			continue
		}
		drift, err := driftCheck(ctx, name, plugin)
		if err != nil {
			return err
		}
		plan.PluginDrift = append(plan.PluginDrift, PluginDrift{Plugin: name, Status: string(drift.Status), Path: drift.MarkerPath})
		if drift.Status == DriftStatusFresh {
			continue
		}
		blocked := gate.Status == GateStatusBlocked
		plan.Rebuilds = append(plan.Rebuilds, RebuildAction{Plugin: name, Needed: true, Reason: drift.Reason})
		plan.Actions = append(plan.Actions, Action{
			ID:             "rebuild:" + name,
			Kind:           "rebuild",
			Target:         name,
			WouldRun:       true,
			ExecuteAllowed: !blocked,
			BlockedBy:      gate.Blockers,
			Evidence:       []string{"plugin drift status: " + string(drift.Status)},
		})
	}
	return nil
}
