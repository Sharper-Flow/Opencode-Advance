package maintain

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func CheckSessionGate(ctx context.Context, _ Options) GateReport {
	blockers := activeOpenCodeProcessBlockers(ctx)
	if len(blockers) > 0 {
		return GateReport{Status: GateStatusBlocked, Blockers: blockers}
	}
	return GateReport{Status: GateStatusPass}
}

func activeOpenCodeProcessBlockers(ctx context.Context) []Blocker {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	self := os.Getpid()
	processes := make([]processInfo, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return buildOpenCodeProcessBlockers(processes, self)
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
		})
	}
	return buildOpenCodeProcessBlockers(processes, self)
}

type processInfo struct {
	pid  int
	ppid int
	name string
}

func buildOpenCodeProcessBlockers(processes []processInfo, self int) []Blocker {
	ancestorPIDs := callerAncestorPIDs(processes, self)
	var blockers []Blocker
	for _, process := range processes {
		if process.pid == self || process.name != "opencode" {
			continue
		}
		if ancestorPIDs[process.pid] {
			blockers = append(blockers, Blocker{
				Code:    "CALLER_OPENCODE_PROCESS",
				Message: fmt.Sprintf("caller OpenCode process pid %d is still running", process.pid),
				Hint:    "run `oca maintain --execute` from an external shell after closing OpenCode sessions; commands launched inside OpenCode self-block by design",
			})
			continue
		}
		blockers = append(blockers, Blocker{
			Code:    "ACTIVE_OPENCODE_PROCESS",
			Message: fmt.Sprintf("opencode pid %d is still running", process.pid),
			Hint:    "close OpenCode/OCA sessions before running `oca maintain --execute`",
		})
	}
	return blockers
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
