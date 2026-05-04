package advruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectWorkspaceStates_ReadsRegistrySnapshot(t *testing.T) {
	advRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", advRoot)
	projectID := "abc123"
	stateDir := filepath.Join(advRoot, "opencode", "plugins", "advance", projectID)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	snapshot := map[string]interface{}{
		"worktree_registry": map[string]interface{}{
			"change/foo": map[string]interface{}{
				"branch":             "change/foo",
				"path":               "/tmp/worktrees/change/foo",
				"materialized":       true,
				"changeId":           "foo",
				"status":             "active",
				"setupReady":         true,
				"baseRef":            "main",
				"headSha":            "deadbeef",
				"source":             "tool",
			},
			"change/bar": map[string]interface{}{
				"branch":             "change/bar",
				"materialized":       false,
				"changeId":           "bar",
				"status":             "idle",
				"baseRef":            "main",
				"source":             "tool",
			},
			"change/boom": map[string]interface{}{
				"branch":              "change/boom",
				"path":                "/tmp/worktrees/change/boom",
				"materialized":        true,
				"changeId":            "boom",
				"status":              "setup_failed",
				"setupReady":          false,
				"setupFailureReason":  "hook exited 1",
				"baseRef":             "main",
				"headSha":             "cafe",
				"source":              "tool",
			},
		},
	}
	data, _ := json.Marshal(snapshot)
	if err := os.WriteFile(filepath.Join(stateDir, "snapshot.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	proj, err := ProjectWorkspaceStates(projectID)
	if err != nil {
		t.Fatalf("ProjectWorkspaceStates: %v", err)
	}
	if proj.ProjectID != projectID {
		t.Errorf("ProjectID = %q, want %q", proj.ProjectID, projectID)
	}
	if len(proj.Worktrees) != 3 {
		t.Fatalf("len(Worktrees) = %d, want 3", len(proj.Worktrees))
	}

	byBranch := map[string]*WorktreeWorkspaceState{}
	for i := range proj.Worktrees {
		byBranch[proj.Worktrees[i].Branch] = &proj.Worktrees[i]
	}

	// Verify active worktree
	if wt, ok := byBranch["change/foo"]; !ok {
		t.Error("missing change/foo")
	} else {
		if wt.Status != "active" {
			t.Errorf("foo status = %q, want active", wt.Status)
		}
		if !wt.Materialized {
			t.Error("foo materialized = false, want true")
		}
		if !wt.SetupReady {
			t.Error("foo setupReady = false, want true")
		}
	}

	// Verify unmaterialized idle worktree
	if wt, ok := byBranch["change/bar"]; !ok {
		t.Error("missing change/bar")
	} else {
		if wt.Status != "idle" {
			t.Errorf("bar status = %q, want idle", wt.Status)
		}
		if wt.Materialized {
			t.Error("bar materialized = true, want false")
		}
	}

	// Verify setup_failed worktree
	if wt, ok := byBranch["change/boom"]; !ok {
		t.Error("missing change/boom")
	} else {
		if wt.Status != "setup_failed" {
			t.Errorf("boom status = %q, want setup_failed", wt.Status)
		}
		if wt.SetupReady {
			t.Error("boom setupReady = true, want false")
		}
		if wt.SetupFailureReason != "hook exited 1" {
			t.Errorf("boom failure = %q, want 'hook exited 1'", wt.SetupFailureReason)
		}
	}
}

func TestProjectWorkspaceStates_GracefulWhenNoSnapshot(t *testing.T) {
	advRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", advRoot)

	proj, err := ProjectWorkspaceStates("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj == nil {
		t.Fatal("proj = nil, want non-nil empty projection")
	}
	if len(proj.Worktrees) != 0 {
		t.Errorf("len(Worktrees) = %d, want 0", len(proj.Worktrees))
	}
}

func TestProjectWorkspaceStates_EmptyProjectID(t *testing.T) {
	proj, err := ProjectWorkspaceStates("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.ProjectID != "" {
		t.Errorf("ProjectID = %q, want empty", proj.ProjectID)
	}
	if len(proj.Worktrees) != 0 {
		t.Errorf("len(Worktrees) = %d, want 0", len(proj.Worktrees))
	}
}

func TestEnrichSessionsWithWorkspaceState(t *testing.T) {
	projection := &WorkspaceProjection{
		ProjectID: "test",
		Worktrees: []WorktreeWorkspaceState{
			{Branch: "change/foo", Path: "/wt/foo", Materialized: true, ChangeID: "foo", Status: "active", SetupReady: true},
			{Branch: "change/boom", Path: "/wt/boom", Materialized: true, ChangeID: "boom", Status: "setup_failed", SetupReady: false, SetupFailureReason: "hook error"},
		},
	}

	sessions := []SessionView{
		{Name: "foo", Attached: true, Path: "/wt/foo"},
		{Name: "boom", Attached: false, Path: "/wt/boom"},
		{Name: "unknown", Attached: false, Path: "/tmp/other"},
	}

	enriched := EnrichSessionsWithWorkspaceState(sessions, projection)

	if len(enriched) != 3 {
		t.Fatalf("len(enriched) = %d, want 3", len(enriched))
	}

	// Session "foo" should match workspace "change/foo" by name
	if enriched[0].WorkspaceStatus != "active" {
		t.Errorf("foo workspaceStatus = %q, want active", enriched[0].WorkspaceStatus)
	}
	if enriched[0].WorkspaceChangeID != "foo" {
		t.Errorf("foo changeID = %q, want foo", enriched[0].WorkspaceChangeID)
	}
	if !enriched[0].WorkspaceSetupReady {
		t.Error("foo setupReady = false, want true")
	}

	// Session "boom" should match workspace "change/boom" — setup_failed
	if enriched[1].WorkspaceStatus != "setup_failed" {
		t.Errorf("boom workspaceStatus = %q, want setup_failed", enriched[1].WorkspaceStatus)
	}
	if enriched[1].WorkspaceFailure != "hook error" {
		t.Errorf("boom failure = %q, want 'hook error'", enriched[1].WorkspaceFailure)
	}

	// Session "unknown" should have empty workspace fields
	if enriched[2].WorkspaceStatus != "" {
		t.Errorf("unknown workspaceStatus = %q, want empty", enriched[2].WorkspaceStatus)
	}
}

func TestEnrichSessionsWithWorkspaceState_NilProjection(t *testing.T) {
	sessions := []SessionView{
		{Name: "test", Attached: true, Path: "/tmp/test"},
	}

	enriched := EnrichSessionsWithWorkspaceState(sessions, nil)
	if len(enriched) != 1 {
		t.Fatalf("len = %d, want 1", len(enriched))
	}
	if enriched[0].WorkspaceStatus != "" {
		t.Errorf("workspaceStatus = %q, want empty on nil projection", enriched[0].WorkspaceStatus)
	}
}

func TestMatchSessionToWorkspace(t *testing.T) {
	cases := []struct {
		name        string
		sessionName string
		sessionPath string
		wt          WorktreeWorkspaceState
		want        bool
	}{
		{"path match", "anything", "/wt/foo", WorktreeWorkspaceState{Branch: "change/foo", Path: "/wt/foo"}, true},
		{"name match change id", "foo", "", WorktreeWorkspaceState{Branch: "change/foo", ChangeID: "foo"}, true},
		{"name match branch short", "bar", "", WorktreeWorkspaceState{Branch: "change/bar"}, true},
		{"no match", "other", "/other", WorktreeWorkspaceState{Branch: "change/x", Path: "/wt/x"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchSessionToWorkspace(tc.sessionName, tc.sessionPath, tc.wt)
			if got != tc.want {
				t.Errorf("matchSessionToWorkspace(%q, %q, {%q, %q}) = %v, want %v",
					tc.sessionName, tc.sessionPath, tc.wt.Branch, tc.wt.Path, got, tc.want)
			}
		})
	}
}
