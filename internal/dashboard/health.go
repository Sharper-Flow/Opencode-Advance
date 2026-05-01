package dashboard

import (
	"context"
	"log/slog"
)

// healthChecker is the narrow interface for health checks.
type healthChecker interface {
	checkTemporal(ctx context.Context) (bool, error)
	checkWorker(ctx context.Context) bool
}

// HealthPoller implements Poller by checking Temporal and worker health.
type HealthPoller struct {
	checker healthChecker
	logger  *slog.Logger
}

// NewHealthPoller creates a new health poller.
func NewHealthPoller(checker healthChecker, logger *slog.Logger) *HealthPoller {
	if logger == nil {
		logger = slog.Default()
	}
	return &HealthPoller{checker: checker, logger: logger}
}

// Poll checks Temporal reachability and worker liveness.
func (p *HealthPoller) Poll(ctx context.Context, s *State) error {
	var health HealthRow

	logger := p.logger
	if logger == nil {
		logger = slog.Default()
	}

	if p.checker != nil {
		reachable, err := p.checker.checkTemporal(ctx)
		if err != nil {
			logger.Warn("temporal health check failed", "error", err)
		}
		health.TemporalReachable = reachable
		health.WorkerRunning = p.checker.checkWorker(ctx)
	}

	s.Update(func(snap *Snapshot) {
		snap.Health = health
	})
	return nil
}
