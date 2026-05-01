package render

import "os"

// Fragment is one rendered MCP entry under opencode.json .mcp.
type Fragment map[string]any

// TargetOp is one planned file operation.
type TargetOp struct {
	Name           string
	Path           string
	Op             string // merge | write | noop
	Before         []byte
	After          []byte
	Mode           os.FileMode
	BackupPath     string
	SuppressBackup bool
	Reason         string
}

// Plan is the deterministic render plan for one apply run.
type Plan struct {
	Source   string
	LockPath string
	Targets  []TargetOp
}

// ApplyOptions controls write execution.
type ApplyOptions struct {
	DryRun     bool
	MaxBackups int
	LockPath   string
	// NoRollback, when true, skips rollback on mid-plan failure.
	// Use this when the caller composes multiple targets in a single plan
	// and wants earlier successful writes to remain on disk (AC4).
	// Default false preserves single-target rollback behavior.
	NoRollback bool
}

// TargetResult summarizes one applied target.
type TargetResult struct {
	Path       string
	Op         string
	BackupPath string
	Wrote      bool
}

// ApplyResult aggregates target write results.
type ApplyResult struct {
	Targets []TargetResult
}
