package render

import (
	"path/filepath"
	"time"
)

// Apply executes a render plan sequentially.
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

	for _, t := range plan.Targets {
		r := TargetResult{Path: t.Path, Op: t.Op}
		if t.Op != "noop" {
			backup, err := WriteAtomic(t.Path, t.After, t.Mode, opts.MaxBackups)
			if err != nil {
				res.Targets = append(res.Targets, r)
				return res, err
			}
			r.Wrote = true
			r.BackupPath = backup
		}
		res.Targets = append(res.Targets, r)
	}
	return res, nil
}
