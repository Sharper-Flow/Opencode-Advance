package render

import (
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanToggles builds the render plan for opencode.json top-level toggle keys
// from the [opencode] section (theme, default_agent, share, snapshot, autoupdate,
// compaction, disabled_providers, enabled_providers, plus Extra passthrough).
func PlanToggles(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	declared := translateOpenCodeToOpencode(stack.OpenCode)
	if len(declared) == 0 {
		// No opencode section — skip (no-op plan).
		return &Plan{
			Source:   source,
			LockPath: paths.ApplyLockPath(),
		}, nil
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
			Reason: "merge declared opencode toggles into top-level keys preserving user keys",
		}},
	}, nil
}

// translateOpenCodeToOpencode converts the typed OpenCodeSection into
// top-level opencode.json keys. Handles the autoupdate bool/string union,
// compaction passthrough, provider lists, and Extra passthrough.
func translateOpenCodeToOpencode(oc cfg.OpenCodeSection) map[string]any {
	result := make(map[string]any)

	if oc.Theme != "" {
		result["theme"] = oc.Theme
	}
	if oc.DefaultAgent != "" {
		result["default_agent"] = oc.DefaultAgent
	}
	if oc.Share != "" {
		result["share"] = oc.Share
	}
	if oc.Snapshot != nil {
		result["snapshot"] = *oc.Snapshot
	}
	if oc.Autoupdate != nil {
		if oc.Autoupdate.Bool != nil {
			result["autoupdate"] = *oc.Autoupdate.Bool
		} else if oc.Autoupdate.Str != nil {
			result["autoupdate"] = *oc.Autoupdate.Str
		}
	}
	if oc.Compaction != nil {
		comp := make(map[string]any)
		if oc.Compaction.Auto != nil {
			comp["auto"] = *oc.Compaction.Auto
		}
		if oc.Compaction.Prune != nil {
			comp["prune"] = *oc.Compaction.Prune
		}
		if oc.Compaction.Reserved > 0 {
			comp["reserved"] = oc.Compaction.Reserved
		}
		// Extra passthrough for compaction
		if oc.Compaction.Extra != nil {
			for k, v := range oc.Compaction.Extra {
				comp[k] = v
			}
		}
		result["compaction"] = comp
	}
	if oc.DisabledProviders != nil && len(oc.DisabledProviders.List) > 0 {
		result["disabled_providers"] = map[string]any{"list": stringSliceToAny(oc.DisabledProviders.List)}
	}
	if oc.EnabledProviders != nil && len(oc.EnabledProviders.List) > 0 {
		result["enabled_providers"] = map[string]any{"list": stringSliceToAny(oc.EnabledProviders.List)}
	}
	// Extra passthrough for top-level opencode keys
	if oc.Extra != nil {
		for k, v := range oc.Extra {
			result[k] = v
		}
	}

	return result
}
