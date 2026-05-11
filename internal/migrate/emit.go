package migrate

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
)

// sanitizePluginKey returns the plugin name unchanged when it matches the TOML
// safe-name pattern. It returns an error otherwise. This is a defense-in-depth
// guard at the emit boundary; the canonical fix is at read time via
// classifyPluginEntry. If a future read path forgets to classify, this guard
// prevents an unparseable [plugins.<bad-name>] from being written.
func sanitizePluginKey(name string) (string, error) {
	if !tomlSafeNamePattern.MatchString(name) {
		return "", fmt.Errorf("plugin name %q contains TOML-unsafe characters; expected pattern %s",
			name, tomlSafeNamePattern.String())
	}
	return name, nil
}

// EmitTOML converts an OpenChadState into a stack.toml string.
func EmitTOML(state *OpenChadState) (string, error) {
	var buf bytes.Buffer

	headerTmpl := `# stack.toml — migrated from open-chad
# Generated: {{ .Timestamp }}
{{ if .HasLocalSource }}# Note: Plugins with source = "local:<path>" reference absolute paths on this
# operator's machine and are not portable across hosts. Migrate or rewrite
# them when sharing this stack.
{{ end }}{{ range .Warnings }}# Warning: {{ . }}
{{ end }}{{ range .Skipped }}# Skipped: {{ . }}
{{ end }}
[meta]
version = "1.0.0"
name = "migrated"
description = "Migrated from open-chad"

{{ .Body }}
`

	body := emitBody(state)

	hasLocalSource := false
	for _, p := range state.Plugins {
		if strings.HasPrefix(p.Source, "local:") {
			hasLocalSource = true
			break
		}
	}

	data := struct {
		Timestamp      string
		Warnings       []string
		Skipped        []string
		Body           string
		HasLocalSource bool
	}{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Warnings:       state.Warnings,
		Skipped:        state.SkippedSources,
		Body:           body,
		HasLocalSource: hasLocalSource,
	}

	tmpl, err := template.New("header").Parse(headerTmpl)
	if err != nil {
		return "", fmt.Errorf("parse header template: %w", err)
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute header template: %w", err)
	}

	return buf.String(), nil
}

// EmitTOMLToFile writes the emitted TOML to a file path.
func EmitTOMLToFile(state *OpenChadState, path string) error {
	content, err := EmitTOML(state)
	if err != nil {
		return err
	}
	return writeFile(path, []byte(content))
}

// emitBody generates the TOML body (all sections after meta).
func emitBody(state *OpenChadState) string {
	var parts []string

	// MCP servers — skip servers that cannot satisfy OCA's schema port
	// constraint (port required and ∈ [6275, 6325]). External services
	// with no Vision-side port (like https://mcp.grep.app) fall here and
	// must be hand-added post-migration.
	const (
		visionPortMin = 6275
		visionPortMax = 6325
	)
	if len(state.MCPServers) > 0 {
		var serverParts []string
		for name, srv := range state.MCPServers {
			if srv.Port == 0 || srv.Port < visionPortMin || srv.Port > visionPortMax {
				state.Warnings = append(state.Warnings, fmt.Sprintf(
					"mcp.%s: port %d outside Vision range [%d, %d]; server skipped (hand-add post-migration)",
					name, srv.Port, visionPortMin, visionPortMax))
				continue
			}
			serverParts = append(serverParts, fmt.Sprintf("[mcp.servers.%s]", name))
			serverParts = append(serverParts, fmt.Sprintf("port = %d", srv.Port))
			if srv.Type != "" {
				serverParts = append(serverParts, fmt.Sprintf("type = %q", srv.Type))
			}
			if srv.URL != "" {
				serverParts = append(serverParts, fmt.Sprintf("url = %q", srv.URL))
			}
			if srv.Command != "" {
				serverParts = append(serverParts, fmt.Sprintf("command = %q", srv.Command))
			}
			if len(srv.Args) > 0 {
				serverParts = append(serverParts, fmt.Sprintf("args = %s", stringArray(srv.Args)))
			}
			if srv.Timeout > 0 {
				serverParts = append(serverParts, fmt.Sprintf("timeout = %d", srv.Timeout))
			}
			if srv.Autostart {
				serverParts = append(serverParts, "autostart = true")
			}
			if srv.Required {
				serverParts = append(serverParts, "required = true")
			}
			if srv.Source != "" {
				serverParts = append(serverParts, fmt.Sprintf("source = %q", srv.Source))
			}
			if srv.EnvFile != "" {
				serverParts = append(serverParts, fmt.Sprintf("env_file = %q", srv.EnvFile))
			}
			if len(srv.Env) > 0 {
				serverParts = append(serverParts, fmt.Sprintf("[mcp.servers.%s.env]", name))
				for k, v := range srv.Env {
					serverParts = append(serverParts, fmt.Sprintf("%s = %q", k, v))
				}
			}
			serverParts = append(serverParts, "")
		}
		if len(serverParts) > 0 {
			parts = append(parts, "# ─── MCP servers ─────────────────────────────────────────────────────────────")
			parts = append(parts, serverParts...)
		}
	}

	// Slot groups
	if len(state.SlotGroups) > 0 {
		parts = append(parts, "# ─── MCP slot groups ─────────────────────────────────────────────────────────")
		for name, sg := range state.SlotGroups {
			parts = append(parts, fmt.Sprintf("[mcp.slot_groups.%s]", name))
			if len(sg.Servers) > 0 {
				parts = append(parts, fmt.Sprintf("servers = %s", stringArray(sg.Servers)))
			}
			if sg.GroupPort > 0 {
				parts = append(parts, fmt.Sprintf("group_port = %d", sg.GroupPort))
			}
			if sg.Template != "" {
				parts = append(parts, fmt.Sprintf("template = %q", sg.Template))
			}
			// base_port and count are the canonical Vision YAML field names
			// and OCA's stack schema's required slot group fields (validate.go).
			if sg.BasePort > 0 {
				parts = append(parts, fmt.Sprintf("base_port = %d", sg.BasePort))
			}
			if sg.Count > 0 {
				parts = append(parts, fmt.Sprintf("count = %d", sg.Count))
			}
			parts = append(parts, "")
		}
	}

	// Plugins — sanitize each name defensively. classifyPluginEntry rejects
	// TOML-unsafe names at read time, but state may also be constructed
	// programmatically (tests, custom callers). Names that fail sanitization
	// are skipped with a warning rather than emitted as invalid TOML.
	if len(state.Plugins) > 0 {
		var pluginParts []string
		for _, p := range state.Plugins {
			cleanKey, err := sanitizePluginKey(p.Name)
			if err != nil {
				state.Warnings = append(state.Warnings,
					fmt.Sprintf("plugin %q has TOML-unsafe name; skipped (%v)", p.Name, err))
				continue
			}
			pluginParts = append(pluginParts, fmt.Sprintf("[plugins.%s]", cleanKey))
			if p.Source != "" {
				pluginParts = append(pluginParts, fmt.Sprintf("source = %q", p.Source))
			}
			if p.Ref != "" {
				pluginParts = append(pluginParts, fmt.Sprintf("ref = %q", p.Ref))
			}
			if p.Checkout != "" {
				pluginParts = append(pluginParts, fmt.Sprintf("checkout = %q", p.Checkout))
			}
			if len(p.Build) > 0 {
				pluginParts = append(pluginParts, fmt.Sprintf("build = %s", stringArray(p.Build)))
			}
			if len(p.Provides) > 0 {
				pluginParts = append(pluginParts, fmt.Sprintf("provides = %s", stringArray(p.Provides)))
			}
			pluginParts = append(pluginParts, "")
		}
		if len(pluginParts) > 0 {
			parts = append(parts, "# ─── Plugins ─────────────────────────────────────────────────────────────────")
			parts = append(parts, pluginParts...)
		}
	}

	// Instructions
	if len(state.Instructions) > 0 {
		parts = append(parts, "# ─── Instructions ────────────────────────────────────────────────────────────")
		parts = append(parts, "[instructions]")
		parts = append(parts, fmt.Sprintf("order = %s", stringArray(state.Instructions)))
		parts = append(parts, "")
	}

	// Providers
	if len(state.Providers) > 0 {
		parts = append(parts, "# ─── Providers ───────────────────────────────────────────────────────────────")
		for name, prov := range state.Providers {
			parts = append(parts, fmt.Sprintf("[providers.%s]", name))
			for _, m := range prov.Models {
				parts = append(parts, fmt.Sprintf("[providers.%s.models.%s]", name, m.Name))
				parts = append(parts, fmt.Sprintf("name = %q", m.Name))
				if m.Limit != nil {
					if m.Limit.Tokens != nil {
						parts = append(parts, fmt.Sprintf("context = %d", *m.Limit.Tokens))
					}
				}
				if len(m.Modalities) > 0 {
					parts = append(parts, fmt.Sprintf("inputs = %s", stringArray(m.Modalities)))
				}
			}
			parts = append(parts, "")
		}
	}

	// Permissions
	if state.Permissions != nil {
		parts = append(parts, "# ─── Permissions ─────────────────────────────────────────────────────────────")
		parts = append(parts, "[permissions]")
		if state.Permissions.Default != "" {
			parts = append(parts, fmt.Sprintf("default = %q", state.Permissions.Default))
		}
		if state.Permissions.DoomLoop != "" {
			parts = append(parts, fmt.Sprintf("doom_loop = %q", state.Permissions.DoomLoop))
		}
		parts = append(parts, "")
	}

	// Watcher
	if state.Watcher != nil && len(state.Watcher.Ignore) > 0 {
		parts = append(parts, "# ─── Watcher ─────────────────────────────────────────────────────────────────")
		parts = append(parts, "[watcher]")
		parts = append(parts, fmt.Sprintf("ignore = %s", stringArray(state.Watcher.Ignore)))
		parts = append(parts, "")
	}

	// LSP
	if len(state.LSP) > 0 {
		parts = append(parts, "# ─── LSP ─────────────────────────────────────────────────────────────────────")
		for name, lsp := range state.LSP {
			parts = append(parts, fmt.Sprintf("[lsp.%s]", name))
			if lsp.Command != "" {
				cmd := append([]string{lsp.Command}, lsp.Args...)
				parts = append(parts, fmt.Sprintf("command = %s", stringArray(cmd)))
			}
			parts = append(parts, "")
		}
	}

	// Skills
	if len(state.Skills) > 0 {
		parts = append(parts, "# ─── Skills ──────────────────────────────────────────────────────────────────")
		parts = append(parts, "[skills]")
		parts = append(parts, fmt.Sprintf("order = %s", stringArray(state.Skills)))
		parts = append(parts, "")
	}

	// OpenCode toggles
	if state.OpenCode != nil {
		parts = append(parts, "# ─── OpenCode toggles ────────────────────────────────────────────────────────")
		parts = append(parts, "[opencode]")
		if state.OpenCode.Theme != "" {
			parts = append(parts, fmt.Sprintf("theme = %q", state.OpenCode.Theme))
		}
		if state.OpenCode.DefaultAgent != "" {
			parts = append(parts, fmt.Sprintf("default_agent = %q", state.OpenCode.DefaultAgent))
		}
		parts = append(parts, "")
	}

	return strings.Join(parts, "\n")
}

// stringArray formats a []string as a TOML array: ["a", "b", "c"]
func stringArray(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	var parts []string
	for _, s := range ss {
		parts = append(parts, fmt.Sprintf("%q", s))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
