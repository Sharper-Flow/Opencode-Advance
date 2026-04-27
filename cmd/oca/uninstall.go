package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/Sharper-Flow/Opencode-Advance/internal/install"
	"github.com/spf13/cobra"
)

func newUninstallCmd(state *commandState) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove OCA shell integration and managed config",
		Long: `Uninstall removes managed shell profile blocks from ~/.bashrc and ~/.zshrc.

If a managed block has been manually edited, uninstall will refuse to remove it
unless --force is passed. Use --force to override this protection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			result, err := install.Uninstall(install.UninstallOptions{Force: force})
			if err != nil {
				return formatUninstallError(err, state)
			}

			if state.output == "json" {
				return printJSON(state.opts.Stdout, result)
			}
			return printUninstallResult(state.opts.Stdout, result)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Remove managed blocks even if edited")
	return cmd
}

func printUninstallResult(w io.Writer, result *install.UninstallResult) error {
	if len(result.RemovedRCFiles) > 0 {
		fmt.Fprintln(w, "Removed managed blocks from:")
		for _, rc := range result.RemovedRCFiles {
			fmt.Fprintf(w, "  %s\n", rc)
		}
	} else {
		fmt.Fprintln(w, "No managed blocks found.")
	}
	return nil
}

func formatUninstallError(err error, state *commandState) error {
	var be *install.ErrBlockEdited
	if errors.As(err, &be) {
		if state.output == "json" {
			return newCLIError(2, "edited block in %s: %w", be.FilePath, err)
		}
		return newCLIError(2, "Managed block in %s has been edited.\n\n%s\n\nUse --force to remove anyway.", be.FilePath, be.Diff)
	}
	return newCLIError(3, "uninstall: %w", err)
}
