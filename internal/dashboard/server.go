package dashboard

import (
	"context"
	"encoding/json"
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
	cfg    Config
	logger *slog.Logger
	mux    *http.ServeMux
	state  *State
}

// NewServer creates a new dashboard server with all routes registered.
func NewServer(cfg Config) (*Server, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		cfg:    cfg,
		logger: logger,
		mux:    http.NewServeMux(),
		state:  NewState(),
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
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/changes", s.handleChanges)
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/events", s.handleEvents)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>OCA Dashboard</h1><p>Loading...</p>"))
}

func (s *Server) handleChanges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data := s.state.Snapshot()
	json.NewEncoder(w).Encode(data.Changes)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data := s.state.Snapshot()
	json.NewEncoder(w).Encode(data.Sessions)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data := s.state.Snapshot()
	json.NewEncoder(w).Encode(data.Health)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Write([]byte(": ping\n\n"))
	w.(http.Flusher).Flush()
}
