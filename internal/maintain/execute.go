package maintain

import (
	"context"
	"fmt"
	"strings"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	pluginpkg "github.com/Sharper-Flow/Opencode-Advance/internal/plugin"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

func ExecutePlan(ctx context.Context, opts Options, plan Plan) error {
	if !opts.Execute {
		return nil
	}
	if len(plan.Blockers) > 0 {
		return fmt.Errorf("blocked: %s", blockerCodeList(plan.Blockers))
	}
	stack, err := cfg.Load(opts.ConfigPath)
	if err != nil {
		return err
	}
	for _, action := range plan.Actions {
		if !action.ExecuteAllowed {
			return fmt.Errorf("action %s is not execute-allowed", action.ID)
		}
		switch action.Kind {
		case "merge":
			if err := executeMerge(ctx, stack, action); err != nil {
				return err
			}
		case "rebuild":
			if err := executeRebuild(ctx, stack, action); err != nil {
				return err
			}
		}
	}
	return nil
}

func executeMerge(ctx context.Context, stack *cfg.Stack, action Action) error {
	plugin, ok := stack.Plugins[action.Target]
	if !ok {
		return fmt.Errorf("merge target plugin %q not found", action.Target)
	}
	if plugin.Checkout == "" {
		return fmt.Errorf("merge target plugin %q has no checkout", action.Target)
	}
	if err := requireCleanGitTree(ctx, plugin.Checkout); err != nil {
		return err
	}
	branch := action.Branch
	if branch == "" {
		branch = strings.TrimPrefix(action.ID, "merge:")
		branch = "change/" + branch
	}
	if _, err := gitCommand(ctx, plugin.Checkout, "merge", "--ff-only", branch); err != nil {
		return fmt.Errorf("ff-only merge %s: %w", branch, err)
	}
	return nil
}

func executeRebuild(ctx context.Context, stack *cfg.Stack, action Action) error {
	plugin, ok := stack.Plugins[action.Target]
	if !ok {
		return fmt.Errorf("rebuild target plugin %q not found", action.Target)
	}
	if err := pluginpkg.RunBuild(ctx, plugin); err != nil {
		return err
	}
	_, err := WriteBuildMarker(ctx, action.Target, plugin)
	return err
}

func requireCleanGitTree(ctx context.Context, dir string) error {
	out, err := gitCommand(ctx, dir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Errorf("git tree not clean in %s", dir)
	}
	return nil
}

func gitCommand(ctx context.Context, dir string, args ...string) (string, error) {
	res, err := subprocess.Run(ctx, subprocess.Cmd{Name: "git", Args: args, Dir: dir, Timeout: 30 * time.Second})
	if err != nil {
		return strings.TrimSpace(string(res.Output)), err
	}
	return strings.TrimSpace(string(res.Output)), nil
}

func blockerCodeList(blockers []Blocker) string {
	codes := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		codes = append(codes, blocker.Code)
	}
	return strings.Join(codes, ", ")
}
