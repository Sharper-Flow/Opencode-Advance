package advruntime

// RecoveryPlanSynthesizer turns a runtime report into an ordered list of
// dry-run recovery steps. It does not mutate the report or execute any
// commands.
type RecoveryPlanSynthesizer struct{}

// NewRecoveryPlanSynthesizer creates a synthesizer.
func NewRecoveryPlanSynthesizer() *RecoveryPlanSynthesizer {
	return &RecoveryPlanSynthesizer{}
}

// Synthesize produces recovery steps from the given report. The report is not
// modified. Steps are ordered by priority: search attributes → queues →
// session debt → worktree debt.
func (s *RecoveryPlanSynthesizer) Synthesize(report Report) []RecoveryStep {
	var steps []RecoveryStep
	order := 1

	// Priority 1: Search attribute issues (foundational — visibility relies on these).
	for _, attr := range report.SearchAttributes {
		if attr.Status == StatusFail || attr.Status == StatusUnknown {
			steps = append(steps, RecoveryStep{
				Order:       order,
				Title:       "Register missing or fix incorrect ADV search attribute: " + attr.Name,
				Command:     []string{"oca", "adv", "recover", "--dry-run"},
				DryRunOnly:  true,
				Destructive: false,
				Reason:      "Search attribute " + attr.Name + " is " + string(attr.Status) + ": " + attr.Message,
			})
			order++
		}
	}

	// Priority 2: Stale queues / missing pollers.
	for _, queue := range report.WorkflowQueues {
		if queue.Status == StatusFail || queue.Status == StatusWarn {
			if queue.Pollers == 0 && queue.RunningWorkflows > 0 {
				steps = append(steps, RecoveryStep{
					Order:       order,
					Title:       "Restart worker for stale queue: " + queue.TaskQueue,
					Command:     []string{"oca", "adv", "recover", "--dry-run", "--project", queue.ProjectID},
					DryRunOnly:  true,
					Destructive: false,
					Reason:      queue.Message + " (" + queue.Hint + ")",
				})
				order++
			}
		}
	}

	// Priority 3: Session debt.
	for _, debt := range report.SessionDebt {
		if debt.Status == StatusWarn && debt.Repairable > 0 {
			steps = append(steps, RecoveryStep{
				Order:       order,
				Title:       "Clean up stale blank assistant messages in session DB",
				Command:     []string{"oca", "session", "doctor", "--apply", "--backup-dir", "<backup-dir>"},
				DryRunOnly:  true,
				Destructive: true,
				Reason:      debt.Message + " — " + debt.Hint,
			})
			order++
		}
	}

	// Priority 4: Worktree debt.
	for _, wt := range report.Worktrees {
		if wt.Status == StatusWarn {
			steps = append(steps, RecoveryStep{
				Order:       order,
				Title:       "Review stale worktree: " + wt.Branch,
				Command:     []string{"oca", "adv", "recover", "--dry-run"},
				DryRunOnly:  true,
				Destructive: false,
				Reason:      wt.Message + " — " + wt.Hint,
			})
			order++
		}
	}

	// Priority 5: ADV gate/archive split-brain hints from findings.
	for _, finding := range report.Findings {
		if finding.Code == "ADV_SEARCH_ATTRIBUTES_UNKNOWN" {
			// Already covered by search attribute steps; skip duplicate.
			continue
		}
		steps = append(steps, RecoveryStep{
			Order:       order,
			Title:       "Investigate ADV runtime finding: " + finding.Code,
			Command:     []string{"oca", "doctor", "--scope", "adv-runtime"},
			DryRunOnly:  true,
			Destructive: false,
			Reason:      finding.Message + " — " + finding.Hint,
		})
		order++
	}

	return steps
}
