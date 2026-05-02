package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	itemporal "github.com/Sharper-Flow/Opencode-Advance/internal/temporal"
	"github.com/spf13/cobra"
)

var temporalSupervisorFactory = func() itemporal.Supervisor { return itemporal.Supervisor{} }

func newTemporalCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "temporal",
		Short: "Manage the local Temporal dev server",
	}
	cmd.AddCommand(newTemporalStatusCmd(state))
	cmd.AddCommand(newTemporalStartCmd(state))
	cmd.AddCommand(newTemporalStopCmd(state))
	cmd.AddCommand(newTemporalRestartCmd(state))
	return cmd
}

func newTemporalStatusCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Temporal dev-server status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			status, err := temporalSupervisorFactory().Status(ctx, stack)
			if err != nil {
				return newCLIError(3, "temporal status: %w", err)
			}
			return printTemporalStatus(state, status)
		},
	}
}

func newTemporalStartCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the local Temporal dev server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemporalLifecycle(state, func(ctx context.Context, sup itemporal.Supervisor, stack *cfg.Stack) (itemporal.Status, error) {
				return sup.Start(ctx, stack)
			})
		},
	}
}

func newTemporalStopCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the OCA-managed Temporal dev server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemporalLifecycle(state, func(ctx context.Context, sup itemporal.Supervisor, stack *cfg.Stack) (itemporal.Status, error) {
				return sup.Stop(ctx, stack)
			})
		},
	}
}

func newTemporalRestartCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the OCA-managed Temporal dev server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemporalLifecycle(state, func(ctx context.Context, sup itemporal.Supervisor, stack *cfg.Stack) (itemporal.Status, error) {
				return sup.Restart(ctx, stack)
			})
		},
	}
}

func runTemporalLifecycle(state *commandState, run func(context.Context, itemporal.Supervisor, *cfg.Stack) (itemporal.Status, error)) error {
	if err := validateOutputMode(state.output); err != nil {
		return err
	}
	stack, err := loadStack(state)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	status, err := run(ctx, temporalSupervisorFactory(), stack)
	if printErr := printTemporalStatus(state, status); printErr != nil {
		return printErr
	}
	if err != nil {
		if errors.Is(err, itemporal.ErrUnmanagedServer) {
			return newCLIError(1, "%w", err)
		}
		return newCLIError(3, "%w", err)
	}
	return nil
}

func printTemporalStatus(state *commandState, status itemporal.Status) error {
	if state.output == "json" {
		return printJSON(state.opts.Stdout, status)
	}
	_, err := fmt.Fprintf(state.opts.Stdout, "%s\t%s\tmanaged=%v\trunning=%v\thealthy=%v\n", status.State, status.Address, status.Managed, status.Running, status.Healthy)
	return err
}
