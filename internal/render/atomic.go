package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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
		if err := pruneBackups(path, maxBackups-1); err != nil {
			return "", err
		}
		backupPath = fmt.Sprintf("%s.bak.%d", path, time.Now().UnixNano())
		if err := os.WriteFile(backupPath+".tmp", existing, mode); err != nil {
			return "", err
		}
		if err := os.Rename(backupPath+".tmp", backupPath); err != nil {
			return "", err
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
		_ = tmp.Close()
		return backupPath, err
	}
	if err := tmp.Close(); err != nil {
		return backupPath, err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return backupPath, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}
