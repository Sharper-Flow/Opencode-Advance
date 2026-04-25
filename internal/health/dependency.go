package health

import (
	"context"
	"fmt"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/temporal"
)

// CheckDependencies probes for Node.js and Temporal CLI installations.
// It returns health checks for each dependency.
func CheckDependencies(ctx context.Context, _ *cfg.Stack, _ Options) ([]Check, error) {
	checks := []Check{}

	nodeRes, major, err := temporal.DetectNode(ctx)
	if err != nil {
		checks = append(checks, Check{
			Name:    "dependencies.node",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Node.js not found: %v", err),
			Hint:    "install Node.js v20+ (e.g. via nvm, fnm, or your system package manager)",
		})
	} else if major < 20 {
		checks = append(checks, Check{
			Name:    "dependencies.node",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Node.js version too old: %s (major %d)", nodeRes.Version, major),
			Hint:    "upgrade to Node.js v20+ for Temporal worker compatibility",
		})
	} else {
		checks = append(checks, Check{
			Name:    "dependencies.node",
			Status:  StatusPass,
			Message: fmt.Sprintf("Node.js %s (major %d)", nodeRes.Version, major),
		})
	}

	cliRes, err := temporal.DetectCLI(ctx)
	if err != nil {
		checks = append(checks, Check{
			Name:    "dependencies.temporal_cli",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Temporal CLI not found: %v", err),
			Hint:    "install Temporal CLI (see https://docs.temporal.io/cli)",
		})
	} else {
		checks = append(checks, Check{
			Name:    "dependencies.temporal_cli",
			Status:  StatusPass,
			Message: fmt.Sprintf("Temporal CLI %s", cliRes.Version),
		})
	}

	return checks, nil
}

func init() {
	registerBuiltin("dependencies", CheckDependencies)
}
