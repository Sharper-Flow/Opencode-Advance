package dashboard_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/dashboard"
)

func TestState_VersionMonotonic(t *testing.T) {
	s := dashboard.NewState()
	var maxVersion uint64

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Update(func(snap *dashboard.Snapshot) {
				snap.Changes = append(snap.Changes, dashboard.ChangeRow{ID: "test"})
			})
			v := s.Version()
			for {
				old := maxVersion
				if v <= old || atomic.CompareAndSwapUint64(&maxVersion, old, v) {
					break
				}
			}
		}()
	}
	wg.Wait()

	if maxVersion != 100 {
		t.Errorf("expected 100 versions, got %d", maxVersion)
	}
	if s.Version() != 100 {
		t.Errorf("final version = %d, want 100", s.Version())
	}
}

func TestState_ConcurrentReadWrite(t *testing.T) {
	s := dashboard.NewState()

	var wg sync.WaitGroup
	// Writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Update(func(snap *dashboard.Snapshot) {
				snap.Changes = append(snap.Changes, dashboard.ChangeRow{ID: "change-" + string(rune('A'+i%26))})
			})
		}(i)
	}
	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snap := s.Snapshot()
			_ = snap.Changes // read should not panic
		}()
	}
	wg.Wait()
}

func TestState_WarmFlag(t *testing.T) {
	s := dashboard.NewState()

	if snap := s.Snapshot(); snap.Warm {
		t.Error("new state should not be warm")
	}

	s.SetWarm(true)
	if snap := s.Snapshot(); !snap.Warm {
		t.Error("state should be warm after SetWarm(true)")
	}

	s.SetWarm(false)
	if snap := s.Snapshot(); snap.Warm {
		t.Error("state should not be warm after SetWarm(false)")
	}
}

type mockPoller struct {
	calls atomic.Int64
}

func (m *mockPoller) Poll(ctx context.Context, s *dashboard.State) error {
	m.calls.Add(1)
	return nil
}

func TestStartPollers(t *testing.T) {
	s := dashboard.NewState()
	mp := &mockPoller{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		dashboard.RunPollers(ctx, s, []dashboard.Poller{mp}, 10*time.Millisecond)
	}()

	<-ctx.Done()
	<-done

	calls := mp.calls.Load()
	if calls < 3 { // should have polled at least 3 times in 100ms with 10ms interval
		t.Errorf("expected at least 3 poll calls, got %d", calls)
	}
}
