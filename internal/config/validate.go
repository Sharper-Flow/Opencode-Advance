package config

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Port range allocated to Vision MCP servers (mirrors Vision's
// internal/config.MinPort/MaxPort constants). Bumped to 6325 in v1.1.1
// to accommodate slot group pools (e.g., Playwright headless+headed+auth
// pools occupy 6300-6310 in the current curated stack).
const (
	minPort = 6275 // Vision daemon (admin MCP) itself lives here
	maxPort = 6325
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

// ValidationErrorContains returns true if any ValidationError's message
// contains the given substring. Used by tests to assert a specific error
// was raised without matching the exact path+message format.
func ValidationErrorContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), substr)
}

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

	// [mcp.slot_groups.*]
	errs = append(errs, s.validateMCPSlotGroups()...)

	// [plugins.*]
	errs = append(errs, validatePlugins(s)...)

	// [instructions]
	errs = append(errs, validateInstructions(s)...)

	// [providers.*]
	errs = append(errs, validateProviders(s)...)

	// [permissions.*]
	errs = append(errs, validatePermissions(s)...)

	// [watcher]
	errs = append(errs, validateWatcher(s)...)

	// [lsp.*]
	errs = append(errs, validateLSP(s)...)

	// Phase 3.5 typed sections
	errs = append(errs, validateSkills(s)...)
	errs = append(errs, validateFormatters(s)...)
	errs = append(errs, validateCommands(s)...)
	errs = append(errs, validateOpenCode(s)...)

	// [session] — promoted from deferred
	errs = append(errs, validateSession(s)...)

	// [temporal] — Phase 5
	errs = append(errs, validateTemporal(s)...)

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

// validateMCPServers is the MCP server portion of Validate, kept as a
// method so it can call inferTransport without exporting it.
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

		// request_timeout must be a positive duration when set because OCA
		// derives the OpenCode-side timeout field from it.
		if srv.RequestTimeout != "" {
			d, err := time.ParseDuration(srv.RequestTimeout)
			if err != nil || d <= 0 {
				message := fmt.Sprintf("invalid duration %q", srv.RequestTimeout)
				if err == nil && d <= 0 {
					message += " (must be > 0)"
				}
				errs = append(errs, ValidationError{
					Path:    path + ".request_timeout",
					Message: message,
				})
			}
		}
	}

	return errs
}

// validateMCPSlotGroups validates [mcp.slot_groups.*] against Vision's
// SlotGroupConfig contract: required fields, count >= 2, port range,
// `port` rejected on defaults, and global port + name collision checks
// across declared servers and other slot groups.
//
// Pre-validating template name collisions (`<template>-N` vs declared
// servers.* keys) lets users see errors at oca apply --dry-run instead of
// at Vision boot. Mirrors Vision's expand.go collision check.
func (s *Stack) validateMCPSlotGroups() ValidationErrors {
	var errs ValidationErrors
	if len(s.MCP.SlotGroups) == 0 {
		return errs
	}

	// Build the set of ports already claimed by [mcp.servers.*]. Reuse the
	// first server name encountered at each port for clear error messages.
	serverPorts := make(map[int]string, len(s.MCP.Servers))
	for name, srv := range s.MCP.Servers {
		if srv.Port > 0 {
			if _, taken := serverPorts[srv.Port]; !taken {
				serverPorts[srv.Port] = "mcp.servers." + name
			}
		}
	}

	// Stable iteration order for deterministic error messages.
	groupNames := make([]string, 0, len(s.MCP.SlotGroups))
	for name := range s.MCP.SlotGroups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	// Track ports claimed by slot groups so cross-group collisions are
	// surfaced. Each port maps to its origin path (e.g.
	// "mcp.slot_groups.pw1.base_port" or "mcp.slot_groups.pw1[2]").
	groupPorts := make(map[int]string)

	for _, name := range groupNames {
		group := s.MCP.SlotGroups[name]
		path := fmt.Sprintf("mcp.slot_groups.%s", name)

		// Required fields.
		if group.Template == "" {
			errs = append(errs, ValidationError{
				Path:    path + ".template",
				Message: "required field missing",
			})
		}
		if group.BasePort == 0 {
			errs = append(errs, ValidationError{
				Path:    path + ".base_port",
				Message: "required field missing",
			})
		}
		if group.GroupPort == 0 {
			errs = append(errs, ValidationError{
				Path:    path + ".group_port",
				Message: "required field missing",
			})
		}
		// count == 0 is "missing"; count == 1 is "below minimum".
		if group.Count == 0 {
			errs = append(errs, ValidationError{
				Path:    path + ".count",
				Message: "required field missing",
			})
		} else if group.Count < 2 {
			errs = append(errs, ValidationError{
				Path:    path + ".count",
				Message: fmt.Sprintf("count %d below minimum (Vision requires count >= 2)", group.Count),
			})
		}

		// Port range checks (only when fields populated).
		if group.GroupPort > 0 && (group.GroupPort < minPort || group.GroupPort > maxPort) {
			errs = append(errs, ValidationError{
				Path:    path + ".group_port",
				Message: fmt.Sprintf("port %d out of range [%d, %d]", group.GroupPort, minPort, maxPort),
			})
		}
		if group.BasePort > 0 && group.Count >= 2 {
			lastSlotPort := group.BasePort + group.Count - 1
			if group.BasePort < minPort {
				errs = append(errs, ValidationError{
					Path:    path + ".base_port",
					Message: fmt.Sprintf("port %d out of range [%d, %d]", group.BasePort, minPort, maxPort),
				})
			} else if lastSlotPort > maxPort {
				errs = append(errs, ValidationError{
					Path:    path + ".base_port",
					Message: fmt.Sprintf("port %d out of range [%d, %d] (base_port %d + count %d - 1 = %d)", lastSlotPort, minPort, maxPort, group.BasePort, group.Count, lastSlotPort),
				})
			}
		}

		// Defaults.port is forbidden — Vision derives slot ports from base_port.
		if group.Defaults != nil && group.Defaults.Port > 0 {
			errs = append(errs, ValidationError{
				Path:    path + ".defaults.port",
				Message: "must not be set (Vision derives slot ports from base_port)",
			})
		}

		// Port collision: group_port vs declared servers and other groups.
		if group.GroupPort > 0 {
			if owner, taken := serverPorts[group.GroupPort]; taken {
				errs = append(errs, ValidationError{
					Path:    path + ".group_port",
					Message: fmt.Sprintf("port %d already used by %s", group.GroupPort, owner),
				})
			}
			if owner, taken := groupPorts[group.GroupPort]; taken {
				errs = append(errs, ValidationError{
					Path:    path + ".group_port",
					Message: fmt.Sprintf("port %d already used by %s", group.GroupPort, owner),
				})
			} else {
				groupPorts[group.GroupPort] = path + ".group_port"
			}
		}

		// Port collision: each synthesized slot port vs servers and other groups.
		// Also reject template-name collisions vs declared servers.
		if group.BasePort > 0 && group.Count >= 2 {
			for i := 0; i < group.Count; i++ {
				slotPort := group.BasePort + i
				slotPath := fmt.Sprintf("%s[slot %d]", path, i+1)

				if owner, taken := serverPorts[slotPort]; taken {
					errs = append(errs, ValidationError{
						Path:    path + ".base_port",
						Message: fmt.Sprintf("port %d already used by %s", slotPort, owner),
					})
				}
				if owner, taken := groupPorts[slotPort]; taken {
					errs = append(errs, ValidationError{
						Path:    path + ".base_port",
						Message: fmt.Sprintf("port %d already used by %s", slotPort, owner),
					})
				} else {
					groupPorts[slotPort] = slotPath
				}

				// Template-name collision check.
				if group.Template != "" {
					synthName := fmt.Sprintf("%s-%d", group.Template, i+1)
					if _, taken := s.MCP.Servers[synthName]; taken {
						errs = append(errs, ValidationError{
							Path:    path + ".template",
							Message: fmt.Sprintf("synthesized slot name %q collides with declared mcp.servers.%s (Vision will reject at load)", synthName, synthName),
						})
					}
				}
			}
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

// validatePlugins checks per-plugin rules:
//   - source must be a git URL (https://...git or git@...:...) or npm:pkg@ver
//   - git source: checkout is required
//   - git source: path is required after {checkout}/{subdir} expansion
//   - provides entries must be valid ProvidesCategory values
//
// Trust boundary: no host or registry allowlist is enforced here. stack.toml
// is a user-authored file; if the user declares a plugin source, oca trusts
// them to vet it. Hardening against hostile remotes is layered elsewhere
// (see internal/plugin/git.go gitHardeningArgs + validateGitRef, and
// internal/plugin/prepare.go's Lstat/remote-URL-drift checks). Adding a
// host allowlist here would block legitimate private forks and mirrors
// without meaningfully raising the security bar. See
// docs/design/stack-toml-schema.md for the rationale.
func validatePlugins(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Plugins == nil {
		return errs
	}
	for name, plugin := range s.Plugins {
		pathPrefix := fmt.Sprintf("plugins.%s", name)

		// source required
		if plugin.Source == "" {
			errs = append(errs, ValidationError{
				Path:    pathPrefix + ".source",
				Message: "required field missing",
			})
			continue // can't validate further without source
		}

		if plugin.IsNPMSource() || plugin.IsLocalSource() {
			// npm and local sources only need source; checkout/path/build are not applicable.
			// Still validate provides enum below.
		} else {
			// Git source requires checkout and path.
			if plugin.Checkout == "" {
				errs = append(errs, ValidationError{
					Path:    pathPrefix + ".checkout",
					Message: "required for git source",
				})
			}
			if plugin.Path == "" {
				errs = append(errs, ValidationError{
					Path:    pathPrefix + ".path",
					Message: "required for git source",
				})
			}
		}

		// provides enum validation
		for i, p := range plugin.Provides {
			if !p.IsValid() {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.provides[%d]", pathPrefix, i),
					Message: fmt.Sprintf("unknown category %q (allowed: adv-commands, adv-agents, adv-skills, adv-overlays, adv-instructions, adv-temporal)", p),
				})
			}
		}
	}
	return errs
}

// validateInstructions checks that the instructions section is well-formed.
func validateInstructions(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Instructions.Order == nil {
		// Empty order is allowed (means no extra instructions beyond plugin defaults).
		return errs
	}
	if len(s.Instructions.Order) == 0 {
		errs = append(errs, ValidationError{
			Path:    "instructions.order",
			Message: "must be non-empty when declared",
		})
	}
	// Each order entry must be a non-empty string.
	for i, entry := range s.Instructions.Order {
		if entry == "" {
			errs = append(errs, ValidationError{
				Path:    fmt.Sprintf("instructions.order[%d]", i),
				Message: "entry must not be empty",
			})
		}
	}
	return errs
}

// validateTemporal validates the [temporal] section and applies defaults.
// When Temporal is nil or disabled, it is a no-op. When enabled (or
// Enabled absent, which defaults to true), defaults are applied:
//   - Address defaults to "127.0.0.1:7233"
//   - Namespace defaults to "default"
//
// A non-loopback address requires AllowRemote=true.
func validateTemporal(s *Stack) ValidationErrors {
	if s.Temporal == nil {
		return nil
	}
	if !s.Temporal.IsEnabled() {
		return nil
	}

	var errs ValidationErrors

	// Apply defaults.
	if s.Temporal.Address == "" {
		s.Temporal.Address = "127.0.0.1:7233"
	}
	if s.Temporal.Namespace == "" {
		s.Temporal.Namespace = "default"
	}

	// Validate: non-loopback requires allow_remote=true.
	if !isLoopbackAddress(s.Temporal.Address) {
		if s.Temporal.AllowRemote == nil || !*s.Temporal.AllowRemote {
			errs = append(errs, ValidationError{
				Path:    "temporal.allow_remote",
				Message: "must be true when address is non-loopback (security: prevents accidental exposure)",
			})
		}
	}

	return errs
}

// isLoopbackAddress checks if a host:port string points to a loopback
// address. Handles 127.x.x.x, localhost, ::1, and IPv6-mapped IPv4 (::ffff:127.x.x.x).
func isLoopbackAddress(addr string) bool {
	host := addr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		host = addr[:idx]
	}
	// Strip brackets from [::1]:port
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")

	switch host {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	// 127.x.x.x range
	if strings.HasPrefix(host, "127.") {
		return true
	}
	// IPv6-mapped IPv4 loopback (::ffff:127.x.x.x)
	if strings.HasPrefix(host, "::ffff:") {
		mappedHost := strings.TrimPrefix(host, "::ffff:")
		if strings.HasPrefix(mappedHost, "127.") || mappedHost == "127.0.0.1" {
			return true
		}
	}
	return false
}

// validateProviders checks per-provider and per-model rules.
// Design K8: each [providers.<name>] must be a non-empty table;
// each models.<id> must have either name or some variant;
// context/output positive if set; inputs/outputs non-empty strings if set.
func validateProviders(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Providers == nil || len(s.Providers) == 0 {
		return errs
	}
	for pname, prov := range s.Providers {
		path := fmt.Sprintf("providers.%s", pname)
		if len(prov.Models) == 0 && len(prov.Extra) == 0 {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: "provider must have at least one models entry or known fields",
			})
			continue
		}
		for mname, model := range prov.Models {
			mPath := fmt.Sprintf("%s.models.%s", path, mname)
			hasName := model.Name != ""
			hasVariants := model.Variants != nil && len(model.Variants) > 0
			if !hasName && !hasVariants {
				errs = append(errs, ValidationError{
					Path:    mPath,
					Message: "model must have at least one of: name, variants",
				})
			}
			if model.Context < 0 {
				errs = append(errs, ValidationError{
					Path:    mPath + ".context",
					Message: fmt.Sprintf("context must be non-negative, got %d", model.Context),
				})
			}
			if model.Output < 0 {
				errs = append(errs, ValidationError{
					Path:    mPath + ".output",
					Message: fmt.Sprintf("output must be non-negative, got %d", model.Output),
				})
			}
			for i, input := range model.Inputs {
				if input == "" {
					errs = append(errs, ValidationError{
						Path:    fmt.Sprintf("%s.inputs[%d]", mPath, i),
						Message: "inputs entries must not be empty strings",
					})
				}
			}
			for i, output := range model.Outputs {
				if output == "" {
					errs = append(errs, ValidationError{
						Path:    fmt.Sprintf("%s.outputs[%d]", mPath, i),
						Message: "outputs entries must not be empty strings",
					})
				}
			}
		}
	}
	return errs
}

// validPermissionActions is the set of allowed permission action values.
var validPermissionActions = map[string]bool{
	"allow": true,
	"ask":   true,
	"deny":  true,
}

// validatePermissions checks that permission action values are valid.
// Design K8: default, doom_loop in {"allow","ask","deny"} if set;
// external_directory and bash entries are string→action in same set.
func validatePermissions(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Permissions.Default != "" && !validPermissionActions[s.Permissions.Default] {
		errs = append(errs, ValidationError{
			Path:    "permissions.default",
			Message: fmt.Sprintf("unknown action %q (allowed: allow, ask, deny)", s.Permissions.Default),
		})
	}
	if s.Permissions.DoomLoop != "" && !validPermissionActions[s.Permissions.DoomLoop] {
		errs = append(errs, ValidationError{
			Path:    "permissions.doom_loop",
			Message: fmt.Sprintf("unknown action %q (allowed: allow, ask, deny)", s.Permissions.DoomLoop),
		})
	}
	for path, action := range s.Permissions.ExternalDirectory {
		if !validPermissionActions[action] {
			errs = append(errs, ValidationError{
				Path:    fmt.Sprintf("permissions.external_directory.%s", path),
				Message: fmt.Sprintf("unknown action %q (allowed: allow, ask, deny)", action),
			})
		}
	}
	for path, action := range s.Permissions.Bash {
		if !validPermissionActions[action] {
			errs = append(errs, ValidationError{
				Path:    fmt.Sprintf("permissions.bash.%s", path),
				Message: fmt.Sprintf("unknown action %q (allowed: allow, ask, deny)", action),
			})
		}
	}
	return errs
}

// validateWatcher checks that watcher.ignore is []string and entries are non-empty.
// Design K8.
func validateWatcher(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Watcher.Ignore == nil {
		return errs
	}
	for i, entry := range s.Watcher.Ignore {
		if entry == "" {
			errs = append(errs, ValidationError{
				Path:    fmt.Sprintf("watcher.ignore[%d]", i),
				Message: "ignore entries must not be empty strings",
			})
		}
	}
	return errs
}

// validateLSP checks per-LSP rules.
// Design K8: each [lsp.<name>] is a non-empty table; if not disabled, command is non-empty.
func validateLSP(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.LSP == nil || len(s.LSP) == 0 {
		return errs
	}
	for lname, lsp := range s.LSP {
		path := fmt.Sprintf("lsp.%s", lname)
		// Must be non-empty (has at least one field or Extra entry).
		hasContent := (lsp.Command != nil && len(lsp.Command) > 0) ||
			(lsp.Extensions != nil && len(lsp.Extensions) > 0) ||
			lsp.Disabled ||
			(lsp.Extra != nil && len(lsp.Extra) > 0)
		if !hasContent {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: "LSP server must have at least one of: command, extensions, disabled, or additional fields",
			})
			continue
		}
		// If not disabled, command must be non-empty.
		if !lsp.Disabled && (lsp.Command == nil || len(lsp.Command) == 0) {
			errs = append(errs, ValidationError{
				Path:    path + ".command",
				Message: "command is required when LSP server is not disabled",
			})
		}
	}
	return errs
}

// ---------------------------------------------------------------------------
// Phase 3.5 validation
// ---------------------------------------------------------------------------

// validShareValues are the allowed values for opencode.share.
var validShareValues = map[string]bool{
	"":         true,
	"manual":   true,
	"auto":     true,
	"disabled": true,
}

// validateSkills checks skills.order for uniqueness, non-empty entries,
// and rejection of reserved adv-* namespace.
func validateSkills(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Skills.Order == nil {
		return errs
	}
	seen := make(map[string]bool, len(s.Skills.Order))
	for i, name := range s.Skills.Order {
		path := fmt.Sprintf("skills.order[%d]", i)
		if name == "" {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: "entry must not be empty",
			})
			continue
		}
		if strings.HasPrefix(name, "adv-") {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("reserved namespace %q (adv-* skills are managed by the Advance plugin)", name),
			})
		}
		if seen[name] {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("duplicate skill %q", name),
			})
		}
		seen[name] = true
	}
	return errs
}

// validateFormatters checks that each formatter (unless disabled) has
// both command and extensions set.
func validateFormatters(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Formatters == nil {
		return errs
	}
	names := sortedKeys(s.Formatters)
	for _, name := range names {
		f := s.Formatters[name]
		path := fmt.Sprintf("formatters.%s", name)
		if f.Disabled {
			continue
		}
		if len(f.Command) == 0 {
			errs = append(errs, ValidationError{
				Path:    path + ".command",
				Message: "command is required when formatter is not disabled",
			})
		}
		if len(f.Extensions) == 0 {
			errs = append(errs, ValidationError{
				Path:    path + ".extensions",
				Message: "extensions is required when formatter is not disabled",
			})
		}
	}
	return errs
}

// validateCommands checks that each command has description and template.
func validateCommands(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Commands == nil {
		return errs
	}
	names := sortedKeys(s.Commands)
	for _, name := range names {
		cmd := s.Commands[name]
		path := fmt.Sprintf("commands.%s", name)
		if cmd.Description == "" {
			errs = append(errs, ValidationError{
				Path:    path + ".description",
				Message: "required field missing",
			})
		}
		if cmd.Template == "" {
			errs = append(errs, ValidationError{
				Path:    path + ".template",
				Message: "required field missing",
			})
		}
	}
	return errs
}

// validateOpenCode checks opencode.share enum and opencode.autoupdate
// union (true|false|"notify").
func validateOpenCode(s *Stack) ValidationErrors {
	var errs ValidationErrors

	if s.OpenCode.Share != "" && !validShareValues[s.OpenCode.Share] {
		errs = append(errs, ValidationError{
			Path:    "opencode.share",
			Message: fmt.Sprintf("unknown value %q (allowed: manual, auto, disabled)", s.OpenCode.Share),
		})
	}

	if au := s.OpenCode.Autoupdate; au != nil {
		if au.Str != nil && *au.Str != "notify" {
			errs = append(errs, ValidationError{
				Path:    "opencode.autoupdate",
				Message: fmt.Sprintf("unknown value %q (allowed: true, false, \"notify\")", *au.Str),
			})
		}
	}

	return errs
}

// sortedKeys returns the keys of a map[string]T in sorted order.
func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func knownSectionNames() []string {
	names := make([]string, 0, len(knownSections))
	for k := range knownSections {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// validateSession checks [session.watchdog] constraints.
func validateSession(s *Stack) ValidationErrors {
	var errs ValidationErrors
	if s.Session == nil || s.Session.Watchdog == nil {
		return errs
	}
	wd := s.Session.Watchdog
	if wd.MaxBumps < 0 {
		errs = append(errs, ValidationError{
			Path:    "session.watchdog.max_bumps",
			Message: "must be >= 0",
		})
	}
	if wd.IdleTimeout > 0 && wd.IdleTimeout < time.Second {
		errs = append(errs, ValidationError{
			Path:    "session.watchdog.idle_timeout",
			Message: "must be >= 1s",
		})
	}
	// Zero idle_timeout is only valid when watchdog is disabled.
	if wd.IdleTimeout == 0 && wd.Enabled {
		errs = append(errs, ValidationError{
			Path:    "session.watchdog.idle_timeout",
			Message: "must be > 0 when watchdog is enabled",
		})
	}
	return errs
}
