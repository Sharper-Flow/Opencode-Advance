package health

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func init() {
	registerBuiltin("runtime", CheckRuntime)
}

// CheckRuntime runs runtime canaries that verify live integration health
// beyond static configuration checks. Canaries are time-bounded and
// degrade gracefully when components are absent.
func CheckRuntime(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	var checks []Check

	checks = append(checks, canaryADVPrompts(stack, opts)...)
	checks = append(checks, canaryOCAPlugin(stack, opts)...)
	checks = append(checks, canaryVisionReachability(ctx, opts)...)

	return checks, nil
}

// canaryADVPrompts checks that ADV provider prompt files exist and contain
// the expected runtime marker. This catches the "config looks correct but
// runtime resolves to stub" class of failures.
func canaryADVPrompts(stack *cfg.Stack, opts Options) []Check {
	var checks []Check

	if opts.ConfigDir == "" {
		checks = append(checks, Check{
			Name:    "runtime.adv-prompts",
			Status:  StatusWarn,
			Message: "ConfigDir not set — ADV prompt canary skipped",
		})
		return checks
	}

	agentPartsDir := filepath.Join(opts.ConfigDir, "agent-parts", "advance")
	if _, err := os.Stat(agentPartsDir); os.IsNotExist(err) {
		checks = append(checks, Check{
			Name:    "runtime.adv-prompts",
			Status:  StatusPass,
			Message: "ADV agent-parts directory not found — prompt canary skipped (plugin not deployed)",
		})
		return checks
	}

	// Check canonical prompt part
	canonicalPath := filepath.Join(agentPartsDir, "adv.md")
	if _, err := os.Stat(canonicalPath); err != nil {
		checks = append(checks, Check{
			Name:    "runtime.adv-prompts.canonical",
			Status:  StatusWarn,
			Message: fmt.Sprintf("ADV canonical prompt part missing: %s", canonicalPath),
			Hint:    "run the plugin sync script (e.g. scripts/sync-global.sh --fix) to regenerate",
		})
		return checks
	}

	// Verify the canonical prompt part contains substantive content (not just a stub marker)
	data, err := os.ReadFile(canonicalPath)
	if err != nil {
		checks = append(checks, Check{
			Name:    "runtime.adv-prompts.canonical",
			Status:  StatusWarn,
			Message: fmt.Sprintf("cannot read ADV prompt part: %v", err),
		})
		return checks
	}

	content := string(data)
	if len(strings.TrimSpace(content)) < 100 {
		checks = append(checks, Check{
			Name:    "runtime.adv-prompts.canonical",
			Status:  StatusWarn,
			Message: "ADV canonical prompt part is suspiciously short (<100 chars) — may be a stub",
			Hint:    "regenerate with the plugin sync script",
		})
	} else {
		checks = append(checks, Check{
			Name:    "runtime.adv-prompts.canonical",
			Status:  StatusPass,
			Message: fmt.Sprintf("ADV canonical prompt part present (%d bytes)", len(content)),
		})
	}

	// Check provider-specific prompt parts for each declared provider
	for providerName := range stack.Providers {
		providerPath := filepath.Join(agentPartsDir, "providers", providerName+".md")
		if _, err := os.Stat(providerPath); err != nil {
			checks = append(checks, Check{
				Name:    fmt.Sprintf("runtime.adv-prompts.provider.%s", providerName),
				Status:  StatusWarn,
				Message: fmt.Sprintf("Provider prompt part missing: %s", providerPath),
				Hint:    "run the plugin sync script to generate provider prompt parts",
			})
			continue
		}

		pData, err := os.ReadFile(providerPath)
		if err != nil {
			checks = append(checks, Check{
				Name:    fmt.Sprintf("runtime.adv-prompts.provider.%s", providerName),
				Status:  StatusWarn,
				Message: fmt.Sprintf("cannot read provider prompt: %v", err),
			})
			continue
		}

		checks = append(checks, Check{
			Name:    fmt.Sprintf("runtime.adv-prompts.provider.%s", providerName),
			Status:  StatusPass,
			Message: fmt.Sprintf("Provider prompt part present (%d bytes)", len(pData)),
		})
	}

	return checks
}

// canaryOCAPlugin checks that the OCA plugin build artifact exists and
// the plugin entry appears in opencode.json.
func canaryOCAPlugin(stack *cfg.Stack, opts Options) []Check {
	var checks []Check

	if opts.ConfigDir == "" {
		checks = append(checks, Check{
			Name:    "runtime.oca-plugin",
			Status:  StatusWarn,
			Message: "ConfigDir not set — OCA plugin canary skipped",
		})
		return checks
	}

	// Check if OCA plugin is declared in stack.toml
	_, hasOCA := stack.Plugins["oca"]
	if !hasOCA {
		checks = append(checks, Check{
			Name:    "runtime.oca-plugin.declared",
			Status:  StatusPass,
			Message: "OCA plugin not declared in stack.toml — plugin canary skipped",
		})
		return checks
	}

	// Check plugin node_modules or build artifact
	pluginCacheDir := filepath.Join(opts.ConfigDir, "node_modules", "@sharperflow", "oca-plugin")
	if _, err := os.Stat(pluginCacheDir); err != nil {
		checks = append(checks, Check{
			Name:    "runtime.oca-plugin.artifact",
			Status:  StatusWarn,
			Message: "OCA plugin artifact not found in node_modules",
			Hint:    "run 'oca apply --target plugins' to install declared plugins",
		})
	} else {
		checks = append(checks, Check{
			Name:    "runtime.oca-plugin.artifact",
			Status:  StatusPass,
			Message: "OCA plugin artifact present",
		})
	}

	return checks
}

// canaryVisionReachability checks Vision daemon reachability with a short timeout.
// This supplements the static MCP scope by testing actual HTTP connectivity.
func canaryVisionReachability(ctx context.Context, opts Options) []Check {
	var checks []Check

	base := opts.VisionAdminURL
	if base == "" {
		base = "http://127.0.0.1:6275"
	}

	client := opts.HTTPClient
	if client == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 3 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/version", nil)
	if err != nil {
		checks = append(checks, Check{
			Name:    "runtime.vision",
			Status:  StatusWarn,
			Message: fmt.Sprintf("cannot construct Vision request: %v", err),
			Elapsed: time.Since(start),
		})
		return checks
	}

	resp, err := client.Do(req)
	if err != nil {
		checks = append(checks, Check{
			Name:    "runtime.vision",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Vision unreachable: %v", err),
			Hint:    "start Vision daemon or verify admin port configuration",
			Elapsed: time.Since(start),
		})
		return checks
	}
	resp.Body.Close()

	checks = append(checks, Check{
		Name:    "runtime.vision",
		Status:  StatusPass,
		Message: fmt.Sprintf("Vision reachable at %s (HTTP %d)", base, resp.StatusCode),
		Elapsed: time.Since(start),
	})

	return checks
}
