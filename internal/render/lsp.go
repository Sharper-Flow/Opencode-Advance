package render

import (
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanLSP builds the render plan for opencode.json .lsp section.
// Uses MergeObject for per-server preservation (command/extensions/disabled)
// with Extra passthrough for future fields.
func PlanLSP(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	declared := translateLSPToOpencode(stack.LSP)

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
			Reason: "merge declared LSP server entries into .lsp preserving per-server keys",
		}},
	}, nil
}

// translateLSPToOpencode converts the typed LSPSection into opencode.json .lsp shape.
func translateLSPToOpencode(lsp cfg.LSPSection) map[string]any {
	if lsp == nil {
		return nil
	}
	lspMap := make(map[string]any, len(lsp))
	for name, srv := range lsp {
		srvMap := make(map[string]any)
		if len(srv.Command) > 0 {
			srvMap["command"] = srv.Command
		}
		if len(srv.Extensions) > 0 {
			srvMap["extensions"] = srv.Extensions
		}
		if srv.Disabled {
			srvMap["disabled"] = true
		}
		// Extra passthrough
		if srv.Extra != nil {
			for k, v := range srv.Extra {
				srvMap[k] = v
			}
		}
		lspMap[name] = srvMap
	}
	return map[string]any{"lsp": lspMap}
}
