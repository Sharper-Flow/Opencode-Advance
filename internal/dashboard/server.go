package dashboard

import (
	"context"
	"log/slog"
	"net"
)

// Config holds dashboard server configuration.
type Config struct {
	Listener net.Listener
	Logger   *slog.Logger
}

// Server is the dashboard HTTP server.
type Server struct {
	cfg    Config
	logger *slog.Logger
}

// NewServer creates a new dashboard server.
func NewServer(cfg Config) (*Server, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{cfg: cfg, logger: logger}, nil
}

// Run starts the dashboard server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	<-ctx.Done()
	s.logger.Info("dashboard shutting down")
	return nil
}
