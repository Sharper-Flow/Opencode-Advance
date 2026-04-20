package main

import (
	"context"

	pluginpkg "github.com/Sharper-Flow/Opencode-Advance/internal/plugin"
	"github.com/spf13/cobra"
)

func newPinCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin [plugin...]",
		Short: "Capture current plugin SHAs into stack.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			selected := map[string]bool{}
			for _, arg := range args {
				selected[arg] = true
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30_000_000_000)
			defer cancel()
			for _, name := range sortedPluginNames(stack.Plugins) {
				plugin := stack.Plugins[name]
				if len(selected) > 0 && !selected[name] {
					continue
				}
				if plugin.IsNPMSource() {
					continue
				}
				sha, err := pluginpkg.CapturePin(ctx, plugin)
				if err != nil {
					return newCLIError(2, "pin %s: %w", name, err)
				}
				if _, err := pluginpkg.WritePin(state.configPath, name, plugin, sha); err != nil {
					return newCLIError(3, "pin %s: %w", name, err)
				}
			}
			return nil
		},
	}
	return cmd
}
