package main

import (
	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	"github.com/spf13/cobra"
)

func newDebugCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{Use: "debug", Short: "Debug planning and validation"}
	cmd.AddCommand(newDebugPlanCmd(state), newDebugValidateCmd(state))
	return cmd
}

func newDebugPlanCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Print the internal render plan as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			plan, err := render.PlanMCP(stack, config.ResolvePaths(), state.configPath)
			if err != nil {
				return newCLIError(3, "debug plan: %w", err)
			}
			return printJSON(state.opts.Stdout, render.RedactPlan(plan))
		},
	}
}

func newDebugValidateCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate stack.toml without writing files",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			if state.output == "json" {
				return printJSON(state.opts.Stdout, map[string]any{"ok": true, "warnings": stack.Warnings})
			}
			if len(stack.Warnings) > 0 {
				if err := printWarningsText(state.opts.Stdout, stack.Warnings); err != nil {
					return err
				}
			}
			_, err = state.opts.Stdout.Write([]byte("ok\n"))
			return err
		},
	}
}
