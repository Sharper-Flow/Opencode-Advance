package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
)

func TestSessionDoctorCmd_Structure(t *testing.T) {
	state := &commandState{output: "text"}
	cmd := newSessionDoctorCmd(state)

	if cmd.Name() != "doctor" {
		t.Fatalf("cmd name = %q, want doctor", cmd.Name())
	}

	flags := cmd.Flags()
	if f := flags.Lookup("db"); f == nil {
		t.Fatal("missing --db flag")
	}
	if f := flags.Lookup("threshold"); f == nil {
		t.Fatal("missing --threshold flag")
	}
	if f := flags.Lookup("apply"); f == nil {
		t.Fatal("missing --apply flag")
	}
	if f := flags.Lookup("backup-dir"); f == nil {
		t.Fatal("missing --backup-dir flag")
	}

	// Default threshold should be 5 minutes
	if f := flags.Lookup("threshold"); f.DefValue != "5m0s" {
		t.Fatalf("threshold default = %q, want 5m0s", f.DefValue)
	}
}

func TestSessionDoctorJSONOutput(t *testing.T) {
	finding := advruntime.SessionDebtFinding{
		Path:       "/path/to/db",
		Status:     advruntime.StatusWarn,
		Repairable: 3,
		Ignored:    1,
		OldestAge:  10 * time.Minute,
		Message:    "3 repairable rows",
		Hint:       "run with --apply",
	}

	// Verify JSON marshaling works for the finding type
	b, err := json.Marshal(finding)
	if err != nil {
		t.Fatalf("marshal finding: %v", err)
	}
	var parsed advruntime.SessionDebtFinding
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal finding: %v", err)
	}
	if parsed.Repairable != 3 {
		t.Fatalf("repairable = %d, want 3", parsed.Repairable)
	}
	if parsed.Status != advruntime.StatusWarn {
		t.Fatalf("status = %q, want warn", parsed.Status)
	}
}

func TestSessionDoctorDefaultDBPath(t *testing.T) {
	// Verify the default DB path function produces a reasonable path
	path := defaultOpenCodeDBPath()
	if path == "" {
		t.Fatal("defaultOpenCodeDBPath returned empty")
	}
	if filepath.Base(path) != "opencode.db" {
		t.Fatalf("base = %q, want opencode.db", filepath.Base(path))
	}
}
