package dashboard

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
)

// ADVOpsPoller populates the ADV Ops panel in the dashboard snapshot.
// It runs the worktree census and a read-only session debt scan.
// Temporal-based checks (workflows, search attributes) are optional and
// degrade gracefully.
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

	// Session debt scan (read-only)
	var sessionDebt int
	dbPath := advruntime.DefaultOpenCodeDBPath()
	if dbPath != "" {
		if _, err := os.Stat(dbPath); err == nil {
			db, err := sql.Open("sqlite", dbPath)
			if err == nil {
				defer db.Close()
				scanner := advruntime.NewSessionDebtScanner(db, p.config)
				finding, err := scanner.Scan(ctx)
				if err == nil {
					sessionDebt = finding.Repairable
					report.SessionDebt = append(report.SessionDebt, finding)
				}
			}
		}
	}

	// Worktree census
	worktreeRoot := advruntime.DefaultWorktreeRoot()
	advRoot := advruntime.DefaultADVStateRoot()
	var staleWorktrees int
	if worktreeRoot != "" && advRoot != "" {
		census := advruntime.NewWorktreeCensus(worktreeRoot, advRoot, p.config)
		worktrees, err := census.Scan(ctx)
		if err == nil {
			report.Worktrees = worktrees
			for _, wt := range worktrees {
				if wt.Status == advruntime.StatusWarn {
					staleWorktrees++
				}
			}
		}
	}

	// Synthesize recovery plan
	synth := advruntime.NewRecoveryPlanSynthesizer()
	plan := synth.Synthesize(report)

	// Summary status
	status := string(advruntime.StatusPass)
	if staleWorktrees > 0 || sessionDebt > 0 {
		status = string(advruntime.StatusWarn)
	}

	s.Update(func(snap *Snapshot) {
		snap.ADVOps = ADVOpsRow{
			Status:        status,
			SessionDebt:   sessionDebt,
			WorktreeDebt:  staleWorktrees,
			RecoverySteps: len(plan),
			Message:       fmt.Sprintf("ADV runtime panel (v1: %d worktree(s), %d session debt)", staleWorktrees, sessionDebt),
		}
	})

	return nil
}
