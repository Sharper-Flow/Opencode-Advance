package maintain

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func CheckSessionGate(ctx context.Context, opts Options) GateReport {
	blockers := activeOpenCodeProcessBlockers(ctx, opts.ProjectRoot)
	if len(blockers) > 0 {
		return GateReport{Status: GateStatusBlocked, Blockers: blockers}
	}
	return GateReport{Status: GateStatusPass}
}

func activeOpenCodeProcessBlockers(ctx context.Context, projectRoot string) []Blocker {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	self := os.Getpid()
	projectRoot = normalizeProjectRoot(projectRoot)
	processes := make([]processInfo, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return buildOpenCodeProcessBlockers(processes, self, projectRoot)
		default:
		}
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err != nil {
			continue
		}
		processes = append(processes, processInfo{
			pid:  pid,
			ppid: readProcessParentPID(entry.Name()),
			name: strings.TrimSpace(string(comm)),
			cwd:  readProcessCWD(entry.Name()),
		})
	}
	return buildOpenCodeProcessBlockers(processes, self, projectRoot)
}

type processInfo struct {
	pid  int
	ppid int
	name string
	cwd  string
}

func buildOpenCodeProcessBlockers(processes []processInfo, self int, projectRoot string) []Blocker {
	ancestorPIDs := callerAncestorPIDs(processes, self)
	var blockers []Blocker
	for _, process := range processes {
		if process.pid == self || process.name != "opencode" {
			continue
		}
		if !processInProject(process, projectRoot) {
			continue
		}
		if ancestorPIDs[process.pid] {
			blockers = append(blockers, Blocker{
				Code:    "CALLER_OPENCODE_PROCESS",
				Message: fmt.Sprintf("caller OpenCode process pid %d is still running in project %s", process.pid, projectRoot),
				Hint:    "run `oca maintain --execute` from an external shell after closing OpenCode sessions for this project; OpenCode sessions in other repos do not block",
			})
			continue
		}
		blockers = append(blockers, Blocker{
			Code:    "ACTIVE_OPENCODE_PROCESS",
			Message: fmt.Sprintf("opencode pid %d is still running in project %s", process.pid, projectRoot),
			Hint:    "close OpenCode/OCA sessions for this project before running `oca maintain --execute`",
		})
	}
	return blockers
}

func processInProject(process processInfo, projectRoot string) bool {
	if projectRoot == "" || projectRoot == "." || process.cwd == "" {
		return false
	}
	cwd := normalizeProjectRoot(process.cwd)
	return cwd == projectRoot || strings.HasPrefix(cwd, projectRoot+string(os.PathSeparator))
}

func normalizeProjectRoot(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}
	if abs, err := filepath.Abs(projectRoot); err == nil {
		projectRoot = abs
	}
	if resolved, err := filepath.EvalSymlinks(projectRoot); err == nil {
		projectRoot = resolved
	}
	return filepath.Clean(projectRoot)
}

func callerAncestorPIDs(processes []processInfo, self int) map[int]bool {
	byPID := make(map[int]processInfo, len(processes))
	for _, process := range processes {
		byPID[process.pid] = process
	}
	ancestors := map[int]bool{}
	for pid := self; ; {
		process, ok := byPID[pid]
		if !ok || process.ppid <= 0 || ancestors[process.ppid] {
			break
		}
		ancestors[process.ppid] = true
		pid = process.ppid
	}
	return ancestors
}

func readProcessParentPID(pidDir string) int {
	stat, err := os.ReadFile(filepath.Join("/proc", pidDir, "stat"))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(stat))
	if len(fields) < 4 {
		return 0
	}
	ppid, err := strconv.Atoi(fields[3])
	if err != nil {
		return 0
	}
	return ppid
}

func readProcessCWD(pidDir string) string {
	cwd, err := os.Readlink(filepath.Join("/proc", pidDir, "cwd"))
	if err != nil {
		return ""
	}
	return cwd
}
