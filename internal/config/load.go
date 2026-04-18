package config

import (
	"os"
)

// Load runs the full pipeline for a stack.toml file on disk:
//   1. Parse (TOML decode → typed Stack with deferred sections)
//   2. Resolve (variable expansion: $HOME, ~/, $XDG_*, ${VAR})
//   3. Validate (aggregate ValidationErrors with field paths)
//   4. Collect warnings (e.g. env_file path does not exist)
//
// Order rationale: Resolve runs before Validate so validation sees
// resolved paths (e.g. env_file paths with ~/expanded). This corrects
// the original Parse→Validate→Resolve order from the initial design.
//
// Load returns the Stack on success and ValidationErrors (as a wrapped
// error) on failure. Warnings are attached to Stack.Warnings for the
// caller to surface; they never fail Load.
func Load(path string) (*Stack, error) {
	stack, err := ParseFile(path)
	if err != nil {
		return nil, err
	}

	stack.Resolve()

	if err := stack.Validate(); err != nil {
		return nil, err
	}

	stack.Warnings = append(stack.Warnings, stack.collectEnvFileWarnings()...)
	return stack, nil
}

// collectEnvFileWarnings produces non-fatal warnings for missing
// env_file paths. LBP semantics: we do NOT fail apply if a path is
// missing (systemd EnvironmentFile=- semantics adapted) because the
// user may run oca apply before secrets are in place. Doctor surfaces
// the same warnings at check time.
func (s *Stack) collectEnvFileWarnings() []Warning {
	var ws []Warning
	for name, srv := range s.MCP.Servers {
		if srv.EnvFile == "" {
			continue
		}
		if _, err := os.Stat(srv.EnvFile); err != nil {
			ws = append(ws, Warning{
				Path:    "mcp.servers." + name + ".env_file",
				Message: "env_file path does not exist: " + srv.EnvFile,
				Hint:    "create the file or remove the env_file reference; secrets are never read by OCA",
			})
		}
	}
	return ws
}
