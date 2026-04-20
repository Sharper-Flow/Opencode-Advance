package render

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// backupGlob returns the glob pattern matching all historical backups for
// a rendered target (e.g. opencode.json.bak.1729875000).
func backupGlob(path string) string { return path + ".bak.*" }

// orphanBackupTempGlob returns the glob pattern matching half-written
// backup temp files left over from a crashed apply (e.g. opencode.json.bak.NNNN.tmp).
// These are cleaned up at the start of each apply run.
func orphanBackupTempGlob(path string) string { return path + ".bak.*.tmp" }

// removeAll unlinks every path in paths. If multiple removals fail, the
// first filesystem error is returned with the offending path; subsequent
// removals are still attempted so a single stubborn file does not leave
// the rest behind. Missing files are ignored (idempotent).
func removeAll(paths []string) error {
	var firstErr error
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = fmt.Errorf("remove %s: %w", p, err)
		}
	}
	return firstErr
}

func cleanupOrphanBackupTemps(path string) error {
	matches, err := filepath.Glob(orphanBackupTempGlob(path))
	if err != nil {
		return err
	}
	return removeAll(matches)
}

func pruneBackups(path string, keep int) error {
	matches, err := filepath.Glob(backupGlob(path))
	if err != nil {
		return err
	}
	if keep <= 0 {
		return removeAll(matches)
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
	victims := make([]string, 0, len(items)-keep)
	for _, it := range items[keep:] {
		victims = append(victims, it.path)
	}
	return removeAll(victims)
}
