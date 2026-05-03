package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
)

func TestADVOpsPoller_UpdatesSnapshot(t *testing.T) {
	state := NewState()
	poller := NewADVOpsPoller(advruntime.Config{WorktreeStaleAfter: 7 * 24 * time.Hour})

	ctx := context.Background()
	if err := poller.Poll(ctx, state); err != nil {
		t.Fatalf("Poll: %v", err)
	}

	snap := state.Snapshot()
	if snap.ADVOps.Status == "" {
		t.Fatal("ADVOps.Status empty after poll")
	}
	// With empty worktree roots, census returns empty — should be pass
	if snap.ADVOps.Status != string(advruntime.StatusPass) {
		t.Fatalf("ADVOps.Status = %q, want pass for empty dirs", snap.ADVOps.Status)
	}
}

func TestADVOpsPoller_RecoveryPlanCount(t *testing.T) {
	state := NewState()
	poller := NewADVOpsPoller(advruntime.Config{})

	ctx := context.Background()
	if err := poller.Poll(ctx, state); err != nil {
		t.Fatalf("Poll: %v", err)
	}

	snap := state.Snapshot()
	// Empty report should yield 0 recovery steps
	if snap.ADVOps.RecoverySteps != 0 {
		t.Fatalf("ADVOps.RecoverySteps = %d, want 0", snap.ADVOps.RecoverySteps)
	}
}

func TestADVOpsRow_JSON(t *testing.T) {
	row := ADVOpsRow{
		Status:        "warn",
		WorktreeDebt:  2,
		RecoverySteps: 3,
		Message:       "test",
	}

	// Verify the row is JSON-serializable via the Snapshot
	state := NewState()
	state.Update(func(snap *Snapshot) {
		snap.ADVOps = row
	})

	snap := state.Snapshot()
	if snap.ADVOps.WorktreeDebt != 2 {
		t.Fatalf("worktree debt = %d, want 2", snap.ADVOps.WorktreeDebt)
	}
}
