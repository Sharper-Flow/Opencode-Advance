package advstatus

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── Helpers ────────────────────────────────────────────────

func setupFixtureState(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	// Project A with an active change
	projectA := filepath.Join(tmp, "project-a")
	changesA := filepath.Join(projectA, "changes", "testChange01")
	if err := os.MkdirAll(changesA, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(changesA, "change.json"), map[string]any{
		"id":     "testChange01",
		"title":  "Test change for status bar",
		"status": "draft",
		"gates": map[string]any{
			"proposal":   map[string]string{"status": "done"},
			"discovery":  map[string]string{"status": "done"},
			"design":     map[string]string{"status": "pending"},
			"planning":   map[string]string{"status": "pending"},
			"execution":  map[string]string{"status": "pending"},
			"acceptance": map[string]string{"status": "pending"},
			"release":    map[string]string{"status": "pending"},
		},
		"tasks": []any{
			map[string]string{"id": "tk-001", "status": "done"},
			map[string]string{"id": "tk-002", "status": "pending"},
		},
	})

	// Project A snapshot with worktree registry
	writeJSON(t, filepath.Join(projectA, "snapshot.json"), map[string]any{
		"worktree_registry": map[string]any{
			"change/testChange01": map[string]any{
				"branch":       "change/testChange01",
				"path":         "/tmp/worktree-test",
				"materialized": true,
				"changeId":     "testChange01",
				"status":       "active",
				"setupReady":   true,
			},
		},
	})

	// Project B with an archived change (should be excluded)
	projectB := filepath.Join(tmp, "project-b")
	changesB := filepath.Join(projectB, "changes", "archivedChange")
	if err := os.MkdirAll(changesB, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(changesB, "change.json"), map[string]any{
		"id":     "archivedChange",
		"title":  "Archived",
		"status": "archived",
		"gates":  map[string]any{},
		"tasks":  []any{},
	})

	// Project C with setup_failed worktree
	projectC := filepath.Join(tmp, "project-c")
	if err := os.MkdirAll(filepath.Join(projectC, "changes"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(projectC, "snapshot.json"), map[string]any{
		"worktree_registry": map[string]any{
			"change/brokenWT": map[string]any{
				"branch":             "change/brokenWT",
				"changeId":           "brokenWT",
				"status":             "setup_failed",
				"setupReady":         false,
				"setupFailureReason": "hook timeout",
			},
			"change/activeWT": map[string]any{
				"branch":     "change/activeWT",
				"changeId":   "activeWT",
				"status":     "active",
				"setupReady": true,
			},
		},
	})

	return tmp
}

func setupTemporalEnv(t *testing.T, addr string) string {
	t.Helper()
	cacheDir := t.TempDir()
	if addr != "" {
		err := os.WriteFile(filepath.Join(cacheDir, "temporal.env"), []byte("ADV_TEMPORAL_ADDRESS="+addr+"\n"), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}
	return cacheDir
}

func writeJSON(t *testing.T, path string, data map[string]any) {
	t.Helper()
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── FindActiveChanges Tests ────────────────────────────────

func TestFindActiveChanges_FindsActiveExcludesArchived(t *testing.T) {
	advRoot := setupFixtureState(t)
	changesDir := filepath.Join(advRoot, "project-a", "changes")

	result, err := FindActiveChanges(changesDir)
	if err != nil {
		t.Fatalf("FindActiveChanges returned error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 active change, got %d: %v", len(result), result)
	}
	if result[0] != "testChange01" {
		t.Fatalf("expected testChange01, got %s", result[0])
	}
}

func TestFindActiveChanges_ExcludesArchivedAndClosed(t *testing.T) {
	advRoot := setupFixtureState(t)
	changesDir := filepath.Join(advRoot, "project-b", "changes")

	result, err := FindActiveChanges(changesDir)
	if err != nil {
		t.Fatalf("FindActiveChanges returned error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 active changes from archived project, got %d", len(result))
	}
}

func TestFindActiveChanges_NonexistentDir(t *testing.T) {
	result, err := FindActiveChanges("/nonexistent/path")
	if err != nil {
		t.Fatalf("FindActiveChanges should not error on missing dir: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 results for missing dir, got %d", len(result))
	}
}

func TestFindActiveChanges_SortsNewestFirst(t *testing.T) {
	changesDir := t.TempDir()
	oldDir := filepath.Join(changesDir, "oldChange")
	newDir := filepath.Join(changesDir, "newChange")
	for _, dir := range []string{oldDir, newDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeJSON(t, filepath.Join(dir, "change.json"), map[string]any{
			"id":     filepath.Base(dir),
			"status": "draft",
			"gates":  map[string]any{},
		})
	}

	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newDir, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	result, err := FindActiveChanges(changesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0] != "newChange" || result[1] != "oldChange" {
		t.Fatalf("expected newest-first order [newChange oldChange], got %v", result)
	}
}

func TestFindActiveChanges_ScansAllProjects(t *testing.T) {
	advRoot := setupFixtureState(t)

	result, err := FindActiveChanges(filepath.Join(advRoot, "project-a", "changes"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

// ── SummarizeChange Tests ─────────────────────────────────

func TestSummarizeChange_ParsesGates(t *testing.T) {
	advRoot := setupFixtureState(t)
	changeDir := filepath.Join(advRoot, "project-a", "changes", "testChange01")

	summary, err := SummarizeChange(changeDir)
	if err != nil {
		t.Fatalf("SummarizeChange error: %v", err)
	}
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if summary.CurrentGate != "design" {
		t.Fatalf("expected current gate 'design', got '%s'", summary.CurrentGate)
	}
	if summary.Status != "draft" {
		t.Fatalf("expected status 'draft', got '%s'", summary.Status)
	}
}

func TestSummarizeChange_AllGatesDone(t *testing.T) {
	tmp := t.TempDir()
	changeDir := filepath.Join(tmp, "completedChange")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(changeDir, "change.json"), map[string]any{
		"id":     "completedChange",
		"status": "active",
		"gates": map[string]any{
			"proposal":   map[string]string{"status": "done"},
			"discovery":  map[string]string{"status": "done"},
			"design":     map[string]string{"status": "done"},
			"planning":   map[string]string{"status": "done"},
			"execution":  map[string]string{"status": "done"},
			"acceptance": map[string]string{"status": "done"},
			"release":    map[string]string{"status": "done"},
		},
	})

	summary, err := SummarizeChange(changeDir)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CurrentGate != "✓" {
		t.Fatalf("expected '✓' for all-done, got '%s'", summary.CurrentGate)
	}
}

func TestSummarizeChange_MissingFile(t *testing.T) {
	summary, err := SummarizeChange("/nonexistent")
	if err != nil {
		t.Fatalf("should not error on missing file: %v", err)
	}
	if summary != nil {
		t.Fatal("expected nil summary for missing file")
	}
}

func TestSummarizeChange_ShortID(t *testing.T) {
	tmp := t.TempDir()
	changeDir := filepath.Join(tmp, "veryLongChangeIdName123456789")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(changeDir, "change.json"), map[string]any{
		"id":     "veryLongChangeIdName123456789",
		"status": "draft",
		"gates": map[string]any{
			"proposal": map[string]string{"status": "pending"},
		},
	})

	summary, err := SummarizeChange(changeDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.ShortID) > 15 {
		t.Fatalf("shortID should be ≤15 chars, got %d: '%s'", len(summary.ShortID), summary.ShortID)
	}
}

// ── WorkspaceState Tests ──────────────────────────────────

func TestWorkspaceState_SetupFailedWinsOverActive(t *testing.T) {
	// Test the status-to-glyph mapping directly.
	glyph := StatusToGlyph("setup_failed")
	if glyph == "" {
		t.Fatal("expected non-empty glyph for setup_failed")
	}
	if !strings.Contains(glyph, "✗") {
		t.Fatalf("expected ✗ in glyph for setup_failed, got '%s'", glyph)
	}
}

func TestStatusToGlyph_AllStatuses(t *testing.T) {
	tests := []struct {
		status   string
		wantChar string
	}{
		{"setup_failed", "✗"},
		{"stale", "ѻ"},
		{"merged", "✓"},
		{"pending_delete", "␡"},
		{"active", ""},
		{"idle", ""},
		{"materializing", ""},
		{"unknown_status", ""},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := StatusToGlyph(tt.status)
			if tt.wantChar == "" {
				if got != "" {
					t.Fatalf("expected empty glyph for '%s', got '%s'", tt.status, got)
				}
			} else if !strings.Contains(got, tt.wantChar) {
				t.Fatalf("expected '%s' in glyph for '%s', got '%s'", tt.wantChar, tt.status, got)
			}
		})
	}
}

// ── BranchSafety Helper Tests ──────────────────────────────

func TestIsDefaultBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   bool
	}{
		{"main", true},
		{"master", true},
		{"trunk", true},
		{"develop", true},
		{"change/myFeature", false},
		{"feature/foo", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			if got := IsDefaultBranch(tt.branch); got != tt.want {
				t.Fatalf("IsDefaultBranch(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

func TestIsActionableWorktree(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"active", true},
		{"idle", true},
		{"materializing", true},
		{"setup_failed", true},
		{"pending_delete", true},
		{"unmaterialized", true},
		{"merged", false},
		{"stale", false},
		{"deleted", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := IsActionableWorktree(tt.status); got != tt.want {
				t.Fatalf("IsActionableWorktree(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// ── TemporalHealthProbe Tests ──────────────────────────────

func TestParseTemporalAddress_Plain(t *testing.T) {
	host, port := parseTemporalAddress("127.0.0.1:7233")
	if host != "127.0.0.1" {
		t.Fatalf("expected host '127.0.0.1', got '%s'", host)
	}
	if port != "7233" {
		t.Fatalf("expected port '7233', got '%s'", port)
	}
}

func TestParseTemporalAddress_IPv6(t *testing.T) {
	host, port := parseTemporalAddress("[::1]:7233")
	if host != "::1" {
		t.Fatalf("expected host '::1', got '%s'", host)
	}
	if port != "7233" {
		t.Fatalf("expected port '7233', got '%s'", port)
	}
}

func TestParseTemporalAddress_NoPort(t *testing.T) {
	host, port := parseTemporalAddress("192.168.1.1")
	if host != "192.168.1.1" {
		t.Fatalf("expected host '192.168.1.1', got '%s'", host)
	}
	if port != "7233" {
		t.Fatalf("expected default port '7233', got '%s'", port)
	}
}

func TestParseTemporalEnvFile(t *testing.T) {
	cacheDir := t.TempDir()
	envFile := filepath.Join(cacheDir, "temporal.env")
	content := "ADV_TEMPORAL_ADDRESS=127.0.0.1:7233\nADV_TEMPORAL_NAMESPACE=default\n"
	if err := os.WriteFile(envFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	addr := readTemporalAddress(cacheDir)
	if addr != "127.0.0.1:7233" {
		t.Fatalf("expected '127.0.0.1:7233', got '%s'", addr)
	}
}

func TestParseTemporalEnvFile_Missing(t *testing.T) {
	addr := readTemporalAddress(t.TempDir())
	if addr != "" {
		t.Fatalf("expected empty address for missing file, got '%s'", addr)
	}
}

func TestProbeTCP_IPv6Loopback(t *testing.T) {
	listener, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	if !probeTCP("::1", port) {
		t.Fatal("expected IPv6 loopback probe to succeed")
	}
	<-done
}

// ── Format Summary Tests ──────────────────────────────────

func TestFormatChangeSummary(t *testing.T) {
	summary := &ChangeSummary{
		ID:          "testChange01",
		ShortID:     "testChange01",
		CurrentGate: "design",
		Status:      "draft",
	}
	got := FormatChangeSummaryText(summary)
	want := "testChange01:design"
	if got != want {
		t.Fatalf("FormatChangeSummaryText = '%s', want '%s'", got, want)
	}
}

func TestFormatChangeSummary_AllDone(t *testing.T) {
	summary := &ChangeSummary{
		ID:          "completedChange",
		ShortID:     "completedChange",
		CurrentGate: "✓",
		Status:      "active",
	}
	got := FormatChangeSummaryText(summary)
	want := "completedChange:✓"
	if got != want {
		t.Fatalf("FormatChangeSummaryText = '%s', want '%s'", got, want)
	}
}

// ── JSON Output Tests ──────────────────────────────────────

func TestFormatJSON_ActiveChange(t *testing.T) {
	summary := &ChangeSummary{
		ID:          "testChange01",
		ShortID:     "testChange01",
		CurrentGate: "design",
		Status:      "draft",
	}
	got, err := FormatJSON("active-change", summary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"query"`) || !strings.Contains(got, `"active-change"`) {
		t.Fatalf("JSON output missing query field: %s", got)
	}
	if !strings.Contains(got, `"currentGate"`) {
		t.Fatalf("JSON output missing result fields: %s", got)
	}
}

func TestFormatJSON_TemporalHealth(t *testing.T) {
	health := &TemporalHealth{
		Reachable: true,
		Address:   "127.0.0.1:7233",
		Glyph:     "T:✓",
	}
	got, err := FormatJSON("temporal-health", health)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"temporal-health"`) {
		t.Fatalf("JSON output missing query: %s", got)
	}
	if !strings.Contains(got, `"reachable":true`) {
		t.Fatalf("JSON output missing reachable: %s", got)
	}
}

// ── Golden File Test ──────────────────────────────────────

func TestGolden_ActiveChangeText(t *testing.T) {
	summary := &ChangeSummary{
		ID:          "testChange01",
		ShortID:     "testChange01",
		CurrentGate: "design",
		Status:      "draft",
	}
	got := FormatChangeSummaryText(summary)
	golden := filepath.Join("testdata", "active-change-text.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Skipf("golden file not found: %v", err)
	}
	if got != strings.TrimSpace(string(want)) {
		t.Fatalf("golden mismatch:\ngot:  %q\nwant: %q", got, strings.TrimSpace(string(want)))
	}
}

func TestGolden_TemporalReachable(t *testing.T) {
	health := &TemporalHealth{Reachable: true, Address: "127.0.0.1:7233", Glyph: "T:✓"}
	got := FormatTemporalHealthText(health)
	golden := filepath.Join("testdata", "temporal-health-reachable.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Skipf("golden file not found: %v", err)
	}
	if got != strings.TrimSpace(string(want)) {
		t.Fatalf("golden mismatch:\ngot:  %q\nwant: %q", got, strings.TrimSpace(string(want)))
	}
}

func TestGolden_BranchSafetyUnsafe(t *testing.T) {
	result := &BranchSafetyResult{
		Unsafe: true,
		Branch: "trunk",
		Glyph:  fmt.Sprintf("#[fg=#E5A649]⚡#[default]"),
	}
	got := result.Glyph
	golden := filepath.Join("testdata", "branch-safety-unsafe.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Skipf("golden file not found: %v", err)
	}
	if got != strings.TrimSpace(string(want)) {
		t.Fatalf("golden mismatch:\ngot:  %q\nwant: %q", got, strings.TrimSpace(string(want)))
	}
}

func TestGolden_WorkspaceLookup(t *testing.T) {
	entries := []WorkspaceLookupEntry{
		{ChangeID: "testChange01", Status: "active"},
		{ChangeID: "brokenWT", Status: "setup_failed"},
	}
	got := FormatWorkspaceLookupTSV(entries)
	golden := filepath.Join("testdata", "workspace-lookup.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Skipf("golden file not found: %v", err)
	}
	if got != strings.TrimSpace(string(want)) {
		t.Fatalf("golden mismatch:\ngot:  %q\nwant: %q", got, strings.TrimSpace(string(want)))
	}
}
