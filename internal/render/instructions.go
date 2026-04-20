package render

import (
	"sort"
	"strings"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// RenderInstructionsList emits base instructions order followed by enabled
// plugin-provided instructions in deterministic plugin-name order. When any
// plugin provides adv-instructions, ADV_INSTRUCTIONS.md entries are omitted
// from plugin instruction append because sync-global.sh patches them later.
func RenderInstructionsList(stack *cfg.Stack) []string {
	if stack == nil {
		return nil
	}

	out := append([]string(nil), stack.Instructions.Order...)
	suppressAdvInstructions := anyProvides(stack.Plugins, cfg.ProvidesInstructions)

	names := make([]string, 0, len(stack.Plugins))
	for name := range stack.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		plugin := stack.Plugins[name]
		if !plugin.IsEnabled() {
			continue
		}
		for _, path := range plugin.Instructions {
			if suppressAdvInstructions && strings.HasSuffix(path, "ADV_INSTRUCTIONS.md") {
				continue
			}
			out = append(out, path)
		}
	}

	return out
}

func anyProvides(plugins cfg.PluginsSection, category cfg.ProvidesCategory) bool {
	for _, plugin := range plugins {
		for _, provided := range plugin.Provides {
			if provided == category {
				return true
			}
		}
	}
	return false
}
