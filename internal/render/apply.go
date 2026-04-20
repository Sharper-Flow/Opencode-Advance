package render

import (
	"os"
	"path/filepath"
	"time"
)

// Apply executes a render plan sequentially.
//
// On write failure mid-plan, previously written targets are rolled back:
// targets that had a backup are restored from it; targets that were newly
// created (no backup) are removed. Rollback errors are best-effort and do
// not override the original error returned to the caller.
func Apply(plan *Plan, opts ApplyOptions) (*ApplyResult, error) {
	res := &ApplyResult{Targets: make([]TargetResult, 0, len(plan.Targets))}
	if opts.MaxBackups == 0 {
		opts.MaxBackups = 3
	}
	if opts.DryRun {
		for _, t := range plan.Targets {
			res.Targets = append(res.Targets, TargetResult{Path: t.Path, Op: t.Op, Wrote: false})
		}
		return res, nil
	}
	lockPath := opts.LockPath
	if lockPath == "" {
		lockPath = plan.LockPath
	}
	lock, err := AcquireApplyLock(filepath.Dir(lockPath), 30*time.Second)
	if err != nil {
		return res, err
	}
	defer ReleaseApplyLock(lock)

	for i, t := range plan.Targets {
		r := TargetResult{Path: t.Path, Op: t.Op}
		if t.Op != "noop" {
			maxBackups := opts.MaxBackups
			if t.SuppressBackup {
				maxBackups = 0
			}
			backup, err := WriteAtomic(t.Path, t.After, t.Mode, maxBackups)
			if err != nil {
				r.BackupPath = backup
				res.Targets = append(res.Targets, r)
				rollback(plan.Targets[:i], res.Targets[:i])
				return res, err
			}
			r.Wrote = true
			r.BackupPath = backup
		}
		res.Targets = append(res.Targets, r)
	}
	return res, nil
}

// rollback restores previously written targets after a mid-plan failure.
// Walks in reverse: restores from backup if present, otherwise removes
// newly created files. Errors are swallowed — rollback is best-effort.
func rollback(targets []TargetOp, results []TargetResult) {
	for i := len(results) - 1; i >= 0; i-- {
		r := results[i]
		if !r.Wrote {
			continue
		}
		t := targets[i]
		if r.BackupPath != "" {
			if data, err := os.ReadFile(r.BackupPath); err == nil {
				_ = os.WriteFile(r.Path, data, t.Mode)
			}
			continue
		}
		_ = os.Remove(r.Path)
	}
}
