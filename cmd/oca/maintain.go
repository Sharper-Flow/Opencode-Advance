package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/maintain"
	"github.com/Sharper-Flow/Opencode-Advance/internal/session"
	"github.com/spf13/cobra"
)

var maintainPlanFunc = maintain.BuildPlan
var maintainExecuteFunc = maintain.ExecutePlan

func newMaintainCmd(state *commandState) *cobra.Command {
	var (
		dryRun         bool
		execute        bool
		includeMerge   bool
		includeRebuild bool
		includeCleanup bool
		projectRoot    string
	)

	cmd := &cobra.Command{
		Use:   "maintain",
		Short: "Plan and run offline OCA/ADV maintenance",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if execute && dryRun {
				return newCLIError(2, "maintain: --execute and --dry-run are mutually exclusive")
			}
			if projectRoot == "" {
				ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
				defer cancel()
				if root, err := session.ProjectRoot(ctx, ""); err == nil {
					projectRoot = root
				}
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			opts := maintain.Options{
				ProjectRoot:    projectRoot,
				ConfigPath:     state.configPath,
				DryRun:         !execute,
				Execute:        execute,
				IncludeMerge:   includeMerge,
				IncludeRebuild: includeRebuild,
				IncludeCleanup: includeCleanup,
			}
			if dryRun {
				opts.DryRun = true
				opts.Execute = false
			}
			plan, err := maintainPlanFunc(ctx, opts)
			if err != nil {
				return newCLIError(3, "maintain plan: %w", err)
			}
			if state.output == "json" {
				if err := printJSON(state.opts.Stdout, map[string]any{
					"dry_run": opts.DryRun,
					"execute": opts.Execute,
					"plan":    plan,
				}); err != nil {
					return err
				}
			} else if err := printMaintainPlanText(state, opts, plan); err != nil {
				return err
			}
			if opts.Execute && len(plan.Blockers) > 0 {
				return newCLIError(3, "active maintenance blockers: %s", blockerCodes(plan.Blockers))
			}
			if opts.Execute {
				if err := maintainExecuteFunc(ctx, opts, plan); err != nil {
					return newCLIError(3, "maintain execute: %w", err)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show maintenance plan without mutation")
	cmd.Flags().BoolVar(&execute, "execute", false, "Execute safe maintenance actions after hard gates pass")
	cmd.Flags().BoolVar(&includeMerge, "include-merge", true, "Include verified merge candidates in the maintenance plan")
	cmd.Flags().BoolVar(&includeRebuild, "include-rebuild", true, "Include plugin rebuild actions in the maintenance plan")
	cmd.Flags().BoolVar(&includeCleanup, "include-cleanup", true, "Include conservative worktree cleanup candidates in the maintenance plan")
	cmd.Flags().StringVar(&projectRoot, "project", "", "Project root to inspect (default: current git root)")
	return cmd
}

func printMaintainPlanText(state *commandState, opts maintain.Options, plan maintain.Plan) error {
	mode := "dry-run"
	if opts.Execute {
		mode = "execute"
	}
	if _, err := fmt.Fprintf(state.opts.Stdout, "OCA Maintenance Plan (%s)\n", mode); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(state.opts.Stdout, "  Project: %s\n", plan.ProjectRoot); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(state.opts.Stdout, "  Session gate: %s\n", plan.SessionGate.Status); err != nil {
		return err
	}
	for _, blocker := range plan.Blockers {
		if _, err := fmt.Fprintf(state.opts.Stdout, "  Blocker: %s — %s\n", blocker.Code, blocker.Message); err != nil {
			return err
		}
	}
	for _, action := range plan.Actions {
		if _, err := fmt.Fprintf(state.opts.Stdout, "  Action: %s (%s)\n", action.ID, action.Kind); err != nil {
			return err
		}
	}
	return nil
}

func blockerCodes(blockers []maintain.Blocker) string {
	codes := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		codes = append(codes, blocker.Code)
	}
	return strings.Join(codes, ", ")
}
