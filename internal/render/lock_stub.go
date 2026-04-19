//go:build !unix

package render

import (
	"os"
	"time"
)

// AcquireApplyLock is a no-op on non-Unix platforms. The Unix implementation
// in lock_unix.go uses flock(2) for cross-process mutual exclusion during
// `oca apply`. On platforms lacking flock, concurrent applies are not
// serialized; callers should avoid running `oca apply` in parallel on
// non-Unix systems. Returns (nil, nil) to satisfy the interface.
func AcquireApplyLock(_ string, _ time.Duration) (*os.File, error) { return nil, nil }

// ReleaseApplyLock is a no-op on non-Unix platforms. See AcquireApplyLock.
func ReleaseApplyLock(_ *os.File) error { return nil }
