package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
		Use:   "list",
		Short: "List OCA tmux sessions",
		Long:  "List all OCA-managed tmux sessions on the current socket.",
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
		Long:  "Kill OCA-managed tmux sessions with no activity for the specified duration.",
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
			res, err := subprocess.Run(ctx, subprocess.Cmd{
				Name:    "tmux",
				Args:    []string{"-L", socket, "list-sessions", "-F", "#{session_activity}\t#{session_attached}\t#{session_name}"},
				Timeout: 10 * time.Second,
			})
			if err != nil {
				output := string(res.Output)
				if strings.Contains(output, "no server running") ||
					strings.Contains(output, "no sessions") ||
					strings.Contains(output, "error connecting to") {
					if state.output == "json" {
						return printJSON(cmd.OutOrStdout(), map[string]any{"reaped": 0, "skipped": 0})
					}
					fmt.Fprintln(cmd.OutOrStdout(), "no sessions to reap")
					return nil
				}
				return newCLIError(1, "list sessions: %v", err)
			}
			now := time.Now().Unix()
			minAge := int64(5 * time.Minute / time.Second)
			threshold := int64(ageThreshold / time.Second)
			if threshold < minAge {
				threshold = minAge
			}
			var reaped []string
			var skipped []string
			for _, line := range strings.Split(string(res.Output), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "\t", 3)
				if len(parts) != 3 {
					continue
				}
				activityUnix, _ := strconv.ParseInt(parts[0], 10, 64)
				attached := parts[1] == "1"
				name := parts[2]
				if !strings.HasPrefix(name, "oca-") {
					continue
				}
				if attached {
					skipped = append(skipped, fmt.Sprintf("%s (attached)", name))
					continue
				}
				age := now - activityUnix
				if age > threshold {
					if dryRun {
						reaped = append(reaped, fmt.Sprintf("[DRY] %s (idle %s)", name, formatDuration(time.Duration(age)*time.Second)))
					} else {
						if err := mgr.Kill(ctx, name); err != nil {
							return newCLIError(1, "kill %s: %v", name, err)
						}
						reaped = append(reaped, fmt.Sprintf("%s (idle %s)", name, formatDuration(time.Duration(age)*time.Second)))
					}
				} else {
					skipped = append(skipped, fmt.Sprintf("%s (recent)", name))
				}
			}
			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]any{
					"reaped":  reaped,
					"skipped": skipped,
					"dry_run": dryRun,
				})
			}
			for _, r := range reaped {
				fmt.Fprintln(cmd.OutOrStdout(), r)
			}
			if len(reaped) == 0 && len(skipped) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no sessions to reap")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be reaped without killing")
	cmd.Flags().DurationVar(&ageThreshold, "age", 30*time.Minute, "Minimum age to consider stale")
	return cmd
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
