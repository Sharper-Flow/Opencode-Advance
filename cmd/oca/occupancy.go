package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/occupancy"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
	"github.com/spf13/cobra"
)

// occupancyJSONOutput is the JSON envelope for `oca occupancy --json`.
type occupancyJSONOutput struct {
	Groups    []occupancyGroupJSON `json:"groups"`
	Warnings  []occupancyWarnJSON  `json:"warnings"`
	Malformed int                  `json:"malformed"`
	Stale     int                  `json:"stale"`
}

type occupancyGroupJSON struct {
	ProjectID    string         `json:"projectId,omitempty"`
	WorktreePath string         `json:"worktreePath"`
	Branch       string         `json:"branch,omitempty"`
	Occupants    []occupantJSON `json:"occupants"`
}

type occupantJSON struct {
	SessionID   string `json:"sessionId"`
	PaneID      string `json:"paneID,omitempty"`
	Agent       string `json:"agent,omitempty"`
	LastSeenAgo string `json:"lastSeenAgo,omitempty"`
	Liveness    string `json:"liveness"`
}

type occupancyWarnJSON struct {
	WorktreePath string `json:"worktreePath"`
	ActiveCount  int    `json:"activeCount"`
}

func newOccupancyCmd(state *commandState) *cobra.Command {
	var showAll bool
	var jsonOut bool
	var socket string
	var pathFilter string
	var statusMode bool
	var paneID string

	cmd := &cobra.Command{
		Use:   "occupancy",
		Short: "Show worktree occupancy across sessions",
		Long:  "Display which sessions occupy which worktrees. Groups by project/worktree and warns on multiple active occupants.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if statusMode {
				return runOccupancyStatus(cmd, paneID)
			}
			return runOccupancyFull(cmd, state.output == "json" || jsonOut, showAll, socket, pathFilter)
		},
	}
	cmd.Flags().BoolVar(&showAll, "all", false, "Show stale records in addition to active")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&socket, "socket", "", "Filter by tmux socket name (default: all sockets)")
	cmd.Flags().StringVar(&pathFilter, "path", "", "Filter by directory path prefix")
	cmd.Flags().BoolVar(&statusMode, "status", false, "Compact single-pane status output (requires --pane)")
	cmd.Flags().StringVar(&paneID, "pane", "", "Target pane ID for --status mode")
	return cmd
}

func occupancyStateDir() string {
	return filepath.Join(xdgStateHome(), "oca", "panes")
}

func runOccupancyFull(cmd *cobra.Command, jsonOut, showAll bool, socket, pathFilter string) error {
	stateDir := occupancyStateDir()

	records, malformed, err := occupancy.DiscoverRecords(stateDir)
	if err != nil {
		return newCLIError(1, "discover records: %v", err)
	}

	// Filter by socket if specified
	if socket != "" {
		var filtered []occupancy.PaneRecord
		for _, r := range records {
			if r.Socket == socket {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	// Filter by path prefix if specified
	if pathFilter != "" {
		var filtered []occupancy.PaneRecord
		for _, r := range records {
			if strings.HasPrefix(r.Directory, pathFilter) || strings.HasPrefix(r.WorktreePath, pathFilter) {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	// Get live tmux panes for liveness classification
	livePanes := getLivePanes()
	tmuxCwd := getTmuxCwdMap()

	// Reconcile and classify
	classified := occupancy.Reconcile(records, livePanes, tmuxCwd)

	// Count stale
	staleCount := 0
	for _, c := range classified {
		if c.Liveness == occupancy.Stale {
			staleCount++
		}
	}

	// Filter out stale unless --all
	var visible []occupancy.ClassifiedRecord
	for _, c := range classified {
		if c.Liveness == occupancy.Stale && !showAll {
			continue
		}
		visible = append(visible, c)
	}

	// Get warnings (from all classified, not just visible)
	warnings := occupancy.OccupancyWarnings(classified)

	if jsonOut {
		return printOccupancyJSON(cmd.OutOrStdout(), visible, warnings, malformed, staleCount)
	}
	return printOccupancyHuman(cmd.OutOrStdout(), visible, warnings, malformed, staleCount)
}

func runOccupancyStatus(cmd *cobra.Command, paneID string) error {
	if paneID == "" {
		paneID = os.Getenv("TMUX_PANE")
	}
	if paneID == "" {
		fmt.Fprint(cmd.OutOrStdout(), "?")
		return nil
	}

	stateDir := occupancyStateDir()
	records, _, err := occupancy.DiscoverRecords(stateDir)
	if err != nil {
		fmt.Fprint(cmd.OutOrStdout(), "?")
		return nil
	}

	// Find record matching this pane (compare sanitized IDs)
	sanitized := strings.TrimPrefix(paneID, "%")
	for _, r := range records {
		recSanitized := strings.TrimPrefix(r.PaneID, "%")
		if recSanitized == sanitized {
			worktree := r.WorktreePath
			if worktree == "" {
				worktree = r.Directory
			}
			count := 0
			for _, r2 := range records {
				wt2 := r2.WorktreePath
				if wt2 == "" {
					wt2 = r2.Directory
				}
				if wt2 == worktree {
					count++
				}
			}

			branch := r.WorktreeBranch
			if branch == "" {
				branch = "?"
			}
			if count == 1 {
				fmt.Fprintf(cmd.OutOrStdout(), "1× %s", branch)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%d× occupied ⚠", count)
			}
			return nil
		}
	}

	fmt.Fprint(cmd.OutOrStdout(), "?")
	return nil
}

func printOccupancyHuman(w io.Writer, records []occupancy.ClassifiedRecord, warnings []occupancy.OccupancyWarning, malformed, stale int) error {
	// Group by project
	byProject := make(map[string][]occupancy.ClassifiedRecord)
	for _, r := range records {
		pid := r.ProjectID
		if pid == "" {
			pid = r.GitRoot
		}
		if pid == "" {
			pid = r.Directory
		}
		byProject[pid] = append(byProject[pid], r)
	}

	var pids []string
	for pid := range byProject {
		pids = append(pids, pid)
	}
	sort.Strings(pids)

	for _, pid := range pids {
		recs := byProject[pid]
		shortID := pid
		if len(pid) > 8 {
			shortID = pid[:8]
		}
		fmt.Fprintf(w, "Project: %s  %s\n", filepath.Base(recs[0].Directory), shortID)

		byWT := make(map[string][]occupancy.ClassifiedRecord)
		for _, r := range recs {
			wt := r.WorktreePath
			if wt == "" {
				wt = r.Directory
			}
			byWT[wt] = append(byWT[wt], r)
		}

		var wts []string
		for wt := range byWT {
			wts = append(wts, wt)
		}
		sort.Strings(wts)

		for _, wt := range wts {
			wtRecs := byWT[wt]
			branch := wtRecs[0].WorktreeBranch
			if branch == "" {
				branch = "?"
			}
			fmt.Fprintf(w, "  Worktree: %s  %s\n", branch, wt)
			for _, r := range wtRecs {
				icon := "✓"
				if r.Liveness == occupancy.Stale {
					icon = "✗"
				} else if r.Liveness == occupancy.Unknown {
					icon = "?"
				}
				agent := r.Agent
				if agent == "" {
					agent = "unknown"
				}
				lastSeen := ""
				if r.LastSeenAt > 0 {
					lastSeen = formatDuration(time.Since(time.UnixMilli(r.LastSeenAt)))
				}
				fmt.Fprintf(w, "    %s %s  %s:%s  %s  last-seen %s\n",
					icon, r.PaneID, r.SessionName, r.WindowName, agent, lastSeen)
			}
		}
	}

	for _, warn := range warnings {
		fmt.Fprintf(w, "    warning: %d active sessions share this worktree\n", warn.ActiveCount)
	}

	if malformed > 0 || stale > 0 {
		fmt.Fprintf(w, "\nWarnings: ")
		var parts []string
		if malformed > 0 {
			parts = append(parts, fmt.Sprintf("%d malformed record(s) skipped", malformed))
		}
		if stale > 0 {
			parts = append(parts, fmt.Sprintf("%d stale record(s) hidden (use --all)", stale))
		}
		fmt.Fprintf(w, "%s\n", strings.Join(parts, ", "))
	}
	return nil
}

func printOccupancyJSON(w io.Writer, records []occupancy.ClassifiedRecord, warnings []occupancy.OccupancyWarning, malformed, stale int) error {
	byWT := make(map[string][]occupancy.ClassifiedRecord)
	for _, r := range records {
		wt := r.WorktreePath
		if wt == "" {
			wt = r.Directory
		}
		byWT[wt] = append(byWT[wt], r)
	}

	var groups []occupancyGroupJSON
	for wt, recs := range byWT {
		g := occupancyGroupJSON{
			ProjectID:    recs[0].ProjectID,
			WorktreePath: wt,
			Branch:       recs[0].WorktreeBranch,
		}
		for _, r := range recs {
			lastSeen := ""
			if r.LastSeenAt > 0 {
				lastSeen = formatDuration(time.Since(time.UnixMilli(r.LastSeenAt)))
			}
			g.Occupants = append(g.Occupants, occupantJSON{
				SessionID:   r.SessionID,
				PaneID:      r.PaneID,
				Agent:       r.Agent,
				LastSeenAgo: lastSeen,
				Liveness:    r.Liveness.String(),
			})
		}
		groups = append(groups, g)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].WorktreePath < groups[j].WorktreePath
	})

	var warnJSON []occupancyWarnJSON
	for _, w := range warnings {
		warnJSON = append(warnJSON, occupancyWarnJSON{
			WorktreePath: w.WorktreePath,
			ActiveCount:  w.ActiveCount,
		})
	}

	return printJSON(w, occupancyJSONOutput{
		Groups:    groups,
		Warnings:  warnJSON,
		Malformed: malformed,
		Stale:     stale,
	})
}

// getLivePanes queries tmux for currently active pane IDs.
// Returns empty map if tmux is unavailable.
func getLivePanes() map[string]bool {
	panes := make(map[string]bool)
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return panes
	}

	for _, sock := range listPaneSockets() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		res, err := subprocess.Run(ctx, subprocess.Cmd{
			Name:    tmuxPath,
			Args:    []string{"-L", sock, "list-panes", "-a", "-F", "#{pane_id}"},
			Timeout: 2 * time.Second,
		})
		cancel()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.TrimSpace(string(res.Output)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				panes[line] = true
			}
		}
	}
	return panes
}

// getTmuxCwdMap returns paneID → current working directory from tmux.
func getTmuxCwdMap() map[string]string {
	cwd := make(map[string]string)
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return cwd
	}

	for _, sock := range listPaneSockets() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		res, err := subprocess.Run(ctx, subprocess.Cmd{
			Name:    tmuxPath,
			Args:    []string{"-L", sock, "list-panes", "-a", "-F", "#{pane_id}\t#{pane_current_path}"},
			Timeout: 2 * time.Second,
		})
		cancel()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.TrimSpace(string(res.Output)), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				cwd[parts[0]] = parts[1]
			}
		}
	}
	return cwd
}

// listPaneSockets returns socket directory names found under XDG_STATE_HOME/oca/panes/.
func listPaneSockets() []string {
	stateDir := occupancyStateDir()
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil
	}
	var sockets []string
	for _, e := range entries {
		if e.IsDir() {
			sockets = append(sockets, e.Name())
		}
	}
	return sockets
}
