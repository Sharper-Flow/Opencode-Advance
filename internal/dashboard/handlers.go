package dashboard

import (
	"io/fs"
	"log/slog"
	"net/http"
	"sync"
)

// subscriber represents a connected SSE client.
type subscriber struct {
	ch chan Snapshot
}

// subscriberRegistry manages SSE subscribers.
type subscriberRegistry struct {
	mu          sync.RWMutex
	subscribers map[*subscriber]struct{}
	logger      *slog.Logger
}

func newSubscriberRegistry() *subscriberRegistry {
	return &subscriberRegistry{
		subscribers: make(map[*subscriber]struct{}),
		logger:      slog.Default(),
	}
}

// Subscribe adds a new subscriber and returns its channel.
func (r *subscriberRegistry) Subscribe() *subscriber {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub := &subscriber{ch: make(chan Snapshot, 8)}
	r.subscribers[sub] = struct{}{}
	return sub
}

// Unsubscribe removes a subscriber.
func (r *subscriberRegistry) Unsubscribe(sub *subscriber) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.subscribers, sub)
	close(sub.ch)
}

// Notify sends the current snapshot to all subscribers.
func (r *subscriberRegistry) Notify(snap Snapshot) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for sub := range r.subscribers {
		select {
		case sub.ch <- snap:
		default:
			// Drop if subscriber is slow
			r.logger.Warn("dropping SSE event for slow subscriber")
		}
	}
}

// jsonChangesHandler returns JSON for /api/changes.
func jsonChangesHandler(s *State) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap := s.Snapshot()
		writeJSON(w, snap.Changes)
	})
}

// jsonSessionsHandler returns JSON for /api/sessions.
func jsonSessionsHandler(s *State) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap := s.Snapshot()
		writeJSON(w, snap.Sessions)
	})
}

// jsonHealthHandler returns JSON for /api/health.
func jsonHealthHandler(s *State) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap := s.Snapshot()
		writeJSON(w, snap.Health)
	})
}

// jsonFSHandler returns JSON listing of the embedded FS (for /api/fs).
func jsonFSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entries, err := fs.ReadDir(staticFS, "frontend/dist")
		if err != nil {
			http.Error(w, "fs error", http.StatusInternalServerError)
			return
		}
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		writeJSON(w, names)
	})
}
