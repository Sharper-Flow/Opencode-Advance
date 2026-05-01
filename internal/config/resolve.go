package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// envVarPattern matches ${VAR} and ${VAR:-default}. Mirrors Vision's
// ExpandEnvVars pattern so behavior is consistent across the two tools.
var envVarPattern = regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

// pluginTokenPattern matches {token} placeholders in plugin fields.
var pluginTokenPattern = regexp.MustCompile(`\{([^}]+)\}`)

// Resolve expands OCA-owned tokens in-place on the Stack:
//   - $HOME and ~/... → user home directory
//   - $XDG_CONFIG_HOME, $XDG_DATA_HOME → env lookup with defaults
//   - ${VAR} / ${VAR:-default} → env lookup
//
// Native OpenCode tokens — {env:...}, {file:...} — are left UNCHANGED
// so OpenCode resolves them at runtime. This is the deliberate handoff
// point that keeps OCA secret-blind: OCA records these tokens verbatim
// and never reads the underlying files or env vars.
func (s *Stack) Resolve() {
	for name, srv := range s.MCP.Servers {
		srv.Command = expandAll(srv.Command)
		srv.URL = expandAll(srv.URL)
		srv.EnvFile = expandAll(srv.EnvFile)
		for i, arg := range srv.Args {
			srv.Args[i] = expandAll(arg)
		}
		for k, v := range srv.Env {
			srv.Env[k] = expandAll(v)
		}
		s.MCP.Servers[name] = srv
	}

	// Expand plugin block-local tokens ({checkout}, {subdir}) in path/sync fields.
	for name, plugin := range s.Plugins {
		if err := expandPluginTokens(&plugin); err != nil {
			// Log and continue — validation will surface the error.
			continue
		}
		s.Plugins[name] = plugin
	}

	// Apply session/watchdog defaults.
	if s.Session != nil && s.Session.Watchdog != nil {
		if s.Session.Watchdog.IdleTimeout == 0 {
			s.Session.Watchdog.IdleTimeout = 5 * time.Minute
		}
		if s.Session.Watchdog.MaxBumps == 0 {
			s.Session.Watchdog.MaxBumps = 3
		}
	}
}

// expandPluginTokens substitutes {checkout} and {subdir} in a plugin's
// Path, Sync, and Instructions fields. Unknown {token} values produce
// an error. Empty checkout is tolerated (tokens pass through unchanged
// so validation can report the missing field).
func expandPluginTokens(p *Plugin) error {
	if p == nil {
		return nil
	}

	// Resolve general OCA tokens first so {checkout} operates on an
	// already-expanded path (e.g. ~/... → /home/user/...).
	checkout := expandAll(p.Checkout)

	allowed := map[string]string{
		"checkout": checkout,
		"subdir":   p.Subdir,
	}

	// unknownTokenIn returns the unknown token name if field contains one.
	unknownTokenIn := func(field string) string {
		matches := pluginTokenPattern.FindAllStringSubmatch(field, -1)
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			name := m[1]
			if _, ok := allowed[name]; !ok {
				return "{" + name + "}"
			}
		}
		return ""
	}

	replaceTokens := func(field string) string {
		if field == "" {
			return field
		}
		return pluginTokenPattern.ReplaceAllStringFunc(field, func(token string) string {
			m := pluginTokenPattern.FindStringSubmatch(token)
			if len(m) < 2 {
				return token
			}
			name := m[1]
			if val, ok := allowed[name]; ok && val != "" {
				return val // only substitute when replacement is non-empty
			}
			return token // unknown or empty value — leave unchanged
		})
	}

	// Check for unknown tokens before substitution.
	if unknown := unknownTokenIn(p.Path); unknown != "" {
		return fmt.Errorf("[plugins.*].path: unrecognized token %s (allowed: {checkout}, {subdir})", unknown)
	}
	if unknown := unknownTokenIn(p.Sync); unknown != "" {
		return fmt.Errorf("[plugins.*].sync: unrecognized token %s (allowed: {checkout}, {subdir})", unknown)
	}
	for i, instr := range p.Instructions {
		if unknown := unknownTokenIn(instr); unknown != "" {
			return fmt.Errorf("[plugins.*].instructions[%d]: unrecognized token %s", i, unknown)
		}
	}

	// Safe to substitute — all tokens are known.
	//
	// Normalize path-shaped fields with filepath.Clean after substitution
	// so `{checkout}/../other` and similar token combinations collapse to
	// their canonical form before any downstream code uses them as a
	// filesystem target. Sync is deliberately NOT cleaned: it is a shell
	// command string, not a filesystem path, and filepath.Clean would
	// corrupt legitimate shell syntax.
	p.Path = filepath.Clean(replaceTokens(p.Path))
	p.Sync = replaceTokens(p.Sync)
	for i, instr := range p.Instructions {
		p.Instructions[i] = filepath.Clean(replaceTokens(instr))
	}

	return nil
}

// expandAll applies $HOME/~/XDG/${VAR} expansion.
// Native OpenCode tokens ({env:...}, {file:...}) are preserved verbatim.
func expandAll(in string) string {
	if in == "" {
		return in
	}
	out := in
	out = expandHome(out)
	out = expandEnv(out)
	return out
}

// expandHome handles $HOME and leading ~/.
func expandHome(in string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return in
	}
	if strings.HasPrefix(in, "~/") {
		in = home + in[1:]
	}
	in = strings.ReplaceAll(in, "$HOME", home)
	return in
}

// expandEnv handles ${VAR} and ${VAR:-default}, plus the XDG defaults.
// Native OpenCode tokens ({env:...}, {file:...}) pass through because
// they don't match the ${...} shape.
func expandEnv(in string) string {
	// XDG defaults: only substitute when the env var is unset OR empty.
	// This matches XDG Base Directory behavior.
	in = expandXDG(in, "XDG_CONFIG_HOME", ".config")
	in = expandXDG(in, "XDG_DATA_HOME", ".local/share")

	return envVarPattern.ReplaceAllStringFunc(in, func(match string) string {
		parts := envVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		varName := parts[1]
		defaultVal := ""
		if len(parts) > 2 {
			defaultVal = parts[2]
		}
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return defaultVal
	})
}

// expandXDG substitutes $XDG_<name> with its env value, falling back to
// $HOME/<fallback> when the env var is unset or empty.
func expandXDG(in, varName, fallbackRel string) string {
	token := "$" + varName
	if !strings.Contains(in, token) {
		return in
	}
	val := os.Getenv(varName)
	if val == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			val = home + "/" + fallbackRel
		}
	}
	return strings.ReplaceAll(in, token, val)
}
