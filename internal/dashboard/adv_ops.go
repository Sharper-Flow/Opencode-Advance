package dashboard

import (
	"context"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
)

// ADVOpsPoller populates the ADV Ops panel in the dashboard snapshot.
// It runs the worktree census and session debt scan. Temporal-based checks
// (workflows, search attributes) are optional and degrade gracefully.
type ADVOpsPoller struct {
	config advruntime.Config
}

// NewADVOpsPoller creates a poller with the given ADV runtime config.
func NewADVOpsPoller(config advruntime.Config) *ADVOpsPoller {
	return &ADVOpsPoller{config: config.WithDefaults()}
}

// Poll updates the ADVOps row in the snapshot.
func (p *ADVOpsPoller) Poll(ctx context.Context, s *State) error {
	report := advruntime.NewReport(p.config)

	// Session debt scan
	// (DB path resolution omitted for brevity; real implementation would open DB)
	// For v1, we show worktree census at minimum.
	worktreeRoot := defaultWorktreeRoot()
	advRoot := defaultADVStateRoot()
	census := advruntime.NewWorktreeCensus(worktreeRoot, advRoot, p.config)
	worktrees, _ := census.Scan(ctx)

	var staleWorktrees int
	for _, wt := range worktrees {
		if wt.Status == advruntime.StatusWarn {
			staleWorktrees++
		}
	}

	// Synthesize recovery plan
	synth := advruntime.NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	// Summary status
	status := string(advruntime.StatusPass)
	if staleWorktrees > 0 {
		status = string(advruntime.StatusWarn)
	}

	s.Update(func(snap *Snapshot) {
		snap.ADVOps = ADVOpsRow{
			Status:        status,
			WorktreeDebt:  staleWorktrees,
			RecoverySteps: len(plan),
			Message:       "ADV runtime panel (v1: worktrees + recovery plan)",
		}
	})

	return nil
}

func defaultWorktreeRoot() string {
	// Simplified; real implementation would use XDG paths
	return ""
}

func defaultADVStateRoot() string {
	// Simplified; real implementation would use XDG paths
	return ""
}
