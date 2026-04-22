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
