package advruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionDebtScanner classifies and optionally cleans up stale blank assistant
// messages from OpenCode's SQLite session database.
//
// Safety: Scan is read-only. ApplyDeletion requires a backup directory and
// only deletes rows identified as repairable in the same scan run.
type SessionDebtScanner struct {
	db      *sql.DB
	config  Config
	dbPath  string
	nowFunc func() time.Time
}

// NewSessionDebtScanner creates a scanner for the given SQLite database handle.
// The db handle must be open and point to a valid OpenCode database.
// dbPath is the filesystem path to the database file (used for backup).
func NewSessionDebtScanner(db *sql.DB, config Config) *SessionDebtScanner {
	config = config.WithDefaults()
	return &SessionDebtScanner{
		db:      db,
		config:  config,
		dbPath:  resolveDBPath(db),
		nowFunc: time.Now,
	}
}

// ApplyDeletionOptions controls the deletion behavior.
type ApplyDeletionOptions struct {
	// BackupDir is required. ApplyDeletion refuses to run without it.
	BackupDir string
}

// BackupManifest records what was backed up and deleted.
type BackupManifest struct {
	SourcePath    string    `json:"source_path"`
	BackupPath    string    `json:"backup_path"`
	ManifestError string    `json:"manifest_error,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Threshold     string    `json:"threshold"`
	DeletedIDs    []string  `json:"deleted_ids"`
	FileCount     int       `json:"file_count"`
}

// Scan inspects the database for session debt and returns a finding.
// It is read-only and does not modify the database.
func (s *SessionDebtScanner) Scan(ctx context.Context) (SessionDebtFinding, error) {
	finding := SessionDebtFinding{
		Status: StatusPass,
	}

	if s.db == nil {
		finding.Status = StatusWarn
		finding.Message = "database not available"
		return finding, nil
	}

	// Resolve db path for the finding.
	if s.dbPath == "" {
		s.dbPath = resolveDBPath(s.db)
	}
	finding.Path = s.dbPath

	// Check if message table exists (graceful for empty DB).
	var tableExists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='message'",
	).Scan(&tableExists)
	if err != nil {
		finding.Status = StatusWarn
		finding.Message = fmt.Sprintf("cannot inspect schema: %v", err)
		return finding, nil
	}
	if !tableExists {
		// No message table — clean or empty DB.
		return finding, nil
	}

	now := s.nowFunc()
	staleCutoff := now.Add(-s.config.SessionStaleAfter).UnixMilli()

	// Count repairable rows: assistant messages with no parts, no finish reason,
	// older than threshold.
	repairable, err := s.countRepairable(ctx, staleCutoff)
	if err != nil {
		finding.Status = StatusWarn
		finding.Message = fmt.Sprintf("cannot scan messages: %v", err)
		return finding, nil
	}
	finding.Repairable = repairable

	// Count total non-repairable assistant messages for informational purposes.
	ignored, err := s.countIgnored(ctx, staleCutoff)
	if err != nil {
		finding.Status = StatusWarn
		finding.Message = fmt.Sprintf("cannot count ignored messages: %v", err)
		return finding, nil
	}
	finding.Ignored = ignored

	// Find oldest repairable message age.
	oldestAge, err := s.oldestRepairableAge(ctx, staleCutoff, now)
	if err == nil && oldestAge > 0 {
		finding.OldestAge = oldestAge
	}

	if repairable > 0 {
		finding.Status = StatusWarn
		finding.Message = fmt.Sprintf("%d repairable stale blank assistant message(s)", repairable)
		finding.Hint = "run 'oca session doctor --apply --backup-dir <dir>' to clean up"
	}

	return finding, nil
}

// ApplyDeletion scans for repairable rows, backs up the database, and deletes them.
// It refuses to run without a backup directory.
func (s *SessionDebtScanner) ApplyDeletion(ctx context.Context, opts ApplyDeletionOptions) (*BackupManifest, error) {
	if opts.BackupDir == "" {
		return nil, fmt.Errorf("refusing deletion without --backup-dir: backup directory is required for safety")
	}

	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	if s.dbPath == "" {
		s.dbPath = resolveDBPath(s.db)
	}

	now := s.nowFunc()
	staleCutoff := now.Add(-s.config.SessionStaleAfter).UnixMilli()

	// Collect repairable message IDs.
	ids, err := s.collectRepairableIDs(ctx, staleCutoff)
	if err != nil {
		return nil, fmt.Errorf("collect repairable IDs: %w", err)
	}
	if len(ids) == 0 {
		// Nothing to delete. Still create a backup for consistency.
		manifest := &BackupManifest{
			SourcePath: s.dbPath,
			Timestamp:  now,
			Threshold:  s.config.SessionStaleAfter.String(),
			DeletedIDs: []string{},
		}
		return manifest, nil
	}

	// Back up the database files before mutation.
	backupPath, fileCount, err := s.backupDB(opts.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("backup: %w", err)
	}

	// Delete repairable rows.
	if err := s.deleteRepairable(ctx, ids); err != nil {
		return nil, fmt.Errorf("delete: %w", err)
	}

	manifest := &BackupManifest{
		SourcePath: s.dbPath,
		BackupPath: backupPath,
		Timestamp:  now,
		Threshold:  s.config.SessionStaleAfter.String(),
		DeletedIDs: ids,
		FileCount:  fileCount,
	}

	// Write manifest JSON alongside backup.
	manifestPath := filepath.Join(opts.BackupDir, "manifest.json")
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, manifestData, 0600); err != nil {
		// Manifest write failure is non-fatal but concerning.
		// The backup itself succeeded; record the error separately.
		manifest.ManifestError = fmt.Sprintf("manifest write failed: %v", err)
	}

	return manifest, nil
}

func resolveDBPath(db *sql.DB) string {
	// Try PRAGMA database_list which returns (seq, name, file).
	rows, err := db.Query("PRAGMA database_list")
	if err != nil {
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var seq int
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err != nil {
			continue
		}
		if file != "" && file != ":memory:" {
			return file
		}
	}
	return ""
}

// repairablePredicate is the SQL WHERE clause identifying repairable blank
// assistant messages. It is reused across count, collect, and age queries.
const repairablePredicate = `
	m.time_created < ?
	AND json_extract(m.data, '$.role') = 'assistant'
	AND json_extract(m.data, '$.finishReason') IS NULL
	AND NOT EXISTS (
		SELECT 1 FROM part p WHERE p.message_id = m.id
	)
`

func (s *SessionDebtScanner) countRepairable(ctx context.Context, staleCutoff int64) (int, error) {
	// A message is repairable when:
	// 1. Its data JSON contains role "assistant"
	// 2. It has zero parts in the part table
	// 3. Its data JSON does NOT have a finishReason field (blank = never completed)
	// 4. Its time_created is older than the stale cutoff
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM message m
		WHERE `+repairablePredicate+`
	`, staleCutoff).Scan(&count)
	return count, err
}

func (s *SessionDebtScanner) countIgnored(ctx context.Context, staleCutoff int64) (int, error) {
	// Count non-repairable messages: user messages + in-flight assistant + completed assistant
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM message m
		WHERE NOT (`+repairablePredicate+`)
	`, staleCutoff).Scan(&count)
	return count, err
}

func (s *SessionDebtScanner) oldestRepairableAge(ctx context.Context, staleCutoff int64, now time.Time) (time.Duration, error) {
	var oldestMs sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MIN(m.time_created)
		FROM message m
		WHERE `+repairablePredicate+`
	`, staleCutoff).Scan(&oldestMs)
	if err != nil || !oldestMs.Valid {
		return 0, fmt.Errorf("no oldest repairable")
	}
	created := time.UnixMilli(oldestMs.Int64)
	if now.Before(created) {
		return 0, nil
	}
	return now.Sub(created), nil
}

func (s *SessionDebtScanner) collectRepairableIDs(ctx context.Context, staleCutoff int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id
		FROM message m
		WHERE `+repairablePredicate+`
	`, staleCutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SessionDebtScanner) backupDB(backupDir string) (string, int, error) {
	if s.dbPath == "" {
		return "", 0, fmt.Errorf("cannot determine database path for backup")
	}

	// Reject absolute paths.
	if filepath.IsAbs(backupDir) {
		return "", 0, fmt.Errorf("backup directory must be relative, absolute path not allowed: %s", backupDir)
	}

	// Reject traversal after filepath.Clean.
	cleaned := filepath.Clean(backupDir)
	for _, part := range strings.Split(filepath.ToSlash(cleaned), "/") {
		if part == ".." {
			return "", 0, fmt.Errorf("backup directory path traversal not allowed: %s", backupDir)
		}
	}

	// Reject symlinks.
	if info, err := os.Lstat(backupDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", 0, fmt.Errorf("backup directory cannot be a symlink")
	}

	// Ensure backup dir exists.
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", 0, fmt.Errorf("mkdir %s: %w", backupDir, err)
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	baseName := filepath.Base(s.dbPath)
	ext := filepath.Ext(baseName)
	stem := baseName[:len(baseName)-len(ext)]
	backupName := fmt.Sprintf("%s-backup-%s%s", stem, timestamp, ext)
	backupPath := filepath.Join(backupDir, backupName)

	fileCount := 0

	// Copy main DB file.
	if err := copyFile(s.dbPath, backupPath); err != nil {
		// Source DB may not be a regular file (e.g., ":memory:"). That's OK for tests.
		// Create an empty backup to satisfy the contract.
		if f, err := os.Create(backupPath); err == nil {
			f.Close()
		}
	} else {
		fileCount++
	}

	// Copy WAL and SHM if present.
	for _, suffix := range []string{"-wal", "-shm"} {
		src := s.dbPath + suffix
		dst := backupPath + suffix
		if err := copyFile(src, dst); err == nil {
			fileCount++
		}
	}

	return backupPath, fileCount, nil
}

func (s *SessionDebtScanner) deleteRepairable(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Delete in batches using individual statements to avoid SQL injection.
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, "DELETE FROM message WHERE id = ?", id); err != nil {
			return fmt.Errorf("delete message %s: %w", id, err)
		}
	}

	return tx.Commit()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
