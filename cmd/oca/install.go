package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/Sharper-Flow/Opencode-Advance/internal/health"
	"github.com/Sharper-Flow/Opencode-Advance/internal/install"
	"github.com/spf13/cobra"
)

func newInstallCmd(state *commandState) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Bootstrap OCA environment on a fresh machine",
		Long: `Install creates the directory structure, checks prerequisites,
and writes managed shell profile blocks to ~/.bashrc and ~/.zshrc.

After install completes, run "oca apply" to render the full configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			result, err := install.Install(cmd.Context(), install.InstallOptions{Yes: yes})
			if err != nil {
				return formatInstallError(err, state)
			}

			if state.output == "json" {
				return printJSON(state.opts.Stdout, result)
			}
			return printInstallResult(state.opts.Stdout, result)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip interactive prompts (CI mode)")
	return cmd
}

func printInstallResult(w io.Writer, result *install.InstallResult) error {
	if len(result.CreatedDirs) > 0 {
		fmt.Fprintln(w, "Directories created:")
		for _, dir := range result.CreatedDirs {
			fmt.Fprintf(w, "  %s\n", dir)
		}
	}

	fmt.Fprintln(w, "Shell profile updated:")
	for _, rc := range result.WroteRCFiles {
		fmt.Fprintf(w, "  %s\n", rc)
	}

	// Prereq summary
	for _, c := range result.PrereqChecks {
		switch c.Status {
		case health.StatusPass:
			fmt.Fprintf(w, "  ✓ %s\n", c.Name)
		case health.StatusWarn:
			fmt.Fprintf(w, "  ⚠ %s: %s\n", c.Name, c.Message)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Next step: run `oca apply` to render your configuration.")
	return nil
}

func formatInstallError(err error, state *commandState) error {
	var pe *install.ErrPrereqFailed
	if errors.As(err, &pe) {
		if state.output == "json" {
			return newCLIError(2, "prerequisites failed: %w", err)
		}
		var msg string
		for _, f := range pe.Failures {
			msg += fmt.Sprintf("  ✗ %s: %s\n", f.Name, f.Message)
			if f.Hint != "" {
				msg += fmt.Sprintf("    hint: %s\n", f.Hint)
			}
		}
		return newCLIError(2, "Prerequisites failed:\n%s", msg)
	}
	return newCLIError(3, "install: %w", err)
}
