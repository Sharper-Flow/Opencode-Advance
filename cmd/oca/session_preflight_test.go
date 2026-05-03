package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRunSessionPreflight_NoTemporalConfig(t *testing.T) {
	var buf bytes.Buffer
	stack := &cfg.Stack{} // No Temporal section
	ctx := context.Background()

	runSessionPreflight(ctx, &buf, stack)

	// Should produce no output when Temporal is not configured
	if buf.Len() != 0 {
		t.Fatalf("expected no output without Temporal config, got: %q", buf.String())
	}
}

func TestRunSessionPreflight_WithTemporalConfig(t *testing.T) {
	var buf bytes.Buffer
	stack := &cfg.Stack{
		Temporal: &cfg.TemporalSection{
			Address:   "127.0.0.1:7233",
			Namespace: "default",
		},
	}
	ctx := context.Background()

	runSessionPreflight(ctx, &buf, stack)

	// Temporal is configured but likely unreachable in test — should print
	// a warning about Temporal being unreachable, not panic.
	output := buf.String()
	if output == "" {
		// If Temporal happens to be running, no output is fine too.
		return
	}
	if !strings.Contains(output, "warning:") {
		t.Fatalf("expected warning prefix, got: %q", output)
	}
}
