package health

import (
	"context"
	"fmt"
	"sort"
	"strings"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func init() {
	registerBuiltin("tool-suites", CheckToolSuites)
}

// CheckToolSuites validates MCP tool suite definitions against declared
// servers and reports per-suite server availability. It also reports
// per-suite membership counts for doctor visibility.
func CheckToolSuites(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	var checks []Check

	if len(stack.MCP.ToolSuites) == 0 {
		checks = append(checks, Check{
			Name:    "tool-suites",
			Status:  StatusPass,
			Message: "No MCP tool suites defined in stack.toml — tool suite audit skipped",
		})
		return checks, nil
	}

	// Build declared server set
	declaredServers := make(map[string]bool)
	for name := range stack.MCP.Servers {
		declaredServers[name] = true
	}

	// Validate each suite
	suiteNames := sortedStringKeys(stack.MCP.ToolSuites)
	for _, suiteName := range suiteNames {
		suite := stack.MCP.ToolSuites[suiteName]

		var missing []string
		for _, srv := range suite.Servers {
			if !declaredServers[srv] {
				missing = append(missing, srv)
			}
		}

		if len(missing) > 0 {
			checks = append(checks, Check{
				Name:    fmt.Sprintf("tool-suites.%s", suiteName),
				Status:  StatusWarn,
				Message: fmt.Sprintf("Suite %q references undeclared servers: %s", suiteName, strings.Join(missing, ", ")),
				Hint:    fmt.Sprintf("add [mcp.servers.%s] or remove from suite", missing[0]),
			})
		} else {
			desc := suite.Description
			if desc == "" {
				desc = fmt.Sprintf("%d servers", len(suite.Servers))
			}
			checks = append(checks, Check{
				Name:    fmt.Sprintf("tool-suites.%s", suiteName),
				Status:  StatusPass,
				Message: fmt.Sprintf("Suite %q: %s (%s)", suiteName, strings.Join(suite.Servers, ", "), desc),
			})
		}
	}

	// Report servers not covered by any suite (informational)
	covered := make(map[string]bool)
	for _, suite := range stack.MCP.ToolSuites {
		for _, srv := range suite.Servers {
			covered[srv] = true
		}
	}

	var uncovered []string
	for name := range stack.MCP.Servers {
		if !covered[name] {
			uncovered = append(uncovered, name)
		}
	}

	if len(uncovered) > 0 {
		sort.Strings(uncovered)
		checks = append(checks, Check{
			Name:    "tool-suites.uncovered-servers",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Servers not in any suite: %s", strings.Join(uncovered, ", ")),
			Hint:    "consider adding these to a suite or they will be excluded from agent profile rendering",
		})
	}

	return checks, nil
}

func sortedStringKeys(m map[string]cfg.ToolSuite) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
