package render

import (
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanCommands builds the render plan for opencode.json .command section.
// Uses MergeObject for per-command preservation with Extra passthrough.
func PlanCommands(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	declared := translateCommandsToOpencode(stack.Commands)
	if declared == nil {
		declared = map[string]any{}
	}

	merged, err := MergeObject(opBefore, declared)
	if err != nil {
		return nil, err
	}
	opOp := "merge"
	if len(opBefore) > 0 && string(opBefore) == string(merged) {
		opOp = "noop"
	}

	return &Plan{
		Source:   source,
		LockPath: paths.ApplyLockPath(),
		Targets: []TargetOp{{
			Name:   "opencode.json",
			Path:   opPath,
			Op:     opOp,
			Before: opBefore,
			After:  merged,
			Mode:   0o644,
			Reason: "merge declared command entries into .command preserving user-added commands",
		}},
	}, nil
}

// translateCommandsToOpencode converts the typed CommandsSection into
// opencode.json .command shape with description/template/agent/model/subtask
// passthrough and Extra field passthrough.
func translateCommandsToOpencode(commands cfg.CommandsSection) map[string]any {
	if commands == nil || len(commands) == 0 {
		return nil
	}
	cmdMap := make(map[string]any, len(commands))
	for name, cmd := range commands {
		entry := make(map[string]any)
		entry["description"] = cmd.Description
		entry["template"] = cmd.Template
		if cmd.Agent != "" {
			entry["agent"] = cmd.Agent
		}
		if cmd.Model != "" {
			entry["model"] = cmd.Model
		}
		if cmd.Subtask != nil {
			entry["subtask"] = *cmd.Subtask
		}
		// Extra passthrough
		if cmd.Extra != nil {
			for k, v := range cmd.Extra {
				entry[k] = v
			}
		}
		cmdMap[name] = entry
	}
	return map[string]any{"command": cmdMap}
}
