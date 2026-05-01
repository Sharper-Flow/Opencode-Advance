package dashboard

import "sync"

// ChangeRow represents a single ADV change in the dashboard.
type ChangeRow struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	ActiveGate   string `json:"active_gate"`
	TasksDone    int    `json:"tasks_done"`
	TasksTotal   int    `json:"tasks_total"`
	DoomLoop     bool   `json:"doom_loop"`
	Project      string `json:"project"`
	LastSeenAgo  string `json:"last_seen_ago,omitempty"`
}

// SessionRow represents a tmux session in the dashboard.
type SessionRow struct {
	Name     string `json:"name"`
	Attached bool   `json:"attached"`
	Path     string `json:"path"`
}

// HealthRow represents system health status.
type HealthRow struct {
	TemporalReachable bool   `json:"temporal_reachable"`
	WorkerRunning     bool   `json:"worker_running"`
	Uptime            string `json:"uptime,omitempty"`
}

// Snapshot is a point-in-time view of all dashboard data.
type Snapshot struct {
	Changes  []ChangeRow  `json:"changes"`
	Sessions []SessionRow `json:"sessions"`
	Health   HealthRow    `json:"health"`
	Warm     bool         `json:"warm"`
	Version  uint64       `json:"version"`
}

// State is the shared in-memory dashboard state cache.
type State struct {
	mu      sync.RWMutex
	snap    Snapshot
	version uint64
}

// NewState creates an empty dashboard state.
func NewState() *State {
	return &State{
		snap: Snapshot{
			Changes:  []ChangeRow{},
			Sessions: []SessionRow{},
		},
	}
}

// Snapshot returns a copy of the current state.
func (s *State) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

// Update atomically replaces the state snapshot and bumps the version.
func (s *State) Update(fn func(snap *Snapshot)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.snap)
	s.version++
	s.snap.Version = s.version
}

// Warm marks the state as fully populated.
func (s *State) SetWarm(warm bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.Warm = warm
}

// Version returns the current monotonic version counter.
func (s *State) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}
