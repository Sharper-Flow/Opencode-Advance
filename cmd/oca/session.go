package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/session"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
	"github.com/spf13/cobra"
	operatorservicepb "go.temporal.io/api/operatorservice/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
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
	cmd.AddCommand(newSessionEnsureWindowCmd(state))
	cmd.AddCommand(newSessionReconcileCmd(state))
	cmd.AddCommand(newSessionDoctorCmd(state))
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
			return runSessionNew(cmd, state, sessionName, noSplash)
		},
	}

	cmd.Flags().StringVar(&sessionName, "name", "", "Session name (default: auto-generated oca-<slug>-<n>)")
	cmd.Flags().BoolVar(&noSplash, "no-splash", false, "Skip boot splash on session creation")

	return cmd
}

func runSessionNew(cmd *cobra.Command, state *commandState, sessionName string, noSplash bool) error {
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

	// Load stack config once — drives theme, watchdog, reaper, and
	// boot-splash decisions below. Failures are non-fatal: each
	// consumer falls back to v1 defaults so `oca session new` still
	// works in fresh environments before stack.toml exists.
	stack, stackErr := loadStack(state)

	// Preflight: warn about ADV runtime debt before creating session.
	// Never blocks session creation — warnings only.
	if stackErr == nil {
		runSessionPreflight(ctx, cmd.ErrOrStderr(), stack)
	}

	// Auto-generate name if not provided
	if sessionName == "" {
		sessionName, err = mgr.NextSessionName(ctx, repoSlug)
		if err != nil {
			return newCLIError(1, "generate session name: %v", err)
		}
	}

	// Resolve tmux conf from configured theme (default "obsidian"
	// on missing config or unset theme; empty conf path on unknown
	// theme = graceful degrade per design KD4).
	theme := defaultTheme
	if stackErr == nil && stack.Session != nil && stack.Session.Theme != "" {
		theme = stack.Session.Theme
	}
	tmuxConf := resolveTmuxConf(theme)

	err = mgr.Create(ctx, sessionName, workingDir, tmuxConf)
	if err != nil {
		return newCLIError(1, "create session: %v", err)
	}

	// Inject OCA_REPO_ROOT into tmux global env so status_bar.sh
	// can be found by obsidian.tmux.conf #() format expansions.
	repoRoot := mustGetRepoRoot()
	if repoRoot != "." {
		if err := mgr.SetGlobalEnv(ctx, "OCA_REPO_ROOT", repoRoot); err != nil {
			// Non-fatal: status bar degrades gracefully when unset
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to set OCA_REPO_ROOT: %v\n", err)
		}
	}

	// Inject watchdog config into tmux global environment so the OCA
	// plugin can read it at startup. Non-fatal on error.
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

	// Trigger boot splash. Default ON; controlled by [session].boot_splash.
	// CLI --no-splash always wins, even when config says true.
	bootSplash := true
	if stackErr == nil && stack.Session != nil {
		bootSplash = stack.Session.BootSplash
	}
	if bootSplash && !noSplash {
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

			// Enrich with ADV workspace projection when available.
			var projectID string
			if root, err := session.ProjectRoot(ctx, ""); err == nil {
				projectID = resolveProjectID(root)
			}
			projection, _ := advruntime.ProjectWorkspaceStates(projectID)
			views := make([]advruntime.SessionView, len(sessions))
			for i, s := range sessions {
				views[i] = advruntime.SessionView{Name: s.Name, Attached: s.Attached, Path: s.Path}
			}
			enriched := advruntime.EnrichSessionsWithWorkspaceState(views, projection)

			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), enriched)
			}

			if len(sessions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no OCA sessions")
				return nil
			}

			for _, e := range enriched {
				attached := "detached"
				if e.Attached {
					attached = "attached"
				}
				line := fmt.Sprintf("%s\t%s", e.Name, attached)
				if e.WorkspaceStatus != "" {
					line += fmt.Sprintf("\t%s", e.WorkspaceStatus)
					if e.WorkspaceFailure != "" {
						line += fmt.Sprintf(" (%s)", e.WorkspaceFailure)
					}
				}
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}

	return cmd
}

// resolveProjectID computes the ADV project ID (root commit SHA) for a git
// repository root. Returns empty string on any failure.
func resolveProjectID(repoRoot string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return ""
	}
	// Match ADV's project-id logic: first commit hash of the repo.
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    gitPath,
		Args:    []string{"-C", repoRoot, "rev-list", "--max-parents=0", "HEAD"},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(res.Output))
}

// sessionSocket returns the tmux socket name for OCA sessions.
// Defaults to "oca" but can be overridden via OCA_TMUX_SOCKET.
func sessionSocket() string {
	if s := os.Getenv("OCA_TMUX_SOCKET"); s != "" {
		return s
	}
	return "oca"
}

// defaultTheme is the v1 default theme name. Used when [session].theme is
// unset (or config load fails entirely).
const defaultTheme = "obsidian"

// validThemeName accepts only basename-like theme IDs: letters, digits,
// underscore, hyphen, and dot. Rejects path separators, "..", and empty.
func validThemeName(theme string) bool {
	if theme == "" {
		return false
	}
	if strings.Contains(theme, "..") {
		return false
	}
	for _, r := range theme {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

// resolveTmuxConf returns the path to the named tmux theme conf asset, or
// the empty string if no asset can be found. An empty theme name is treated
// as "use the default" (obsidian). An unknown theme name (no matching file)
// degrades to "" so Manager.Create runs without a -f flag rather than
// failing — matches the design's invalid-theme contract (KD4).
//
// Lookup order matches resolveTmuxConf's pre-Phase 6 behavior:
//  1. $OCA_ASSETS_ROOT/themes/<theme>.tmux.conf (test override)
//  2. <executable-dir>/../assets/themes/<theme>.tmux.conf (installed)
//  3. assets/themes/<theme>.tmux.conf (development checkout, relative to cwd)
func resolveTmuxConf(theme string) string {
	if theme == "" {
		theme = defaultTheme
	}
	if !validThemeName(theme) {
		return ""
	}
	filename := theme + ".tmux.conf"

	// 1. OCA_ASSETS_ROOT (test/dev override)
	if root := os.Getenv("OCA_ASSETS_ROOT"); root != "" {
		p := filepath.Join(root, "themes", filename)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// 2. Relative to the running executable (installed binary).
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "assets", "themes", filename)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// 3. Relative to cwd (development checkout).
	dev := filepath.Join("assets", "themes", filename)
	if _, err := os.Stat(dev); err == nil {
		abs, _ := filepath.Abs(dev)
		return abs
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
			// Resolve theme from config (default "obsidian"). Same fallback
			// rules as `oca session new` — config load failure → default.
			theme := defaultTheme
			if stack, stackErr := loadStack(state); stackErr == nil && stack.Session != nil && stack.Session.Theme != "" {
				theme = stack.Session.Theme
			}
			tmuxConf := resolveTmuxConf(theme)
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

func newSessionEnsureWindowCmd(state *commandState) *cobra.Command {
	var sessionName string
	var windowName string
	var workingDir string
	cmd := &cobra.Command{
		Use:   "ensure-window",
		Short: "Ensure a Pattern B project window exists",
		Long:  "Create or reuse a named window in a Pattern B project tmux session with cwd set to a worktree path.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if workingDir == "" {
				wd, err := os.Getwd()
				if err != nil {
					return newCLIError(1, "get working directory: %v", err)
				}
				workingDir = wd
			}
			if windowName == "" {
				windowName = filepath.Base(workingDir)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if sessionName == "" {
				projectRoot, err := session.ProjectRoot(ctx, workingDir)
				if err != nil {
					return newCLIError(2, "infer project session: %v", err)
				}
				name, err := session.ProjectSessionName(projectRoot)
				if err != nil {
					return newCLIError(2, "infer project session name: %v", err)
				}
				sessionName = name
			}

			mgr, err := session.NewManager(sessionSocket())
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}
			w, created, err := mgr.EnsureWindow(ctx, sessionName, windowName, workingDir)
			if err != nil {
				return newCLIError(1, "ensure window: %v", err)
			}
			status := "existing"
			if created {
				status = "created"
			}
			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]any{"session": sessionName, "window": w, "status": status})
			}
			if created {
				fmt.Fprintf(cmd.OutOrStdout(), "window %s ensured in session %s (created)\n", windowName, sessionName)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "window %s ensured in session %s (already existed)\n", windowName, sessionName)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionName, "session", "", "Project session name (default: inferred from git root)")
	cmd.Flags().StringVar(&windowName, "name", "", "Window name (default: cwd basename)")
	cmd.Flags().StringVar(&workingDir, "cwd", "", "Window working directory (default: current directory)")
	return cmd
}

func newSessionReconcileCmd(state *commandState) *cobra.Command {
	var sessionName string
	var worktreeRoot string
	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Ensure Pattern B windows for ADV worktrees",
		Long:  "Scan the ADV worktree layout and create missing project-session windows. No filesystem watcher is required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if sessionName == "" {
				wd, err := os.Getwd()
				if err != nil {
					return newCLIError(1, "get working directory: %v", err)
				}
				projectRoot, err := session.ProjectRoot(ctx, wd)
				if err != nil {
					return newCLIError(2, "infer project session: %v", err)
				}
				name, err := session.ProjectSessionName(projectRoot)
				if err != nil {
					return newCLIError(2, "infer project session name: %v", err)
				}
				sessionName = name
			}

			changeDirs, err := advChangeWorktreeDirs(worktreeRoot)
			if err != nil {
				return newCLIError(1, "scan ADV worktrees: %v", err)
			}
			mgr, err := session.NewManager(sessionSocket())
			if err != nil {
				return newCLIError(1, "session manager: %v", err)
			}

			type ensuredWindow struct {
				Name   string `json:"name"`
				Path   string `json:"path"`
				Status string `json:"status"`
			}
			ensured := make([]ensuredWindow, 0, len(changeDirs))
			for _, dir := range changeDirs {
				name := filepath.Base(dir)
				_, created, err := mgr.EnsureWindow(ctx, sessionName, name, dir)
				if err != nil {
					return newCLIError(1, "ensure %s: %v", name, err)
				}
				status := "existing"
				if created {
					status = "created"
				}
				ensured = append(ensured, ensuredWindow{Name: name, Path: dir, Status: status})
			}

			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]any{"session": sessionName, "windows": ensured})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "reconciled %d window(s) in session %s\n", len(ensured), sessionName)
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionName, "session", "", "Project session name (default: inferred from git root)")
	cmd.Flags().StringVar(&worktreeRoot, "worktree-root", "", "ADV worktree root (default: $XDG_DATA_HOME/opencode/worktree)")
	return cmd
}

func advChangeWorktreeDirs(worktreeRoot string) ([]string, error) {
	if worktreeRoot == "" {
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("resolve home directory: %w", err)
			}
			xdg = filepath.Join(home, ".local", "share")
		}
		worktreeRoot = filepath.Join(xdg, "opencode", "worktree")
	}
	matches, err := filepath.Glob(filepath.Join(worktreeRoot, "*", "change", "*"))
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || !info.IsDir() {
			continue
		}
		dirs = append(dirs, match)
	}
	return dirs, nil
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

func newSessionDoctorCmd(state *commandState) *cobra.Command {
	var (
		dbPath    string
		threshold time.Duration
		apply     bool
		backupDir string
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose and clean up stale OpenCode session debt",
		Long: "Scan the OpenCode SQLite database for stale blank assistant messages. " +
			"By default, runs in dry-run mode. Use --apply --backup-dir to delete repairable rows.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			// Resolve DB path
			if dbPath == "" {
				dbPath = defaultOpenCodeDBPath()
			}

			// Open DB
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return newCLIError(1, "open database %s: %v", dbPath, err)
			}
			defer db.Close()

			config := advruntime.Config{SessionStaleAfter: threshold}
			scanner := advruntime.NewSessionDebtScanner(db, config)

			ctx := context.Background()
			finding, err := scanner.Scan(ctx)
			if err != nil {
				return newCLIError(1, "scan session debt: %v", err)
			}

			// Dry-run mode: just print findings
			if !apply {
				if state.output == "json" {
					return printJSON(cmd.OutOrStdout(), finding)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Session doctor (dry-run)\n")
				fmt.Fprintf(cmd.OutOrStdout(), "  DB:         %s\n", dbPath)
				fmt.Fprintf(cmd.OutOrStdout(), "  Status:     %s\n", finding.Status)
				fmt.Fprintf(cmd.OutOrStdout(), "  Repairable: %d\n", finding.Repairable)
				fmt.Fprintf(cmd.OutOrStdout(), "  Ignored:    %d\n", finding.Ignored)
				if finding.OldestAge > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "  Oldest:     %s\n", finding.OldestAge.Round(time.Second))
				}
				if finding.Message != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Message:    %s\n", finding.Message)
				}
				if finding.Hint != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Hint:       %s\n", finding.Hint)
				}
				return nil
			}

			// Apply mode: require backup dir
			if backupDir == "" {
				return newCLIError(2, "--apply requires --backup-dir for safety")
			}

			manifest, err := scanner.ApplyDeletion(ctx, advruntime.ApplyDeletionOptions{
				BackupDir: backupDir,
			})
			if err != nil {
				return newCLIError(1, "apply deletion: %v", err)
			}

			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), map[string]any{
					"finding":  finding,
					"manifest": manifest,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Session doctor (APPLY)\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  DB:        %s\n", dbPath)
			fmt.Fprintf(cmd.OutOrStdout(), "  Backup:    %s\n", manifest.BackupPath)
			fmt.Fprintf(cmd.OutOrStdout(), "  Deleted:   %d row(s)\n", len(manifest.DeletedIDs))
			fmt.Fprintf(cmd.OutOrStdout(), "  Threshold: %s\n", threshold)
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to OpenCode SQLite database (default: $XDG_DATA_HOME/opencode/opencode.db)")
	cmd.Flags().DurationVar(&threshold, "threshold", 5*time.Minute, "Age threshold for stale messages")
	cmd.Flags().BoolVar(&apply, "apply", false, "Actually delete repairable rows (requires --backup-dir)")
	cmd.Flags().StringVar(&backupDir, "backup-dir", "", "Directory to store database backup before deletion (required with --apply)")

	return cmd
}

// operatorServiceAdapter adapts the gRPC client to advruntime.OperatorService.
type operatorServiceAdapter struct {
	client interface {
		ListSearchAttributes(context.Context, *operatorservicepb.ListSearchAttributesRequest, ...grpc.CallOption) (*operatorservicepb.ListSearchAttributesResponse, error)
	}
}

func (a *operatorServiceAdapter) ListSearchAttributes(ctx context.Context, req *operatorservicepb.ListSearchAttributesRequest) (*operatorservicepb.ListSearchAttributesResponse, error) {
	return a.client.ListSearchAttributes(ctx, req)
}

// workflowServiceAdapter adapts the gRPC client to advruntime.WorkflowService.
type workflowServiceAdapter struct {
	client interface {
		ListWorkflowExecutions(context.Context, *workflowservicepb.ListWorkflowExecutionsRequest, ...grpc.CallOption) (*workflowservicepb.ListWorkflowExecutionsResponse, error)
		DescribeTaskQueue(context.Context, *workflowservicepb.DescribeTaskQueueRequest, ...grpc.CallOption) (*workflowservicepb.DescribeTaskQueueResponse, error)
	}
}

func (a *workflowServiceAdapter) ListWorkflowExecutions(ctx context.Context, req *workflowservicepb.ListWorkflowExecutionsRequest) (*workflowservicepb.ListWorkflowExecutionsResponse, error) {
	return a.client.ListWorkflowExecutions(ctx, req)
}

func (a *workflowServiceAdapter) DescribeTaskQueue(ctx context.Context, req *workflowservicepb.DescribeTaskQueueRequest) (*workflowservicepb.DescribeTaskQueueResponse, error) {
	return a.client.DescribeTaskQueue(ctx, req)
}

// runSessionPreflight inspects ADV runtime state and prints warnings to stderr.
// It never blocks or fails — warnings are purely advisory.
func runSessionPreflight(ctx context.Context, stderr io.Writer, stack *cfg.Stack) {
	if stack == nil || stack.Temporal == nil || stack.Temporal.Address == "" {
		return // Temporal not configured — nothing to preflight
	}

	config := advruntime.Config{
		Address:   stack.Temporal.Address,
		Namespace: stack.Temporal.Namespace,
	}
	if config.Namespace == "" {
		config.Namespace = advruntime.DefaultNamespace
	}

	provider := advruntime.NewTemporalClientProvider(config, nil)
	defer provider.Close()

	client, err := provider.Client(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "warning: ADV preflight: Temporal unreachable: %v\n", err)
		return
	}

	// Check search attributes
	checker := advruntime.NewSearchAttributeChecker(&operatorServiceAdapter{client.OperatorService()}, config)
	report, _ := checker.Check(ctx)
	for _, attr := range report.SearchAttributes {
		if attr.Status == advruntime.StatusFail || attr.Status == advruntime.StatusUnknown {
			fmt.Fprintf(stderr, "warning: ADV search attribute %s is %s: %s\n", attr.Name, attr.Status, attr.Message)
		}
	}

	// Check workflow queues
	classifier := advruntime.NewWorkflowClassifier(&workflowServiceAdapter{client.WorkflowService()}, config, nil)
	report, _ = classifier.Classify(ctx)
	for _, queue := range report.WorkflowQueues {
		if queue.Status == advruntime.StatusWarn || queue.Status == advruntime.StatusFail {
			fmt.Fprintf(stderr, "warning: ADV queue %s: %s\n", queue.TaskQueue, queue.Message)
		}
	}
}

func defaultOpenCodeDBPath() string {
	return advruntime.DefaultOpenCodeDBPath()
}
