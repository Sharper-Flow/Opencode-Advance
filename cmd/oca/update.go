package main

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	pluginpkg "github.com/Sharper-Flow/Opencode-Advance/internal/plugin"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	syncpkg "github.com/Sharper-Flow/Opencode-Advance/internal/sync"
	"github.com/spf13/cobra"
)

var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func newUpdateCmd(state *commandState) *cobra.Command {
	var force bool
	var check bool
	cmd := &cobra.Command{
		Use:   "update [plugin...]",
		Short: "Update git plugins and re-apply plugin config",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			selected := map[string]bool{}
			for _, arg := range args {
				selected[arg] = true
			}
			if check {
				return runUpdateCheck(cmd.Context(), state, stack, selected)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60_000_000_000)
			defer cancel()
			skipped := false
			for _, name := range sortedPluginNames(stack.Plugins) {
				plugin := stack.Plugins[name]
				if len(selected) > 0 && !selected[name] {
					continue
				}
				if !plugin.IsEnabled() || plugin.IsNPMSource() {
					continue
				}
				if shaPattern.MatchString(plugin.Ref) && !force {
					skipped = true
					_, _ = fmt.Fprintf(state.opts.Stdout, "skip\t%s\tpinned ref %s\n", name, plugin.Ref)
					continue
				}
				if err := pluginpkg.Fetch(ctx, plugin.Checkout); err != nil {
					return newCLIError(2, "update %s fetch: %w", name, err)
				}
				ref := plugin.Ref
				if ref == "" {
					ref = "trunk"
				}
				checkoutRef := ref
				if !shaPattern.MatchString(ref) {
					checkoutRef = "origin/" + ref
				}
				if err := pluginpkg.Checkout(ctx, plugin.Checkout, checkoutRef); err != nil {
					return newCLIError(2, "update %s checkout: %w", name, err)
				}
				if err := pluginpkg.RunBuild(ctx, plugin); err != nil {
					return newCLIError(3, "update %s build: %w", name, err)
				}
			}
			paths := config.ResolvePaths()
			plan, err := render.PlanPlugins(stack, paths, state.configPath)
			if err != nil {
				return newCLIError(3, "plan plugins: %w", err)
			}
			result, err := render.Apply(plan, render.ApplyOptions{DryRun: false, MaxBackups: 3, LockPath: plan.LockPath})
			if err != nil {
				return newCLIError(3, "apply plugins: %w", err)
			}
			if state.output == "json" {
				if err := printJSON(state.opts.Stdout, result); err != nil {
					return err
				}
			} else {
				for _, t := range result.Targets {
					if _, err := fmt.Fprintf(state.opts.Stdout, "%s\t%s\n", t.Op, t.Path); err != nil {
						return err
					}
				}
			}
			for _, name := range sortedPluginNames(stack.Plugins) {
				plugin := stack.Plugins[name]
				if !plugin.IsEnabled() || plugin.Sync == "" {
					continue
				}
				res, err := syncpkg.InvokeAdvance(ctx, plugin)
				if err != nil {
					return newCLIError(3, "%w", syncpkg.FormatSyncError(name, res, err))
				}
			}
			if err := render.WriteShellEnvAndStamp(config.ResolvePaths()); err != nil {
				return newCLIError(3, "update shell env refresh: %w", err)
			}
			if skipped {
				return newCLIError(1, "some plugins skipped")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force update even when plugin ref is pinned to a SHA (does not mutate stack.toml ref)")
	cmd.Flags().BoolVar(&check, "check", false, "Check for plugin updates without fetching, building, or applying")
	return cmd
}

func runUpdateCheck(ctx context.Context, state *commandState, stack *config.Stack, selected map[string]bool) error {
	paths := config.ResolvePaths()
	opts := pluginpkg.ProbeOptions{
		TimeoutPerPlugin: time.Duration(stack.UpdateProbe.TimeoutPerPluginMS) * time.Millisecond,
		TimeoutGlobal:    time.Duration(stack.UpdateProbe.TimeoutGlobalMS) * time.Millisecond,
		Parallelism:      4,
	}
	results, err := pluginpkg.ProbeAll(ctx, stack.Plugins, selected, opts)
	if err != nil {
		return newCLIError(3, "update check: %w", err)
	}
	if err := pluginpkg.WriteDriftCache(paths.DriftCachePath(), results); err != nil {
		return newCLIError(3, "write drift cache: %w", err)
	}
	if state.quiet {
		return nil
	}
	if state.output == "json" {
		return printJSON(state.opts.Stdout, map[string]any{"results": results})
	}
	for _, res := range results {
		if _, err := fmt.Fprintf(state.opts.Stdout, "%s\t%s\t%s\t%s\t%s\n", res.Status, res.Name, res.Ref, res.LocalSHA, res.RemoteSHA); err != nil {
			return err
		}
	}
	return nil
}
