package dashboard

import (
	"context"
	"errors"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/internal/session"
)

// mockSessionLister implements sessionLister for testing.
type mockSessionLister struct {
	sessions []session.Session
	err      error
}

func (m *mockSessionLister) List() ([]session.Session, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.sessions, nil
}

func TestSessionPoller_Success(t *testing.T) {
	s := NewState()
	mock := &mockSessionLister{
		sessions: []session.Session{
			{Name: "work", Attached: true, Path: "/home/user/project"},
			{Name: "scratch", Attached: false, Path: "/tmp"},
		},
	}

	poller := &SessionPoller{lister: mock}
	err := poller.Poll(context.Background(), s)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}

	snap := s.Snapshot()
	if len(snap.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(snap.Sessions))
	}
	if snap.Sessions[0].Name != "work" {
		t.Errorf("session[0].Name = %q, want %q", snap.Sessions[0].Name, "work")
	}
	if !snap.Sessions[0].Attached {
		t.Error("session[0].Attached = false, want true")
	}
}

func TestSessionPoller_DegradationOnError(t *testing.T) {
	s := NewState()
	mock := &mockSessionLister{err: errors.New("tmux not found")}

	poller := &SessionPoller{lister: mock}
	err := poller.Poll(context.Background(), s)
	if err != nil {
		t.Fatalf("Poll should not error on tmux failure: %v", err)
	}

	snap := s.Snapshot()
	if snap.Sessions == nil {
		t.Error("expected Sessions to be initialized even on error")
	}
}

func TestSessionPoller_NilLister(t *testing.T) {
	s := NewState()
	poller := &SessionPoller{lister: nil}

	err := poller.Poll(context.Background(), s)
	if err != nil {
		t.Fatalf("Poll with nil lister: %v", err)
	}

	snap := s.Snapshot()
	if len(snap.Sessions) != 0 {
		t.Errorf("expected 0 sessions with nil lister, got %d", len(snap.Sessions))
	}
}

// mockHealthChecker implements healthChecker for testing.
type mockHealthChecker struct {
	temporalReachable bool
	workerRunning     bool
	err               error
}

func (m *mockHealthChecker) checkTemporal(ctx context.Context) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.temporalReachable, nil
}

func (m *mockHealthChecker) checkWorker(ctx context.Context) bool {
	return m.workerRunning
}

func TestHealthPoller_AllHealthy(t *testing.T) {
	s := NewState()
	mock := &mockHealthChecker{temporalReachable: true, workerRunning: true}

	poller := &HealthPoller{checker: mock}
	err := poller.Poll(context.Background(), s)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}

	snap := s.Snapshot()
	if !snap.Health.TemporalReachable {
		t.Error("expected TemporalReachable = true")
	}
	if !snap.Health.WorkerRunning {
		t.Error("expected WorkerRunning = true")
	}
}

func TestHealthPoller_TemporalDown(t *testing.T) {
	s := NewState()
	mock := &mockHealthChecker{temporalReachable: false, workerRunning: false}

	poller := &HealthPoller{checker: mock}
	err := poller.Poll(context.Background(), s)
	if err != nil {
		t.Fatalf("Poll should not error when Temporal down: %v", err)
	}

	snap := s.Snapshot()
	if snap.Health.TemporalReachable {
		t.Error("expected TemporalReachable = false when Temporal down")
	}
}
