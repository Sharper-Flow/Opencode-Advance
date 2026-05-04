package main

import (
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/health"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
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
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			ctx, cancel := withContext()
			defer cancel()
			paths := cfg.ResolvePaths()
			opts := health.Options{
				Timeout:          timeout,
				SkillsAssetsRoot: render.AssetsSkillsRoot(),
				SkillsTargetDir:  paths.OpencodeSkillsDir(),
				ConfigDir:        paths.OpencodeConfigDir,
			}
			checks, err := health.Run(scope, ctx, stack, opts)
			if err != nil {
				if err == health.ErrUnknownScope {
					return newCLIError(2, "unknown scope %q; supported: mcp, plugins, skills, temporal, adv-assets, adv-plugin, adv-runtime, cross, runtime, context-budget, tool-suites", scope)
				}
				return newCLIError(3, "doctor %s: %w", scope, err)
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
				return newCLIError(2, "%s", health.Summary(checks))
			}
			if health.HasWarnings(checks) {
				return newCLIError(1, "%s", health.Summary(checks))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "mcp", "Scope to check (supported: mcp, plugins, skills, temporal, adv-assets, adv-plugin, adv-runtime, cross, runtime, context-budget, tool-suites)")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "HTTP timeout for doctor checks")
	_ = cmd.RegisterFlagCompletionFunc("scope", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"mcp", "plugins", "skills", "temporal", "adv-assets", "adv-plugin", "adv-runtime", "cross", "runtime", "context-budget", "tool-suites"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}
