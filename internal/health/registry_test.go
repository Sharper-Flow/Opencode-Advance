package health

import (
	"context"
	"errors"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRegistry_RegisterRunKnownScopesAndReset(t *testing.T) {
	ResetForTesting()
	t.Cleanup(ResetForTesting)

	called := false
	Register("fake", func(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
		called = true
		return []Check{{Name: "fake.ok", Status: StatusPass, Message: "ok"}}, nil
	})

	scopes := KnownScopes()
	if !containsScope(scopes, "fake") {
		t.Fatalf("fake scope missing from %#v", scopes)
	}

	checks, err := Run("fake", context.Background(), &cfg.Stack{}, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Fatal("registered check not invoked")
	}
	if len(checks) != 1 || checks[0].Name != "fake.ok" {
		t.Fatalf("unexpected checks: %#v", checks)
	}

	ResetForTesting()
	if containsScope(KnownScopes(), "fake") {
		t.Fatalf("reset should clear non-builtin registrations: %#v", KnownScopes())
	}
	if !containsScope(KnownScopes(), "mcp") {
		t.Fatalf("reset should preserve builtin mcp registration: %#v", KnownScopes())
	}
}

func TestRegistry_RunUnknownScope(t *testing.T) {
	ResetForTesting()
	t.Cleanup(ResetForTesting)

	_, err := Run("missing", context.Background(), &cfg.Stack{}, Options{})
	if !errors.Is(err, ErrUnknownScope) {
		t.Fatalf("err=%v want ErrUnknownScope", err)
	}
}

func TestRegistry_ResetReplaysBuiltinRegistrations(t *testing.T) {
	ResetForTesting()
	t.Cleanup(ResetForTesting)

	if !containsScope(KnownScopes(), "mcp") {
		t.Fatalf("expected builtin mcp scope after reset, got %#v", KnownScopes())
	}

	Register("temp", func(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
		return nil, nil
	})
	if !containsScope(KnownScopes(), "temp") {
		t.Fatalf("expected temp scope before reset, got %#v", KnownScopes())
	}

	ResetForTesting()
	if containsScope(KnownScopes(), "temp") {
		t.Fatalf("temp scope should be cleared by reset, got %#v", KnownScopes())
	}
	if !containsScope(KnownScopes(), "mcp") {
		t.Fatalf("builtin mcp scope lost after reset, got %#v", KnownScopes())
	}
}

func containsScope(scopes []string, target string) bool {
	for _, scope := range scopes {
		if scope == target {
			return true
		}
	}
	return false
}
