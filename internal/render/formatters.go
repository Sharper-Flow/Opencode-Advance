package render

import (
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanFormatters builds the render plan for opencode.json .formatter section.
// Uses MergeObject for per-formatter preservation with Extra passthrough.
func PlanFormatters(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	declared := translateFormattersToOpencode(stack.Formatters)
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
			Reason: "merge declared formatter entries into .formatter preserving user-added formatters",
		}},
	}, nil
}

// translateFormattersToOpencode converts the typed FormattersSection into
// opencode.json .formatter shape with command/extensions/disabled/environment
// passthrough and Extra field passthrough.
func translateFormattersToOpencode(formatters cfg.FormattersSection) map[string]any {
	if formatters == nil || len(formatters) == 0 {
		return nil
	}
	fmtMap := make(map[string]any, len(formatters))
	for name, f := range formatters {
		entry := make(map[string]any)
		if len(f.Command) > 0 {
			entry["command"] = stringSliceToAny(f.Command)
		}
		if len(f.Extensions) > 0 {
			entry["extensions"] = stringSliceToAny(f.Extensions)
		}
		if f.Disabled {
			entry["disabled"] = true
		}
		if len(f.Environment) > 0 {
			entry["environment"] = stringMapToAny(f.Environment)
		}
		// Extra passthrough
		if f.Extra != nil {
			for k, v := range f.Extra {
				entry[k] = v
			}
		}
		fmtMap[name] = entry
	}
	return map[string]any{"formatter": fmtMap}
}

// stringSliceToAny converts []string to []any for JSON compatibility.
func stringSliceToAny(s []string) []any {
	if s == nil {
		return nil
	}
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

// stringMapToAny converts map[string]string to map[string]any for JSON compatibility.
func stringMapToAny(m map[string]string) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
