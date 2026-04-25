package health

import (
	"context"
	"errors"
	"sort"
	"sync"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

type CheckFn func(context.Context, *cfg.Stack, Options) ([]Check, error)

var ErrUnknownScope = errors.New("unknown health scope")

var (
	registryMu           sync.RWMutex
	registry             = map[string]CheckFn{}
	builtinRegistrations []builtinRegistration
)

type builtinRegistration struct {
	scope string
	fn    CheckFn
}

func Register(scope string, fn CheckFn) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[scope] = fn
}

func registerBuiltin(scope string, fn CheckFn) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[scope] = fn
	builtinRegistrations = append(builtinRegistrations, builtinRegistration{scope: scope, fn: fn})
}

func Run(scope string, ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	registryMu.RLock()
	fn, ok := registry[scope]
	registryMu.RUnlock()
	if !ok {
		return nil, ErrUnknownScope
	}
	return fn(ctx, stack, opts)
}

func KnownScopes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	scopes := make([]string, 0, len(registry))
	for scope := range registry {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scopes
}

func ResetForTesting() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]CheckFn{}
	for _, builtin := range builtinRegistrations {
		registry[builtin.scope] = builtin.fn
	}
}
