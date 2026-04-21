package render

import (
	"errors"
	"fmt"
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
				// Skip rollback when NoRollback is true (composed all-apply path).
				// Prior successful WriteAtomic results remain on disk per AC4.
				if !opts.NoRollback {
					if rbErrs := rollback(plan.Targets[:i], res.Targets[:i]); len(rbErrs) > 0 {
						joined := append([]error{err}, rbErrs...)
						return res, fmt.Errorf("apply failed with rollback errors: %w",
							errors.Join(joined...))
					}
				}
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
// newly created files.
//
// Rollback is best-effort on the successful-path sense (one failed
// restore must not stop us from trying to restore the rest), but
// individual failures are NOT silent: they are returned as an error
// slice so the caller can log them. A silent rollback failure would
// leave the user's config in a corrupt state with no signal, so we
// surface every error even though we cannot act on any of them from
// inside this function.
func rollback(targets []TargetOp, results []TargetResult) []error {
	var errs []error
	for i := len(results) - 1; i >= 0; i-- {
		r := results[i]
		if !r.Wrote {
			continue
		}
		t := targets[i]
		if r.BackupPath != "" {
			data, err := os.ReadFile(r.BackupPath)
			if err != nil {
				errs = append(errs, fmt.Errorf("rollback read backup %s: %w", r.BackupPath, err))
				continue
			}
			if err := os.WriteFile(r.Path, data, t.Mode); err != nil {
				errs = append(errs, fmt.Errorf("rollback restore %s: %w", r.Path, err))
			}
			continue
		}
		if err := os.Remove(r.Path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("rollback remove %s: %w", r.Path, err))
		}
	}
	return errs
}
