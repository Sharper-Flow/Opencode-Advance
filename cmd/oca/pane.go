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
// v1: SessionID, Directory, Ts only.
// v2: all fields, identified by schemaVersion == 2.
type paneState struct {
	// v1 fields (always present)
	SessionID string `json:"sessionID"`
	Directory string `json:"directory"`
	Ts        int64  `json:"ts"`

	// v2 fields (optional — zero-valued when absent)
	SchemaVersion    int    `json:"schemaVersion,omitempty"`
	StartedAt        int64  `json:"startedAt,omitempty"`
	LastSeenAt       int64  `json:"lastSeenAt,omitempty"`
	PaneID           string `json:"paneID,omitempty"`
	Socket           string `json:"socket,omitempty"`
	Agent            string `json:"agent,omitempty"`
	GitRoot          string `json:"gitRoot,omitempty"`
	WorktreePath     string `json:"worktreePath,omitempty"`
	GitCommonDir     string `json:"gitCommonDir,omitempty"`
	DefaultBranch    string `json:"defaultBranch,omitempty"`
	MainCheckoutPath string `json:"mainCheckoutPath,omitempty"`
	IsMainCheckout   bool   `json:"isMainCheckout,omitempty"`
	BranchSafety     string `json:"branchSafety,omitempty"`
	ProjectID        string `json:"projectId,omitempty"`
	WorktreeBranch   string `json:"worktreeBranch,omitempty"`
	ChangeID         string `json:"changeID,omitempty"`
	Role             string `json:"role,omitempty"`
}

// paneContext holds derived context computed from a paneState at read time.
// These fields are NOT persisted — they are computed on demand.
type paneContext struct {
	ChangeID         string // from explicit changeID or worktreeBranch "change/*"
	ProjectRoot      string // gitRoot
	ProjectID        string // projectId
	IsWorktree       bool   // worktreePath differs from gitRoot
	IsMainCheckout   bool   // worktreePath/gitRoot matches mainCheckoutPath
	WorktreePath     string // materialized checkout path
	WorktreeBranch   string // current branch
	DefaultBranch    string // default branch, when known
	MainCheckoutPath string // main checkout root, when known
	BranchSafety     string // worktree, safe_main_checkout, unsafe_main_branch, or unknown
	Role             string // explicit role or agent name
	Agent            string // agent name
}

// derivePaneContext computes derived context from a paneState without
// persisting stale window/project fields. Pure function, no side effects.
func derivePaneContext(ps *paneState) *paneContext {
	ctx := &paneContext{
		ProjectRoot:      ps.GitRoot,
		ProjectID:        ps.ProjectID,
		Agent:            ps.Agent,
		Role:             ps.Role,
		WorktreePath:     ps.WorktreePath,
		WorktreeBranch:   ps.WorktreeBranch,
		DefaultBranch:    ps.DefaultBranch,
		MainCheckoutPath: ps.MainCheckoutPath,
		BranchSafety:     ps.BranchSafety,
	}

	// Prefer git common-dir derived mainCheckoutPath over legacy gitRoot/path
	// comparison. In linked worktrees, gitRoot and worktreePath are both the
	// worktree root, while mainCheckoutPath points at the shared main checkout.
	if ps.MainCheckoutPath != "" {
		ctx.ProjectRoot = ps.MainCheckoutPath
		ctx.IsMainCheckout = ps.IsMainCheckout || ps.WorktreePath == ps.MainCheckoutPath || ps.GitRoot == ps.MainCheckoutPath
		ctx.IsWorktree = ps.WorktreePath != "" && ps.WorktreePath != ps.MainCheckoutPath
	} else {
		ctx.IsMainCheckout = ps.IsMainCheckout
		ctx.IsWorktree = ps.WorktreePath != "" && ps.WorktreePath != ps.GitRoot
	}

	if ctx.BranchSafety == "" {
		ctx.BranchSafety = deriveBranchSafety(ctx.IsWorktree, ctx.IsMainCheckout, ps.WorktreeBranch, ps.DefaultBranch)
	}

	// Derive changeID: prefer explicit field, then extract from branch.
	if ps.ChangeID != "" {
		ctx.ChangeID = ps.ChangeID
	} else if strings.HasPrefix(ps.WorktreeBranch, "change/") {
		id := strings.TrimPrefix(ps.WorktreeBranch, "change/")
		if id != "" {
			ctx.ChangeID = id
		}
	}

	// Role defaults to agent when not explicitly set.
	if ctx.Role == "" {
		ctx.Role = ctx.Agent
	}

	return ctx
}

func deriveBranchSafety(isWorktree, isMainCheckout bool, branch, defaultBranch string) string {
	if isWorktree {
		return "worktree"
	}
	if !isMainCheckout {
		return "unknown"
	}
	if defaultBranch != "" && branch != "" && branch != defaultBranch {
		return "unsafe_main_branch"
	}
	return "safe_main_checkout"
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
