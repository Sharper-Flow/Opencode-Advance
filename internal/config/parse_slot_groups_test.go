package config

import "testing"

// TestParse_SlotGroupsSectionTyped asserts that [mcp.slot_groups.<name>]
// blocks parse into the typed MCPSection.SlotGroups map mirroring Vision's
// SlotGroupConfig schema (template, base_port, count, group_port, defaults).
//
// This is the red phase for tk-677c6a72 — adding the SlotGroup type and
// MCPSection.SlotGroups field so OCA can declare Vision slot groups in
// stack.toml with the same shape Vision expects in servers.yaml.
func TestParse_SlotGroupsSectionTyped(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.context7]
port = 6276
command = "context7-mcp"

[mcp.slot_groups.playwright-headless]
template   = "playwright-headless-slot"
base_port  = 6301
count      = 4
group_port = 6300

[mcp.slot_groups.playwright-headless.defaults]
command = "npx"
args    = ["-y", "@playwright/mcp@latest", "--browser", "chromium", "--no-sandbox", "--headless", "--isolated"]
autostart = true
session_timeout = "10m"
`
	stack := mustParse(t, src)

	if stack.MCP.SlotGroups == nil {
		t.Fatal("MCP.SlotGroups is nil; expected typed map")
	}

	group, ok := stack.MCP.SlotGroups["playwright-headless"]
	if !ok {
		t.Fatalf("playwright-headless missing; got groups: %v", slotGroupKeys(stack))
	}

	if group.Template != "playwright-headless-slot" {
		t.Errorf("Template = %q, want playwright-headless-slot", group.Template)
	}
	if group.BasePort != 6301 {
		t.Errorf("BasePort = %d, want 6301", group.BasePort)
	}
	if group.Count != 4 {
		t.Errorf("Count = %d, want 4", group.Count)
	}
	if group.GroupPort != 6300 {
		t.Errorf("GroupPort = %d, want 6300", group.GroupPort)
	}

	if group.Defaults == nil {
		t.Fatal("Defaults is nil; expected nested defaults block to parse")
	}
	if group.Defaults.Command != "npx" {
		t.Errorf("Defaults.Command = %q, want npx", group.Defaults.Command)
	}
	if got, want := len(group.Defaults.Args), 7; got != want {
		t.Errorf("Defaults.Args len = %d, want %d", got, want)
	}
	if !group.Defaults.Autostart {
		t.Errorf("Defaults.Autostart = false, want true")
	}
	if group.Defaults.SessionTimeout != "10m" {
		t.Errorf("Defaults.SessionTimeout = %q, want 10m", group.Defaults.SessionTimeout)
	}

	// Existing servers must continue to parse alongside slot groups.
	if _, ok := stack.MCP.Servers["context7"]; !ok {
		t.Error("regular server context7 lost when slot_groups present")
	}
}

// TestParse_SlotGroupsSectionAbsent asserts that omitting slot_groups
// leaves MCPSection.SlotGroups as nil/empty without error.
func TestParse_SlotGroupsSectionAbsent(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.context7]
port = 6276
command = "context7-mcp"
`
	stack := mustParse(t, src)
	if len(stack.MCP.SlotGroups) != 0 {
		t.Errorf("SlotGroups should be empty when absent; got %d entries", len(stack.MCP.SlotGroups))
	}
}

func slotGroupKeys(s *Stack) []string {
	keys := make([]string, 0, len(s.MCP.SlotGroups))
	for k := range s.MCP.SlotGroups {
		keys = append(keys, k)
	}
	return keys
}
