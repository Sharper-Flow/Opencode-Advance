package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/session"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
	"github.com/spf13/cobra"
)

// newSessionCmd creates the "oca session" command group.
// This is the first cobra subcommand group in the OCA CLI.
// Convention: parent command has no RunE (shows help); children
// handle all logic. Future groups (e.g. "oca theme") should
// follow the same pattern.
func newSessionCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage OCA tmux sessions",
		Long:  "Create and list OpenCode Advance tmux sessions with Obsidian theming.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newSessionNewCmd(state))
	cmd.AddCommand(newSessionListCmd(state))
	cmd.AddCommand(newSessionAttachCmd(state))
	cmd.AddCommand(newSessionSwitchCmd(state))
	cmd.AddCommand(newSessionKillCmd(state))
	cmd.AddCommand(newSessionKillallCmd(state))
	cmd.AddCommand(newSessionRestartCmd(state))
	cmd.AddCommand(newSessionReapCmd(state))
	return cmd
}

func newSessionNewCmd(state *commandState) *cobra.Command {
	var sessionName string
	var noSplash bool

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new OCA tmux session",
		Long:  "Create a new detached tmux session with Obsidian theming and optional boot splash.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			socket := sessionSocket()
			mgr, err := session.NewManager(socket)
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}

			workingDir, err := os.Getwd()
			if err != nil {
				return newCLIError(1, "get working directory: %v", err)
			}

			// Resolve repo slug from working dir basename
			repoSlug := filepath.Base(workingDir)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			// Auto-generate name if not provided
			if sessionName == "" {
				sessionName, err = mgr.NextSessionName(ctx, repoSlug)
				if err != nil {
					return newCLIError(1, "generate session name: %v", err)
				}
			}

			// Resolve tmux conf path
			tmuxConf := resolveTmuxConf()

			err = mgr.Create(ctx, sessionName, workingDir, tmuxConf)
			if err != nil {
				return newCLIError(1, "create session: %v", err)
			}

			// Inject OCA_REPO_ROOT into tmux global env so status_bar.sh
			// can be found by obsidian.tmux.conf #() expansions.
			repoRoot := mustGetRepoRoot()
			if repoRoot != "." {
				if err := mgr.SetGlobalEnv(ctx, "OCA_REPO_ROOT", repoRoot); err != nil {
					// Non-fatal: status bar degrades gracefully when unset
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to set OCA_REPO_ROOT: %v\n", err)
				}
			}

			// Inject watchdog config into tmux global env so the OCA
			// plugin can read it at startup. Non-fatal on error.
			stack, stackErr := loadStack(state)
			if stackErr == nil && stack.Session != nil && stack.Session.Watchdog != nil {
				if err := mgr.ApplyWatchdogEnv(ctx, stack.Session.Watchdog); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to set watchdog env: %v\n", err)
				}
			}

			// Auto-reap stale sessions in the background. Default ON; controlled
			// by [session].reaper. Fire-and-forget — must not block session
			// creation. Uses context.Background() with its own timeout so the
			// 15s cobra ctx cancelling on RunE return does not cancel the
			// reap mid-flight.
			autoReap := true
			reapThreshold := 4 * time.Hour
			if stackErr == nil && stack.Session != nil {
				autoReap = stack.Session.Reaper
				if stack.Session.ReaperThreshold > 0 {
					reapThreshold = stack.Session.ReaperThreshold
				}
			}
			if autoReap {
				go func(threshold time.Duration) {
					bgCtx, bgCancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer bgCancel()
					if _, err := mgr.ReapStale(bgCtx, threshold); err != nil {
						fmt.Fprintf(os.Stderr, "warning: auto-reap: %v\n", err)
					}
				}(reapThreshold)
			}

			// Trigger boot splash unless --no-splash
			if !noSplash {
				triggerSplash(ctx, socket, sessionName, state)
			}

			if state.output == "json" {
				type result struct {
					Session string `json:"session"`
					Socket  string `json:"socket"`
					Status  string `json:"status"`
				}
				return printJSON(cmd.OutOrStdout(), result{
					Session: sessionName,
					Socket:  socket,
					Status:  "created",
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "session %s created (socket: %s)\n", sessionName, socket)
			return nil
		},
	}

	cmd.Flags().StringVar(&sessionName, "name", "", "Session name (default: auto-generated oca-<slug>-<n>)")
	cmd.Flags().BoolVar(&noSplash, "no-splash", false, "Skip boot splash on session creation")

	return cmd
}

func newSessionListCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List OCA tmux sessions",
		Long:    "List all OCA-managed tmux sessions on the current socket.",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			socket := sessionSocket()
			mgr, err := session.NewManager(socket)
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			sessions, err := mgr.List(ctx)
			if err != nil {
				return newCLIError(1, "list sessions: %v", err)
			}

			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), sessions)
			}

			if len(sessions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no OCA sessions")
				return nil
			}

			for _, s := range sessions {
				attached := "detached"
				if s.Attached {
					attached = "attached"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", s.Name, attached)
			}
			return nil
		},
	}

	return cmd
}

// sessionSocket returns the tmux socket name for OCA sessions.
// Defaults to "oca" but can be overridden via OCA_TMUX_SOCKET.
func sessionSocket() string {
	if s := os.Getenv("OCA_TMUX_SOCKET"); s != "" {
		return s
	}
	return "oca"
}

// resolveTmuxConf returns the path to the Obsidian tmux conf asset.
// Returns empty string if not found (session created without theming).
func resolveTmuxConf() string {
	// Check OCA_ASSETS_ROOT first (for testing)
	if root := os.Getenv("OCA_ASSETS_ROOT"); root != "" {
		p := filepath.Join(root, "themes", "obsidian.tmux.conf")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "assets", "themes", "obsidian.tmux.conf")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Try relative to repo root (for development)
	candidates := []string{
		"assets/themes/obsidian.tmux.conf",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	return ""
}

// triggerSplash sends the boot splash command into the new session.
func triggerSplash(ctx context.Context, socket, sessionName string, state *commandState) {
	version := state.opts.Version.Version
	if version == "" {
		version = "dev"
	}

	// Build the splash command
	splashCmd := fmt.Sprintf("OCA_SPLASH_VERSION=%s OCA_SPLASH_DIR=%s %s/lib/boot_splash.sh",
		version,
		shellEscape(mustGetwd()),
		mustGetRepoRoot(),
	)

	subprocess.Run(ctx, subprocess.Cmd{
		Name: "tmux",
		Args: []string{"-L", socket, "send-keys", "-t", sessionName, splashCmd, "Enter"},
	})
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func mustGetRepoRoot() string {
	// Walk up from cwd to find lib/boot_splash.sh
	wd := mustGetwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "lib", "boot_splash.sh")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "."
}

// shellEscape wraps a string in single quotes for shell safety.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func newSessionAttachCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach <name>",
		Short: "Attach to an existing OCA tmux session",
		Long:  "Attach to an existing OCA tmux session, replacing the current process.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			socket := sessionSocket()
			mgr, err := session.NewManager(socket)
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return mgr.Attach(ctx, args[0])
		},
	}
	return cmd
}

func newSessionSwitchCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch the current tmux client to a different session",
		Long:  "Switch the current tmux client to a different OCA-managed session.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			socket := sessionSocket()
			mgr, err := session.NewManager(socket)
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := mgr.SwitchClient(ctx, args[0]); err != nil {
				return newCLIError(1, "switch client: %v", err)
			}
			return nil
		},
	}
	return cmd
}

func newSessionKillCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kill <name>",
		Short: "Kill an OCA tmux session",
		Long:  "Destroy a specific OCA-managed tmux session.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			socket := sessionSocket()
			mgr, err := session.NewManager(socket)
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := mgr.Kill(ctx, args[0]); err != nil {
				return newCLIError(1, "kill session: %v", err)
			}
			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]string{"session": args[0], "status": "killed"})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "session %s killed\n", args[0])
			return nil
		},
	}
	return cmd
}

func newSessionKillallCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "killall",
		Short: "Kill all OCA tmux sessions",
		Long:  "Destroy all OCA-managed tmux sessions on the current socket.",
		RunE: func(cmd *cobra.Command, args []string) error {
			socket := sessionSocket()
			mgr, err := session.NewManager(socket)
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			count, err := mgr.KillAll(ctx)
			if err != nil {
				return newCLIError(1, "kill all sessions: %v", err)
			}
			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]any{"count": count, "status": "killed"})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "killed %d session(s)\n", count)
			return nil
		},
	}
	return cmd
}

func newSessionRestartCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart an OCA tmux session",
		Long:  "Kill and recreate an OCA-managed tmux session with the same name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			socket := sessionSocket()
			mgr, err := session.NewManager(socket)
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			workingDir, err := os.Getwd()
			if err != nil {
				return newCLIError(1, "get working directory: %v", err)
			}
			tmuxConf := resolveTmuxConf()
			if err := mgr.Restart(ctx, args[0], workingDir, tmuxConf); err != nil {
				return newCLIError(1, "restart session: %v", err)
			}
			// Re-inject OCA_REPO_ROOT after restart (new server may not inherit)
			repoRoot := mustGetRepoRoot()
			if repoRoot != "." {
				_ = mgr.SetGlobalEnv(ctx, "OCA_REPO_ROOT", repoRoot)
			}
			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]string{"session": args[0], "status": "restarted"})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "session %s restarted\n", args[0])
			return nil
		},
	}
	return cmd
}

func newSessionReapCmd(state *commandState) *cobra.Command {
	var dryRun bool
	var ageThreshold time.Duration
	cmd := &cobra.Command{
		Use:   "reap",
		Short: "Reap stale OCA tmux sessions",
		Long: "Kill OCA-managed tmux sessions with no activity for the specified duration. " +
			"Threshold has a 5-minute floor enforced by session.Manager.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			socket := sessionSocket()
			mgr, err := session.NewManager(socket)
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			candidates, err := mgr.ReapCandidates(ctx, ageThreshold)
			if err != nil {
				return newCLIError(1, "list sessions: %v", err)
			}

			if dryRun {
				return printReapResult(cmd, state, candidates, true)
			}

			// Production path — actually kill them.
			var reaped []session.ReapCandidate
			for _, c := range candidates {
				if err := mgr.Kill(ctx, c.Name); err != nil {
					return newCLIError(1, "kill %s: %v", c.Name, err)
				}
				reaped = append(reaped, c)
			}
			return printReapResult(cmd, state, reaped, false)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be reaped without killing")
	cmd.Flags().DurationVar(&ageThreshold, "age", 30*time.Minute, "Minimum age to consider stale (5m floor enforced)")
	return cmd
}

// printReapResult renders the reap output for both dry-run and production
// paths. For dry-run, names are prefixed with "[DRY]".
func printReapResult(cmd *cobra.Command, state *commandState, candidates []session.ReapCandidate, dryRun bool) error {
	if state.output == "json" {
		names := make([]string, 0, len(candidates))
		for _, c := range candidates {
			label := c.Name
			if dryRun {
				label = "[DRY] " + label
			}
			names = append(names, fmt.Sprintf("%s (idle %s)", label, formatDuration(c.Age)))
		}
		return printJSON(cmd.OutOrStdout(), map[string]any{
			"reaped":  names,
			"dry_run": dryRun,
		})
	}
	if len(candidates) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no sessions to reap")
		return nil
	}
	for _, c := range candidates {
		prefix := ""
		if dryRun {
			prefix = "[DRY] "
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s%s (idle %s)\n", prefix, c.Name, formatDuration(c.Age))
	}
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
