package render

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanMCP builds the deterministic render plan for opencode.json + vision/servers.yaml.
func PlanMCP(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	declared := map[string]Fragment{}
	for name, srv := range stack.MCP.Servers {
		declared[name] = RenderMCPFragment(srv)
	}
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}
	opAfter, err := MergeMCP(opBefore, declared)
	if err != nil {
		return nil, err
	}
	opOp := "merge"
	if bytes.Equal(opBefore, opAfter) {
		opOp = "noop"
	}

	visionPath := paths.VisionServersYAML()
	visionBefore, err := readIfExists(visionPath)
	if err != nil {
		return nil, err
	}
	visionAfter, err := RenderVisionServers(stack, source)
	if err != nil {
		return nil, err
	}
	visionOp := "write"
	if bytes.Equal(visionBefore, visionAfter) {
		visionOp = "noop"
	}

	return &Plan{
		Source:   source,
		LockPath: paths.ApplyLockPath(),
		Targets: []TargetOp{
			{Name: "opencode.json", Path: opPath, Op: opOp, Before: opBefore, After: opAfter, Mode: 0o644, Reason: "merge declared MCP entries into .mcp preserving user-added keys"},
			{Name: "vision/servers.yaml", Path: visionPath, Op: visionOp, Before: visionBefore, After: visionAfter, Mode: 0o600, Reason: "write authoritative Vision server registry"},
		},
	}, nil
}

// PlanPlugins builds deterministic render plan for opencode.json .plugin.
func PlanPlugins(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	declared := make([]string, 0, len(stack.Plugins))
	hasSync := false
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
		declared = append(declared, RenderPluginFragment(plugin))
		if plugin.Sync != "" {
			hasSync = true
		}
	}

	merged, err := MergeArray(opBefore, "plugin", declared, MergeArrayOptions{StripStaleWorktree: true})
	if err != nil {
		return nil, err
	}
	opOp := "merge"
	if bytes.Equal(opBefore, merged.Bytes) {
		opOp = "noop"
	}

	return &Plan{
		Source:   source,
		LockPath: paths.ApplyLockPath(),
		Targets: []TargetOp{{
			Name:           "opencode.json",
			Path:           opPath,
			Op:             opOp,
			Before:         opBefore,
			After:          merged.Bytes,
			Mode:           0o644,
			SuppressBackup: hasSync,
			Reason:         "merge declared plugin entries into .plugin preserving user-added entries and pruning stale worktree paths",
		}},
	}, nil
}

// PlanInstructions builds deterministic render plan for opencode.json .instructions.
func PlanInstructions(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	declared := RenderInstructionsList(stack)
	merged, err := MergeArray(opBefore, "instructions", declared, MergeArrayOptions{})
	if err != nil {
		return nil, err
	}
	opOp := "merge"
	if bytes.Equal(opBefore, merged.Bytes) {
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
			After:  merged.Bytes,
			Mode:   0o644,
			Reason: "merge declared instruction entries into .instructions preserving user-added entries",
		}},
	}, nil
}

func readIfExists(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		return b, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, fmt.Errorf("read %s: %w", path, err)
}
