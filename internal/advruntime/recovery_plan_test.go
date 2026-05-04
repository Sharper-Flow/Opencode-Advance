package advruntime

import (
	"testing"
	"time"
)

func TestRecoveryPlanSynthesizer_EmptyReportReturnsNoSteps(t *testing.T) {
	report := NewReport(Config{})
	synth := NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	if len(plan) != 0 {
		t.Fatalf("plan = %d steps, want 0 for healthy report", len(plan))
	}
}

func TestRecoveryPlanSynthesizer_SearchAttributeIssuesFirst(t *testing.T) {
	report := NewReport(Config{})
	report.SearchAttributes = []SearchAttributeStatus{
		{Name: "AdvProjectId", Expected: "INDEXED_VALUE_TYPE_KEYWORD", Actual: "", Status: StatusFail},
	}
	report.Summary.MissingSearchAttributes = 1

	synth := NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	if len(plan) == 0 {
		t.Fatal("expected at least one recovery step")
	}

	first := plan[0]
	if first.Order != 1 {
		t.Fatalf("first step order = %d, want 1", first.Order)
	}
	if !contains(first.Title, "search attribute") {
		t.Fatalf("first step title = %q, want search-attribute related", first.Title)
	}
	if !first.DryRunOnly {
		t.Fatal("all v1 recovery steps must be dry-run only")
	}
}

func TestRecoveryPlanSynthesizer_StaleQueueStep(t *testing.T) {
	report := NewReport(Config{})
	report.WorkflowQueues = []WorkflowQueueStatus{
		{
			ProjectID:        "proj-abc",
			TaskQueue:        "advance-proj-abc",
			Status:           StatusFail,
			Pollers:          0,
			RunningWorkflows: 347,
			OldestRunAge:     30 * time.Minute,
			Message:          "no pollers and stale workflows",
		},
	}
	report.Summary.StaleWorkflowQueues = 1

	synth := NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	var found bool
	for _, step := range plan {
		if contains(step.Title, "queue") || contains(step.Title, "poller") {
			found = true
			if !step.DryRunOnly {
				t.Fatal("queue step must be dry-run only")
			}
			if step.Reason == "" {
				t.Fatal("queue step must have a reason")
			}
		}
	}
	if !found {
		t.Fatalf("no queue/poller recovery step in plan: %+v", plan)
	}
}

func TestRecoveryPlanSynthesizer_SessionDebtStep(t *testing.T) {
	report := NewReport(Config{})
	report.SessionDebt = []SessionDebtFinding{
		{
			Path:       "/path/to/opencode.db",
			Status:     StatusWarn,
			Repairable: 5,
			Ignored:    2,
			OldestAge:  10 * time.Minute,
		},
	}
	report.Summary.SessionDebt = 5

	synth := NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	var found bool
	for _, step := range plan {
		if contains(step.Title, "session") {
			found = true
			if !step.DryRunOnly {
				t.Fatal("session step must be dry-run only")
			}
			if len(step.Command) == 0 {
				t.Fatal("session step should suggest a command")
			}
		}
	}
	if !found {
		t.Fatalf("no session debt recovery step in plan: %+v", plan)
	}
}

func TestRecoveryPlanSynthesizer_WorktreeDebtStep(t *testing.T) {
	report := NewReport(Config{})
	report.Worktrees = []WorktreeFinding{
		{
			Path:    "/worktree/change/old-change",
			Branch:  "change/old-change",
			Status:  StatusWarn,
			Age:     8 * 24 * time.Hour,
			Message: "stale worktree",
		},
	}
	report.Summary.WorktreeDebt = 1

	synth := NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	var found bool
	for _, step := range plan {
		if contains(step.Title, "worktree") {
			found = true
			if !step.DryRunOnly {
				t.Fatal("worktree step must be dry-run only")
			}
		}
	}
	if !found {
		t.Fatalf("no worktree recovery step in plan: %+v", plan)
	}
}

func TestRecoveryPlanSynthesizer_OrdersStepsCorrectly(t *testing.T) {
	report := NewReport(Config{})
	// All issue types present
	report.SearchAttributes = []SearchAttributeStatus{
		{Name: "AdvProjectId", Status: StatusFail},
	}
	report.WorkflowQueues = []WorkflowQueueStatus{
		{TaskQueue: "advance-proj", Status: StatusFail, Pollers: 0, RunningWorkflows: 10},
	}
	report.SessionDebt = []SessionDebtFinding{
		{Status: StatusWarn, Repairable: 3},
	}
	report.Worktrees = []WorktreeFinding{
		{Status: StatusWarn, Age: 8 * 24 * time.Hour},
	}

	synth := NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	// Verify monotonic ordering
	for i := 1; i < len(plan); i++ {
		if plan[i].Order <= plan[i-1].Order {
			t.Fatalf("step ordering violated at index %d: %d <= %d", i, plan[i].Order, plan[i-1].Order)
		}
	}

	// Verify expected count: search attrs (1) + queue (1) + session (1) + worktree (1) = 4+
	if len(plan) < 4 {
		t.Fatalf("plan = %d steps, want >= 4", len(plan))
	}
}

func TestRecoveryPlanSynthesizer_NoMutation(t *testing.T) {
	report := NewReport(Config{})
	report.SearchAttributes = []SearchAttributeStatus{
		{Name: "AdvProjectId", Status: StatusFail},
	}

	synth := NewRecoveryPlanSynthesizer()
	_ = synth.Synthesize(report)

	// Report must remain unchanged
	if len(report.RecoveryPlan) != 0 {
		t.Fatal("synthesize must not mutate report.RecoveryPlan")
	}
	if report.Summary.Status != StatusPass {
		t.Fatalf("synthesize mutated report status: %q", report.Summary.Status)
	}
}

func TestRecoveryPlanSynthesizer_JSONRoundTrip(t *testing.T) {
	report := NewReport(Config{})
	report.WorkflowQueues = []WorkflowQueueStatus{
		{TaskQueue: "advance-proj", Status: StatusFail, Pollers: 0, RunningWorkflows: 10},
	}

	synth := NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	if len(plan) == 0 {
		t.Fatal("expected at least one step")
	}

	// Each step should have non-empty required fields
	for _, step := range plan {
		if step.Order == 0 {
			t.Fatalf("step order must be > 0: %+v", step)
		}
		if step.Title == "" {
			t.Fatalf("step title must not be empty: %+v", step)
		}
		if step.Reason == "" {
			t.Fatalf("step reason must not be empty: %+v", step)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(s != substr && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
