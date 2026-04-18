package config

import (
	"fmt"
	"os"
	"sort"

	"github.com/BurntSushi/toml"
)

// knownSections lists top-level stack.toml sections Phase 1 recognizes.
// "meta" and "mcp" are parsed into typed fields on Stack. The remaining
// entries are known-but-unimplemented (deferred to later phases):
// parsed into Stack.DeferredSections as raw maps so round-trip works,
// but not rendered by any Phase 1 target.
//
// Truly unknown top-level keys (typos, not in this set) produce a
// validation error in Validate().
var knownSections = map[string]bool{
	// Phase 1 typed sections
	"meta": true,
	"mcp":  true,
	// Phase 2+ — parsed as deferred raw maps
	"plugins":      true,
	"instructions": true,
	"providers":    true,
	"agents":       true,
	"permissions":  true,
	"watcher":      true,
	"lsp":          true,
	"session":      true,
	"discord":      true,
	"skills":       true,
	"formatters":   true,
	"commands":     true,
	"opencode":     true,
}

// ParseError wraps a TOML decode error with the source path for better
// user-facing diagnostics. Exposed so callers can distinguish it from
// Validate-time errors.
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("parse %s: %v", e.Path, e.Err)
	}
	return fmt.Sprintf("parse: %v", e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// ParseFile reads and parses a stack.toml file from disk. Returns a
// *ParseError wrapping any IO or TOML decode failure.
func ParseFile(path string) (*Stack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ParseError{Path: path, Err: err}
	}
	stack, perr := Parse(data)
	if perr != nil {
		if pe, ok := perr.(*ParseError); ok {
			pe.Path = path
		}
		return nil, perr
	}
	return stack, nil
}

// Parse decodes TOML bytes into a Stack. Unknown top-level keys (not in
// knownSections) are collected into DeferredSections with their raw map
// representation; the caller's Validate run decides which ones are
// truly-unknown (error) vs known-but-deferred (ok).
//
// This deliberately decodes permissively: BurntSushi/toml's MetaData
// gives us the key set so we can compare against knownSections, but we
// don't use DisallowUnknownFields at the root — Vision may add fields
// to [mcp.servers.*] that OCA wants to pass through.
func Parse(data []byte) (*Stack, error) {
	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, &ParseError{Err: err}
	}

	stack := &Stack{
		DeferredSections: map[string]any{},
	}

	// Meta and MCP use typed decode; everything else is cached raw.
	for k, v := range raw {
		switch k {
		case "meta":
			if err := decodeInto(v, &stack.Meta); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[meta]: %w", err)}
			}
		case "mcp":
			if err := decodeInto(v, &stack.MCP); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[mcp]: %w", err)}
			}
		default:
			// Known-but-unimplemented OR truly unknown — defer the
			// classification to Validate so all errors can be aggregated
			// with field paths.
			stack.DeferredSections[k] = v
		}
	}

	if stack.MCP.Servers == nil {
		stack.MCP.Servers = map[string]Server{}
	}
	return stack, nil
}

// decodeInto round-trips a raw TOML value through TOML encoding back
// into a typed Go struct. This is the cleanest way to re-use BurntSushi's
// tag-driven decode after an initial untyped decode.
func decodeInto(raw any, dst any) error {
	tomlBytes, err := encodeTOML(raw)
	if err != nil {
		return err
	}
	_, err = toml.Decode(tomlBytes, dst)
	return err
}

// encodeTOML marshals a raw map back to TOML. BurntSushi's encoder
// requires a struct or map[string]any at the top level; we wrap single
// values in a synthetic table key if needed.
func encodeTOML(v any) (string, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", fmt.Errorf("expected table, got %T", v)
	}
	// Stable key ordering via sorted iteration for deterministic output.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Use the BurntSushi encoder via a temporary struct-like wrapper.
	// Since we only need round-trip fidelity for typed decode, we can
	// leverage toml.Marshal's support for map[string]any.
	b, err := toml.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
