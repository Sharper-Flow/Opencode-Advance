package main

import (
	"fmt"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	"github.com/spf13/cobra"
)

func newVersionCmd(opts commandOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show OpenCode Advance version",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := brand.DetectColorMode(opts.Environment)
			if _, err := fmt.Fprintln(opts.Stdout, brand.Render(brand.VariantFull, mode)); err != nil {
				return err
			}
			_, err := fmt.Fprintf(opts.Stdout, "\nversion: %s\n", opts.Version.Display())
			return err
		},
	}
}
