package config

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// knownSections lists top-level stack.toml sections Phase 1+ recognizes.
// "meta" and "mcp" are parsed into typed fields on Stack. "plugins",
// "instructions", and "temporal" are Phase 2 typed. Remaining entries are
// known-but-unimplemented (deferred): parsed into Stack.DeferredSections
// as raw maps so round-trip works, but not rendered by Phase 1 targets.
//
// Truly unknown top-level keys (typos, not in this set) produce a
// validation error in Validate().
var knownSections = map[string]bool{
	// Phase 1 typed sections
	"meta": true,
	"mcp":  true,
	// Phase 2 typed sections
	"plugins":      true,
	"instructions": true,
	"temporal":     true, // advisory tolerance — parsed but validation handled by validateTemporal
	// Phase 2+ — parsed as deferred raw maps
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
	"shell":        true,
	"update_probe": true,
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

	// Typed sections: Meta, MCP, Plugins, Instructions, Temporal.
	// Everything else deferred for later-phase handling.
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
		case "plugins":
			if err := decodeInto(v, &stack.Plugins); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[plugins]: %w", err)}
			}
		case "instructions":
			if err := decodeInto(v, &stack.Instructions); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[instructions]: %w", err)}
			}
		case "temporal":
			ts := &TemporalSection{}
			if err := decodeInto(v, ts); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[temporal]: %w", err)}
			}
			stack.Temporal = ts

		// Phase 3 typed sections: lift from deferred, capturing Extra unknown fields.
		case "providers":
			// ProvidersSection is map[string]Provider — toml.Decode handles maps.
			// We still run decodeIntoWithExtra to capture per-Provider Extra fields.
			var rawMap map[string]any
			if rawm, ok := v.(map[string]any); ok {
				rawMap = rawm
			}
			ps := make(ProvidersSection)
			for name, provRaw := range rawMap {
				provMap, ok := provRaw.(map[string]any)
				if !ok {
					continue
				}
				prov := Provider{}
				if err := decodeIntoWithExtra(provMap, &prov); err != nil {
					return nil, &ParseError{Err: fmt.Errorf("[providers.%s]: %w", name, err)}
				}
				ps[name] = prov
			}
			stack.Providers = ps

		case "permissions":
			perm := PermissionsSection{}
			if err := decodeIntoWithExtra(v, &perm); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[permissions]: %w", err)}
			}
			stack.Permissions = perm

		case "watcher":
			ws := WatcherSection{}
			if err := decodeIntoWithExtra(v, &ws); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[watcher]: %w", err)}
			}
			stack.Watcher = ws

		case "lsp":
			// LSPSection is map[string]LSP — handle each server individually.
			var rawMap map[string]any
			if rawm, ok := v.(map[string]any); ok {
				rawMap = rawm
			}
			ls := make(LSPSection)
			for name, lspRaw := range rawMap {
				lspMap, ok := lspRaw.(map[string]any)
				if !ok {
					continue
				}
				lsp := LSP{}
				if err := decodeIntoWithExtra(lspMap, &lsp); err != nil {
					return nil, &ParseError{Err: fmt.Errorf("[lsp.%s]: %w", name, err)}
				}
				ls[name] = lsp
			}
			stack.LSP = ls

		// Phase 3.5 typed sections: skills, formatters, commands, opencode.
		case "skills":
			ss := SkillsSection{}
			if err := decodeInto(v, &ss); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[skills]: %w", err)}
			}
			stack.Skills = ss

		case "formatters":
			// FormattersSection is map[string]Formatter — handle each individually.
			var rawMap map[string]any
			if rawm, ok := v.(map[string]any); ok {
				rawMap = rawm
			}
			fs := make(FormattersSection)
			for name, fmtRaw := range rawMap {
				fmtMap, ok := fmtRaw.(map[string]any)
				if !ok {
					continue
				}
				f := Formatter{}
				if err := decodeIntoWithExtra(fmtMap, &f); err != nil {
					return nil, &ParseError{Err: fmt.Errorf("[formatters.%s]: %w", name, err)}
				}
				fs[name] = f
			}
			stack.Formatters = fs

		case "commands":
			// CommandsSection is map[string]Command — handle each individually.
			var rawMap map[string]any
			if rawm, ok := v.(map[string]any); ok {
				rawMap = rawm
			}
			cs := make(CommandsSection)
			for name, cmdRaw := range rawMap {
				cmdMap, ok := cmdRaw.(map[string]any)
				if !ok {
					continue
				}
				cmd := Command{}
				if err := decodeIntoWithExtra(cmdMap, &cmd); err != nil {
					return nil, &ParseError{Err: fmt.Errorf("[commands.%s]: %w", name, err)}
				}
				cs[name] = cmd
			}
			stack.Commands = cs

		case "opencode":
			oc := OpenCodeSection{}
			if err := decodeIntoWithExtra(v, &oc); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[opencode]: %w", err)}
			}
			stack.OpenCode = oc

		case "session":
			ss := &SessionSection{}
			if err := decodeInto(v, ss); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[session]: %w", err)}
			}
			stack.Session = ss

		case "shell":
			sh := ShellSection{}
			if err := decodeInto(v, &sh); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[shell]: %w", err)}
			}
			stack.Shell = sh

		case "update_probe":
			up := UpdateProbeSection{}
			if err := decodeInto(v, &up); err != nil {
				return nil, &ParseError{Err: fmt.Errorf("[update_probe]: %w", err)}
			}
			stack.UpdateProbe = up

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
	if stack.Plugins == nil {
		stack.Plugins = PluginsSection{}
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

// reflectFieldByName finds a field with the given TOML tag name using
// reflection. Returns the reflect.Value pointer to the field so callers
// can Set it.
func reflectFieldByName(v any, tomlKey string) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return nil
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("toml")
		if tag == "" {
			continue
		}
		// Handle "name,omitempty" tags.
		name := strings.Split(tag, ",")[0]
		if name == tomlKey {
			return rv.Field(i).Addr().Interface()
		}
	}
	return nil
}

// decodeIntoWithExtra decodes a raw TOML value into a typed struct,
// populating any field named "Extra" (tagged `toml:"-"`) with keys from
// the raw map that did not decode into any named struct field.
//
// If dst implements encoding.Unmarshaler (e.g., Provider, LSP), the
// custom UnmarshalTOML is called directly to handle Extra population
// without a round-trip through toml.Encode (which drops "toml:\"-\""
// fields). Otherwise, the standard decodeInto round-trip is used and
// Extra is computed from unconsumed keys via reflection.
func decodeIntoWithExtra(raw any, dst any) error {
	m, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("expected table, got %T", raw)
	}

	// If dst has a custom UnmarshalTOML, call it directly.
	// This avoids the toml.Encode round-trip which drops "toml:\"-\"" fields
	// (like Extra) and avoids double-decoding for types that already handled
	// nested map Extra fields (Provider.Models, etc.).
	if u, ok := dst.(interface{ UnmarshalTOML(any) error }); ok {
		return u.UnmarshalTOML(raw)
	}

	// Standard round-trip decode for non-custom types.
	if err := decodeInto(raw, dst); err != nil {
		return err
	}

	// Populate Extra from keys not consumed by the round-trip.
	consumed := collectDecodedKeys(dst)
	remaining := make(map[string]any)
	for k, v := range m {
		if !consumed[k] {
			remaining[k] = v
		}
	}
	if len(remaining) == 0 {
		return nil
	}
	extraIface := reflectFieldByName(dst, "-")
	if extraIface == nil {
		return nil
	}
	extraMap := reflect.ValueOf(extraIface).Elem()
	extraMap.Set(reflect.ValueOf(remaining))
	return nil
}

// collectDecodedKeys returns a set of top-level toml keys that would be
// consumed when decoding into dst using decodeInto (which round-trips via
// toml encoding). We determine this by round-tripping a synthetic document
// through toml encoding and checking what keys appear.
func collectDecodedKeys(dst any) map[string]bool {
	// Use the known struct field TOML tag names as the consumed set.
	rv := reflect.ValueOf(dst)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	consumed := make(map[string]bool)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("toml")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name != "" {
			consumed[name] = true
		}
	}
	return consumed
}
