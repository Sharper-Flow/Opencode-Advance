package session

import (
	"testing"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// TestWatchdogEnvConstruction verifies that the watchdog environment
// variable values are correctly constructed from WatchdogConfig.
func TestWatchdogEnvConstruction(t *testing.T) {
	tests := []struct {
		name          string
		wd            cfg.WatchdogConfig
		wantEnabled   string
		wantTimeoutMs string
		wantMaxBumps  string
	}{
		{
			name:          "defaults",
			wd:            cfg.WatchdogConfig{Enabled: false, IdleTimeout: 5 * time.Minute, MaxBumps: 3},
			wantEnabled:   "0",
			wantTimeoutMs: "300000",
			wantMaxBumps:  "3",
		},
		{
			name:          "enabled_custom",
			wd:            cfg.WatchdogConfig{Enabled: true, IdleTimeout: 3 * time.Minute, MaxBumps: 5},
			wantEnabled:   "1",
			wantTimeoutMs: "180000",
			wantMaxBumps:  "5",
		},
		{
			name:          "disabled_nil_watchdog",
			wd:            cfg.WatchdogConfig{},
			wantEnabled:   "0",
			wantTimeoutMs: "0",
			wantMaxBumps:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := watchdogEnvVars(&tt.wd)
			if env["OCA_WATCHDOG_ENABLED"] != tt.wantEnabled {
				t.Errorf("ENABLED = %q, want %q", env["OCA_WATCHDOG_ENABLED"], tt.wantEnabled)
			}
			if env["OCA_WATCHDOG_IDLE_TIMEOUT_MS"] != tt.wantTimeoutMs {
				t.Errorf("IDLE_TIMEOUT_MS = %q, want %q", env["OCA_WATCHDOG_IDLE_TIMEOUT_MS"], tt.wantTimeoutMs)
			}
			if env["OCA_WATCHDOG_MAX_BUMPS"] != tt.wantMaxBumps {
				t.Errorf("MAX_BUMPS = %q, want %q", env["OCA_WATCHDOG_MAX_BUMPS"], tt.wantMaxBumps)
			}
		})
	}
}
