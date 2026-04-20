package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var renameFile = os.Rename

// SetRenameForTesting overrides the rename function used by WriteAtomic.
// Tests can inject rename failures or delays to exercise rollback/locking.
func SetRenameForTesting(fn func(oldPath, newPath string) error) func() {
	prev := renameFile
	renameFile = fn
	return func() { renameFile = prev }
}

// WriteAtomic writes data to path using temp-file + rename in the same
// directory. If path exists, a .bak.<UnixNano> copy is created first.
//
// The returned backupPath is non-empty whenever a backup file was created,
// even if a subsequent step (tmp write, chmod, rename) failed. This lets the
// caller restore the previous content on partial failure. backupPath is
// empty only when no backup was needed (file did not exist, or new content
// matched existing) or when the failure occurred before the backup was
// written.
func WriteAtomic(path string, data []byte, mode os.FileMode, maxBackups int) (string, error) {
	if err := cleanupOrphanBackupTemps(path); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	var backupPath string
	existing, err := os.ReadFile(path)
	if err == nil {
		if bytes.Equal(existing, data) {
			return "", nil
		}
		if maxBackups > 0 {
			if err := pruneBackups(path, maxBackups-1); err != nil {
				return "", err
			}
			backupPath = fmt.Sprintf("%s.bak.%d", path, time.Now().UnixNano())
			if err := os.WriteFile(backupPath+".tmp", existing, mode); err != nil {
				return "", err
			}
			if err := renameFile(backupPath+".tmp", backupPath); err != nil {
				return "", err
			}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".oca-write-*.tmp")
	if err != nil {
		return backupPath, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		// Close is called so the fd is released, but its error is
		// intentionally dropped in favor of the Write error which is
		// the root cause and more actionable. The temp file itself is
		// removed by the deferred os.Remove above.
		_ = tmp.Close()
		return backupPath, fmt.Errorf("write temp %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return backupPath, err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return backupPath, err
	}
	if err := renameFile(tmpPath, path); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}
