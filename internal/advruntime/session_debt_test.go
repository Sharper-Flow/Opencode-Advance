package advruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// openTestDB creates an isolated SQLite database with the OpenCode message/part
// schema. Callers must call db.Close() and may clean up the temp directory.
func openTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "opencode.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db.SetMaxOpenConns(1)

	// Create minimal OpenCode-compatible schema.
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
		db.Close()
		t.Fatalf("create schema: %v", err)
	}
	return db, dir
}

// insertMessage inserts a message row. data is JSON-encoded message info.
func insertMessage(t *testing.T, db *sql.DB, id, sessionID string, createdAt int64, data any) {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal message data: %v", err)
	}
	_, err = db.Exec(
		"INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?)",
		id, sessionID, createdAt, createdAt, string(b),
	)
	if err != nil {
		t.Fatalf("insert message %s: %v", id, err)
	}
}

// insertPart inserts a part row for a message.
func insertPart(t *testing.T, db *sql.DB, id, messageID, sessionID string, createdAt int64, data any) {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal part data: %v", err)
	}
	_, err = db.Exec(
		"INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?, ?)",
		id, messageID, sessionID, createdAt, createdAt, string(b),
	)
	if err != nil {
		t.Fatalf("insert part %s: %v", id, err)
	}
}

// msgInfo returns a message data JSON shape matching OpenCode's Info type.
// role should be "assistant" or "user".
func msgInfo(role string) map[string]any {
	return map[string]any{
		"role": role,
	}
}

func TestSessionDebtScanner_ClassifiesRepairableBlankAssistantMessages(t *testing.T) {
	db, dir := openTestDB(t)
	defer db.Close()

	now := time.Now().UnixMilli()
	staleTime := now - (10 * time.Minute).Milliseconds() // 10 minutes ago — exceeds 5-min threshold

	// Insert a session.
	_, _ = db.Exec("INSERT INTO session (id, title) VALUES ('sess-1', 'test')")

	// Repairable: assistant message, no parts, stale, no finish timestamp in data
	insertMessage(t, db, "msg-stale-1", "sess-1", staleTime, msgInfo("assistant"))
	insertMessage(t, db, "msg-stale-2", "sess-1", staleTime, msgInfo("assistant"))

	// Not repairable: user message, no parts, stale
	insertMessage(t, db, "msg-user-1", "sess-1", staleTime, msgInfo("user"))

	// Not repairable: assistant message WITH parts (has content)
	insertMessage(t, db, "msg-with-parts", "sess-1", staleTime, msgInfo("assistant"))
	insertPart(t, db, "part-1", "msg-with-parts", "sess-1", staleTime, map[string]any{"type": "text", "text": "hello"})

	// Not repairable: assistant message, recent (in-flight)
	insertMessage(t, db, "msg-recent", "sess-1", now, msgInfo("assistant"))

	scanner := NewSessionDebtScanner(db, Config{SessionStaleAfter: 5 * time.Minute})
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if result.Path != filepath.Join(dir, "opencode.db") {
		t.Fatalf("Path = %q, want db path", result.Path)
	}
	if result.Repairable != 2 {
		t.Fatalf("Repairable = %d, want 2 (only stale blank assistant messages)", result.Repairable)
	}
	if result.Ignored < 2 {
		// At minimum: the user message + the recent in-flight message should be counted.
		// The message with parts is also non-repairable but not necessarily "ignored" in the same way.
		t.Fatalf("Ignored = %d, want >= 2 (user message + in-flight)", result.Ignored)
	}
	if result.Status != StatusWarn {
		t.Fatalf("Status = %q, want %q", result.Status, StatusWarn)
	}
	if result.OldestAge < 10*time.Minute-time.Second {
		t.Fatalf("OldestAge = %v, want approximately 10 minutes", result.OldestAge)
	}
}

func TestSessionDebtScanner_CleanDBReportsPassWithZeroDebt(t *testing.T) {
	db, dir := openTestDB(t)
	defer db.Close()

	// Insert a session with a healthy completed assistant message (has parts)
	_, _ = db.Exec("INSERT INTO session (id, title) VALUES ('sess-1', 'test')")
	now := time.Now().UnixMilli()
	insertMessage(t, db, "msg-ok", "sess-1", now, msgInfo("assistant"))
	insertPart(t, db, "part-ok", "msg-ok", "sess-1", now, map[string]any{"type": "text", "text": "response"})

	scanner := NewSessionDebtScanner(db, Config{SessionStaleAfter: 5 * time.Minute})
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if result.Status != StatusPass {
		t.Fatalf("Status = %q, want %q for clean DB", result.Status, StatusPass)
	}
	if result.Repairable != 0 {
		t.Fatalf("Repairable = %d, want 0", result.Repairable)
	}
	_ = dir // suppress unused
}

func TestSessionDebtScanner_HandlesMissingDBGracefully(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nonexistent.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Don't create any tables — simulate empty/missing schema.
	defer db.Close()

	scanner := NewSessionDebtScanner(db, Config{SessionStaleAfter: 5 * time.Minute})
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan should not error on empty/missing schema: %v", err)
	}
	if result.Status != StatusPass {
		t.Fatalf("Status = %q, want pass for empty DB", result.Status)
	}
	if result.Repairable != 0 {
		t.Fatalf("Repairable = %d, want 0", result.Repairable)
	}
}

func TestSessionDebtScanner_FinishedMessagesNotRepairable(t *testing.T) {
	db, _ := openTestDB(t)
	defer db.Close()

	_, _ = db.Exec("INSERT INTO session (id, title) VALUES ('sess-1', 'test')")
	staleTime := time.Now().Add(-10 * time.Minute).UnixMilli()

	// Assistant message with finishReason (completed) but no parts — should NOT be repairable
	info := msgInfo("assistant")
	info["finishReason"] = "stop"
	insertMessage(t, db, "msg-finished", "sess-1", staleTime, info)

	scanner := NewSessionDebtScanner(db, Config{SessionStaleAfter: 5 * time.Minute})
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.Repairable != 0 {
		t.Fatalf("Repairable = %d, want 0 (finished messages should not be repairable)", result.Repairable)
	}
}

func TestSessionDebtBackup_RefusesWithoutBackupDir(t *testing.T) {
	db, _ := openTestDB(t)
	defer db.Close()

	_, _ = db.Exec("INSERT INTO session (id, title) VALUES ('sess-1', 'test')")
	staleTime := time.Now().Add(-10 * time.Minute).UnixMilli()
	insertMessage(t, db, "msg-stale", "sess-1", staleTime, msgInfo("assistant"))

	scanner := NewSessionDebtScanner(db, Config{SessionStaleAfter: 5 * time.Minute})

	_, err := scanner.ApplyDeletion(context.Background(), ApplyDeletionOptions{
		BackupDir: "", // empty — should refuse
	})
	if err == nil {
		t.Fatal("ApplyDeletion should refuse without backup dir")
	}
}

func TestSessionDebtBackup_CopiesDBAndCreatesManifest(t *testing.T) {
	db, dir := openTestDB(t)
	defer db.Close()

	_, _ = db.Exec("INSERT INTO session (id, title) VALUES ('sess-1', 'test')")
	staleTime := time.Now().Add(-10 * time.Minute).UnixMilli()
	insertMessage(t, db, "msg-stale-1", "sess-1", staleTime, msgInfo("assistant"))
	insertMessage(t, db, "msg-stale-2", "sess-1", staleTime, msgInfo("assistant"))
	// Non-repairable: assistant with parts
	insertMessage(t, db, "msg-with-parts", "sess-1", staleTime, msgInfo("assistant"))
	insertPart(t, db, "part-1", "msg-with-parts", "sess-1", staleTime, map[string]any{"type": "text", "text": "hello"})

	backupDir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatalf("mkdir backups: %v", err)
	}

	scanner := NewSessionDebtScanner(db, Config{SessionStaleAfter: 5 * time.Minute})
	manifest, err := scanner.ApplyDeletion(context.Background(), ApplyDeletionOptions{
		BackupDir: backupDir,
	})
	if err != nil {
		t.Fatalf("ApplyDeletion: %v", err)
	}

	// Verify manifest structure
	if len(manifest.DeletedIDs) != 2 {
		t.Fatalf("DeletedIDs = %d, want 2", len(manifest.DeletedIDs))
	}
	if manifest.SourcePath == "" {
		t.Fatal("SourcePath empty")
	}
	if manifest.Timestamp.IsZero() {
		t.Fatal("Timestamp zero")
	}
	if manifest.Threshold == "" {
		t.Fatal("Threshold empty")
	}

	// Verify backup file exists
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".db" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no .db backup file in backup dir")
	}

	// Verify repairable rows were deleted from DB
	var remaining int
	err = db.QueryRow("SELECT COUNT(*) FROM message WHERE id IN (?, ?)", "msg-stale-1", "msg-stale-2").Scan(&remaining)
	if err != nil {
		t.Fatalf("count remaining: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("remaining stale rows = %d, want 0 (deleted)", remaining)
	}

	// Verify non-repairable row still exists
	var kept int
	err = db.QueryRow("SELECT COUNT(*) FROM message WHERE id = ?", "msg-with-parts").Scan(&kept)
	if err != nil {
		t.Fatalf("count kept: %v", err)
	}
	if kept != 1 {
		t.Fatalf("kept rows = %d, want 1 (message with parts should not be deleted)", kept)
	}
}

func TestSessionDebtBackup_ManifestJSON(t *testing.T) {
	db, dir := openTestDB(t)
	defer db.Close()

	// We need to close and reopen to ensure PRAGMA database_list returns a valid path.
	db.Close()
	dbPath := filepath.Join(dir, "opencode.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	staleTime := time.Now().Add(-10 * time.Minute).UnixMilli()
	insertMessage(t, db, "msg-stale-1", "sess-1", staleTime, msgInfo("assistant"))

	backupDir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	scanner := NewSessionDebtScanner(db, Config{SessionStaleAfter: 5 * time.Minute})
	manifest, err := scanner.ApplyDeletion(context.Background(), ApplyDeletionOptions{
		BackupDir: backupDir,
	})
	if err != nil {
		t.Fatalf("ApplyDeletion: %v", err)
	}

	// Verify manifest is JSON-serializable
	b, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("empty manifest JSON")
	}

	// Verify manifest.json is written alongside backup
	manifestPath := filepath.Join(backupDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatal("manifest.json not written to backup dir")
	}

	// Round-trip: parse it back
	var parsed BackupManifest
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if len(parsed.DeletedIDs) != 1 || parsed.DeletedIDs[0] != "msg-stale-1" {
		t.Fatalf("parsed manifest DeletedIDs = %v", parsed.DeletedIDs)
	}

	_ = fmt.Sprintf("manifest JSON length: %d", len(b))
}
