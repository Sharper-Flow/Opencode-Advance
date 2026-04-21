package main

import (
	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	"github.com/spf13/cobra"
)

// newDiffCmd returns a read-only command that reports drift between the
// declared stack.toml and the current opencode.json.
func newDiffCmd(state *commandState) *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between stack.toml and current opencode.json",
		Long: `Read-only comparison of declared configuration against the current
opencode.json. Exits 0 if all targets are in sync; exits 1 if any target
has drift. Does not modify any files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if target != "" {
				switch target {
				case "mcp", "plugins", "instructions", "providers", "permissions", "watcher", "lsp":
					// known
				case "temporal":
					return newCLIError(2, "target %q reserved for Phase 6.5; see docs/proposals/phases.md § Phase 6.5", target)
				default:
					return newCLIError(2, "unknown target %q; supported: mcp, plugins, instructions, providers, permissions, watcher, lsp", target)
				}
			}
			stack, err := loadStack(state)
			if err != nil {
				// loadStack returns cliError with code 2 for validation/parse
				// and code 3 for runtime errors — propagate both.
				return err
			}
			paths := config.ResolvePaths()

			// Build the target list: single target if --target specified,
			// otherwise all in-scope targets.
			targets := render.AllTargets
			if target != "" {
				targets = []render.TargetName{render.TargetName(target)}
			}

			plan, err := render.ComposeApplyPlan(stack, paths, state.configPath, targets)
			if err != nil {
				return newCLIError(3, "compose plan: %w", err)
			}

			// Count drift: any opencode.json target with Op != "noop".
			hasDrift := false
			for _, t := range plan.Targets {
				if t.Op != "noop" {
					hasDrift = true
					break
				}
			}

			if state.output == "json" {
				type diffTarget struct {
					Path  string `json:"path"`
					Op    string `json:"op"`
					Drift bool   `json:"drift"`
					// Reason is not included in JSON output for now (can be
					// added later if there's demand; text output covers it).
				}
				diff := struct {
					HasDrift bool         `json:"hasDrift"`
					Targets  []diffTarget `json:"targets"`
				}{
					HasDrift: hasDrift,
					Targets:  make([]diffTarget, 0, len(plan.Targets)),
				}
				for _, t := range plan.Targets {
					diff.Targets = append(diff.Targets, diffTarget{
						Path:  t.Path,
						Op:    t.Op,
						Drift: t.Op != "noop",
					})
				}
				return printJSON(state.opts.Stdout, diff)
			}

			// Text output: per-target line.
			for _, t := range plan.Targets {
				driftMark := " "
				if t.Op != "noop" {
					driftMark = "*"
				}
				if _, err := state.opts.Stdout.Write([]byte(driftMark + "\t" + t.Op + "\t" + t.Name + "\t" + t.Reason + "\n")); err != nil {
					return err
				}
			}

			if !hasDrift {
				// Exit 0: no drift.
				return nil
			}
			// Exit 1: drift detected.
			return newCLIError(1, "drift detected")
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "Show only this target (supported: mcp, plugins, instructions, providers, permissions, watcher, lsp)")
	_ = cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"mcp", "plugins", "instructions", "providers", "permissions", "watcher", "lsp"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}
