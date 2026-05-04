package occupancy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Parse tests ---

func TestParseV1(t *testing.T) {
	raw := `{"sessionID":"ses_abc","directory":"/home/user/project","ts":1710000000000}`
	records, malformed := Parse([]byte(raw))
	if len(malformed) > 0 {
		t.Fatalf("unexpected malformed: %v", malformed)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.SessionID != "ses_abc" {
		t.Errorf("SessionID = %q, want ses_abc", r.SessionID)
	}
	if r.Directory != "/home/user/project" {
		t.Errorf("Directory = %q, want /home/user/project", r.Directory)
	}
	if r.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", r.SchemaVersion)
	}
	if r.ProjectID != "" {
		t.Errorf("ProjectID should be empty for v1, got %q", r.ProjectID)
	}
}

func TestParseV2(t *testing.T) {
	raw := `{
		"schemaVersion": 2,
		"sessionID": "ses_def",
		"directory": "/home/user/project",
		"ts": 1710000000000,
		"startedAt": 1710000000000,
		"lastSeenAt": 1710000010000,
		"socket": "oca",
		"paneID": "%42",
		"sessionName": "oca-project-0",
		"windowID": "@7",
		"windowName": "mychange",
		"agent": "adv",
		"gitRoot": "/home/user/project",
		"gitCommonDir": "/home/user/project/.git",
		"projectId": "abc123",
		"worktreePath": "/home/user/project",
		"worktreeBranch": "trunk"
	}`
	records, malformed := Parse([]byte(raw))
	if len(malformed) > 0 {
		t.Fatalf("unexpected malformed: %v", malformed)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", r.SchemaVersion)
	}
	if r.ProjectID != "abc123" {
		t.Errorf("ProjectID = %q, want abc123", r.ProjectID)
	}
	if r.WorktreePath != "/home/user/project" {
		t.Errorf("WorktreePath = %q", r.WorktreePath)
	}
	if r.WorktreeBranch != "trunk" {
		t.Errorf("WorktreeBranch = %q, want trunk", r.WorktreeBranch)
	}
	if r.Agent != "adv" {
		t.Errorf("Agent = %q, want adv", r.Agent)
	}
	if r.PaneID != "%42" {
		t.Errorf("PaneID = %q, want %%42", r.PaneID)
	}
	if r.StartedAt != 1710000000000 {
		t.Errorf("StartedAt = %d, want 1710000000000", r.StartedAt)
	}
	if r.LastSeenAt != 1710000010000 {
		t.Errorf("LastSeenAt = %d, want 1710000010000", r.LastSeenAt)
	}
}

func TestParseV2Partial(t *testing.T) {
	raw := `{"schemaVersion":2,"sessionID":"ses_ghi","directory":"/tmp","ts":100}`
	records, malformed := Parse([]byte(raw))
	if len(malformed) > 0 {
		t.Fatalf("unexpected malformed: %v", malformed)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", r.SchemaVersion)
	}
	if r.WorktreeBranch != "" {
		t.Errorf("partial v2 should have empty WorktreeBranch, got %q", r.WorktreeBranch)
	}
}

func TestParseMalformed(t *testing.T) {
	raw := `not json at all`
	records, malformed := Parse([]byte(raw))
	if len(records) != 0 {
		t.Errorf("expected 0 records for malformed input, got %d", len(records))
	}
	if len(malformed) != 1 {
		t.Fatalf("expected 1 malformed entry, got %d", len(malformed))
	}
}

func TestParseEmpty(t *testing.T) {
	records, malformed := Parse([]byte(""))
	if len(records) != 0 {
		t.Errorf("expected 0 records for empty input, got %d", len(records))
	}
	if len(malformed) != 0 {
		t.Errorf("expected 0 malformed for empty input, got %d", len(malformed))
	}
}

// --- DiscoverRecords tests ---

func TestDiscoverRecords(t *testing.T) {
	dir := t.TempDir()
	socketDir := filepath.Join(dir, "oca")
	os.MkdirAll(socketDir, 0o755)

	// Write v1
	v1 := `{"sessionID":"s1","directory":"/a","ts":100}`
	os.WriteFile(filepath.Join(socketDir, "42.json"), []byte(v1), 0o644)

	// Write v2
	v2 := `{"schemaVersion":2,"sessionID":"s2","directory":"/b","ts":200,"projectId":"p1","worktreePath":"/b","worktreeBranch":"trunk"}`
	os.WriteFile(filepath.Join(socketDir, "43.json"), []byte(v2), 0o644)

	// Write malformed
	os.WriteFile(filepath.Join(socketDir, "99.json"), []byte("bad"), 0o644)

	records, malformedCount, err := DiscoverRecords(dir)
	if err != nil {
		t.Fatalf("DiscoverRecords error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
	if malformedCount != 1 {
		t.Errorf("expected 1 malformed, got %d", malformedCount)
	}
}

func TestDiscoverRecordsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	records, malformedCount, err := DiscoverRecords(dir)
	if err != nil {
		t.Fatalf("DiscoverRecords error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
	if malformedCount != 0 {
		t.Errorf("expected 0 malformed, got %d", malformedCount)
	}
}

func TestDiscoverRecordsNonexistentDir(t *testing.T) {
	records, malformedCount, err := DiscoverRecords("/nonexistent/path")
	if err != nil {
		t.Fatalf("DiscoverRecords should not error on missing dir, got: %v", err)
	}
	if len(records) != 0 || malformedCount != 0 {
		t.Errorf("expected empty results for missing dir")
	}
}

// --- GroupBy tests ---

func TestGroupByProject(t *testing.T) {
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", ProjectID: "p1", WorktreePath: "/a", WorktreeBranch: "trunk"},
		{SessionID: "s2", Directory: "/a", ProjectID: "p1", WorktreePath: "/a/wt1", WorktreeBranch: "change/foo"},
		{SessionID: "s3", Directory: "/b", ProjectID: "p2", WorktreePath: "/b"},
	}
	groups := GroupBy(records, GroupByProject)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	key1 := GroupKey{ID: "p1"}
	if g, ok := groups[key1]; !ok {
		t.Fatal("missing group p1")
	} else if len(g) != 2 {
		t.Errorf("group p1 should have 2 records, got %d", len(g))
	}
}

func TestGroupByWorktree(t *testing.T) {
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", WorktreePath: "/a"},
		{SessionID: "s2", Directory: "/a", WorktreePath: "/a"},
		{SessionID: "s3", Directory: "/b", WorktreePath: "/b"},
	}
	groups := GroupBy(records, GroupByWorktree)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	keyA := GroupKey{ID: "/a"}
	if g, ok := groups[keyA]; !ok {
		t.Fatal("missing group /a")
	} else if len(g) != 2 {
		t.Errorf("group /a should have 2 records, got %d", len(g))
	}
}

// --- Liveness tests ---

func TestClassifyLiveness(t *testing.T) {
	now := time.Now()
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", PaneID: "%1", LastSeenAt: now.UnixMilli()},
		{SessionID: "s2", Directory: "/b", PaneID: "", LastSeenAt: 0},
	}

	// With live panes map: only %1 is live
	livePanes := map[string]bool{"%1": true}
	results := ClassifyLiveness(records, livePanes, now)
	if results[0].Liveness != Active {
		t.Errorf("record 0 should be Active, got %v", results[0].Liveness)
	}
	if results[1].Liveness != Unknown {
		t.Errorf("record 1 (no paneID, no live check) should be Unknown, got %v", results[1].Liveness)
	}
}

func TestClassifyLivenessStale(t *testing.T) {
	old := time.Now().Add(-5 * time.Minute)
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", PaneID: "%1", LastSeenAt: old.UnixMilli()},
	}
	// Pane not in live set → stale
	livePanes := map[string]bool{}
	results := ClassifyLiveness(records, livePanes, time.Now())
	if results[0].Liveness != Stale {
		t.Errorf("record should be Stale (pane gone), got %v", results[0].Liveness)
	}
}

func TestClassifyLivenessNoTmuxFallback(t *testing.T) {
	now := time.Now()
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", PaneID: "", LastSeenAt: now.UnixMilli()},
	}
	// No live panes, but recent lastSeenAt → unknown (no tmux data)
	livePanes := map[string]bool{}
	results := ClassifyLiveness(records, livePanes, now)
	if results[0].Liveness != Unknown {
		t.Errorf("no-paneID record should be Unknown, got %v", results[0].Liveness)
	}
}

// --- Occupancy warning tests ---

func TestOccupancyWarnings(t *testing.T) {
	classified := []ClassifiedRecord{
		{PaneRecord: PaneRecord{SessionID: "s1", WorktreePath: "/a", PaneID: "%1"}, Liveness: Active},
		{PaneRecord: PaneRecord{SessionID: "s2", WorktreePath: "/a", PaneID: "%2"}, Liveness: Active},
		{PaneRecord: PaneRecord{SessionID: "s3", WorktreePath: "/b", PaneID: "%3"}, Liveness: Stale},
	}
	warnings := OccupancyWarnings(classified)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning (2 active in /a), got %d", len(warnings))
	}
	if warnings[0].WorktreePath != "/a" {
		t.Errorf("warning worktree = %q, want /a", warnings[0].WorktreePath)
	}
	if warnings[0].ActiveCount != 2 {
		t.Errorf("warning count = %d, want 2", warnings[0].ActiveCount)
	}
}

// --- Reconciliation tests ---

func TestReconcilePreferTmux(t *testing.T) {
	// Pane state says /a, tmux says /b → prefer tmux, mark drift
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", PaneID: "%1"},
	}
	livePanes := map[string]bool{"%1": true}
	tmuxCwd := map[string]string{"%1": "/b"}

	reconciled := Reconcile(records, livePanes, tmuxCwd)
	if reconciled[0].Directory != "/b" {
		t.Errorf("reconciled dir = %q, want /b (tmux)", reconciled[0].Directory)
	}
	if !reconciled[0].DirectoryDrift {
		t.Error("expected DirectoryDrift=true when pane state and tmux disagree")
	}
}

func TestReconcileNoDrift(t *testing.T) {
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", PaneID: "%1"},
	}
	livePanes := map[string]bool{"%1": true}
	tmuxCwd := map[string]string{"%1": "/a"}

	reconciled := Reconcile(records, livePanes, tmuxCwd)
	if reconciled[0].DirectoryDrift {
		t.Error("should not mark drift when directories agree")
	}
}

func TestReconcilePaneGone(t *testing.T) {
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", PaneID: "%1"},
	}
	livePanes := map[string]bool{} // pane %1 not live
	tmuxCwd := map[string]string{}

	reconciled := Reconcile(records, livePanes, tmuxCwd)
	if reconciled[0].Liveness != Stale {
		t.Errorf("expected Stale, got %v", reconciled[0].Liveness)
	}
}

func TestReconcileNoTmuxData(t *testing.T) {
	records := []PaneRecord{
		{SessionID: "s1", Directory: "/a", PaneID: "%1"},
	}
	livePanes := map[string]bool{"%1": true}
	tmuxCwd := map[string]string{} // no tmux cwd data

	reconciled := Reconcile(records, livePanes, tmuxCwd)
	if reconciled[0].Directory != "/a" {
		t.Errorf("should keep pane state dir when no tmux data, got %q", reconciled[0].Directory)
	}
}

// --- JSON round-trip test ---

func TestPaneRecordJSONRoundTrip(t *testing.T) {
	original := PaneRecord{
		SchemaVersion:  2,
		SessionID:      "ses_test",
		Directory:      "/home/user/project",
		Ts:             1710000000000,
		StartedAt:      1710000000000,
		LastSeenAt:     1710000010000,
		Socket:         "oca",
		PaneID:         "%42",
		SessionName:    "oca-project-0",
		WindowID:       "@7",
		WindowName:     "mychange",
		Agent:          "adv",
		GitRoot:        "/home/user/project",
		GitCommonDir:   "/home/user/project/.git",
		ProjectID:      "abc123",
		WorktreePath:   "/home/user/project",
		WorktreeBranch: "trunk",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	records, malformed := Parse(data)
	if len(malformed) > 0 {
		t.Fatalf("unexpected malformed on round-trip: %v", malformed)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record on round-trip, got %d", len(records))
	}
	got := records[0]
	if got != original {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", got, original)
	}
}
