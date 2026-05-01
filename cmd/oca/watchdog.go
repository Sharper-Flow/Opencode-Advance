package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type watchdogStatus struct {
	SessionID     string `json:"sessionID"`
	Status        string `json:"status"`
	BumpCount     int    `json:"bump_count"`
	LastBumpAt    int64  `json:"last_bump_at"`
	LastActivityAt int64 `json:"last_activity_at"`
}

type paneWatchdogState struct {
	SessionID string `json:"sessionID"`
	Directory string `json:"directory"`
	Ts        int64  `json:"ts"`
	Watchdog  *struct {
		Enabled      bool   `json:"enabled"`
		BumpCount    int    `json:"bump_count"`
		LastBumpAt   int64  `json:"last_bump_at"`
		LastActivityAt int64 `json:"last_activity_at"`
		Status       string `json:"status"`
	} `json:"watchdog"`
}

func newWatchdogCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watchdog",
		Short: "Manage OCA agent watchdog",
		Long:  "Monitor and control the automatic hang detection and recovery watchdog.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newWatchdogStatusCmd(state))
	cmd.AddCommand(newWatchdogResetCmd(state))
	return cmd
}

func newWatchdogStatusCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [<session>]",
		Short: "Show watchdog status for OCA panes",
		Long:  "Display watchdog status for all OCA tmux panes, or a specific session.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			paneFiles, err := findAllPaneStateFiles()
			if err != nil {
				return newCLIError(1, "find pane state files: %v", err)
			}

			var statuses []watchdogStatus
			for _, pf := range paneFiles {
				ps := readPaneWatchdogState(pf)
				if ps == nil {
					continue
				}
				// Filter by session if specified.
				if len(args) > 0 && ps.SessionID != args[0] {
					continue
				}
				ws := watchdogStatus{
					SessionID: ps.SessionID,
					Status:    "disabled",
				}
				if ps.Watchdog != nil {
					ws.Status = ps.Watchdog.Status
					ws.BumpCount = ps.Watchdog.BumpCount
					ws.LastBumpAt = ps.Watchdog.LastBumpAt
					ws.LastActivityAt = ps.Watchdog.LastActivityAt
					if ws.Status == "" {
						if ps.Watchdog.Enabled {
							ws.Status = "active"
						} else {
							ws.Status = "disabled"
						}
					}
				}
				statuses = append(statuses, ws)
			}

			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), statuses)
			}

			if len(statuses) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No OCA panes with watchdog state found.")
				return nil
			}

			for _, s := range statuses {
				fmt.Fprintf(cmd.OutOrStdout(), "  session: %s  status: %s  bumps: %d\n", s.SessionID, s.Status, s.BumpCount)
			}
			return nil
		},
	}
	return cmd
}

func newWatchdogResetCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset [<pane>]",
		Short: "Reset watchdog bump count and status",
		Long:  "Reset the watchdog bump counter and status to defaults for all panes or a specific pane.",
		RunE: func(cmd *cobra.Command, args []string) error {
			paneFiles, err := findAllPaneStateFiles()
			if err != nil {
				return newCLIError(1, "find pane state files: %v", err)
			}

			resetCount := 0
			for _, pf := range paneFiles {
				ps := readPaneWatchdogState(pf)
				if ps == nil || ps.Watchdog == nil {
					continue
				}
				// Filter by pane if specified.
				if len(args) > 0 {
					base := filepath.Base(pf)
					ext := filepath.Ext(base)
					paneID := base[:len(base)-len(ext)]
					if paneID != args[0] {
						continue
					}
				}
				ps.Watchdog.BumpCount = 0
				ps.Watchdog.LastBumpAt = 0
				ps.Watchdog.Status = "active"
				writePaneWatchdogState(pf, ps)
				resetCount++
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Reset %d pane(s)\n", resetCount)
			return nil
		},
	}
	return cmd
}

func findAllPaneStateFiles() ([]string, error) {
	stateDir := filepath.Join(xdgStateHome(), "oca", "panes")
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		socketDir := filepath.Join(stateDir, entry.Name())
		jsonFiles, err := filepath.Glob(filepath.Join(socketDir, "*.json"))
		if err != nil {
			continue
		}
		files = append(files, jsonFiles...)
	}
	return files, nil
}

func readPaneWatchdogState(filePath string) *paneWatchdogState {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	var ps paneWatchdogState
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil
	}
	return &ps
}

func writePaneWatchdogState(filePath string, ps *paneWatchdogState) {
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(filePath, data, 0o644)
}
