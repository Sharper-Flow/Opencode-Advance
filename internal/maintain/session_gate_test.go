package maintain

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOpenCodeProcessBlockers_LabelsCallerAncestor(t *testing.T) {
	processes := []processInfo{
		{pid: 10, ppid: 1, name: "opencode", cwd: "/repo/target"},
		{pid: 20, ppid: 10, name: "bash", cwd: "/repo/target"},
		{pid: 30, ppid: 20, name: "oca", cwd: "/repo/target"},
		{pid: 40, ppid: 1, name: "opencode", cwd: "/repo/target/subdir"},
	}

	blockers := buildOpenCodeProcessBlockers(processes, 30, "/repo/target")
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

func TestBuildOpenCodeProcessBlockers_IgnoresOtherProjects(t *testing.T) {
	processes := []processInfo{
		{pid: 10, ppid: 1, name: "opencode", cwd: "/repo/other"},
		{pid: 20, ppid: 10, name: "bash", cwd: "/repo/other"},
		{pid: 30, ppid: 20, name: "oca", cwd: "/repo/other"},
		{pid: 40, ppid: 1, name: "opencode", cwd: "/repo/target"},
	}

	blockers := buildOpenCodeProcessBlockers(processes, 30, "/repo/target")
	if len(blockers) != 1 {
		t.Fatalf("blockers=%#v", blockers)
	}
	if blockers[0].Code != "ACTIVE_OPENCODE_PROCESS" {
		t.Fatalf("blocker code=%s", blockers[0].Code)
	}
}

func TestPlannerBuild_PassesNormalizedProjectRootToGate(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	var got string
	planner := Planner{
		GateCheck: func(_ context.Context, opts Options) GateReport {
			got = opts.ProjectRoot
			return GateReport{Status: GateStatusPass}
		},
	}
	if _, err := planner.Build(context.Background(), Options{}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got != nested {
		t.Fatalf("gate ProjectRoot=%q, want %q", got, nested)
	}
}
