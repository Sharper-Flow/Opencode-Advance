package main

import (
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/health"
	"github.com/spf13/cobra"
)

func newDoctorCmd(state *commandState) *cobra.Command {
	var scope string
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check current configuration health",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if scope == "" {
				scope = "mcp"
			}
			if scope != "mcp" {
				return newCLIError(2, "unknown scope %q; supported: mcp", scope)
			}
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			ctx, cancel := withContext()
			defer cancel()
			checks, err := health.CheckMCP(ctx, stack, health.Options{Timeout: timeout})
			if err != nil {
				return newCLIError(3, "doctor mcp: %v", err)
			}
			if state.output == "json" {
				if err := printJSON(state.opts.Stdout, checks); err != nil {
					return err
				}
			} else {
				if err := printChecksText(state.opts.Stdout, checks); err != nil {
					return err
				}
			}
			if health.HasFailures(checks) {
				return newCLIError(2, health.Summary(checks))
			}
			if health.HasWarnings(checks) {
				return newCLIError(1, health.Summary(checks))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "mcp", "Scope to check (Phase 1 supports: mcp)")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "HTTP timeout for doctor checks")
	return cmd
}
