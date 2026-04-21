package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	pluginpkg "github.com/Sharper-Flow/Opencode-Advance/internal/plugin"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	syncpkg "github.com/Sharper-Flow/Opencode-Advance/internal/sync"
	"github.com/spf13/cobra"
)

var invokeAdvance = syncpkg.InvokeAdvance

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
			paths := config.ResolvePaths()
			ctx, cancel := withContext()
			defer cancel()

			// No-target: compose all in-scope targets (providers/permissions/watcher/lsp
			// plus the Phase 1+ targets) in dependency order with NoRollback.
			// Chains Before/After bytes through a running in-memory doc so later
			// targets don't clobber earlier ones.
			if len(targets) == 0 {
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
				plan, err := render.ComposeApplyPlan(stack, paths, state.configPath, render.AllTargets)
				if err != nil {
					return newCLIError(3, "compose plan: %w", err)
				}
				applyOpts := render.ApplyOptions{
					DryRun:     false,
					MaxBackups: 3,
					LockPath:   plan.LockPath,
					NoRollback: true, // leave earlier targets on disk if later ones fail
				}
				return emitPlanOrApplyWithOpts(state, plan, dryRun, "apply all targets", applyOpts)
			}

			// Validate all targets before applying any.
			for _, t := range targets {
				switch t {
				case "mcp", "plugins", "instructions", "providers", "permissions", "watcher", "lsp", "skills", "commands", "formatters", "toggles":
					// known
				case "temporal":
					return newCLIError(2, "target %q reserved for Phase 6.5; see docs/proposals/phases.md § Phase 6.5", t)
				default:
					return newCLIError(2, "unknown target %q; supported: mcp, plugins, instructions, providers, permissions, watcher, lsp, skills, commands, formatters, toggles", t)
				}
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
				case "providers":
					if err := applyProviders(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "permissions":
					if err := applyPermissions(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "watcher":
					if err := applyWatcher(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "lsp":
					if err := applyLSP(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "skills":
					if err := applySkills(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "commands":
					if err := applyCommands(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "formatters":
					if err := applyFormatters(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				case "toggles":
					if err := applyToggles(ctx, state, stack, paths, dryRun); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&targets, "target", nil, "Target(s) to apply (supported: mcp, plugins, instructions, providers, permissions, watcher, lsp, skills, commands, formatters, toggles)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the render plan without writing files")
	_ = cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"mcp", "plugins", "instructions", "providers", "permissions", "watcher", "lsp", "skills", "commands", "formatters", "toggles", "temporal"}, cobra.ShellCompDirectiveNoFileComp
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
		if _, err := invokeAdvance(ctx, plugin); err != nil {
			return newCLIError(3, "sync plugin %s: %w", name, err)
		}
	}
	return nil
}

func applyProviders(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanProviders(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan providers: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply providers")
}

func applyPermissions(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanPermissions(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan permissions: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply permissions")
}

func applyWatcher(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanWatcher(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan watcher: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply watcher")
}

func applyLSP(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanLSP(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan lsp: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply lsp")
}

func applySkills(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanSkills(stack, paths, state.configPath, render.AssetsSkillsRoot())
	if err != nil {
		return newCLIError(3, "plan skills: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply skills")
}

func applyCommands(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanCommands(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan commands: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply commands")
}

func applyFormatters(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanFormatters(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan formatters: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply formatters")
}

func applyToggles(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanToggles(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan toggles: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply toggles")
}

func applyInstructions(ctx context.Context, state *commandState, stack *config.Stack, paths config.Paths, dryRun bool) error {
	plan, err := render.PlanInstructions(stack, paths, state.configPath)
	if err != nil {
		return newCLIError(3, "plan instructions: %w", err)
	}
	return emitPlanOrApply(state, plan, dryRun, "apply instructions")
}

func emitPlanOrApply(state *commandState, plan *render.Plan, dryRun bool, action string) error {
	return emitPlanOrApplyWithOpts(state, plan, dryRun, action, render.ApplyOptions{DryRun: false, MaxBackups: 3, LockPath: plan.LockPath})
}

func emitPlanOrApplyWithOpts(state *commandState, plan *render.Plan, dryRun bool, action string, opts render.ApplyOptions) error {
	if dryRun {
		if state.output == "json" {
			return printJSON(state.opts.Stdout, render.RedactPlan(plan))
		}
		return printPlanText(state.opts.Stdout, plan)
	}
	result, err := render.Apply(plan, opts)
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
	for _, t := range plan.Targets {
		if t.Reason != "" && strings.Contains(t.Reason, "pruned ") {
			if _, err := fmt.Fprintln(state.opts.Stdout, t.Reason); err != nil {
				return err
			}
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
