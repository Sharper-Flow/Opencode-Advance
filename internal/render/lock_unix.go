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

// ReleaseApplyLock unlocks and closes the lock file. Both the Flock
// LOCK_UN syscall and Close can fail; a silent failure in unlock would
// be catastrophic because the next oca apply on the same cache dir
// would block waiting for a lock that no one holds. We surface the
// first error (unlock preferred, close as fallback) so callers can log
// it — but we still always call Close so the file descriptor is
// released even when unlock fails.
func ReleaseApplyLock(f *os.File) error {
	if f == nil {
		return nil
	}
	unlockErr := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	closeErr := f.Close()
	if unlockErr != nil {
		return fmt.Errorf("release apply lock: flock LOCK_UN: %w", unlockErr)
	}
	return closeErr
}
