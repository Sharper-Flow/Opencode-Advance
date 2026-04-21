package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/health"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	"github.com/spf13/cobra"
)

type commandOptions struct {
	Stdout      io.Writer
	Stderr      io.Writer
	Version     VersionInfo
	Environment brand.Environment
}

type commandState struct {
	opts       commandOptions
	configPath string
	output     string
	verbose    bool
	quiet      bool
	logger     *slog.Logger
}

type exitCoder interface{ ExitCode() int }

type cliError struct {
	code int
	err  error
}

func (e *cliError) Error() string { return e.err.Error() }
func (e *cliError) Unwrap() error { return e.err }
func (e *cliError) ExitCode() int { return e.code }

func newCLIError(code int, format string, args ...any) error {
	return &cliError{code: code, err: fmt.Errorf(format, args...)}
}

func execute() error {
	cmd := newRootCmd(defaultCommandOptions())
	return cmd.Execute()
}

func newRootCmd(opts commandOptions) *cobra.Command {
	state := &commandState{opts: opts}
	cmd := &cobra.Command{
		Use:           "oca",
		Short:         "OpenCode Advance",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if state.configPath == "" {
				state.configPath = defaultStackPath()
			}
			state.logger = newLogger(opts, state.output, state.verbose, state.quiet)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.SetOut(opts.Stdout)
	cmd.SetErr(opts.Stderr)
	cmd.PersistentFlags().StringVar(&state.configPath, "config", "", "Path to stack.toml (default: ./stack.toml if present, else $XDG_CONFIG_HOME/opencode-advance/stack.toml)")
	cmd.PersistentFlags().StringVar(&state.output, "output", "text", "Output format: text|json")
	cmd.PersistentFlags().BoolVar(&state.verbose, "verbose", false, "Enable debug logging")
	cmd.PersistentFlags().BoolVar(&state.quiet, "quiet", false, "Only print errors")
	cmd.AddCommand(newVersionCmd(opts))
	cmd.AddCommand(newApplyCmd(state))
	cmd.AddCommand(newDiffCmd(state))
	cmd.AddCommand(newDoctorCmd(state))
	cmd.AddCommand(newPinCmd(state))
	cmd.AddCommand(newUpdateCmd(state))
	cmd.AddCommand(newDebugCmd(state))

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

func defaultStackPath() string {
	if _, err := os.Stat("stack.toml"); err == nil {
		return "stack.toml"
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "opencode-advance", "stack.toml")
}

func newLogger(opts commandOptions, output string, verbose, quiet bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	if quiet {
		level = slog.LevelError
	}
	hopts := &slog.HandlerOptions{Level: level}
	if output == "json" {
		return slog.New(slog.NewJSONHandler(opts.Stderr, hopts))
	}
	return slog.New(slog.NewTextHandler(opts.Stderr, hopts))
}

func printJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func printWarningsText(w io.Writer, warnings []cfg.Warning) error {
	for _, warning := range warnings {
		if _, err := fmt.Fprintf(w, "warning: [%s] %s", warning.Path, warning.Message); err != nil {
			return err
		}
		if warning.Hint != "" {
			if _, err := fmt.Fprintf(w, " (hint: %s)", warning.Hint); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func printChecksText(w io.Writer, checks []health.Check) error {
	for _, c := range checks {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s", c.Status, c.Name, c.Message); err != nil {
			return err
		}
		if c.Hint != "" {
			if _, err := fmt.Fprintf(w, "\t%s", c.Hint); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func printPlanText(w io.Writer, plan *render.Plan) error {
	for _, target := range plan.Targets {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", target.Op, target.Name, target.Path); err != nil {
			return err
		}
	}
	return nil
}

func loadStack(state *commandState) (*cfg.Stack, error) {
	stack, err := cfg.Load(state.configPath)
	if err != nil {
		var ve cfg.ValidationErrors
		if errors.As(err, &ve) {
			return nil, &cliError{code: 2, err: err}
		}
		var pe *cfg.ParseError
		if errors.As(err, &pe) {
			return nil, &cliError{code: 2, err: err}
		}
		return nil, &cliError{code: 3, err: err}
	}
	return stack, nil
}

func validateOutputMode(output string) error {
	if output != "" && output != "text" && output != "json" {
		return newCLIError(2, "invalid --output %q (allowed: text, json)", output)
	}
	return nil
}

func withContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
