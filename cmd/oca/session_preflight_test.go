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

func TestRunSessionPreflight_NeverPanics(t *testing.T) {
	// runSessionPreflight must never panic or block, regardless of input.
	cases := []struct {
		name  string
		stack *cfg.Stack
	}{
		{"nil stack", nil},
		{"empty stack", &cfg.Stack{}},
		{"temporal unreachable", &cfg.Stack{Temporal: &cfg.TemporalSection{Address: "127.0.0.1:1", Namespace: "default"}}},
		{"temporal no namespace", &cfg.Stack{Temporal: &cfg.TemporalSection{Address: "127.0.0.1:7233"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			// Must not panic.
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("runSessionPreflight panicked: %v", r)
					}
				}()
				runSessionPreflight(context.Background(), &buf, tc.stack)
			}()
		})
	}
}
