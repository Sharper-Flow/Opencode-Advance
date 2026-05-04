// Package temporal provides helpers for detecting Temporal CLI and Node.js
// installations on the host system. Both use subprocess.Run with a 2s timeout.
package temporal

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

const detectTimeout = 2 * time.Second

// DetectResult holds the outcome of a detect call.
type DetectResult struct {
	Path    string // Absolute path to the binary
	Version string // Raw version string from the binary
}

// DetectCLI searches for the Temporal CLI on PATH (or at the absolute path
// given by the env var TEMPORAL_CLI_PATH). Returns the path and version
// string on success, or an error describing what went wrong.
func DetectCLI(ctx context.Context) (DetectResult, error) {
	name := "temporal"
	if p := os.Getenv("TEMPORAL_CLI_PATH"); p != "" {
		name = p
	}
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    name,
		Args:    []string{"--version"},
		Timeout: detectTimeout,
	})
	if err != nil {
		return DetectResult{}, fmt.Errorf("temporal CLI: %w", err)
	}
	if res.ExitClass != subprocess.ExitSuccess {
		return DetectResult{}, fmt.Errorf("temporal CLI: %s: %s", res.ExitClass, string(res.Output))
	}

	version := strings.TrimSpace(string(res.Output))
	// temporal --version outputs something like "temporal version 1.x.x"
	return DetectResult{
		Path:    name,
		Version: version,
	}, nil
}

// DetectNode searches for Node.js on PATH (or at the path given by ADV_NODE_PATH).
// Returns the path and the parsed major version number on success.
func DetectNode(ctx context.Context) (DetectResult, int, error) {
	name := "node"
	if p := os.Getenv("ADV_NODE_PATH"); p != "" {
		name = p
	}
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    name,
		Args:    []string{"--version"},
		Timeout: detectTimeout,
	})
	if err != nil {
		return DetectResult{}, 0, fmt.Errorf("node: %w", err)
	}
	if res.ExitClass != subprocess.ExitSuccess {
		return DetectResult{}, 0, fmt.Errorf("node: %s: %s", res.ExitClass, string(res.Output))
	}

	raw := strings.TrimSpace(string(res.Output))
	major, err := parseNodeMajorVersion(raw)
	if err != nil {
		return DetectResult{}, 0, fmt.Errorf("node: parse version %q: %w", raw, err)
	}

	return DetectResult{
		Path:    name,
		Version: raw,
	}, major, nil
}

// DetectGH searches for GitHub CLI on PATH and verifies auth status.
// GitHub CLI is optional for core OCA operation but enables ADV agent mesh
// issue workflows.
func DetectGH(ctx context.Context) (DetectResult, error) {
	name := "gh"
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    name,
		Args:    []string{"--version"},
		Timeout: detectTimeout,
	})
	if err != nil {
		return DetectResult{}, fmt.Errorf("gh CLI: %w", err)
	}
	if res.ExitClass != subprocess.ExitSuccess {
		return DetectResult{}, fmt.Errorf("gh CLI: %s: %s", res.ExitClass, string(res.Output))
	}

	version := firstNonEmptyLine(string(res.Output))
	authRes, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    name,
		Args:    []string{"auth", "status"},
		Timeout: detectTimeout,
	})
	if err != nil {
		return DetectResult{}, fmt.Errorf("gh auth status: %w", err)
	}
	if authRes.ExitClass != subprocess.ExitSuccess {
		return DetectResult{}, fmt.Errorf("gh auth status: %s: %s", authRes.ExitClass, string(authRes.Output))
	}

	return DetectResult{Path: name, Version: version}, nil
}

// parseNodeMajorVersion extracts the major version from a node --version
// output. Accepts formats like "v20.1.0", "v22.0.0-nightly", etc.
func parseNodeMajorVersion(raw string) (int, error) {
	s := raw
	s = strings.TrimPrefix(s, "v")
	// Handle nightly/pre-release like "v22.0.0-nightly"
	if idx := strings.IndexByte(s, '-'); idx != -1 {
		s = s[:idx]
	}
	parts := strings.SplitN(s, ".", 2)
	if len(parts) < 1 {
		return 0, fmt.Errorf("malformed version %q", raw)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("malformed major version in %q: %w", raw, err)
	}
	return major, nil
}

func firstNonEmptyLine(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(raw)
}
