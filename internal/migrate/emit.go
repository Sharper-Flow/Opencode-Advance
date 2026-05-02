package migrate

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
)

// EmitTOML converts an OpenChadState into a stack.toml string.
func EmitTOML(state *OpenChadState) (string, error) {
	var buf bytes.Buffer

	headerTmpl := `# stack.toml — migrated from open-chad
# Generated: {{ .Timestamp }}
{{ range .Warnings }}# Warning: {{ . }}
{{ end }}{{ range .Skipped }}# Skipped: {{ . }}
{{ end }}
[meta]
version = "1.0.0"
name = "migrated"
description = "Migrated from open-chad"

{{ .Body }}
`

	body := emitBody(state)

	data := struct {
		Timestamp string
		Warnings  []string
		Skipped   []string
		Body      string
	}{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Warnings:  state.Warnings,
		Skipped:   state.SkippedSources,
		Body:      body,
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

	// MCP servers
	if len(state.MCPServers) > 0 {
		parts = append(parts, "# ─── MCP servers ─────────────────────────────────────────────────────────────")
		for name, srv := range state.MCPServers {
			parts = append(parts, fmt.Sprintf("[mcp.servers.%s]", name))
			if srv.Port > 0 {
				parts = append(parts, fmt.Sprintf("port = %d", srv.Port))
			}
			if srv.Type != "" {
				parts = append(parts, fmt.Sprintf("type = %q", srv.Type))
			}
			if srv.Command != "" {
				parts = append(parts, fmt.Sprintf("command = %q", srv.Command))
			}
			if len(srv.Args) > 0 {
				parts = append(parts, fmt.Sprintf("args = %s", stringArray(srv.Args)))
			}
			if srv.Timeout > 0 {
				parts = append(parts, fmt.Sprintf("timeout = %d", srv.Timeout))
			}
			if srv.Autostart {
				parts = append(parts, "autostart = true")
			}
			if srv.Required {
				parts = append(parts, "required = true")
			}
			if srv.Source != "" {
				parts = append(parts, fmt.Sprintf("source = %q", srv.Source))
			}
			if srv.EnvFile != "" {
				parts = append(parts, fmt.Sprintf("env_file = %q", srv.EnvFile))
			}
			if len(srv.Env) > 0 {
				parts = append(parts, fmt.Sprintf("[mcp.servers.%s.env]", name))
				for k, v := range srv.Env {
					parts = append(parts, fmt.Sprintf("%s = %q", k, v))
				}
			}
			parts = append(parts, "")
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
			if sg.MinSlots > 0 {
				parts = append(parts, fmt.Sprintf("min_slots = %d", sg.MinSlots))
			}
			if sg.MaxSlots > 0 {
				parts = append(parts, fmt.Sprintf("max_slots = %d", sg.MaxSlots))
			}
			parts = append(parts, "")
		}
	}

	// Plugins
	if len(state.Plugins) > 0 {
		parts = append(parts, "# ─── Plugins ─────────────────────────────────────────────────────────────────")
		for _, p := range state.Plugins {
			parts = append(parts, fmt.Sprintf("[plugins.%s]", p.Name))
			if p.Source != "" {
				parts = append(parts, fmt.Sprintf("source = %q", p.Source))
			}
			if p.Ref != "" {
				parts = append(parts, fmt.Sprintf("ref = %q", p.Ref))
			}
			if p.Checkout != "" {
				parts = append(parts, fmt.Sprintf("checkout = %q", p.Checkout))
			}
			if len(p.Build) > 0 {
				parts = append(parts, fmt.Sprintf("build = %s", stringArray(p.Build)))
			}
			if len(p.Provides) > 0 {
				parts = append(parts, fmt.Sprintf("provides = %s", stringArray(p.Provides)))
			}
			parts = append(parts, "")
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
