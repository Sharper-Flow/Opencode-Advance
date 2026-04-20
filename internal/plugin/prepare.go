package plugin

import (
	"context"
	"fmt"
	"os"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// Prepare orchestrates clone-or-pull + build for a single plugin.
//
// Flow:
//  1. Skip npm-source plugins (no git operations).
//  2. Skip disabled plugins (enabled == false).
//  3. If checkout directory missing → Clone into it.
//  4. If checkout exists and dirty → abort with remediation guidance (AC29).
//  5. If checkout exists and clean → Fetch + Checkout ref.
//  6. If plugin declares build commands → RunBuild.
//
// Ref defaults to "trunk" when empty (matching OCA convention per AC27).
func Prepare(ctx context.Context, p config.Plugin) error {
	// Skip npm-source plugins — no local checkout needed.
	if p.IsNPMSource() {
		return nil
	}

	// Skip disabled plugins.
	if !p.IsEnabled() {
		return nil
	}

	ref := p.Ref
	if ref == "" {
		ref = "trunk"
	}
	checkout := p.Checkout

	// Step 3: clone if checkout doesn't exist.
	if _, err := os.Stat(checkout); os.IsNotExist(err) {
		if err := Clone(ctx, p.Source, checkout, ref); err != nil {
			return fmt.Errorf("prepare plugin: clone failed: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("prepare plugin: stat checkout %s: %w", checkout, err)
	} else {
		// Step 4: abort if dirty.
		clean, err := StatusClean(ctx, checkout)
		if err != nil {
			return fmt.Errorf("prepare plugin: status check: %w", err)
		}
		if !clean {
			return fmt.Errorf("prepare plugin: checkout %s has uncommitted changes (dirty worktree). "+
				"Remediation: commit or stash your local changes before running oca apply. "+
				"Run 'git -C %s status' to see what changed", checkout, checkout)
		}

		// Step 5: fetch + checkout.
		if err := Fetch(ctx, checkout); err != nil {
			return fmt.Errorf("prepare plugin: fetch: %w", err)
		}
		if err := Checkout(ctx, checkout, ref); err != nil {
			return fmt.Errorf("prepare plugin: checkout %s: %w", ref, err)
		}
	}

	// Step 6: run build if declared.
	if len(p.Build) > 0 {
		if err := RunBuild(ctx, p); err != nil {
			return fmt.Errorf("prepare plugin: build: %w", err)
		}
	}

	return nil
}
