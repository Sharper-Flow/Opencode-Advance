//go:build !unix

package render

import (
	"os"
	"time"
)

func AcquireApplyLock(_ string, _ time.Duration) (*os.File, error) { return nil, nil }
func ReleaseApplyLock(_ *os.File) error                            { return nil }
