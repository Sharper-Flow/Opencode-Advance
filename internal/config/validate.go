package config

import (
	"fmt"
	"sort"
	"strings"
)

// Port range allocated to Vision MCP servers (mirrors Vision's
// internal/config.MinPort/MaxPort constants).
const (
	minPort = 6275 // Vision daemon (admin MCP) itself lives here
	maxPort = 6300
)

// validRestartPolicies mirror Vision's enum.
var validRestartPolicies = map[string]bool{
	"":           true, // defaults to "on-failure"
	"always":     true,
	"on-failure": true,
	"never":      true,
}

// validAvailabilityProfiles mirror Vision's enum.
var validAvailabilityProfiles = map[string]bool{
	"":          true,
	"networked": true,
}

// validTypes covers OCA-level transport values. "daemon" is OCA-only
// (marks the Vision admin MCP). "stdio"/"http"/"sse" match Vision.
// Empty string infers from command/url like Vision does.
var validTypes = map[string]bool{
	"":       true,
	"stdio":  true,
	"http":   true,
	"sse":    true,
	"daemon": true,
}

// ValidationError is a single field-scoped problem with the input.
type ValidationError struct {
	Path    string // e.g. mcp.servers.kagi.port
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("[%s]: %s", e.Path, e.Message)
}

// ValidationErrors aggregates multiple ValidationError instances so
// callers can surface every issue in one pass.
type ValidationErrors []ValidationError

func (es ValidationErrors) Error() string {
	if len(es) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "stack.toml validation failed (%d error(s)):\n", len(es))
	for _, e := range es {
		fmt.Fprintf(&b, "  %s\n", e.Error())
	}
	return b.String()
}

// HasErrors returns true when the slice is non-empty.
func (es ValidationErrors) HasErrors() bool { return len(es) > 0 }

// Validate runs all schema checks against a parsed Stack. Errors are
// aggregated — the returned ValidationErrors contains every problem
// found so the user can fix them in one pass. nil is returned when the
// stack is valid.
func (s *Stack) Validate() error {
	var errs ValidationErrors

	// [meta]
	if s.Meta.Version == "" {
		errs = append(errs, ValidationError{
			Path:    "meta.version",
			Message: "required field missing",
		})
	} else if s.Meta.Version != "1.0.0" {
		errs = append(errs, ValidationError{
			Path:    "meta.version",
			Message: fmt.Sprintf("unsupported schema version %q (expected \"1.0.0\")", s.Meta.Version),
		})
	}

	// [mcp.servers.*]
	errs = append(errs, s.validateMCPServers()...)

	// Deferred sections — classify as known-but-deferred (ok) or
	// truly unknown (error).
	for name := range s.DeferredSections {
		if !knownSections[name] {
			errs = append(errs, ValidationError{
				Path:    name,
				Message: fmt.Sprintf("unknown top-level section %q (typo? allowed: %s)", name, strings.Join(knownSectionNames(), ", ")),
			})
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

func (s *Stack) validateMCPServers() ValidationErrors {
	var errs ValidationErrors

	// Stable iteration order for deterministic error messages.
	names := make([]string, 0, len(s.MCP.Servers))
	for name := range s.MCP.Servers {
		names = append(names, name)
	}
	sort.Strings(names)

	// Track port uniqueness and daemon-type singleton.
	portOwners := make(map[int]string, len(names))
	var daemonServer string

	for _, name := range names {
		srv := s.MCP.Servers[name]
		path := fmt.Sprintf("mcp.servers.%s", name)

		// Port range
		if srv.Port == 0 {
			errs = append(errs, ValidationError{
				Path:    path + ".port",
				Message: "required field missing",
			})
		} else if srv.Port < minPort || srv.Port > maxPort {
			errs = append(errs, ValidationError{
				Path:    path + ".port",
				Message: fmt.Sprintf("port %d out of range [%d, %d]", srv.Port, minPort, maxPort),
			})
		}

		// Port uniqueness
		if srv.Port > 0 {
			if existing, ok := portOwners[srv.Port]; ok {
				errs = append(errs, ValidationError{
					Path:    path + ".port",
					Message: fmt.Sprintf("port %d already used by server %q", srv.Port, existing),
				})
			} else {
				portOwners[srv.Port] = name
			}
		}

		// Type validation
		if !validTypes[srv.Type] {
			errs = append(errs, ValidationError{
				Path:    path + ".type",
				Message: fmt.Sprintf("unknown type %q (allowed: stdio, http, sse, daemon)", srv.Type),
			})
		}

		// type="daemon" special-case (OCA-only directive)
		if srv.Type == "daemon" {
			if daemonServer != "" {
				errs = append(errs, ValidationError{
					Path:    path + ".type",
					Message: fmt.Sprintf("only one server may have type=\"daemon\"; already declared on %q", daemonServer),
				})
			} else {
				daemonServer = name
			}
			if name != "vision" {
				errs = append(errs, ValidationError{
					Path:    path + ".type",
					Message: "type=\"daemon\" must be named \"vision\"",
				})
			}
			if srv.Command != "" || srv.URL != "" || len(srv.Args) > 0 {
				errs = append(errs, ValidationError{
					Path:    path,
					Message: "type=\"daemon\" must not specify command, args, or url (OCA-only directive)",
				})
			}
			// daemon-type skips transport inference validation below.
			continue
		}

		// Transport inference + conflict
		transport := inferTransport(srv)
		if srv.Command != "" && srv.URL != "" {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: "cannot specify both command and url",
			})
		}
		switch transport {
		case "stdio":
			if srv.Command == "" {
				errs = append(errs, ValidationError{
					Path:    path + ".command",
					Message: "required for stdio transport",
				})
			}
		case "http":
			if srv.URL == "" {
				errs = append(errs, ValidationError{
					Path:    path + ".url",
					Message: "required for http transport",
				})
			} else if !strings.HasSuffix(srv.URL, "/mcp") {
				errs = append(errs, ValidationError{
					Path:    path + ".url",
					Message: "http transport url must end with /mcp",
				})
			}
		case "sse":
			if srv.URL == "" {
				errs = append(errs, ValidationError{
					Path:    path + ".url",
					Message: "required for sse transport",
				})
			}
		}

		// Restart policy enum
		if !validRestartPolicies[srv.RestartPolicy] {
			errs = append(errs, ValidationError{
				Path:    path + ".restart_policy",
				Message: fmt.Sprintf("unknown value %q (allowed: always, on-failure, never)", srv.RestartPolicy),
			})
		}

		// Availability profile enum
		if !validAvailabilityProfiles[srv.AvailabilityProfile] {
			errs = append(errs, ValidationError{
				Path:    path + ".availability_profile",
				Message: fmt.Sprintf("unknown value %q (allowed: networked)", srv.AvailabilityProfile),
			})
		}
	}

	return errs
}

// inferTransport returns the effective transport for a server, mirroring
// Vision's ServerConfig.InferTransport precedence.
func inferTransport(s Server) string {
	if s.Transport != "" {
		return s.Transport
	}
	if s.Type != "" && s.Type != "daemon" {
		return s.Type
	}
	if s.Command != "" {
		return "stdio"
	}
	if s.URL != "" {
		if strings.HasSuffix(s.URL, "/mcp") {
			return "http"
		}
		return "sse"
	}
	return "stdio"
}

func knownSectionNames() []string {
	names := make([]string, 0, len(knownSections))
	for k := range knownSections {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
