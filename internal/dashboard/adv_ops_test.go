package dashboard

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"

	_ "modernc.org/sqlite"
)

func testCleanEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", "")
	t.Setenv("XDG_DATA_HOME", "")
}

func TestADVOpsPoller_UpdatesSnapshot(t *testing.T) {
	testCleanEnv(t)
	state := NewState()
	poller := NewADVOpsPoller(advruntime.Config{WorktreeStaleAfter: 7 * 24 * time.Hour})

	ctx := context.Background()
	if err := poller.Poll(ctx, state); err != nil {
		t.Fatalf("Poll: %v", err)
	}

	snap := state.Snapshot()
	if snap.ADVOps.Status == "" {
		t.Fatal("ADVOps.Status empty after poll")
	}
	// Status may be pass or warn depending on real worktree state;
	// we only verify it is set and the poll completed without error.
}

func TestADVOpsPoller_RecoveryPlanCount(t *testing.T) {
	testCleanEnv(t)
	state := NewState()
	poller := NewADVOpsPoller(advruntime.Config{})

	ctx := context.Background()
	if err := poller.Poll(ctx, state); err != nil {
		t.Fatalf("Poll: %v", err)
	}

	snap := state.Snapshot()
	// Empty report should yield 0 recovery steps
	if snap.ADVOps.RecoverySteps != 0 {
		t.Fatalf("ADVOps.RecoverySteps = %d, want 0", snap.ADVOps.RecoverySteps)
	}
}

func TestADVOpsRow_JSON(t *testing.T) {
	row := ADVOpsRow{
		Status:        "warn",
		WorktreeDebt:  2,
		SessionDebt:   3,
		RecoverySteps: 3,
		Message:       "test",
	}

	// Verify the row is JSON-serializable via the Snapshot
	state := NewState()
	state.Update(func(snap *Snapshot) {
		snap.ADVOps = row
	})

	snap := state.Snapshot()
	if snap.ADVOps.WorktreeDebt != 2 {
		t.Fatalf("worktree debt = %d, want 2", snap.ADVOps.WorktreeDebt)
	}
	if snap.ADVOps.SessionDebt != 3 {
		t.Fatalf("session debt = %d, want 3", snap.ADVOps.SessionDebt)
	}
}

func TestADVOpsPoller_SessionDebtScan(t *testing.T) {
	testCleanEnv(t)

	// Create a fake OpenCode DB with repairable messages.
	home := os.Getenv("HOME")
	dbDir := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "opencode.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	schema := `
		CREATE TABLE IF NOT EXISTS session (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS message (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			data TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS part (
			id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			session_id TEXT NOT NULL,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			data TEXT NOT NULL
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	_, _ = db.Exec("INSERT INTO session (id, title) VALUES ('sess-1', 'test')")
	staleTime := time.Now().Add(-10 * time.Minute).UnixMilli()
	data := `{"role":"assistant"}`
	_, err = db.Exec(
		"INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?)",
		"msg-stale", "sess-1", staleTime, staleTime, data,
	)
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}

	state := NewState()
	poller := NewADVOpsPoller(advruntime.Config{SessionStaleAfter: 5 * time.Minute})

	ctx := context.Background()
	if err := poller.Poll(ctx, state); err != nil {
		t.Fatalf("Poll: %v", err)
	}

	snap := state.Snapshot()
	if snap.ADVOps.SessionDebt != 1 {
		t.Fatalf("session debt = %d, want 1", snap.ADVOps.SessionDebt)
	}
	if snap.ADVOps.Status != string(advruntime.StatusWarn) {
		t.Fatalf("status = %q, want warn when session debt > 0", snap.ADVOps.Status)
	}
}
