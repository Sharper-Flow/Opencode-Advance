package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advstatus"
	"github.com/spf13/cobra"
)

func newAdvStatusCmd(state *commandState) *cobra.Command {
	var query string
	var jsonOut bool
	var panePath string
	var projectID string

	cmd := &cobra.Command{
		Use:   "adv-status",
		Short: "ADV state queries for status bar and CLI",
		Long:  "Query ADV state (active changes, temporal health, branch safety, worktree status) for tmux status bar and CLI use.",
		RunE: func(cmd *cobra.Command, args []string) error {
			useJSON := state.output == "json" || jsonOut

			switch advstatus.QueryMode(query) {
			case advstatus.QueryActiveChange:
				return runAdvStatusActiveChange(cmd, useJSON)
			case advstatus.QueryTemporalHealth:
				return runAdvStatusTemporalHealth(cmd, useJSON)
			case advstatus.QueryBranchSafety:
				return runAdvStatusBranchSafety(cmd, useJSON, panePath, projectID)
			case advstatus.QueryWorktrees:
				return runAdvStatusWorktrees(cmd, useJSON, projectID)
			case advstatus.QueryWorkspaceLookup:
				return runAdvStatusWorkspaceLookup(cmd, useJSON, projectID)
			default:
				return newCLIError(2, "unknown query mode: %s (valid: active-change, temporal-health, branch-safety, worktrees, workspace-lookup)", query)
			}
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Query mode (required)")
	_ = cmd.MarkFlagRequired("query")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON (shorthand for --output json)")
	cmd.Flags().StringVar(&panePath, "path", "", "Pane path for branch-safety query")
	cmd.Flags().StringVar(&projectID, "project", "", "Project ID for worktree/branch-safety queries")

	return cmd
}

func runAdvStatusActiveChange(cmd *cobra.Command, jsonOut bool) error {
	summary, _ := advstatus.ActiveChangeSummary()
	if jsonOut {
		return printAdvStatusJSON(cmd, "active-change", summary)
	}
	fmt.Fprint(cmd.OutOrStdout(), advstatus.FormatActiveChangeTextOrDefault(summary))
	return nil
}

func runAdvStatusTemporalHealth(cmd *cobra.Command, jsonOut bool) error {
	health, _ := advstatus.TemporalHealthProbe()
	if jsonOut {
		return printAdvStatusJSON(cmd, "temporal-health", health)
	}
	fmt.Fprint(cmd.OutOrStdout(), advstatus.FormatTemporalHealthTextOrDefault(health))
	return nil
}

func runAdvStatusBranchSafety(cmd *cobra.Command, jsonOut bool, panePath, projectID string) error {
	pid := resolveProjectIDForStatus(panePath, projectID)
	result, _ := advstatus.BranchSafety(panePath, pid)
	if jsonOut {
		return printAdvStatusJSON(cmd, "branch-safety", result)
	}
	fmt.Fprint(cmd.OutOrStdout(), advstatus.FormatBranchSafetyText(result))
	return nil
}

func runAdvStatusWorktrees(cmd *cobra.Command, jsonOut bool, projectID string) error {
	result, _ := advstatus.WorkspaceState(projectID)
	if jsonOut {
		return printAdvStatusJSON(cmd, "worktrees", result)
	}
	fmt.Fprint(cmd.OutOrStdout(), advstatus.FormatWorkspaceStateText(result))
	return nil
}

func runAdvStatusWorkspaceLookup(cmd *cobra.Command, jsonOut bool, projectID string) error {
	entries, _ := advstatus.WorkspaceLookup(projectID)
	if jsonOut {
		return printAdvStatusJSON(cmd, "workspace-lookup", entries)
	}
	fmt.Fprint(cmd.OutOrStdout(), advstatus.FormatWorkspaceLookupTSV(entries))
	return nil
}

func printAdvStatusJSON(cmd *cobra.Command, query string, result any) error {
	out, err := advstatus.FormatJSON(query, result)
	if err != nil {
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), out)
	return nil
}

// resolveProjectIDForStatus resolves project ID from explicit flag or pane path.
func resolveProjectIDForStatus(panePath, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if panePath == "" {
		return ""
	}
	// Derive from git root commit SHA (matches _oca_status_resolve_project_id in shell)
	out, err := exec.Command("git", "-C", panePath, "rev-list", "--max-parents=0", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
