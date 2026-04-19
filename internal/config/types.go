// Package config parses, validates, and resolves the OpenCode Advance
// stack.toml configuration file into typed Go structs.
//
// Phase 1 scope covers [meta] and [mcp.servers.*]. Known-but-unimplemented
// top-level sections (plugins, providers, agents, permissions, watcher,
// lsp, session, discord, skills, formatters, commands, opencode,
// instructions) parse into a generic DeferredSections map without error.
// Truly unknown top-level sections produce a validation error.
//
// See docs/design/phase1-mcp-apply.md for the full type design.
package config

// Stack is the in-memory representation of a validated stack.toml file.
// Fields are populated by Parse, with validation run by Validate and
// variable expansion by Resolve. Use Load to run the full pipeline.
type Stack struct {
	Meta Meta       `toml:"meta"`
	MCP  MCPSection `toml:"mcp"`

	// DeferredSections holds known-but-unimplemented top-level sections
	// verbatim so that a complete stack.toml (including future-phase
	// sections) round-trips through Phase 1 without errors. Entries are
	// not rendered by Phase 1 target writers.
	DeferredSections map[string]any `toml:"-"`

	// Warnings are non-fatal issues surfaced during Load (e.g. env_file
	// path does not exist). Apply and Doctor surface these to the user.
	Warnings []Warning `toml:"-"`
}

// Meta is the [meta] table.
type Meta struct {
	Version     string `toml:"version"`
	Name        string `toml:"name,omitempty"`
	Description string `toml:"description,omitempty"`
}

// MCPSection is the [mcp] table. It currently only carries Servers, but
// is a table for forward-compat with future MCP-wide settings.
type MCPSection struct {
	Servers map[string]Server `toml:"servers"`
}

// Server mirrors Vision's internal/config.ServerConfig field-for-field
// plus OCA-only additions (Required is now in Vision post-V3; Source
// and Description are in Vision post-V2; EnvFile and Type="daemon" are
// OCA-only and stripped at render time when appropriate).
//
// Unknown Vision-side fields under [mcp.servers.<name>] land in
// ExtraFields as raw map entries so OCA passes them through to
// servers.yaml without interpretation (forward-compat with Vision
// schema additions).
type Server struct {
	// Core transport (Vision)
	Port      int               `toml:"port" yaml:"port"`
	Type      string            `toml:"type,omitempty" yaml:"-"`                        // OCA "daemon"|"stdio"|"http"|"sse"
	Transport string            `toml:"transport,omitempty" yaml:"transport,omitempty"` // Vision passthrough alias
	Command   string            `toml:"command,omitempty" yaml:"command,omitempty"`
	Args      []string          `toml:"args,omitempty" yaml:"args,omitempty"`
	Env       map[string]string `toml:"env,omitempty" yaml:"env,omitempty"`
	URL       string            `toml:"url,omitempty" yaml:"url,omitempty"`
	Headers   map[string]string `toml:"headers,omitempty" yaml:"headers,omitempty"`

	// Lifecycle (Vision)
	Autostart           bool   `toml:"autostart,omitempty" yaml:"autostart,omitempty"`
	RestartPolicy       string `toml:"restart_policy,omitempty" yaml:"restart_policy,omitempty"`
	MaxRestarts         int    `toml:"max_restarts,omitempty" yaml:"max_restarts,omitempty"`
	Stateful            bool   `toml:"stateful,omitempty" yaml:"stateful,omitempty"`
	AvailabilityProfile string `toml:"availability_profile,omitempty" yaml:"availability_profile,omitempty"`
	SessionTimeout      string `toml:"session_timeout,omitempty" yaml:"session_timeout,omitempty"`
	MaxSessions         int    `toml:"max_sessions,omitempty" yaml:"max_sessions,omitempty"`
	SessionTTL          string `toml:"session_ttl,omitempty" yaml:"session_ttl,omitempty"`
	HealthCheckInterval string `toml:"health_check_interval,omitempty" yaml:"health_check_interval,omitempty"`
	RequestTimeout      string `toml:"request_timeout,omitempty" yaml:"request_timeout,omitempty"`
	Timeout             int    `toml:"timeout,omitempty" yaml:"-"` // OCA-side alias (ms) → Vision request_timeout

	// Resilience (Vision)
	Retry          *Retry          `toml:"retry,omitempty" yaml:"retry,omitempty"`
	CircuitBreaker *CircuitBreaker `toml:"circuit_breaker,omitempty" yaml:"circuit_breaker,omitempty"`

	// Sharing (Vision)
	SharedReadOnlyTools   []string `toml:"shared_read_only_tools,omitempty" yaml:"shared_read_only_tools,omitempty"`
	SharedResultCacheTTL  string   `toml:"shared_result_cache_ttl,omitempty" yaml:"shared_result_cache_ttl,omitempty"`
	SharedResultCacheSize int      `toml:"shared_result_cache_size,omitempty" yaml:"shared_result_cache_size,omitempty"`
	MaxInFlightRequests   int      `toml:"max_in_flight_requests,omitempty" yaml:"max_in_flight_requests,omitempty"`

	// OCA-respected / Vision accepts after V2+V3
	Required    bool   `toml:"required,omitempty" yaml:"required,omitempty"`
	Source      string `toml:"source,omitempty" yaml:"source,omitempty"`
	Description string `toml:"description,omitempty" yaml:"description,omitempty"`

	// Enabled is tri-state (pointer) so "enabled=false" is distinguishable
	// from "field absent (default true)" at render time.
	Enabled *bool `toml:"enabled,omitempty" yaml:"enabled,omitempty"`

	// OCA-side only — stripped at render time.
	EnvFile string `toml:"env_file,omitempty" yaml:"-"`
}

// Retry configures retryable-failure behavior (Vision passthrough).
type Retry struct {
	MaxAttempts     int      `toml:"max_attempts,omitempty" yaml:"max_attempts,omitempty"`
	InitialDelay    string   `toml:"initial_delay,omitempty" yaml:"initial_delay,omitempty"`
	MaxDelay        string   `toml:"max_delay,omitempty" yaml:"max_delay,omitempty"`
	RetryableErrors []string `toml:"retryable_errors,omitempty" yaml:"retryable_errors,omitempty"`
}

// CircuitBreaker configures fast-fail after repeated failures (Vision passthrough).
type CircuitBreaker struct {
	FailureThreshold int    `toml:"failure_threshold,omitempty" yaml:"failure_threshold,omitempty"`
	RecoveryTimeout  string `toml:"recovery_timeout,omitempty" yaml:"recovery_timeout,omitempty"`
}

// Warning carries a non-fatal diagnostic from Load. Surfaced by apply
// and doctor commands but does not abort operations.
type Warning struct {
	Path    string // e.g. "mcp.servers.kagi.env_file"
	Message string
	Hint    string // optional remediation suggestion
}

// IsEnabled returns the effective enabled state for a server. Absent
// (nil) is treated as true; explicit false disables.
func (s Server) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}
