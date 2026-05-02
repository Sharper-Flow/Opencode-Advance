package temporal

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRecentLogLinesReturnsBoundedTail(t *testing.T) {
	path := writeLogFixture(t, 250)
	got, err := RecentLogLines(path, 200)
	if err != nil {
		t.Fatalf("RecentLogLines: %v", err)
	}
	if strings.Contains(got, "line-049") || !strings.Contains(got, "line-050") || !strings.Contains(got, "line-249") {
		t.Fatalf("unexpected tail output: first chunk=%q", got[:80])
	}

	got, err = RecentLogLines(path, 3)
	if err != nil {
		t.Fatalf("RecentLogLines 3: %v", err)
	}
	if got != "line-247\nline-248\nline-249\n" {
		t.Fatalf("tail 3 = %q", got)
	}
}

func TestRecentLogLinesMissingFileIsEmpty(t *testing.T) {
	got, err := RecentLogLines(t.TempDir()+"/missing.log", 200)
	if err != nil {
		t.Fatalf("RecentLogLines missing: %v", err)
	}
	if got != "" {
		t.Fatalf("missing log output = %q", got)
	}
}

func TestFollowLogHonorsContextCancellation(t *testing.T) {
	path := writeLogFixture(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := FollowLog(ctx, path, &strings.Builder{}, time.Millisecond); err != context.Canceled {
		t.Fatalf("FollowLog err=%v, want context.Canceled", err)
	}
}

func writeLogFixture(t *testing.T, lines int) string {
	t.Helper()
	var b strings.Builder
	for i := 0; i < lines; i++ {
		b.WriteString("line-")
		b.WriteString(pad3(i))
		b.WriteByte('\n')
	}
	path := t.TempDir() + "/temporal.log"
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return path
}

func pad3(n int) string {
	if n < 10 {
		return "00" + strconv.Itoa(n)
	}
	if n < 100 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
