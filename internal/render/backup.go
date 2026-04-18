package render

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func backupGlob(path string) string { return path + ".bak.*" }

func orphanBackupTempGlob(path string) string { return path + ".bak.*.tmp" }

func cleanupOrphanBackupTemps(path string) error {
	matches, err := filepath.Glob(orphanBackupTempGlob(path))
	if err != nil {
		return err
	}
	for _, m := range matches {
		_ = os.Remove(m)
	}
	return nil
}

func pruneBackups(path string, keep int) error {
	if keep <= 0 {
		matches, err := filepath.Glob(backupGlob(path))
		if err != nil {
			return err
		}
		for _, m := range matches {
			_ = os.Remove(m)
		}
		return nil
	}
	matches, err := filepath.Glob(backupGlob(path))
	if err != nil {
		return err
	}
	type item struct {
		path  string
		epoch int64
	}
	items := make([]item, 0, len(matches))
	for _, m := range matches {
		parts := strings.Split(m, ".bak.")
		if len(parts) != 2 {
			continue
		}
		epoch, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		items = append(items, item{path: m, epoch: epoch})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].epoch > items[j].epoch })
	if len(items) <= keep {
		return nil
	}
	for _, it := range items[keep:] {
		_ = os.Remove(it.path)
	}
	return nil
}
