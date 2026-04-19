package main

import (
	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	"github.com/spf13/cobra"
)

func newApplyCmd(state *commandState) *cobra.Command {
	var target string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Render and write configuration targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if target == "" {
				return newCLIError(2, "--target is required in Phase 1 (supported: mcp)")
			}
			if target != "mcp" {
				return newCLIError(2, "unknown target %q; supported: mcp", target)
			}
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			if len(stack.Warnings) > 0 {
				if state.output == "json" {
					if err := printJSON(state.opts.Stdout, map[string]any{"warnings": stack.Warnings}); err != nil {
						return err
					}
				} else if err := printWarningsText(state.opts.Stdout, stack.Warnings); err != nil {
					return err
				}
			}
			paths := config.ResolvePaths()
			plan, err := render.PlanMCP(stack, paths, state.configPath)
			if err != nil {
				return newCLIError(3, "plan mcp: %v", err)
			}
			if dryRun {
				if state.output == "json" {
					return printJSON(state.opts.Stdout, plan)
				}
				return printPlanText(state.opts.Stdout, plan)
			}
			result, err := render.Apply(plan, render.ApplyOptions{DryRun: false, MaxBackups: 3, LockPath: paths.ApplyLockPath()})
			if err != nil {
				return newCLIError(3, "apply mcp: %v", err)
			}
			if state.output == "json" {
				return printJSON(state.opts.Stdout, result)
			}
			for _, t := range result.Targets {
				if _, err := state.opts.Stdout.Write([]byte(t.Op + "\t" + t.Path + "\n")); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "Target to apply (Phase 1 supports: mcp)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the render plan without writing files")
	return cmd
}
