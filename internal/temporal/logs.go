package temporal

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

// RecentLogLines returns the last n lines from path. Missing logs are empty.
func RecentLogLines(path string, n int) (string, error) {
	if n <= 0 {
		n = 200
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	text := string(b)
	if text == "" {
		return "", nil
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, ""), nil
}

// FollowLog streams appended log output until ctx is cancelled.
func FollowLog(ctx context.Context, path string, w io.Writer, poll time.Duration) error {
	if poll <= 0 {
		poll = 250 * time.Millisecond
	}
	var offset int64
	if info, err := os.Stat(path); err == nil {
		offset = info.Size()
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(poll):
			b, err := os.ReadFile(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if int64(len(b)) > offset {
				if _, err := w.Write(b[offset:]); err != nil {
					return err
				}
				offset = int64(len(b))
			}
		}
	}
}
