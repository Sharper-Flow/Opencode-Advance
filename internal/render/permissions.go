package render

import (
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// PlanPermissions builds the render plan for opencode.json .permission section.
// Shape translation (design K3):
//
//	permissions.default → .permission["*"]
//	permissions.doom_loop → .permission.doom_loop
//	permissions.external_directory / permissions.bash → passthrough
//	Extra → passthrough unchanged
func PlanPermissions(stack *cfg.Stack, paths cfg.Paths, source string) (*Plan, error) {
	opPath := paths.OpencodeJSON()
	opBefore, err := readIfExists(opPath)
	if err != nil {
		return nil, err
	}

	declared := translatePermissionsToOpencode(stack.Permissions)

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
			Reason: "merge declared permission entries into .permission with default→\"*\" translation",
		}},
	}, nil
}

// translatePermissionsToOpencode converts the typed PermissionsSection into the
// opencode.json .permission shape per design K3.
func translatePermissionsToOpencode(perms cfg.PermissionsSection) map[string]any {
	permissionMap := make(map[string]any)
	// default → "*"
	if perms.Default != "" {
		permissionMap["*"] = perms.Default
	}
	// doom_loop → "doom_loop" key
	if perms.DoomLoop != "" {
		permissionMap["doom_loop"] = perms.DoomLoop
	}
	// external_directory → passthrough (nested map string→string)
	if len(perms.ExternalDirectory) > 0 {
		ed := make(map[string]string, len(perms.ExternalDirectory))
		for k, v := range perms.ExternalDirectory {
			ed[k] = v
		}
		permissionMap["external_directory"] = ed
	}
	// bash → passthrough (nested map string→string)
	if len(perms.Bash) > 0 {
		bash := make(map[string]string, len(perms.Bash))
		for k, v := range perms.Bash {
			bash[k] = v
		}
		permissionMap["bash"] = bash
	}
	// Extra passthrough
	if perms.Extra != nil {
		for k, v := range perms.Extra {
			permissionMap[k] = v
		}
	}
	if len(permissionMap) == 0 {
		return nil
	}
	return map[string]any{"permission": permissionMap}
}
