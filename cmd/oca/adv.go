package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
	"github.com/spf13/cobra"
)

func newAdvCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adv",
		Short: "ADV operator commands",
		Long:  "OpenCode Advance ADV runtime operator commands.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newAdvRecoverCmd(state))
	return cmd
}

func newAdvRecoverCmd(state *commandState) *cobra.Command {
	var (
		projectFilter string
		changeFilter  string
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "recover",
		Short: "Render ADV recovery plan (dry-run only)",
		Long: "Load the current ADV runtime report and render an ordered recovery plan. " +
			"This command is dry-run only in v1 — it does not mutate any state.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			if !dryRun {
				return newCLIError(2, "recover requires --dry-run in v1 (mutation not supported)")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			config := advruntime.Config{
				ProjectID: projectFilter,
				ChangeID:  changeFilter,
			}
			config = config.WithDefaults()

			// Try to load stack.toml for Temporal config
			temporalConfigured := false
			if stack, err := loadStack(state); err == nil && stack.Temporal != nil {
				if stack.Temporal.Address != "" {
					config.Address = stack.Temporal.Address
					temporalConfigured = true
				}
				if stack.Temporal.Namespace != "" {
					config.Namespace = stack.Temporal.Namespace
				}
			}

			report := collectReport(ctx, config, temporalConfigured)

			// Synthesize recovery plan
			synth := advruntime.NewRecoveryPlanSynthesizer()
			plan := synth.Synthesize(report)

			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]any{
					"dry_run":     true,
					"project":     projectFilter,
					"change":      changeFilter,
					"plan":        plan,
					"total_steps": len(plan),
				})
			}

			// Text output
			fmt.Fprintf(cmd.OutOrStdout(), "ADV Recovery Plan (dry-run)\n")
			if projectFilter != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Project filter: %s\n", projectFilter)
			}
			if changeFilter != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Change filter:  %s\n", changeFilter)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  Steps:          %d\n", len(plan))
			if len(plan) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\nNo recovery steps needed — ADV runtime looks healthy.\n")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nOrdered recovery steps:\n")
			for _, step := range plan {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%d. %s\n", step.Order, step.Title)
				if step.Reason != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "   Reason: %s\n", step.Reason)
				}
				if len(step.Command) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "   Command: %s\n", strings.Join(step.Command, " "))
				}
				if step.Destructive {
					fmt.Fprintf(cmd.OutOrStdout(), "   ⚠ Destructive\n")
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectFilter, "project", "", "Filter recovery plan by project ID")
	cmd.Flags().StringVar(&changeFilter, "change", "", "Filter recovery plan by change ID")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show recovery plan without executing (required in v1)")

	return cmd
}

// collectReport gathers a read-only ADV runtime report using existing scanners.
// It is mutation-free and degrades gracefully when Temporal or the DB is unavailable.
// If temporalConfigured is false, Temporal checks are skipped entirely.
func collectReport(ctx context.Context, config advruntime.Config, temporalConfigured bool) advruntime.Report {
	report := advruntime.NewReport(config)

	// --- Temporal checks (only if configured in stack.toml) ---
	if temporalConfigured {
		provider := advruntime.NewTemporalClientProvider(config, nil)
		defer provider.Close()

		client, err := provider.Client(ctx)
		if err != nil {
			// Temporal not reachable — record a warning finding, do not crash.
			report.Findings = append(report.Findings, advruntime.Finding{
				Code:     "ADV_TEMPORAL_UNAVAILABLE",
				Severity: advruntime.SeverityWarn,
				Message:  fmt.Sprintf("Temporal unreachable: %v", err),
				Hint:     "start Temporal dev server with 'oca temporal start' or verify stack.toml [temporal] address",
			})
		} else {
			// Search attributes
			checker := advruntime.NewSearchAttributeChecker(&operatorServiceAdapter{client.OperatorService()}, config)
			if attrReport, err := checker.Check(ctx); err != nil {
				report.Findings = append(report.Findings, advruntime.Finding{
					Code:     "ADV_SEARCH_ATTRIBUTES_SCAN_FAILED",
					Severity: advruntime.SeverityWarn,
					Message:  fmt.Sprintf("search attribute check failed: %v", err),
					Hint:     "verify Temporal search attributes are registered",
				})
			} else {
				report.SearchAttributes = attrReport.SearchAttributes
			}

			// Workflow / task queue classifier
			classifier := advruntime.NewWorkflowClassifier(&workflowServiceAdapter{client.WorkflowService()}, config, nil)
			if wfReport, err := classifier.Classify(ctx); err != nil {
				report.Findings = append(report.Findings, advruntime.Finding{
					Code:     "ADV_WORKFLOW_CLASSIFIER_FAILED",
					Severity: advruntime.SeverityWarn,
					Message:  fmt.Sprintf("workflow classification failed: %v", err),
					Hint:     "verify Temporal workflow service is responsive",
				})
			} else {
				report.WorkflowQueues = wfReport.WorkflowQueues
				report.Findings = append(report.Findings, wfReport.Findings...)
			}
		}
	}

	// --- Session debt scan (optional) ---
	dbPath := advruntime.DefaultOpenCodeDBPath()
	if dbPath != "" {
		if _, err := os.Stat(dbPath); err == nil {
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				report.Findings = append(report.Findings, advruntime.Finding{
					Code:     "ADV_SESSION_DB_OPEN_FAILED",
					Severity: advruntime.SeverityWarn,
					Message:  fmt.Sprintf("cannot open session DB: %v", err),
					Hint:     "verify the database file is readable and not locked",
				})
			} else {
				defer db.Close()
				scanner := advruntime.NewSessionDebtScanner(db, config)
				finding, err := scanner.Scan(ctx)
				if err != nil {
					report.Findings = append(report.Findings, advruntime.Finding{
						Code:     "ADV_SESSION_DEBT_SCAN_FAILED",
						Severity: advruntime.SeverityWarn,
						Message:  fmt.Sprintf("session debt scan failed: %v", err),
						Hint:     "run 'oca doctor --scope adv-runtime' to retry",
					})
				} else {
					report.SessionDebt = append(report.SessionDebt, finding)
				}
			}
		}
	}

	// --- Worktree census ---
	worktreeRoot := advruntime.DefaultWorktreeRoot()
	advRoot := advruntime.DefaultADVStateRoot()
	if worktreeRoot != "" && advRoot != "" {
		census := advruntime.NewWorktreeCensus(worktreeRoot, advRoot, config)
		worktrees, err := census.Scan(ctx)
		if err != nil {
			report.Findings = append(report.Findings, advruntime.Finding{
				Code:     "ADV_WORKTREE_CENSUS_FAILED",
				Severity: advruntime.SeverityWarn,
				Message:  fmt.Sprintf("worktree census failed: %v", err),
				Hint:     "verify worktree root is readable",
			})
		} else {
			report.Worktrees = worktrees
		}
	}

	// Update summary counts
	for _, attr := range report.SearchAttributes {
		if attr.Status == advruntime.StatusFail || attr.Status == advruntime.StatusUnknown {
			report.Summary.MissingSearchAttributes++
			report.Summary.Status = advruntime.StatusWarn
		}
	}
	for _, queue := range report.WorkflowQueues {
		if queue.Status == advruntime.StatusFail || queue.Status == advruntime.StatusWarn {
			report.Summary.StaleWorkflowQueues++
			if report.Summary.Status == advruntime.StatusPass {
				report.Summary.Status = advruntime.StatusWarn
			}
		}
	}
	for _, debt := range report.SessionDebt {
		if debt.Status == advruntime.StatusWarn {
			report.Summary.SessionDebt += debt.Repairable
			if report.Summary.Status == advruntime.StatusPass {
				report.Summary.Status = advruntime.StatusWarn
			}
		}
	}
	for _, wt := range report.Worktrees {
		if wt.Status == advruntime.StatusWarn {
			report.Summary.WorktreeDebt++
			if report.Summary.Status == advruntime.StatusPass {
				report.Summary.Status = advruntime.StatusWarn
			}
		}
	}
	for _, finding := range report.Findings {
		if finding.Severity == advruntime.SeverityError {
			report.Summary.Status = advruntime.StatusFail
		} else if finding.Severity == advruntime.SeverityWarn && report.Summary.Status == advruntime.StatusPass {
			report.Summary.Status = advruntime.StatusWarn
		}
	}

	return report
}
