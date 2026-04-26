package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
	"github.com/spf13/cobra"
)

// paneState is the JSON shape written by the OCA plugin.
type paneState struct {
	SessionID string `json:"sessionID"`
	Directory string `json:"directory"`
	Ts        int64  `json:"ts"`
}

func newPaneCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pane",
		Short: "Manage OCA tmux panes",
		Long:  "Pane-level operations for OpenCode Advance tmux sessions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newPaneRestartTuiCmd(state))
	return cmd
}

func newPaneRestartTuiCmd(state *commandState) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "restart-tui",
		Short: "Restart opencode in the current tmux pane",
		Long:  "Restart the opencode TUI process in the current tmux pane, preserving the chat session.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			paneID := os.Getenv("TMUX_PANE")
			if paneID == "" {
				return newCLIError(1, "TMUX_PANE not set — run inside an OCA tmux pane")
			}

			socket := sessionSocket()
			mgr, err := newTmuxManager(socket)
			if err != nil {
				return newCLIError(1, "tmux manager: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Read state file
			stateFile := paneStateFilePath(paneID)
			ps, err := readPaneState(stateFile)
			if err != nil {
				// State file missing or unreadable — fallback to --continue
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: no state file (%v); falling back to --continue\n", err)
				return mgr.respawnPane(ctx, paneID, "opencode --continue", force)
			}

			// Check pane state
			dead, pid, err := mgr.isPaneDead(ctx, paneID)
			if err != nil {
				return newCLIError(1, "check pane state: %v", err)
			}

			if !dead && !force {
				proc, _ := getPaneProcess(pid)
				return newCLIError(1, "pane is running %s. Use --force to restart.", proc)
			}

			if !dead && force {
				// Verify it's opencode before force-killing
				proc, err := getPaneProcess(pid)
				if err != nil || proc != "opencode" {
					return newCLIError(1, "pane is not running opencode (found: %s). Refusing to force restart.", proc)
				}
			}

			cmdStr := fmt.Sprintf("opencode -s %s", ps.SessionID)
			if err := mgr.respawnPane(ctx, paneID, cmdStr, force || !dead); err != nil {
				return newCLIError(1, "respawn pane: %v", err)
			}

			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]string{
					"pane":      paneID,
					"session":   ps.SessionID,
					"status":    "restarted",
					"directory": ps.Directory,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pane %s restarted (session: %s)\n", paneID, ps.SessionID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force restart even if pane is alive")
	return cmd
}

// paneStateFilePath resolves the per-pane state file path.
func paneStateFilePath(paneID string) string {
	socket := parseTmuxSocket(os.Getenv("TMUX"))
	sanitized := sanitizePaneID(paneID)
	return filepath.Join(xdgStateHome(), "oca", "panes", socket, sanitized+".json")
}

func xdgStateHome() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state")
}

func parseTmuxSocket(tmuxEnv string) string {
	if tmuxEnv == "" {
		return "oca"
	}
	parts := strings.Split(tmuxEnv, ",")
	if len(parts) < 2 {
		return "oca"
	}
	base := filepath.Base(parts[0])
	if base == "" || base == "." {
		return "oca"
	}
	return base
}

func sanitizePaneID(paneID string) string {
	return strings.TrimPrefix(paneID, "%")
}

func readPaneState(filePath string) (*paneState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var ps paneState
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, err
	}
	return &ps, nil
}

// tmuxManager wraps tmux subprocess calls for pane operations.
type tmuxManager struct {
	socket   string
	tmuxPath string
}

func newTmuxManager(socket string) (*tmuxManager, error) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found: %w", err)
	}
	return &tmuxManager{socket: socket, tmuxPath: path}, nil
}

func (m *tmuxManager) isPaneDead(ctx context.Context, paneID string) (bool, int, error) {
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    []string{"-L", m.socket, "list-panes", "-F", "#{pane_dead}\t#{pane_pid}", "-t", paneID},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return false, 0, fmt.Errorf("tmux list-panes: %w", err)
	}
	output := strings.TrimSpace(string(res.Output))
	parts := strings.SplitN(output, "\t", 2)
	if len(parts) < 2 {
		return false, 0, fmt.Errorf("unexpected tmux output: %q", output)
	}
	dead := parts[0] == "1"
	pid, _ := strconv.Atoi(parts[1])
	return dead, pid, nil
}

func (m *tmuxManager) respawnPane(ctx context.Context, paneID, command string, useKill bool) error {
	args := []string{"-L", m.socket, "respawn-pane"}
	if useKill {
		args = append(args, "-k")
	}
	args = append(args, "-t", paneID, command)
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    m.tmuxPath,
		Args:    args,
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("tmux respawn-pane: %w", err)
	}
	return nil
}

func getPaneProcess(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid pid")
	}
	commPath := filepath.Join("/proc", strconv.Itoa(pid), "comm")
	data, err := os.ReadFile(commPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
