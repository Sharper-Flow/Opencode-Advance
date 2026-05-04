package advruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	workerLockFilename       = "worker.lock"
	defaultWorkerStaleAfter  = time.Minute
	workerHeartbeatStaleEnv  = "ADV_WORKER_HEARTBEAT_STALE_MS"
	workerRestartRemediation = "restart OpenCode or run adv_temporal_worker_restart from an ADV-capable session"
)

// WorkerLockFinding reports read-only observations about ADV worker singleton
// locks. OCA never mutates or reclaims these locks; Advance owns worker
// lifecycle and recovery.
type WorkerLockFinding struct {
	ProjectID string        `json:"project_id"`
	Path      string        `json:"path"`
	PID       int           `json:"pid,omitempty"`
	WorkerID  string        `json:"worker_id,omitempty"`
	Status    Status        `json:"status"`
	Age       time.Duration `json:"age,omitempty"`
	Message   string        `json:"message,omitempty"`
	Hint      string        `json:"hint,omitempty"`
}

type WorkerLockScanner struct {
	advRoot    string
	staleAfter time.Duration
	nowFunc    func() time.Time
	pidAlive   func(int) bool
}

func NewWorkerLockScanner(advRoot string, staleAfter time.Duration) *WorkerLockScanner {
	if staleAfter == 0 {
		staleAfter = workerHeartbeatStaleAfterFromEnv()
	}
	return &WorkerLockScanner{
		advRoot:    advRoot,
		staleAfter: staleAfter,
		nowFunc:    time.Now,
		pidAlive:   pidAlive,
	}
}

func (s *WorkerLockScanner) Scan(ctx context.Context) ([]WorkerLockFinding, error) {
	if s.advRoot == "" {
		return []WorkerLockFinding{}, nil
	}
	entries, err := os.ReadDir(s.advRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []WorkerLockFinding{}, nil
		}
		return nil, err
	}

	var findings []WorkerLockFinding
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return findings, ctx.Err()
		default:
		}
		if !entry.IsDir() {
			continue
		}
		projectID := entry.Name()
		lockPath := filepath.Join(s.advRoot, projectID, workerLockFilename)
		finding, ok := s.scanLock(projectID, lockPath)
		if ok {
			findings = append(findings, finding)
		}
	}
	return findings, nil
}

type workerLockContents struct {
	SchemaVersion int    `json:"schema_version,omitempty"`
	PID           int    `json:"pid"`
	WorkerID      string `json:"worker_id"`
	AcquiredAt    string `json:"acquired_at"`
	LastHeartbeat string `json:"last_heartbeat,omitempty"`
}

func (s *WorkerLockScanner) scanLock(projectID, lockPath string) (WorkerLockFinding, bool) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return WorkerLockFinding{}, false
	}

	var contents workerLockContents
	if err := json.Unmarshal(data, &contents); err != nil {
		return WorkerLockFinding{
			ProjectID: projectID,
			Path:      lockPath,
			Status:    StatusWarn,
			Message:   fmt.Sprintf("cannot parse ADV worker lock for project %s", projectID),
			Hint:      workerRestartRemediation,
		}, true
	}

	// V1 lock fallback: no last_heartbeat means older Advance build. Do not warn.
	if contents.LastHeartbeat == "" {
		return WorkerLockFinding{}, false
	}
	heartbeat, err := time.Parse(time.RFC3339, contents.LastHeartbeat)
	if err != nil {
		return WorkerLockFinding{
			ProjectID: projectID,
			Path:      lockPath,
			PID:       contents.PID,
			WorkerID:  contents.WorkerID,
			Status:    StatusWarn,
			Message:   fmt.Sprintf("cannot parse ADV worker lock heartbeat for project %s", projectID),
			Hint:      workerRestartRemediation,
		}, true
	}

	now := s.nowFunc()
	age := now.Sub(heartbeat)
	if age < s.staleAfter {
		return WorkerLockFinding{}, false
	}

	message := fmt.Sprintf("ADV worker lock has stale heartbeat for project %s: age %s exceeds %s", projectID, age.Round(time.Second), s.staleAfter.Round(time.Second))
	if !s.pidAlive(contents.PID) {
		message = fmt.Sprintf("ADV worker lock has stale heartbeat and dead pid for project %s: pid %d", projectID, contents.PID)
	}
	return WorkerLockFinding{
		ProjectID: projectID,
		Path:      lockPath,
		PID:       contents.PID,
		WorkerID:  contents.WorkerID,
		Status:    StatusWarn,
		Age:       age.Round(time.Second),
		Message:   message,
		Hint:      workerRestartRemediation,
	}, true
}

func workerHeartbeatStaleAfterFromEnv() time.Duration {
	raw := os.Getenv(workerHeartbeatStaleEnv)
	if raw == "" {
		return defaultWorkerStaleAfter
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return defaultWorkerStaleAfter
	}
	return time.Duration(ms) * time.Millisecond
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
