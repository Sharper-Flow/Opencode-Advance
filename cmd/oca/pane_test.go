package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTmuxSocket(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/tmp/tmux-1000/oca,12345", "oca"},
		{"/tmp/tmux-1000/default,12345", "default"},
		{"", "oca"},
		{",12345", "oca"},
	}
	for _, tt := range tests {
		got := parseTmuxSocket(tt.input)
		if got != tt.expected {
			t.Errorf("parseTmuxSocket(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSanitizePaneID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"%42", "42"},
		{"%0", "0"},
		{"42", "42"},
		{"0", "0"},
	}
	for _, tt := range tests {
		got := sanitizePaneID(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizePaneID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestXdgStateHome(t *testing.T) {
	// With XDG_STATE_HOME set
	t.Run("env set", func(t *testing.T) {
		original := os.Getenv("XDG_STATE_HOME")
		os.Setenv("XDG_STATE_HOME", "/custom/state")
		defer os.Setenv("XDG_STATE_HOME", original)

		got := xdgStateHome()
		if got != "/custom/state" {
			t.Errorf("xdgStateHome() = %q, want %q", got, "/custom/state")
		}
	})

	// With XDG_STATE_HOME unset
	t.Run("fallback", func(t *testing.T) {
		original := os.Getenv("XDG_STATE_HOME")
		os.Unsetenv("XDG_STATE_HOME")
		defer os.Setenv("XDG_STATE_HOME", original)

		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".local", "state")
		got := xdgStateHome()
		if got != want {
			t.Errorf("xdgStateHome() = %q, want %q", got, want)
		}
	})
}

func TestReadPaneState(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		tmp := t.TempDir()
		file := filepath.Join(tmp, "state.json")
		os.WriteFile(file, []byte(`{"sessionID":"ses_abc","directory":"/tmp","ts":12345}`), 0o644)

		ps, err := readPaneState(file)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ps.SessionID != "ses_abc" || ps.Directory != "/tmp" || ps.Ts != 12345 {
			t.Errorf("unexpected state: %+v", ps)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readPaneState("/nonexistent/path/state.json")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		tmp := t.TempDir()
		file := filepath.Join(tmp, "bad.json")
		os.WriteFile(file, []byte("not json"), 0o644)

		_, err := readPaneState(file)
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})
}

// ── V2 pane state reader tests ─────────────────────────────────────────────

func TestReadPaneStateV2(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "v2.json")
	v2JSON := `{
		"schemaVersion": 2,
		"sessionID": "ses_v2",
		"directory": "/home/user/project",
		"ts": 1700000000,
		"startedAt": 1699999999,
		"lastSeenAt": 1700000000,
		"paneID": "%42",
		"socket": "oca",
		"agent": "adv",
		"gitRoot": "/home/user/project",
		"worktreePath": "/home/user/.local/share/opencode/worktree/abc123/change/mychange",
		"gitCommonDir": "/home/user/project/.git",
		"defaultBranch": "trunk",
		"mainCheckoutPath": "/home/user/project",
		"isMainCheckout": false,
		"branchSafety": "worktree",
		"projectId": "abc123def",
		"worktreeBranch": "change/mychange",
		"changeID": "mychange",
		"role": "adv"
	}`
	os.WriteFile(file, []byte(v2JSON), 0o644)

	ps, err := readPaneState(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// v1 fields still readable
	if ps.SessionID != "ses_v2" {
		t.Errorf("SessionID = %q, want %q", ps.SessionID, "ses_v2")
	}
	if ps.Directory != "/home/user/project" {
		t.Errorf("Directory = %q, want %q", ps.Directory, "/home/user/project")
	}
	if ps.Ts != 1700000000 {
		t.Errorf("Ts = %d, want %d", ps.Ts, 1700000000)
	}

	// v2 fields
	if ps.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", ps.SchemaVersion)
	}
	if ps.ChangeID != "mychange" {
		t.Errorf("ChangeID = %q, want %q", ps.ChangeID, "mychange")
	}
	if ps.Role != "adv" {
		t.Errorf("Role = %q, want %q", ps.Role, "adv")
	}
	if ps.ProjectID != "abc123def" {
		t.Errorf("ProjectID = %q, want %q", ps.ProjectID, "abc123def")
	}
	if ps.WorktreeBranch != "change/mychange" {
		t.Errorf("WorktreeBranch = %q, want %q", ps.WorktreeBranch, "change/mychange")
	}
	if ps.WorktreePath != "/home/user/.local/share/opencode/worktree/abc123/change/mychange" {
		t.Errorf("WorktreePath = %q, want worktree path", ps.WorktreePath)
	}
	if ps.DefaultBranch != "trunk" {
		t.Errorf("DefaultBranch = %q, want trunk", ps.DefaultBranch)
	}
	if ps.MainCheckoutPath != "/home/user/project" {
		t.Errorf("MainCheckoutPath = %q, want main checkout", ps.MainCheckoutPath)
	}
	if ps.IsMainCheckout {
		t.Error("IsMainCheckout = true, want false for worktree state")
	}
	if ps.BranchSafety != "worktree" {
		t.Errorf("BranchSafety = %q, want worktree", ps.BranchSafety)
	}
	if ps.Agent != "adv" {
		t.Errorf("Agent = %q, want %q", ps.Agent, "adv")
	}
}

func TestReadPaneStateV1StillWorksAfterV2(t *testing.T) {
	// Verify v1 format still reads — backward compat.
	tmp := t.TempDir()
	file := filepath.Join(tmp, "v1.json")
	os.WriteFile(file, []byte(`{"sessionID":"ses_abc","directory":"/tmp","ts":12345}`), 0o644)

	ps, err := readPaneState(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ps.SessionID != "ses_abc" || ps.Directory != "/tmp" || ps.Ts != 12345 {
		t.Errorf("unexpected v1 state: %+v", ps)
	}
	// v2 fields should be zero values
	if ps.SchemaVersion != 0 {
		t.Errorf("SchemaVersion = %d, want 0 for v1", ps.SchemaVersion)
	}
	if ps.ChangeID != "" {
		t.Errorf("ChangeID = %q, want empty for v1", ps.ChangeID)
	}
}

// ── Derived context tests ──────────────────────────────────────────────────

func TestDerivePaneContext(t *testing.T) {
	t.Run("extracts changeID from branch", func(t *testing.T) {
		ps := &paneState{
			SessionID:      "ses_1",
			Directory:      "/home/user/project",
			Ts:             1700000000,
			WorktreeBranch: "change/patternBSessionTopologyOne",
			Agent:          "adv",
		}
		ctx := derivePaneContext(ps)
		if ctx.ChangeID != "patternBSessionTopologyOne" {
			t.Errorf("ChangeID = %q, want %q", ctx.ChangeID, "patternBSessionTopologyOne")
		}
	})

	t.Run("uses explicit changeID over branch", func(t *testing.T) {
		ps := &paneState{
			ChangeID:       "explicit-id",
			WorktreeBranch: "change/other",
		}
		ctx := derivePaneContext(ps)
		if ctx.ChangeID != "explicit-id" {
			t.Errorf("ChangeID = %q, want %q", ctx.ChangeID, "explicit-id")
		}
	})

	t.Run("no changeID when not on change branch", func(t *testing.T) {
		ps := &paneState{
			WorktreeBranch: "trunk",
		}
		ctx := derivePaneContext(ps)
		if ctx.ChangeID != "" {
			t.Errorf("ChangeID = %q, want empty for non-change branch", ctx.ChangeID)
		}
	})

	t.Run("role defaults to agent", func(t *testing.T) {
		ps := &paneState{Agent: "build"}
		ctx := derivePaneContext(ps)
		if ctx.Role != "build" {
			t.Errorf("Role = %q, want %q", ctx.Role, "build")
		}
	})

	t.Run("explicit role overrides agent", func(t *testing.T) {
		ps := &paneState{Agent: "build", Role: "primary"}
		ctx := derivePaneContext(ps)
		if ctx.Role != "primary" {
			t.Errorf("Role = %q, want %q", ctx.Role, "primary")
		}
	})

	t.Run("detects worktree", func(t *testing.T) {
		ps := &paneState{
			GitRoot:      "/home/user/project",
			WorktreePath: "/home/user/.local/share/opencode/worktree/abc/change/xyz",
		}
		ctx := derivePaneContext(ps)
		if !ctx.IsWorktree {
			t.Error("IsWorktree = false, want true")
		}
	})

	t.Run("no worktree when paths match", func(t *testing.T) {
		ps := &paneState{
			GitRoot:      "/home/user/project",
			WorktreePath: "/home/user/project",
		}
		ctx := derivePaneContext(ps)
		if ctx.IsWorktree {
			t.Error("IsWorktree = true, want false")
		}
	})

	t.Run("no worktree when worktreePath empty", func(t *testing.T) {
		ps := &paneState{
			GitRoot: "/home/user/project",
		}
		ctx := derivePaneContext(ps)
		if ctx.IsWorktree {
			t.Error("IsWorktree = true, want false when worktreePath empty")
		}
	})

	t.Run("detects worktree via main checkout path when gitRoot is worktree root", func(t *testing.T) {
		ps := &paneState{
			GitRoot:          "/home/user/.local/share/opencode/worktree/abc/change/xyz",
			WorktreePath:     "/home/user/.local/share/opencode/worktree/abc/change/xyz",
			GitCommonDir:     "/home/user/project/.git",
			MainCheckoutPath: "/home/user/project",
			WorktreeBranch:   "change/xyz",
			DefaultBranch:    "trunk",
		}
		ctx := derivePaneContext(ps)
		if !ctx.IsWorktree {
			t.Error("IsWorktree = false, want true for linked worktree")
		}
		if ctx.IsMainCheckout {
			t.Error("IsMainCheckout = true, want false for linked worktree")
		}
		if ctx.ProjectRoot != "/home/user/project" {
			t.Errorf("ProjectRoot = %q, want main checkout path", ctx.ProjectRoot)
		}
		if ctx.BranchSafety != "worktree" {
			t.Errorf("BranchSafety = %q, want worktree", ctx.BranchSafety)
		}
	})

	t.Run("marks non-default main checkout branch unsafe", func(t *testing.T) {
		ps := &paneState{
			GitRoot:          "/home/user/project",
			WorktreePath:     "/home/user/project",
			MainCheckoutPath: "/home/user/project",
			WorktreeBranch:   "feature/unsafe",
			DefaultBranch:    "trunk",
			IsMainCheckout:   true,
		}
		ctx := derivePaneContext(ps)
		if !ctx.IsMainCheckout {
			t.Error("IsMainCheckout = false, want true")
		}
		if ctx.IsWorktree {
			t.Error("IsWorktree = true, want false for main checkout")
		}
		if ctx.BranchSafety != "unsafe_main_branch" {
			t.Errorf("BranchSafety = %q, want unsafe_main_branch", ctx.BranchSafety)
		}
	})
}

func TestPaneStateFilePath(t *testing.T) {
	originalTmux := os.Getenv("TMUX")
	originalXdg := os.Getenv("XDG_STATE_HOME")
	defer func() {
		os.Setenv("TMUX", originalTmux)
		os.Setenv("XDG_STATE_HOME", originalXdg)
	}()

	os.Setenv("XDG_STATE_HOME", "/test/state")
	os.Setenv("TMUX", "/tmp/tmux-1000/oca,12345")

	got := paneStateFilePath("%42")
	want := filepath.Join("/test/state", "oca", "panes", "oca", "42.json")
	if got != want {
		t.Errorf("paneStateFilePath(%%42) = %q, want %q", got, want)
	}
}
