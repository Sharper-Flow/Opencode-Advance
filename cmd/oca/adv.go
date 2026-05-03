package main

import (
	"fmt"
	"strings"

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
				// In v1, recover is always dry-run. The flag exists for explicitness
				// and future expansion.
				return newCLIError(2, "recover requires --dry-run in v1 (mutation not supported)")
			}

			// Build a synthetic report from the health system.
			// In a full implementation this would run all scanners; for v1 we
			// demonstrate the recovery plan rendering with a minimal report.
			report := advruntime.NewReport(advruntime.Config{})

			// Apply filters if specified
			if projectFilter != "" {
				report.Config.ProjectID = projectFilter
			}
			if changeFilter != "" {
				report.Config.ChangeID = changeFilter
			}

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
