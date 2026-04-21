package render

import (
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanProviders builds the render plan for opencode.json .provider section.
// Shape translation (design K3):
//
//	context/output → limit.context / limit.output
//	inputs/outputs → modalities.input / modalities.output
//	options/variants → passthrough unchanged
//	Extra fields → passthrough unchanged
func PlanProviders(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	declared := translateProvidersToOpencode(stack.Providers)

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
			Reason: "merge declared provider entries into .provider with shape translation",
		}},
	}, nil
}

// translateProvidersToOpencode converts the typed ProvidersSection into the
// opencode.json .provider shape per design K3.
func translateProvidersToOpencode(providers cfg.ProvidersSection) map[string]any {
	if providers == nil {
		return nil
	}
	providerMap := make(map[string]any, len(providers))
	for pname, prov := range providers {
		provMap := translateProvider(prov)
		providerMap[pname] = provMap
	}
	return map[string]any{"provider": providerMap}
}

// translateProvider converts one Provider into opencode.json format.
func translateProvider(p cfg.Provider) map[string]any {
	out := make(map[string]any)
	if p.Models != nil {
		modelsOut := make(map[string]any, len(p.Models))
		for mname, model := range p.Models {
			modelsOut[mname] = translateModel(model)
		}
		out["models"] = modelsOut
	}
	if p.Options != nil {
		out["options"] = p.Options
	}
	if p.Variants != nil {
		out["variants"] = p.Variants
	}
	if p.Extra != nil {
		for k, v := range p.Extra {
			out[k] = v
		}
	}
	return out
}

// translateModel converts one ProviderModel into opencode.json format.
// context → limit.context, output → limit.output, inputs → modalities.input,
// outputs → modalities.output.
func translateModel(m cfg.ProviderModel) map[string]any {
	out := make(map[string]any)
	if m.Name != "" {
		out["name"] = m.Name
	}
	// Shape translation: context/output → limit.*
	limit := make(map[string]any)
	if m.Context > 0 {
		limit["context"] = m.Context
	}
	if m.Output > 0 {
		limit["output"] = m.Output
	}
	if len(limit) > 0 {
		out["limit"] = limit
	}
	// Shape translation: inputs/outputs → modalities.*
	modalities := make(map[string]any)
	if len(m.Inputs) > 0 {
		modalities["input"] = m.Inputs
	}
	if len(m.Outputs) > 0 {
		modalities["output"] = m.Outputs
	}
	if len(modalities) > 0 {
		out["modalities"] = modalities
	}
	// options/variants passthrough
	if m.Options != nil {
		out["options"] = m.Options
	}
	if m.Variants != nil {
		out["variants"] = m.Variants
	}
	// Extra passthrough
	if m.Extra != nil {
		for k, v := range m.Extra {
			out[k] = v
		}
	}
	return out
}
