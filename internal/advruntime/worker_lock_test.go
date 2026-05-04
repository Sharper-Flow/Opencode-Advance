package advruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeWorkerLock(t *testing.T, advRoot, projectID, contents string) string {
	t.Helper()
	projectDir := filepath.Join(advRoot, projectID)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project state: %v", err)
	}
	path := filepath.Join(projectDir, "worker.lock")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write worker.lock: %v", err)
	}
	return path
}

func TestWorkerLockScanner_StaleHeartbeatWithLivePIDWarns(t *testing.T) {
	advRoot := t.TempDir()
	now := time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC)
	writeWorkerLock(t, advRoot, "proj-live", `{
		"schema_version": 2,
		"pid": 1234,
		"worker_id": "worker-live",
		"acquired_at": "2026-05-04T00:00:00Z",
		"last_heartbeat": "2026-05-04T00:58:00Z"
	}`)

	scanner := NewWorkerLockScanner(advRoot, time.Minute)
	scanner.nowFunc = func() time.Time { return now }
	scanner.pidAlive = func(pid int) bool { return pid == 1234 }

	findings, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}
	got := findings[0]
	if got.ProjectID != "proj-live" || got.PID != 1234 || got.Status != StatusWarn {
		t.Fatalf("unexpected finding: %#v", got)
	}
	if !strings.Contains(got.Message, "stale heartbeat") {
		t.Fatalf("expected stale heartbeat message, got %q", got.Message)
	}
}

func TestWorkerLockScanner_FreshHeartbeatNoWarning(t *testing.T) {
	advRoot := t.TempDir()
	now := time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC)
	writeWorkerLock(t, advRoot, "proj-fresh", `{
		"schema_version": 2,
		"pid": 2222,
		"worker_id": "worker-fresh",
		"acquired_at": "2026-05-04T00:00:00Z",
		"last_heartbeat": "2026-05-04T00:59:30Z"
	}`)

	scanner := NewWorkerLockScanner(advRoot, time.Minute)
	scanner.nowFunc = func() time.Time { return now }
	scanner.pidAlive = func(pid int) bool { return true }

	findings, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %d, want 0: %#v", len(findings), findings)
	}
}

func TestWorkerLockScanner_StaleHeartbeatWithDeadPIDWarns(t *testing.T) {
	advRoot := t.TempDir()
	now := time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC)
	writeWorkerLock(t, advRoot, "proj-dead", `{
		"schema_version": 2,
		"pid": 3333,
		"worker_id": "worker-dead",
		"acquired_at": "2026-05-04T00:00:00Z",
		"last_heartbeat": "2026-05-04T00:50:00Z"
	}`)

	scanner := NewWorkerLockScanner(advRoot, time.Minute)
	scanner.nowFunc = func() time.Time { return now }
	scanner.pidAlive = func(pid int) bool { return false }

	findings, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 1 || findings[0].Status != StatusWarn {
		t.Fatalf("unexpected findings: %#v", findings)
	}
	if !strings.Contains(findings[0].Message, "dead pid") {
		t.Fatalf("expected dead pid message, got %q", findings[0].Message)
	}
}

func TestWorkerLockScanner_V1AndMissingLocksDoNotWarn(t *testing.T) {
	advRoot := t.TempDir()
	writeWorkerLock(t, advRoot, "proj-v1", `{
		"pid": 4444,
		"worker_id": "worker-v1",
		"acquired_at": "2026-05-04T00:00:00Z"
	}`)
	if err := os.MkdirAll(filepath.Join(advRoot, "proj-missing"), 0755); err != nil {
		t.Fatal(err)
	}

	scanner := NewWorkerLockScanner(advRoot, time.Minute)
	findings, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %d, want 0: %#v", len(findings), findings)
	}
}

func TestWorkerLockScanner_MalformedLockWarns(t *testing.T) {
	advRoot := t.TempDir()
	writeWorkerLock(t, advRoot, "proj-bad", `{not-json`)

	scanner := NewWorkerLockScanner(advRoot, time.Minute)
	findings, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 1 || findings[0].Status != StatusWarn {
		t.Fatalf("unexpected findings: %#v", findings)
	}
	if !strings.Contains(findings[0].Message, "cannot parse") {
		t.Fatalf("expected parse warning, got %q", findings[0].Message)
	}
}
