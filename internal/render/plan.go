package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanMCP builds the deterministic render plan for opencode.json + vision/servers.yaml.
func PlanMCP(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	declared := map[string]Fragment{}
	for name, srv := range stack.MCP.Servers {
		declared[name] = RenderMCPFragment(srv)
	}
	for name, group := range stack.MCP.SlotGroups {
		declared[name] = RenderSlotGroupFragment(group)
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
	reason := "merge declared plugin entries into .plugin preserving user-added entries and pruning stale worktree paths"
	if merged.PrunedCount > 0 {
		reason = fmt.Sprintf("pruned %d stale worktree plugin entrie(s) while merging declared plugin entries", merged.PrunedCount)
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
			Reason:         reason,
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

// AssetsSkillsRoot returns the path to the OCA-owned skills source directory.
// In production this is <repo_root>/assets/skills/. The repo root is detected
// from the binary location (embedded in the oca binary at build time) or
// overridden by the OCA_ASSETS_ROOT env var for testing.
func AssetsSkillsRoot() string {
	if env := os.Getenv("OCA_ASSETS_ROOT"); env != "" {
		return filepath.Join(env, "skills")
	}
	// Default: relative to executable. In tests, use OCA_ASSETS_ROOT.
	exe, err := os.Executable()
	if err != nil {
		return "assets/skills"
	}
	repoRoot := filepath.Dir(filepath.Dir(exe))
	return filepath.Join(repoRoot, "assets", "skills")
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

// TargetName identifies a render target for composeApplyPlan and CLI.
type TargetName string

const (
	TargetMCP          TargetName = "mcp"
	TargetPlugins      TargetName = "plugins"
	TargetInstructions TargetName = "instructions"
	TargetProviders    TargetName = "providers"
	TargetPermissions  TargetName = "permissions"
	TargetWatcher      TargetName = "watcher"
	TargetLSP          TargetName = "lsp"
	TargetSkills       TargetName = "skills"
	TargetCommands     TargetName = "commands"
	TargetFormatters   TargetName = "formatters"
	TargetToggles      TargetName = "toggles"
)

// AllTargets is the ordered list of all apply-able targets in dependency order.
// Design K9. Phase 3.5 adds skills, commands, formatters, toggles after Phase 3.
var AllTargets = []TargetName{
	TargetMCP,
	TargetPlugins,
	TargetInstructions,
	TargetProviders,
	TargetPermissions,
	TargetWatcher,
	TargetLSP,
	TargetSkills,
	TargetCommands,
	TargetFormatters,
	TargetToggles,
}

// planForTarget calls the appropriate PlanX function for the given target.
func planForTarget(stack *cfg.Stack, paths cfg.Paths, source string, target TargetName) (*Plan, error) {
	switch target {
	case TargetMCP:
		return PlanMCP(stack, paths, source)
	case TargetPlugins:
		return PlanPlugins(stack, paths, source)
	case TargetInstructions:
		return PlanInstructions(stack, paths, source)
	case TargetProviders:
		return PlanProviders(stack, paths, source)
	case TargetPermissions:
		return PlanPermissions(stack, paths, source)
	case TargetWatcher:
		return PlanWatcher(stack, paths, source)
	case TargetLSP:
		return PlanLSP(stack, paths, source)
	case TargetSkills:
		return PlanSkills(stack, paths, source, AssetsSkillsRoot())
	case TargetCommands:
		return PlanCommands(stack, paths, source)
	case TargetFormatters:
		return PlanFormatters(stack, paths, source)
	case TargetToggles:
		return PlanToggles(stack, paths, source)
	default:
		return nil, fmt.Errorf("unknown target: %s", target)
	}
}

func composeTargetOps(stack *cfg.Stack, paths cfg.Paths, source string, target TargetName, currentDoc []byte) ([]TargetOp, []byte, error) {
	switch target {
	case TargetMCP:
		declared := map[string]Fragment{}
		for name, srv := range stack.MCP.Servers {
			declared[name] = RenderMCPFragment(srv)
		}
		for name, group := range stack.MCP.SlotGroups {
			declared[name] = RenderSlotGroupFragment(group)
		}
		opAfter, err := MergeMCP(currentDoc, declared)
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, opAfter) {
			opOp = "noop"
		}

		visionPath := paths.VisionServersYAML()
		visionBefore, err := readIfExists(visionPath)
		if err != nil {
			return nil, nil, err
		}
		visionAfter, err := RenderVisionServers(stack, source)
		if err != nil {
			return nil, nil, err
		}
		visionOp := "write"
		if bytes.Equal(visionBefore, visionAfter) {
			visionOp = "noop"
		}

		return []TargetOp{
			{Name: "opencode.json", Path: paths.OpencodeJSON(), Op: opOp, Before: currentDoc, After: opAfter, Mode: 0o644, Reason: "merge declared MCP entries into .mcp preserving user-added keys"},
			{Name: "vision/servers.yaml", Path: visionPath, Op: visionOp, Before: visionBefore, After: visionAfter, Mode: 0o600, Reason: "write authoritative Vision server registry"},
		}, opAfter, nil
	case TargetPlugins:
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
		merged, err := MergeArray(currentDoc, "plugin", declared, MergeArrayOptions{StripStaleWorktree: true})
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged.Bytes) {
			opOp = "noop"
		}
		reason := "merge declared plugin entries into .plugin preserving user-added entries and pruning stale worktree paths"
		if merged.PrunedCount > 0 {
			reason = fmt.Sprintf("pruned %d stale worktree plugin entrie(s) while merging declared plugin entries", merged.PrunedCount)
		}
		return []TargetOp{{
			Name:           "opencode.json",
			Path:           paths.OpencodeJSON(),
			Op:             opOp,
			Before:         currentDoc,
			After:          merged.Bytes,
			Mode:           0o644,
			SuppressBackup: hasSync,
			Reason:         reason,
		}}, merged.Bytes, nil
	case TargetInstructions:
		declared := RenderInstructionsList(stack)
		merged, err := MergeArray(currentDoc, "instructions", declared, MergeArrayOptions{})
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged.Bytes) {
			opOp = "noop"
		}
		return []TargetOp{{
			Name:   "opencode.json",
			Path:   paths.OpencodeJSON(),
			Op:     opOp,
			Before: currentDoc,
			After:  merged.Bytes,
			Mode:   0o644,
			Reason: "merge declared instruction entries into .instructions preserving user-added entries",
		}}, merged.Bytes, nil
	case TargetProviders:
		merged, err := MergeObject(currentDoc, translateProvidersToOpencode(stack.Providers))
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged) {
			opOp = "noop"
		}
		return []TargetOp{{Name: "opencode.json", Path: paths.OpencodeJSON(), Op: opOp, Before: currentDoc, After: merged, Mode: 0o644, Reason: "merge declared provider entries into .provider with shape translation"}}, merged, nil
	case TargetPermissions:
		merged, err := MergeObject(currentDoc, translatePermissionsToOpencode(stack.Permissions))
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged) {
			opOp = "noop"
		}
		return []TargetOp{{Name: "opencode.json", Path: paths.OpencodeJSON(), Op: opOp, Before: currentDoc, After: merged, Mode: 0o644, Reason: "merge declared permission entries into .permission with default→\"*\" translation"}}, merged, nil
	case TargetWatcher:
		merged, err := MergeWatcherIgnore(currentDoc, stack.Watcher.Ignore)
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged) {
			opOp = "noop"
		}
		return []TargetOp{{Name: "opencode.json", Path: paths.OpencodeJSON(), Op: opOp, Before: currentDoc, After: merged, Mode: 0o644, Reason: "merge declared watcher.ignore entries into .watcher preserving user-added entries"}}, merged, nil
	case TargetLSP:
		merged, err := MergeObject(currentDoc, translateLSPToOpencode(stack.LSP))
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged) {
			opOp = "noop"
		}
		return []TargetOp{{Name: "opencode.json", Path: paths.OpencodeJSON(), Op: opOp, Before: currentDoc, After: merged, Mode: 0o644, Reason: "merge declared LSP server entries into .lsp preserving per-server keys"}}, merged, nil
	case TargetSkills:
		// Skills are filesystem writes, not JSON merges. Return TargetOps
		// without mutating currentDoc.
		skillOps := PlanSkillsOps(stack, paths, AssetsSkillsRoot())
		return skillOps, currentDoc, nil
	case TargetCommands:
		declared := translateCommandsToOpencode(stack.Commands)
		if declared == nil {
			declared = map[string]any{}
		}
		merged, err := MergeObject(currentDoc, declared)
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged) {
			opOp = "noop"
		}
		return []TargetOp{{Name: "opencode.json", Path: paths.OpencodeJSON(), Op: opOp, Before: currentDoc, After: merged, Mode: 0o644, Reason: "merge declared command entries into .command preserving user-added commands"}}, merged, nil
	case TargetFormatters:
		declared := translateFormattersToOpencode(stack.Formatters)
		if declared == nil {
			declared = map[string]any{}
		}
		merged, err := MergeObject(currentDoc, declared)
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged) {
			opOp = "noop"
		}
		return []TargetOp{{Name: "opencode.json", Path: paths.OpencodeJSON(), Op: opOp, Before: currentDoc, After: merged, Mode: 0o644, Reason: "merge declared formatter entries into .formatter preserving user-added formatters"}}, merged, nil
	case TargetToggles:
		declared := translateOpenCodeToOpencode(stack.OpenCode)
		if len(declared) == 0 {
			return nil, currentDoc, nil
		}
		merged, err := MergeObject(currentDoc, declared)
		if err != nil {
			return nil, nil, err
		}
		opOp := "merge"
		if bytes.Equal(currentDoc, merged) {
			opOp = "noop"
		}
		return []TargetOp{{Name: "opencode.json", Path: paths.OpencodeJSON(), Op: opOp, Before: currentDoc, After: merged, Mode: 0o644, Reason: "merge declared opencode toggles into top-level keys preserving user keys"}}, merged, nil
	default:
		return nil, nil, fmt.Errorf("unknown target: %s", target)
	}
}

// ComposeApplyPlan builds a single Plan by chaining per-target Before/After bytes
// through a running in-memory opencode.json document, producing one opencode.json
// TargetOp per target in K9 order. Used by both apply-all and oca diff (shared code path).
//
// This solves the stale-Before stomp problem: if each PlanX reads from the original
// on-disk file, later targets clobber earlier ones. By chaining After bytes as the
// next target's Before, each step operates on the fully-merged state.
func ComposeApplyPlan(stack *cfg.Stack, paths cfg.Paths, source string, targets []TargetName) (*Plan, error) {
	currentDoc, err := readIfExists(paths.OpencodeJSON())
	if err != nil {
		return nil, err
	}

	var combinedTargets []TargetOp
	for _, target := range targets {
		targetOps, nextDoc, err := composeTargetOps(stack, paths, source, target, currentDoc)
		if err != nil {
			return nil, fmt.Errorf("plan %s: %w", target, err)
		}
		combinedTargets = append(combinedTargets, targetOps...)
		currentDoc = nextDoc
	}

	return &Plan{
		Source:   source,
		LockPath: paths.ApplyLockPath(),
		Targets:  combinedTargets,
	}, nil
}
