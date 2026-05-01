package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

// buildTimeout is the wall-clock budget for a single build command.
const buildTimeout = 5 * 60 * time.Second // 5 minutes

// RunBuild executes each build command from the plugin's Build field
// sequentially in the checkout directory. Commands are invoked via
// "sh -c <cmd>" so shell features (pipes, &&, etc.) work naturally.
//
// Environment (per jc-pnpm1 resolution):
//   - CI=true, DEBIAN_FRONTEND=noninteractive, npm_config_yes=true
//     are merged onto the inherited process environment.
//   - --frozen-lockfile is the caller's responsibility in the build
//     command string itself.
//
// On failure, the captured combined output is included in the error
// message for remediation per AC29.
func RunBuild(ctx context.Context, p config.Plugin) error {
	if len(p.Build) == 0 {
		return nil
	}

	buildEnv := map[string]string{
		"CI":              "true",
		"DEBIAN_FRONTEND": "noninteractive",
		"npm_config_yes":  "true",
	}

	for i, cmd := range p.Build {
		res, err := subprocess.Run(ctx, subprocess.Cmd{
			Name:    "sh",
			Args:    []string{"-c", cmd},
			Dir:     p.Checkout,
			Env:     buildEnv,
			Timeout: buildTimeout,
		})
		if err != nil {
			output := strings.TrimSpace(string(res.Output))
			// Include step index so multi-step failures point at the
			// exact command that broke, not just "a build step".
			return fmt.Errorf("build step %d/%d %q failed: %w\noutput:\n%s",
				i+1, len(p.Build), cmd, err, output)
		}
	}
	return nil
}
