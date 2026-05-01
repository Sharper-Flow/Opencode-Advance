package dashboard

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Config holds dashboard server configuration.
type Config struct {
	Listener net.Listener
	Logger   *slog.Logger
}

// Server is the dashboard HTTP server.
type Server struct {
	cfg      Config
	logger   *slog.Logger
	mux      *http.ServeMux
	state    *State
	registry *subscriberRegistry
}

// NewServer creates a new dashboard server with all routes registered.
func NewServer(cfg Config) (*Server, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		cfg:      cfg,
		logger:   logger,
		mux:      http.NewServeMux(),
		state:    NewState(),
		registry: newSubscriberRegistry(),
	}

	s.registerRoutes()
	return s, nil
}

// Handler returns the HTTP handler for the server (for testing with httptest).
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Run starts the dashboard server and blocks until ctx is cancelled.
// Shutdown sequence: ctx cancelled (SSE handlers exit) → http.Server.Shutdown.
func (s *Server) Run(ctx context.Context) error {
	// If no listener, just block until cancelled (for testing).
	if s.cfg.Listener == nil {
		<-ctx.Done()
		s.logger.Info("dashboard shutting down")
		return nil
	}

	hsrv := &http.Server{Handler: s.mux}

	// Start serving in background
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- hsrv.Serve(s.cfg.Listener)
	}()

	// Wait for context cancellation or serve error
	select {
	case <-ctx.Done():
		s.logger.Info("dashboard shutting down")
	case err := <-serveErr:
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	}

	// Shutdown: ctx already cancelled → SSE handlers exited → safe to Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return hsrv.Shutdown(shutdownCtx)
}

const shutdownTimeout = 5 * time.Second

func (s *Server) registerRoutes() {
	s.mux.Handle("GET /", indexHandler(s.state))
	s.mux.Handle("GET /api/changes", jsonChangesHandler(s.state))
	s.mux.Handle("GET /api/sessions", jsonSessionsHandler(s.state))
	s.mux.Handle("GET /api/health", jsonHealthHandler(s.state))
	s.mux.Handle("GET /api/events", sseHandler(s.state, s.registry))
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler()))
}
