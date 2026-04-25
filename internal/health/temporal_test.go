package health

import (
	"context"
	"net"
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

func TestCheckTemporal_Disabled(t *testing.T) {
	f := false
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{Enabled: &f}}
	checks, err := CheckTemporal(context.Background(), stack, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d: %#v", len(checks), checks)
	}
	if checks[0].Name != "temporal.enabled" || checks[0].Status != StatusPass {
		t.Fatalf("expected pass for disabled, got %#v", checks[0])
	}
}

func TestCheckTemporal_NilTemporal(t *testing.T) {
	stack := &cfg.Stack{}
	checks, err := CheckTemporal(context.Background(), stack, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d: %#v", len(checks), checks)
	}
	if checks[0].Name != "temporal.enabled" || checks[0].Status != StatusPass {
		t.Fatalf("expected pass for nil temporal, got %#v", checks[0])
	}
}

func TestCheckTemporal_Unreachable(t *testing.T) {
	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{
		Enabled:   boolPtr(true),
		Address:   "127.0.0.1:1",
		Namespace: "default",
	}}
	checks, err := CheckTemporal(context.Background(), stack, Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertCheckStatus(t, checks, "temporal.reachable", StatusWarn)
	// Namespace check should not run when unreachable
	for _, c := range checks {
		if c.Name == "temporal.namespace" {
			t.Fatal("expected no namespace check when unreachable")
		}
	}
}

func TestCheckTemporal_ReachableNamespaceExists(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	origRun := runSubprocess
	runSubprocess = func(ctx context.Context, c subprocess.Cmd) (subprocess.Result, error) {
		return subprocess.Result{ExitClass: subprocess.ExitSuccess, Output: []byte("Namespace: default")}, nil
	}
	defer func() { runSubprocess = origRun }()

	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{
		Enabled:   boolPtr(true),
		Address:   addr,
		Namespace: "default",
	}}
	checks, err := CheckTemporal(context.Background(), stack, Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertCheckStatus(t, checks, "temporal.reachable", StatusPass)
	assertCheckStatus(t, checks, "temporal.namespace", StatusPass)
}

func TestCheckTemporal_ReachableNamespaceNotFound(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	origRun := runSubprocess
	runSubprocess = func(ctx context.Context, c subprocess.Cmd) (subprocess.Result, error) {
		return subprocess.Result{
			ExitClass: subprocess.ExitNonZero,
			ExitCode:  1,
			Output:    []byte("Namespace default not found"),
		}, nil
	}
	defer func() { runSubprocess = origRun }()

	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{
		Enabled:   boolPtr(true),
		Address:   addr,
		Namespace: "default",
	}}
	checks, err := CheckTemporal(context.Background(), stack, Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertCheckStatus(t, checks, "temporal.reachable", StatusPass)
	assertCheckStatus(t, checks, "temporal.namespace", StatusWarn)
	for _, c := range checks {
		if c.Name == "temporal.namespace" {
			if c.Hint == "" {
				t.Fatal("expected hint for namespace not found")
			}
			if !strings.Contains(c.Hint, "temporal operator namespace create") {
				t.Fatalf("expected create hint, got %q", c.Hint)
			}
		}
	}
}

func TestCheckTemporal_ReachableTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	origRun := runSubprocess
	runSubprocess = func(ctx context.Context, c subprocess.Cmd) (subprocess.Result, error) {
		return subprocess.Result{
			ExitClass: subprocess.ExitTimeout,
			Output:    []byte(""),
		}, nil
	}
	defer func() { runSubprocess = origRun }()

	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{
		Enabled:   boolPtr(true),
		Address:   addr,
		Namespace: "default",
	}}
	checks, err := CheckTemporal(context.Background(), stack, Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertCheckStatus(t, checks, "temporal.reachable", StatusPass)
	assertCheckStatus(t, checks, "temporal.namespace", StatusWarn)
	for _, c := range checks {
		if c.Name == "temporal.namespace" {
			if !strings.Contains(c.Message, "timed out") {
				t.Fatalf("expected timeout message, got %q", c.Message)
			}
		}
	}
}

func TestCheckTemporal_ReachableOtherError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	origRun := runSubprocess
	runSubprocess = func(ctx context.Context, c subprocess.Cmd) (subprocess.Result, error) {
		return subprocess.Result{
			ExitClass: subprocess.ExitNonZero,
			ExitCode:  2,
			Output:    []byte("some other error"),
		}, nil
	}
	defer func() { runSubprocess = origRun }()

	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{
		Enabled:   boolPtr(true),
		Address:   addr,
		Namespace: "default",
	}}
	checks, err := CheckTemporal(context.Background(), stack, Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertCheckStatus(t, checks, "temporal.reachable", StatusPass)
	assertCheckStatus(t, checks, "temporal.namespace", StatusWarn)
	for _, c := range checks {
		if c.Name == "temporal.namespace" {
			if !strings.Contains(c.Message, "some other error") {
				t.Fatalf("expected error message, got %q", c.Message)
			}
		}
	}
}

func TestCheckTemporal_RedactsStderr(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	origRun := runSubprocess
	runSubprocess = func(ctx context.Context, c subprocess.Cmd) (subprocess.Result, error) {
		return subprocess.Result{
			ExitClass: subprocess.ExitNonZero,
			ExitCode:  1,
			Output:    []byte("error: API_KEY=secret123"),
		}, nil
	}
	defer func() { runSubprocess = origRun }()

	stack := &cfg.Stack{Temporal: &cfg.TemporalSection{
		Enabled:   boolPtr(true),
		Address:   addr,
		Namespace: "default",
	}}
	checks, err := CheckTemporal(context.Background(), stack, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range checks {
		if c.Name == "temporal.namespace" {
			if strings.Contains(c.Message, "secret123") {
				t.Fatalf("expected secret to be redacted, got %q", c.Message)
			}
			if !strings.Contains(c.Message, "REDACTED") {
				t.Fatalf("expected REDACTED marker, got %q", c.Message)
			}
		}
	}
}

func TestCheckTemporal_BuiltinRegistryPresentAfterReset(t *testing.T) {
	ResetForTesting()
	t.Cleanup(ResetForTesting)
	if !containsScope(KnownScopes(), "temporal") {
		t.Fatalf("expected builtin temporal scope, got %#v", KnownScopes())
	}
}

func boolPtr(b bool) *bool {
	return &b
}
