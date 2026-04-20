package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	pluginpkg "github.com/Sharper-Flow/Opencode-Advance/internal/plugin"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	syncpkg "github.com/Sharper-Flow/Opencode-Advance/internal/sync"
	"github.com/spf13/cobra"
)

func newApplyCmd(state *commandState) *cobra.Command {
	var targets []string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Render and write configuration targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if len(targets) == 0 {
				return newCLIError(2, "--target is required (supported: mcp, plugins, instructions, temporal)")
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
			ctx, cancel := withContext()
			defer cancel()

			for _, target := range targets {
				switch target {
				case "mcp":
					if err := applyMCP(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "plugins":
					if err := applyPlugins(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "instructions":
					if err := applyInstructions(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "temporal":
					return newCLIError(2, "target %q reserved for Phase 6.5; see docs/proposals/phases.md § Phase 6.5", target)
				default:
					return newCLIError(2, "unknown target %q; supported: mcp, plugins, instructions, temporal", target)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&targets, "target", nil, "Target(s) to apply (supported: mcp, plugins, instructions, temporal)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the render plan without writing files")
	_ = cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"mcp", "plugins", "instructions", "temporal"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func applyMCP(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanMCP(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan mcp: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply mcp")
}

func applyPlugins(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	if !dryRun {
		names := sortedPluginNames(stack.Plugins)
		for _, name := range names {
			plugin := stack.Plugins[name]
			if !plugin.IsEnabled() || plugin.IsNPMSource() {
				continue
			}
			if err := pluginpkg.Prepare(ctx, plugin); err != nil {
				return newCLIError(3, "prepare plugin %s: %w", name, err)
			}
		}
	}
	plan, err := render.PlanPlugins(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan plugins: %w", err)
	}
	if err := emitPlanOrApply(state, plan, dryRun, "apply plugins"); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	for _, name := range sortedPluginNames(stack.Plugins) {
		plugin := stack.Plugins[name]
		if !plugin.IsEnabled() || plugin.Sync == "" {
			continue
		}
		if _, err := syncpkg.InvokeAdvance(ctx, plugin); err != nil {
			return newCLIError(3, "sync plugin %s: %w", name, err)
		}
	}
	return nil
}

func applyInstructions(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanInstructions(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan instructions: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply instructions")
}

func emitPlanOrApply(state *commandState, plan *render.Plan, dryRun bool, action string) error {
	if dryRun {
		if state.output == "json" {
			return printJSON(state.opts.Stdout, render.RedactPlan(plan))
		}
		return printPlanText(state.opts.Stdout, plan)
	}
	result, err := render.Apply(plan, render.ApplyOptions{DryRun: false, MaxBackups: 3, LockPath: plan.LockPath})
	if err != nil {
		return newCLIError(3, "%s: %w", action, err)
	}
	if state.output == "json" {
		return printJSON(state.opts.Stdout, result)
	}
	for _, t := range result.Targets {
		if _, err := fmt.Fprintf(state.opts.Stdout, "%s\t%s\n", t.Op, t.Path); err != nil {
			return err
		}
	}
	return nil
}

func sortedPluginNames(plugins config.PluginsSection) []string {
	names := make([]string, 0, len(plugins))
	for name := range plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
