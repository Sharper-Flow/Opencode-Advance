package maintain

import (
	"strings"
	"testing"
)

func TestBuildOpenCodeProcessBlockers_LabelsCallerAncestor(t *testing.T) {
	processes := []processInfo{
		{pid: 10, ppid: 1, name: "opencode"},
		{pid: 20, ppid: 10, name: "bash"},
		{pid: 30, ppid: 20, name: "oca"},
		{pid: 40, ppid: 1, name: "opencode"},
	}

	blockers := buildOpenCodeProcessBlockers(processes, 30)
	if len(blockers) != 2 {
		t.Fatalf("blockers=%#v", blockers)
	}
	if blockers[0].Code != "CALLER_OPENCODE_PROCESS" {
		t.Fatalf("caller blocker code=%s", blockers[0].Code)
	}
	if !strings.Contains(blockers[0].Hint, "external shell") {
		t.Fatalf("caller hint=%q missing external shell guidance", blockers[0].Hint)
	}
	if blockers[1].Code != "ACTIVE_OPENCODE_PROCESS" {
		t.Fatalf("external blocker code=%s", blockers[1].Code)
	}
}
