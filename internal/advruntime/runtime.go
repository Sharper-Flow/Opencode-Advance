// Package advruntime contains read-only ADV runtime health models used by
// doctor, session preflight, recovery planning, and dashboard surfaces.
package advruntime

import "time"

const (
	DefaultAddress   = "127.0.0.1:7233"
	DefaultNamespace = "default"
)

type Status string

const (
	StatusPass    Status = "pass"
	StatusWarn    Status = "warn"
	StatusFail    Status = "fail"
	StatusUnknown Status = "unknown"
)

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

// Thresholds are tunable cutoffs for runtime debt classification. They are
// intentionally conservative and mutation-free; callers decide how to render
// warnings or recovery hints.
type Thresholds struct {
	MaxWorkflows       int           `json:"max_workflows"`
	WorkflowStaleAfter time.Duration `json:"workflow_stale_after"`
	SessionStaleAfter  time.Duration `json:"session_stale_after"`
	WorktreeStaleAfter time.Duration `json:"worktree_stale_after"`
	TaskQueueNoPoller  time.Duration `json:"task_queue_no_poller"`
}

func DefaultThresholds() Thresholds {
	return Thresholds{
		MaxWorkflows:       500,
		WorkflowStaleAfter: 30 * time.Minute,
		SessionStaleAfter:  24 * time.Hour,
		WorktreeStaleAfter: 7 * 24 * time.Hour,
		TaskQueueNoPoller:  5 * time.Minute,
	}
}

// Config captures runtime scan inputs and threshold overrides. Zero values are
// normalized through WithDefaults before use.
type Config struct {
	Address            string        `json:"address"`
	Namespace          string        `json:"namespace"`
	ProjectID          string        `json:"project_id,omitempty"`
	ChangeID           string        `json:"change_id,omitempty"`
	MaxWorkflows       int           `json:"max_workflows"`
	WorkflowStaleAfter time.Duration `json:"workflow_stale_after"`
	SessionStaleAfter  time.Duration `json:"session_stale_after"`
	WorktreeStaleAfter time.Duration `json:"worktree_stale_after"`
	TaskQueueNoPoller  time.Duration `json:"task_queue_no_poller"`
}

func (c Config) WithDefaults() Config {
	if c.Address == "" {
		c.Address = DefaultAddress
	}
	if c.Namespace == "" {
		c.Namespace = DefaultNamespace
	}
	thresholds := DefaultThresholds()
	if c.MaxWorkflows == 0 {
		c.MaxWorkflows = thresholds.MaxWorkflows
	}
	if c.WorkflowStaleAfter == 0 {
		c.WorkflowStaleAfter = thresholds.WorkflowStaleAfter
	}
	if c.SessionStaleAfter == 0 {
		c.SessionStaleAfter = thresholds.SessionStaleAfter
	}
	if c.WorktreeStaleAfter == 0 {
		c.WorktreeStaleAfter = thresholds.WorktreeStaleAfter
	}
	if c.TaskQueueNoPoller == 0 {
		c.TaskQueueNoPoller = thresholds.TaskQueueNoPoller
	}
	return c
}

type Report struct {
	GeneratedAt      time.Time               `json:"generated_at"`
	Config           Config                  `json:"config"`
	Summary          Summary                 `json:"summary"`
	WorkflowQueues   []WorkflowQueueStatus   `json:"workflow_queues"`
	SearchAttributes []SearchAttributeStatus `json:"search_attributes"`
	SessionDebt      []SessionDebtFinding    `json:"session_debt"`
	Worktrees        []WorktreeFinding       `json:"worktrees"`
	Findings         []Finding               `json:"findings"`
	RecoveryPlan     []RecoveryStep          `json:"recovery_plan,omitempty"`
}

func NewReport(config Config) Report {
	config = config.WithDefaults()
	return Report{
		GeneratedAt:      time.Now().UTC(),
		Config:           config,
		Summary:          Summary{Status: StatusPass},
		WorkflowQueues:   []WorkflowQueueStatus{},
		SearchAttributes: []SearchAttributeStatus{},
		SessionDebt:      []SessionDebtFinding{},
		Worktrees:        []WorktreeFinding{},
		Findings:         []Finding{},
	}
}

type Summary struct {
	Status                  Status `json:"status"`
	Projects                int    `json:"projects"`
	WorkflowQueues          int    `json:"workflow_queues"`
	StaleWorkflowQueues     int    `json:"stale_workflow_queues"`
	MissingSearchAttributes int    `json:"missing_search_attributes"`
	SessionDebt             int    `json:"session_debt"`
	WorktreeDebt            int    `json:"worktree_debt"`
	Findings                int    `json:"findings"`
}

type WorkflowQueueStatus struct {
	ProjectID        string        `json:"project_id,omitempty"`
	TaskQueue        string        `json:"task_queue"`
	Status           Status        `json:"status"`
	Pollers          int           `json:"pollers"`
	Backlog          int64         `json:"backlog"`
	RunningWorkflows int           `json:"running_workflows"`
	OldestRunAge     time.Duration `json:"oldest_run_age,omitempty"`
	Message          string        `json:"message,omitempty"`
	Hint             string        `json:"hint,omitempty"`
}

type SearchAttributeStatus struct {
	Name     string `json:"name"`
	Expected string `json:"expected"`
	Actual   string `json:"actual,omitempty"`
	Status   Status `json:"status"`
	Message  string `json:"message,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

type SessionDebtFinding struct {
	Path       string        `json:"path"`
	Status     Status        `json:"status"`
	Repairable int           `json:"repairable"`
	Ignored    int           `json:"ignored"`
	OldestAge  time.Duration `json:"oldest_age,omitempty"`
	Message    string        `json:"message,omitempty"`
	Hint       string        `json:"hint,omitempty"`
}

type WorktreeFinding struct {
	Path    string        `json:"path"`
	Branch  string        `json:"branch,omitempty"`
	Status  Status        `json:"status"`
	Age     time.Duration `json:"age,omitempty"`
	Message string        `json:"message,omitempty"`
	Hint    string        `json:"hint,omitempty"`
}

type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Hint     string   `json:"hint,omitempty"`
}

type RecoveryStep struct {
	Order       int      `json:"order"`
	Title       string   `json:"title"`
	Command     []string `json:"command,omitempty"`
	DryRunOnly  bool     `json:"dry_run_only"`
	Destructive bool     `json:"destructive"`
	Reason      string   `json:"reason,omitempty"`
}
