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
	var blockers []Blocker
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return blockers
		default:
		}
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == self {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(comm))
		if name != "opencode" {
			continue
		}
		blockers = append(blockers, Blocker{
			Code:    "ACTIVE_OPENCODE_PROCESS",
			Message: fmt.Sprintf("opencode pid %d is still running", pid),
			Hint:    "close OpenCode/OCA sessions before running `oca maintain --execute`",
		})
	}
	return blockers
}
