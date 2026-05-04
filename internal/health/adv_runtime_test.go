package health

import (
	"context"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestCheckAdvRuntime_NoTemporalConfigSkipsGracefully(t *testing.T) {
	stack := &cfg.Stack{} // No Temporal section
	checks, err := CheckAdvRuntime(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckAdvRuntime: %v", err)
	}

	var found bool
	for _, c := range checks {
		if c.Name == "adv-runtime.temporal" {
			found = true
			if c.Status != StatusPass {
				t.Fatalf("temporal status = %q, want pass when not configured", c.Status)
			}
			if c.Message == "" {
				t.Fatal("temporal check message empty")
			}
		}
	}
	if !found {
		t.Fatal("missing adv-runtime.temporal check")
	}
}

func TestMapSearchAttribute_PassWarnFail(t *testing.T) {
	tests := []struct {
		input      advruntime.SearchAttributeStatus
		wantStatus Status
	}{
		{input: advruntime.SearchAttributeStatus{Status: advruntime.StatusPass}, wantStatus: StatusPass},
		{input: advruntime.SearchAttributeStatus{Status: advruntime.StatusWarn}, wantStatus: StatusWarn},
		{input: advruntime.SearchAttributeStatus{Status: advruntime.StatusFail}, wantStatus: StatusFail},
		{input: advruntime.SearchAttributeStatus{Status: advruntime.StatusUnknown}, wantStatus: StatusWarn},
	}
	for _, tt := range tests {
		got := mapSearchAttribute(tt.input)
		if got.Status != tt.wantStatus {
			t.Fatalf("mapSearchAttribute(%q) = %q, want %q", tt.input.Status, got.Status, tt.wantStatus)
		}
	}
}

func TestMapWorkflowQueue_StatusMapping(t *testing.T) {
	tests := []struct {
		input      advruntime.WorkflowQueueStatus
		wantStatus Status
	}{
		{input: advruntime.WorkflowQueueStatus{Status: advruntime.StatusPass}, wantStatus: StatusPass},
		{input: advruntime.WorkflowQueueStatus{Status: advruntime.StatusWarn, Pollers: 0, RunningWorkflows: 10}, wantStatus: StatusWarn},
		{input: advruntime.WorkflowQueueStatus{Status: advruntime.StatusFail}, wantStatus: StatusFail},
	}
	for _, tt := range tests {
		got := mapWorkflowQueue(tt.input)
		if got.Status != tt.wantStatus {
			t.Fatalf("mapWorkflowQueue(%q) = %q, want %q", tt.input.Status, got.Status, tt.wantStatus)
		}
	}
}

func TestMapSessionDebt_WarnWhenRepairable(t *testing.T) {
	debt := advruntime.SessionDebtFinding{
		Status:     advruntime.StatusWarn,
		Repairable: 5,
		Message:    "5 repairable stale rows",
		Hint:       "run oca session doctor",
	}
	got := mapSessionDebt(debt, "/path/to/db")
	if got.Status != StatusWarn {
		t.Fatalf("status = %q, want warn", got.Status)
	}
	if got.Message == "" {
		t.Fatal("message empty")
	}
	if got.Hint == "" {
		t.Fatal("hint empty")
	}
}

func TestMapWorktree_StaleWorktree(t *testing.T) {
	wt := advruntime.WorktreeFinding{
		Path:    "/wt/change/old",
		Branch:  "change/old",
		Status:  advruntime.StatusWarn,
		Age:     8 * 24 * time.Hour,
		Message: "stale",
		Hint:    "review",
	}
	got := mapWorktree(wt)
	if got.Status != StatusWarn {
		t.Fatalf("status = %q, want warn", got.Status)
	}
	if got.Name != "adv-runtime.worktree.change/old" {
		t.Fatalf("name = %q, want adv-runtime.worktree.change/old", got.Name)
	}
}

func TestMapFinding_ErrorSeverity(t *testing.T) {
	finding := advruntime.Finding{
		Code:     "ADV_TEMPORAL_UNAVAILABLE",
		Severity: advruntime.SeverityError,
		Message:  "Temporal down",
		Hint:     "start it",
	}
	got := mapFinding(finding)
	if got.Status != StatusFail {
		t.Fatalf("status = %q, want fail for error severity", got.Status)
	}
}

func TestMapFinding_WarnSeverity(t *testing.T) {
	finding := advruntime.Finding{
		Code:     "ADV_QUEUE_STALE",
		Severity: advruntime.SeverityWarn,
		Message:  "Queue stale",
		Hint:     "restart worker",
	}
	got := mapFinding(finding)
	if got.Status != StatusWarn {
		t.Fatalf("status = %q, want warn", got.Status)
	}
}

func TestCheckAdvRuntime_MissingHomeEmitsWarnings(t *testing.T) {
	// Simulate unresolvable home directory by unsetting HOME and XDG_DATA_HOME.
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("XDG_DATA_HOME", "")

	stack := &cfg.Stack{}
	checks, err := CheckAdvRuntime(context.Background(), stack, Options{})
	if err != nil {
		t.Fatalf("CheckAdvRuntime should not error: %v", err)
	}

	var sessionDebtFound, worktreeFound bool
	for _, c := range checks {
		if c.Name == "adv-runtime.session-debt" {
			sessionDebtFound = true
			if c.Status != StatusWarn {
				t.Fatalf("session-debt status = %q, want warn when HOME unset", c.Status)
			}
		}
		if c.Name == "adv-runtime.worktree" {
			worktreeFound = true
			if c.Status != StatusWarn {
				t.Fatalf("worktree status = %q, want warn when HOME unset", c.Status)
			}
		}
	}
	if !sessionDebtFound {
		t.Fatal("missing adv-runtime.session-debt warning when HOME unset")
	}
	if !worktreeFound {
		t.Fatal("missing adv-runtime.worktree warning when HOME unset")
	}
}

func TestAdvRuntimeScopeIsRegistered(t *testing.T) {
	scopes := KnownScopes()
	found := false
	for _, s := range scopes {
		if s == "adv-runtime" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("adv-runtime not in known scopes: %v", scopes)
	}
}
