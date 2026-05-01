package dashboard

import (
	"context"
	"log/slog"

	"github.com/Sharper-Flow/Opencode-Advance/internal/session"
)

// sessionLister is the narrow interface for listing tmux sessions.
type sessionLister interface {
	List() ([]session.Session, error)
}

// SessionPoller implements Poller by fetching tmux sessions.
type SessionPoller struct {
	lister sessionLister
	logger *slog.Logger
}

// NewSessionPoller creates a new session poller.
func NewSessionPoller(lister sessionLister, logger *slog.Logger) *SessionPoller {
	if logger == nil {
		logger = slog.Default()
	}
	return &SessionPoller{lister: lister, logger: logger}
}

// Poll fetches tmux sessions and updates the state.
func (p *SessionPoller) Poll(ctx context.Context, s *State) error {
	if p.lister == nil {
		s.Update(func(snap *Snapshot) {
			snap.Sessions = []SessionRow{}
		})
		return nil
	}

	logger := p.logger
	if logger == nil {
		logger = slog.Default()
	}

	sessions, err := p.lister.List()
	if err != nil {
		logger.Warn("session list failed", "error", err)
		s.Update(func(snap *Snapshot) {
			snap.Sessions = []SessionRow{}
		})
		return nil
	}

	rows := make([]SessionRow, len(sessions))
	for i, sess := range sessions {
		rows[i] = SessionRow{
			Name:     sess.Name,
			Attached: sess.Attached,
			Path:     sess.Path,
		}
	}

	s.Update(func(snap *Snapshot) {
		snap.Sessions = rows
	})
	return nil
}
