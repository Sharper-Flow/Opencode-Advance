package main

import (
	"github.com/spf13/cobra"
)

func newCompletionCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for bash or zsh.

Note: fish completion is not yet supported (planned for v1.1).`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh"},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			root := cmd.Root()

			switch shell {
			case "bash":
				return root.GenBashCompletionV2(state.opts.Stdout, true)
			case "zsh":
				return root.GenZshCompletion(state.opts.Stdout)
			case "fish":
				return newCLIError(2, "fish completion is not yet supported; planned for v1.1")
			default:
				return newCLIError(2, "unsupported shell %q; supported: bash, zsh", shell)
			}
		},
	}
	return cmd
}
