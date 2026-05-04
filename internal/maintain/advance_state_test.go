package maintain

import (
	"context"
	"testing"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestPlannerBuild_ConsumesAdvanceInspectAndPlansOnlyVerifiedMerges(t *testing.T) {
	stack := &cfg.Stack{Plugins: cfg.PluginsSection{
		"advance": {Source: "https://github.com/Sharper-Flow/Advance.git", Checkout: "/repo/advance", Ref: "trunk"},
	}}
	planner := Planner{
		Now: func() time.Time { return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC) },
		GateCheck: func(context.Context, Options) GateReport {
			return GateReport{Status: GateStatusPass}
		},
		LoadStack: func(string) (*cfg.Stack, error) { return stack, nil },
		InspectAdvance: func(context.Context, cfg.Plugin) (AdvanceInspectReport, error) {
			return AdvanceInspectReport{
				SchemaVersion: 1,
				EligibleArchives: []AdvanceArchiveCandidate{
					{ChangeID: "verifiedChange", ReleaseGate: "done", Eligible: true},
					{ChangeID: "pendingRelease", ReleaseGate: "pending", Eligible: false},
				},
			}, nil
		},
		DriftCheck: func(context.Context, string, cfg.Plugin) (PluginDriftReport, error) {
			return PluginDriftReport{Plugin: "advance", Status: DriftStatusFresh}, nil
		},
	}

	plan, err := planner.Build(context.Background(), Options{ConfigPath: "stack.toml", IncludeMerge: true, IncludeRebuild: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.MergeCandidates) != 2 {
		t.Fatalf("MergeCandidates=%#v", plan.MergeCandidates)
	}
	if !plan.MergeCandidates[0].Eligible || plan.MergeCandidates[0].Branch != "change/verifiedChange" {
		t.Fatalf("verified candidate not planned correctly: %#v", plan.MergeCandidates[0])
	}
	if plan.MergeCandidates[1].Eligible || plan.MergeCandidates[1].Reason == "" {
		t.Fatalf("unverified candidate missing skip reason: %#v", plan.MergeCandidates[1])
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("Actions=%#v", plan.Actions)
	}
	if plan.Actions[0].ID != "merge:verifiedChange" || !plan.Actions[0].ExecuteAllowed {
		t.Fatalf("merge action=%#v", plan.Actions[0])
	}
}
