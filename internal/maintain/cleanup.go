package maintain

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type WorktreeInfo struct {
	Branch      string
	Path        string
	Clean       bool
	Merged      bool
	ProcessFree bool
	SessionFree bool
}

func (w WorktreeInfo) Eligible() bool {
	return w.Branch != "" && w.Path != "" && w.Clean && w.Merged && w.ProcessFree && w.SessionFree
}

func (w WorktreeInfo) IneligibleReason() string {
	switch {
	case w.Branch == "":
		return "missing branch"
	case w.Path == "":
		return "missing path"
	case !w.Merged:
		return "branch is not merged into default branch"
	case !w.Clean:
		return "worktree has uncommitted changes"
	case !w.ProcessFree:
		return "process is using worktree"
	case !w.SessionFree:
		return "session is using worktree"
	default:
		return "not eligible"
	}
}

func ListWorktrees(ctx context.Context, projectRoot, defaultBranch string) ([]WorktreeInfo, error) {
	out, err := gitCommand(ctx, projectRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktrees(ctx, projectRoot, defaultBranch, out), nil
}

func parseWorktrees(ctx context.Context, projectRoot, defaultBranch, porcelain string) []WorktreeInfo {
	blocks := strings.Split(strings.TrimSpace(porcelain), "\n\n")
	worktrees := make([]WorktreeInfo, 0, len(blocks))
	for _, block := range blocks {
		var wt WorktreeInfo
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimPrefix(line, "worktree ")
			case strings.HasPrefix(line, "branch "):
				wt.Branch = strings.TrimPrefix(line, "branch refs/heads/")
			}
		}
		if wt.Path == "" || wt.Path == projectRoot || !strings.HasPrefix(wt.Branch, "change/") {
			continue
		}
		wt.Clean = isGitTreeClean(ctx, wt.Path)
		wt.Merged = isBranchMerged(ctx, projectRoot, wt.Branch, defaultBranch)
		wt.ProcessFree = isProcessFree(wt.Path)
		wt.SessionFree = wt.ProcessFree
		worktrees = append(worktrees, wt)
	}
	return worktrees
}

func isGitTreeClean(ctx context.Context, path string) bool {
	out, err := gitCommand(ctx, path, "status", "--porcelain")
	return err == nil && strings.TrimSpace(out) == ""
}

func isBranchMerged(ctx context.Context, projectRoot, branch, defaultBranch string) bool {
	if defaultBranch == "" {
		defaultBranch = "trunk"
	}
	_, err := gitCommand(ctx, projectRoot, "merge-base", "--is-ancestor", branch, defaultBranch)
	return err == nil
}

func isProcessFree(worktreePath string) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return true
	}
	cleanPath := filepath.Clean(worktreePath)
	for _, entry := range entries {
		if !entry.IsDir() || !isDigits(entry.Name()) {
			continue
		}
		cwd, err := os.Readlink(filepath.Join("/proc", entry.Name(), "cwd"))
		if err != nil {
			continue
		}
		cwd = filepath.Clean(cwd)
		if cwd == cleanPath || strings.HasPrefix(cwd, cleanPath+string(os.PathSeparator)) {
			return false
		}
	}
	return true
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
