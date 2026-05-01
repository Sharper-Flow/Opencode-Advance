package tests

import (
	"testing"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// TestSessionSection_Typed verifies that [session] is parsed into the typed
// SessionSection field rather than DeferredSections, following the Phase 3
// graduation pattern.
func TestSessionSection_Typed(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[session]
prefix = "oca-"

[session.watchdog]
enabled = true
idle_timeout = "3m"
max_bumps = 5
`
	stack, err := cfg.Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Must be on typed field, not deferred.
	if stack.Session == nil {
		t.Fatal("Session section not on typed field (nil)")
	}
	if _, ok := stack.DeferredSections["session"]; ok {
		t.Fatal("session should not be in DeferredSections")
	}

	// Watchdog field checks.
	if stack.Session.Watchdog == nil {
		t.Fatal("Watchdog config nil")
	}
	if !stack.Session.Watchdog.Enabled {
		t.Error("Enabled = false, want true")
	}
	if stack.Session.Watchdog.IdleTimeout != 3*time.Minute {
		t.Errorf("IdleTimeout = %v, want 3m", stack.Session.Watchdog.IdleTimeout)
	}
	if stack.Session.Watchdog.MaxBumps != 5 {
		t.Errorf("MaxBumps = %d, want 5", stack.Session.Watchdog.MaxBumps)
	}
}

// loadFromBytes is a test helper that runs the full Load pipeline
// (Parse → Resolve → Validate) on raw TOML bytes.
func loadFromBytes(data []byte) (*cfg.Stack, error) {
	stack, err := cfg.Parse(data)
	if err != nil {
		return nil, err
	}
	stack.Resolve()
	if err := stack.Validate(); err != nil {
		return nil, err
	}
	return stack, nil
}

// TestSessionSection_Defaults verifies defaults are applied when watchdog
// section is absent or partial.
func TestSessionSection_Defaults(t *testing.T) {
	t.Run("no_watchdog_section", func(t *testing.T) {
		toml := `
[meta]
version = "1.0.0"

[session]
prefix = "oca-"
`
		stack, err := loadFromBytes([]byte(toml))
		if err != nil {
			t.Fatalf("loadFromBytes failed: %v", err)
		}
		if stack.Session == nil {
			t.Fatal("Session section nil")
		}
		// Watchdog should have zero-value defaults (enabled=false).
		if stack.Session.Watchdog != nil && stack.Session.Watchdog.Enabled {
			t.Error("Watchdog should default to disabled")
		}
	})

	t.Run("watchdog_with_only_enabled", func(t *testing.T) {
		toml := `
[meta]
version = "1.0.0"

[session.watchdog]
enabled = true
`
		stack, err := loadFromBytes([]byte(toml))
		if err != nil {
			t.Fatalf("loadFromBytes failed: %v", err)
		}
		wd := stack.Session.Watchdog
		if wd == nil {
			t.Fatal("Watchdog nil")
		}
		if !wd.Enabled {
			t.Error("Enabled = false, want true")
		}
		// Defaults: idle_timeout=5m, max_bumps=3
		if wd.IdleTimeout != 5*time.Minute {
			t.Errorf("IdleTimeout default = %v, want 5m", wd.IdleTimeout)
		}
		if wd.MaxBumps != 3 {
			t.Errorf("MaxBumps default = %d, want 3", wd.MaxBumps)
		}
	})
}

// TestSessionSection_Validation verifies validation errors for invalid
// watchdog config values.
func TestSessionSection_Validation(t *testing.T) {
	t.Run("negative_max_bumps", func(t *testing.T) {
		toml := `
[meta]
version = "1.0.0"

[session.watchdog]
enabled = true
max_bumps = -1
`
		_, err := loadFromBytes([]byte(toml))
		if err == nil {
			t.Fatal("expected validation error for negative max_bumps")
		}
	})

	t.Run("invalid_idle_timeout", func(t *testing.T) {
		toml := `
[meta]
version = "1.0.0"

[session.watchdog]
enabled = true
idle_timeout = "not-a-duration"
`
		_, err := cfg.Parse([]byte(toml))
		if err == nil {
			t.Fatal("expected parse error for invalid idle_timeout")
		}
	})

	t.Run("zero_idle_timeout_gets_default", func(t *testing.T) {
		toml := `
[meta]
version = "1.0.0"

[session.watchdog]
enabled = true
idle_timeout = "0s"
`
		stack, err := loadFromBytes([]byte(toml))
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		// Zero idle_timeout is treated as "use default" (5m).
		if stack.Session.Watchdog.IdleTimeout != 5*time.Minute {
			t.Errorf("IdleTimeout = %v, want 5m (default)", stack.Session.Watchdog.IdleTimeout)
		}
	})
}

// TestSessionSection_AbsentSection verifies that [session] is optional and
// results in nil Session with no errors.
func TestSessionSection_AbsentSection(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[mcp.servers.stub]
port = 6276
command = "echo"
`
	stack, err := loadFromBytes([]byte(toml))
	if err != nil {
		t.Fatalf("loadFromBytes failed: %v", err)
	}
	if stack.Session != nil {
		t.Error("Session should be nil when [session] absent")
	}
	if _, ok := stack.DeferredSections["session"]; ok {
		t.Error("session should not be in DeferredSections when absent")
	}
}
