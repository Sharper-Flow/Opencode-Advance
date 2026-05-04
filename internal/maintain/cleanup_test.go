package maintain

import (
	"context"
	"testing"
	"time"
)

func TestPlannerBuild_BlocksDirtyAndSessionActiveCleanupCandidates(t *testing.T) {
	planner := Planner{
		Now: func() time.Time { return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC) },
		GateCheck: func(context.Context, Options) GateReport {
			return GateReport{Status: GateStatusPass}
		},
		ListWorktrees: func(context.Context, string, string) ([]WorktreeInfo, error) {
			return []WorktreeInfo{
				{Branch: "change/mergedClean", Path: "/wt/merged", Clean: true, Merged: true, ProcessFree: true, SessionFree: true},
				{Branch: "change/dirty", Path: "/wt/dirty", Clean: false, Merged: true, ProcessFree: true, SessionFree: true},
				{Branch: "change/session", Path: "/wt/session", Clean: true, Merged: true, ProcessFree: true, SessionFree: false},
			}, nil
		},
	}

	plan, err := planner.Build(context.Background(), Options{ProjectRoot: "/repo/main", IncludeCleanup: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.CleanupCandidates) != 3 {
		t.Fatalf("CleanupCandidates=%#v", plan.CleanupCandidates)
	}
	if !plan.CleanupCandidates[0].Eligible {
		t.Fatalf("merged clean candidate should be eligible: %#v", plan.CleanupCandidates[0])
	}
	if plan.CleanupCandidates[1].Eligible || plan.CleanupCandidates[1].Reason == "" {
		t.Fatalf("dirty candidate should be blocked with reason: %#v", plan.CleanupCandidates[1])
	}
	if plan.CleanupCandidates[2].Eligible || plan.CleanupCandidates[2].Reason == "" {
		t.Fatalf("session candidate should be blocked with reason: %#v", plan.CleanupCandidates[2])
	}
	if len(plan.Actions) != 1 || plan.Actions[0].ID != "cleanup:change/mergedClean" {
		t.Fatalf("cleanup actions=%#v", plan.Actions)
	}
}
