package main

import (
	"io"
	"os"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	"github.com/spf13/cobra"
)

type commandOptions struct {
	Stdout      io.Writer
	Stderr      io.Writer
	Version     VersionInfo
	Environment brand.Environment
}

func execute() error {
	cmd := newRootCmd(defaultCommandOptions())
	return cmd.Execute()
}

func newRootCmd(opts commandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "oca",
		Short:         "OpenCode Advance",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.SetOut(opts.Stdout)
	cmd.SetErr(opts.Stderr)
	cmd.AddCommand(newVersionCmd(opts))

	return cmd
}

func defaultCommandOptions() commandOptions {
	return commandOptions{
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Version:     CurrentVersionInfo(),
		Environment: detectEnvironment(),
	}
}

func detectEnvironment() brand.Environment {
	return brand.Environment{
		IsTTY:     stdoutIsTTY(),
		NoColor:   os.Getenv("NO_COLOR") != "",
		Term:      os.Getenv("TERM"),
		ColorTerm: os.Getenv("COLORTERM"),
	}
}

func stdoutIsTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
