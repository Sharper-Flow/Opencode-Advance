package config

import (
	"os"
	"regexp"
	"strings"
)

// envVarPattern matches ${VAR} and ${VAR:-default}. Mirrors Vision's
// ExpandEnvVars pattern so behavior is consistent across the two tools.
var envVarPattern = regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

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
