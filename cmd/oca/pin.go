package main

import (
	"context"
	"path/filepath"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	pluginpkg "github.com/Sharper-Flow/Opencode-Advance/internal/plugin"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
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

			// Acquire the apply lock so concurrent `oca apply` / `oca pin`
			// cannot interleave reads and writes against stack.toml. The lock
			// directory is the same one used by render.Apply for atomic writes
			// to opencode.json and friends, giving us a single coordination
			// primitive across all stack.toml-adjacent mutations.
			paths := config.ResolvePaths()
			lockDir := filepath.Dir(paths.ApplyLockPath())
			lock, err := render.AcquireApplyLock(lockDir, 30*time.Second)
			if err != nil {
				return newCLIError(3, "pin: acquire apply lock: %w", err)
			}
			defer render.ReleaseApplyLock(lock)

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
