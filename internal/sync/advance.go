// Package sync handles post-apply plugin sync invocation.
//
// Today that means running the plugin's configured `sync` shell command
// (typically the Advance plugin's scripts/sync-global.sh mirror) from
// the plugin's checkout directory and returning combined output with
// best-effort secret redaction applied. Additional sync providers can
// slot in here without changing the plugin lifecycle contract.
package sync

import (
	"context"
	"fmt"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

type SyncResult struct {
	ExitCode  int
	ExitClass subprocess.ExitClass
	Output    []byte
	Duration  time.Duration
}

// InvokeAdvance runs the plugin sync command from the plugin checkout,
// captures combined output, and applies render.Redact to the bytes before
// returning them.
//
// Security note: the redaction is a best-effort pattern match against
// well-known secret shapes (GitHub PATs, OpenAI keys, generic
// key=value forms). It is NOT a security boundary — sync scripts that
// print secrets in exotic formats may leak them to the caller unchanged.
// Treat the redaction as a convenience for operators watching live
// output, not as a guarantee. See docs/design/stack-toml-schema.md
// ("Security & trust boundaries") for the broader trust model.
// FormatSyncError returns a user-visible diagnostic for a plugin sync failure.
// It includes the plugin name, exit code/class, a truncated redacted output
// excerpt, and an actionable doctor command.
func FormatSyncError(name string, res SyncResult, err error) error {
	const maxExcerpt = 200
	excerpt := string(res.Output)
	if len(excerpt) > maxExcerpt {
		excerpt = excerpt[:maxExcerpt] + "..."
	}
	return fmt.Errorf(
		"sync plugin %s failed (exit code %d, class %s)\noutput: %s\naction: run `oca doctor --scope plugins` to diagnose",
		name, res.ExitCode, res.ExitClass, excerpt,
	)
}

func InvokeAdvance(ctx context.Context, plugin cfg.Plugin) (SyncResult, error) {
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "sh",
		Args:    []string{"-c", plugin.Sync},
		Dir:     plugin.Checkout,
		Timeout: 5 * time.Minute,
	})
	out := render.Redact(res.Output)
	return SyncResult{
		ExitCode:  res.ExitCode,
		ExitClass: res.ExitClass,
		Output:    out,
		Duration:  res.Duration,
	}, err
}
