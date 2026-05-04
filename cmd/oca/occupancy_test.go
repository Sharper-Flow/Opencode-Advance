package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/occupancy"
)

// --- occupancyJSONOutput round-trip ---

func TestOccupancyJSONRoundTrip(t *testing.T) {
	original := occupancyJSONOutput{
		Groups: []occupancyGroupJSON{
			{
				ProjectID:    "abc",
				WorktreePath: "/a",
				Branch:       "trunk",
				Occupants: []occupantJSON{
					{SessionID: "s1", PaneID: "%1", Agent: "adv", LastSeenAgo: "5s", Liveness: "active"},
				},
			},
		},
		Warnings: []occupancyWarnJSON{
			{WorktreePath: "/a", ActiveCount: 2},
		},
		Malformed: 1,
		Stale:     2,
	}
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got occupancyJSONOutput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got.Groups) != 1 {
		t.Errorf("groups = %d, want 1", len(got.Groups))
	}
	if got.Groups[0].ProjectID != "abc" {
		t.Errorf("projectId = %q, want abc", got.Groups[0].ProjectID)
	}
	if len(got.Warnings) != 1 {
		t.Errorf("warnings = %d, want 1", len(got.Warnings))
	}
	if got.Malformed != 1 {
		t.Errorf("malformed = %d, want 1", got.Malformed)
	}
	if got.Stale != 2 {
		t.Errorf("stale = %d, want 2", got.Stale)
	}
}

// --- formatDuration ---

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    int64 // seconds
		want string
	}{
		{0, "0s"},
		{5, "5s"},
		{59, "59s"},
		{60, "1m"},
		{300, "5m"},
		{3600, "1h"},
		{7200, "2h"},
	}
	for _, tt := range tests {
		got := formatDurationFromSeconds(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%ds) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// --- Status compact output tests ---

func TestOccupancyStatusCompactSingle(t *testing.T) {
	// Build temp pane state with single occupant
	dir := t.TempDir()
	socketDir := filepath.Join(dir, "oca")
	os.MkdirAll(socketDir, 0o755)

	v2 := `{"schemaVersion":2,"sessionID":"s1","directory":"/a","ts":100,"worktreePath":"/a","worktreeBranch":"trunk","paneID":"%42","lastSeenAt":1710000010000}`
	os.WriteFile(filepath.Join(socketDir, "42.json"), []byte(v2), 0o644)

	// Test status logic directly
	output := computeStatusOutput(dir, "oca", "%42")
	if !strings.Contains(output, "trunk") {
		t.Errorf("status should contain 'trunk', got: %q", output)
	}
	if !strings.Contains(output, "1×") {
		t.Errorf("status should contain '1×' for single occupant, got: %q", output)
	}
}

func TestOccupancyStatusCompactOccupied(t *testing.T) {
	dir := t.TempDir()
	socketDir := filepath.Join(dir, "oca")
	os.MkdirAll(socketDir, 0o755)

	v2a := `{"schemaVersion":2,"sessionID":"s1","directory":"/a","ts":100,"worktreePath":"/a","worktreeBranch":"trunk","paneID":"%42","lastSeenAt":1710000010000}`
	v2b := `{"schemaVersion":2,"sessionID":"s2","directory":"/a","ts":200,"worktreePath":"/a","worktreeBranch":"trunk","paneID":"%43","lastSeenAt":1710000020000}`
	os.WriteFile(filepath.Join(socketDir, "42.json"), []byte(v2a), 0o644)
	os.WriteFile(filepath.Join(socketDir, "43.json"), []byte(v2b), 0o644)

	output := computeStatusOutput(dir, "oca", "%42")
	if !strings.Contains(output, "2× occupied") {
		t.Errorf("status should contain '2× occupied' for 2 occupants, got: %q", output)
	}
}

func TestOccupancyStatusCompactMissing(t *testing.T) {
	dir := t.TempDir()
	output := computeStatusOutput(dir, "oca", "%99")
	if output != "?" {
		t.Errorf("missing pane should output '?', got: %q", output)
	}
}

func TestOccupancyStatusCompactNoBranch(t *testing.T) {
	dir := t.TempDir()
	socketDir := filepath.Join(dir, "oca")
	os.MkdirAll(socketDir, 0o755)

	v2 := `{"schemaVersion":2,"sessionID":"s1","directory":"/a","ts":100,"worktreePath":"/a","paneID":"%1","lastSeenAt":1710000010000}`
	os.WriteFile(filepath.Join(socketDir, "1.json"), []byte(v2), 0o644)

	output := computeStatusOutput(dir, "oca", "%1")
	if !strings.Contains(output, "?") {
		t.Errorf("no branch should show '?', got: %q", output)
	}
}

// --- Human output tests ---

func TestPrintOccupancyHumanBasic(t *testing.T) {
	var buf strings.Builder
	records := []occupancy.ClassifiedRecord{
		{
			PaneRecord: occupancy.PaneRecord{
				SessionID:      "s1",
				Directory:      "/home/user/proj",
				ProjectID:      "abc12345",
				WorktreePath:   "/home/user/proj",
				WorktreeBranch: "trunk",
				PaneID:         "%1",
				Agent:          "adv",
				LastSeenAt:     time.Now().Add(-5 * time.Second).UnixMilli(),
			},
			Liveness: occupancy.Active,
		},
	}
	warnings := []occupancy.OccupancyWarning{}
	err := printOccupancyHuman(&buf, records, warnings, 0, 0)
	if err != nil {
		t.Fatalf("printOccupancyHuman error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Project:") {
		t.Error("human output should contain 'Project:'")
	}
	if !strings.Contains(output, "trunk") {
		t.Error("human output should contain branch 'trunk'")
	}
	if !strings.Contains(output, "✓") {
		t.Error("active record should show ✓ icon")
	}
}

func TestPrintOccupancyHumanWithWarning(t *testing.T) {
	var buf strings.Builder
	records := []occupancy.ClassifiedRecord{
		{
			PaneRecord: occupancy.PaneRecord{SessionID: "s1", WorktreePath: "/a", PaneID: "%1", Agent: "adv"},
			Liveness:   occupancy.Active,
		},
		{
			PaneRecord: occupancy.PaneRecord{SessionID: "s2", WorktreePath: "/a", PaneID: "%2", Agent: "build"},
			Liveness:   occupancy.Active,
		},
	}
	warnings := occupancy.OccupancyWarnings(records)
	err := printOccupancyHuman(&buf, records, warnings, 0, 0)
	if err != nil {
		t.Fatalf("printOccupancyHuman error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "warning:") {
		t.Error("human output should contain warning")
	}
	if !strings.Contains(output, "2 active sessions share") {
		t.Error("warning should mention 2 active sessions")
	}
}

func TestPrintOccupancyHumanMalformedStaleFooter(t *testing.T) {
	var buf strings.Builder
	err := printOccupancyHuman(&buf, nil, nil, 1, 2)
	if err != nil {
		t.Fatalf("printOccupancyHuman error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "1 malformed") {
		t.Error("footer should mention malformed count")
	}
	if !strings.Contains(output, "2 stale") {
		t.Error("footer should mention stale count")
	}
	if !strings.Contains(output, "--all") {
		t.Error("footer should mention --all flag")
	}
}

func TestPrintOccupancyJSONEmpty(t *testing.T) {
	var buf strings.Builder
	err := printOccupancyJSON(&buf, nil, nil, 0, 0)
	if err != nil {
		t.Fatalf("printOccupancyJSON error: %v", err)
	}
	var got occupancyJSONOutput
	if err := json.Unmarshal([]byte(buf.String()), &got); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if len(got.Groups) != 0 {
		t.Errorf("empty should have 0 groups, got %d", len(got.Groups))
	}
}

// --- listPaneSockets ---

func TestListPaneSockets(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "oca"), 0o755)
	os.MkdirAll(filepath.Join(dir, "default"), 0o755)
	os.WriteFile(filepath.Join(dir, "notadir.json"), []byte("{}"), 0o644)

	sockets := listPaneSocketsInDir(dir)
	if len(sockets) != 2 {
		t.Errorf("expected 2 sockets, got %d: %v", len(sockets), sockets)
	}
	found := map[string]bool{}
	for _, s := range sockets {
		found[s] = true
	}
	if !found["oca"] || !found["default"] {
		t.Errorf("expected oca+default sockets, got: %v", sockets)
	}
}

// --- Helper functions for testing ---

func formatDurationFromSeconds(secs int64) string {
	return formatDuration(time.Duration(secs) * time.Second)
}

// computeStatusOutput runs the status computation logic on a test state dir.
func computeStatusOutput(stateDir, socket, paneID string) string {
	records, _, err := occupancy.DiscoverRecords(stateDir)
	if err != nil {
		return "?"
	}

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
				return "1× " + branch
			}
			return "2× occupied ⚠"
		}
	}
	return "?"
}

func listPaneSocketsInDir(dir string) []string {
	entries, err := os.ReadDir(dir)
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
