//go:build unix

package render

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// flockRetryInterval is the sleep between Flock(LOCK_EX|LOCK_NB) attempts
// while waiting for another oca apply to release the lock. Kept short so
// waiters notice release quickly without busy-spinning the CPU.
const flockRetryInterval = 50 * time.Millisecond

func AcquireApplyLock(cacheDir string, timeout time.Duration) (*os.File, error) {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(cacheDir, "apply.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return f, nil
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("timed out acquiring apply lock after %s", timeout)
		}
		time.Sleep(flockRetryInterval)
	}
}

func ReleaseApplyLock(f *os.File) error {
	if f == nil {
		return nil
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return f.Close()
}
