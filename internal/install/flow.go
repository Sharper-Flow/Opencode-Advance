package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sharper-Flow/Opencode-Advance/internal/health"
)

// Package-level function variables for testability.
var (
	userHomeDir   = os.UserHomeDir
	osExecutable  = os.Executable
	checkPrereqs  = CheckPrerequisites
	evalSymlinks  = filepath.EvalSymlinks
)

// InstallOptions holds options for the install operation.
type InstallOptions struct {
	Yes bool // skip interactive prompts (CI mode)
}

// UninstallOptions holds options for the uninstall operation.
type UninstallOptions struct {
	Force bool // skip edited-block protection
}

// InstallResult holds the result of a successful install.
type InstallResult struct {
	BinaryPath   string
	CreatedDirs  []string
	WroteRCFiles []string
	PrereqChecks []health.Check
}

// UninstallResult holds the result of a successful uninstall.
type UninstallResult struct {
	RemovedRCFiles []string
}

// ErrPrereqFailed is returned when a required prerequisite check fails.
type ErrPrereqFailed struct {
	Failures []health.Check
}

func (e *ErrPrereqFailed) Error() string {
	return fmt.Sprintf("prerequisites failed: %d required checks failed", len(e.Failures))
}

// ErrBlockEdited is returned when uninstall finds an edited managed block
// and Force is not set.
type ErrBlockEdited struct {
	FilePath string
	Diff     string
}

func (e *ErrBlockEdited) Error() string {
	return fmt.Sprintf("managed block in %s has been edited; use --force to override", e.FilePath)
}

// Install performs the full installation flow:
// 1. Create directories
// 2. Resolve binary path (KD9: os.Executable + EvalSymlinks)
// 3. Check prerequisites
// 4. Render and write shell profile blocks
func Install(ctx context.Context, opts InstallOptions) (*InstallResult, error) {
	home, err := userHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}

	// Resolve OCA binary path (KD9)
	exePath, err := osExecutable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	resolvedPath, err := evalSymlinks(exePath)
	if err != nil {
		resolvedPath = exePath // fallback to unresolved
	}

	// Create directories
	dirs := []string{
		filepath.Join(home, ".config", "opencode"),
		filepath.Join(home, ".config", "vision"),
	}
	var createdDirs []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create dir %s: %w", dir, err)
			}
			createdDirs = append(createdDirs, dir)
		}
	}

	// Check prerequisites
	prereqChecks, err := checkPrereqs(ctx, nil, health.Options{})
	if err != nil {
		return nil, fmt.Errorf("prerequisite check: %w", err)
	}

	var failures []health.Check
	for _, c := range prereqChecks {
		if c.Status == health.StatusFail {
			failures = append(failures, c)
		}
	}
	if len(failures) > 0 {
		return nil, &ErrPrereqFailed{Failures: failures}
	}

	// Render shell profile block
	blockContent := RenderShellProfile(resolvedPath)

	// Write managed blocks to both rc files (always both, per user decision)
	rcFiles := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	}
	var wroteRCFiles []string
	for _, rc := range rcFiles {
		if err := WriteBlock(rc, blockContent); err != nil {
			return nil, fmt.Errorf("write %s: %w", rc, err)
		}
		wroteRCFiles = append(wroteRCFiles, rc)
	}

	return &InstallResult{
		BinaryPath:   resolvedPath,
		CreatedDirs:  createdDirs,
		WroteRCFiles: wroteRCFiles,
		PrereqChecks: prereqChecks,
	}, nil
}

// Uninstall performs the full uninstallation flow:
// 1. Resolve current binary path for block comparison
// 2. Check each rc file for edited blocks (unless Force)
// 3. Remove managed blocks
func Uninstall(opts UninstallOptions) (*UninstallResult, error) {
	home, err := userHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}

	// Resolve current binary path for block comparison
	exePath, err := osExecutable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	resolvedPath, err := evalSymlinks(exePath)
	if err != nil {
		resolvedPath = exePath
	}
	expectedBlock := RenderShellProfile(resolvedPath)

	rcFiles := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	}

	// Pre-check for edited blocks (before removing anything)
	if !opts.Force {
		for _, rc := range rcFiles {
			if _, err := os.Stat(rc); os.IsNotExist(err) {
				continue
			}
			block, err := ReadBlock(rc)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", rc, err)
			}
			if block == "" {
				continue // no block, skip
			}
			edited, err := IsEdited(rc, expectedBlock)
			if err != nil {
				return nil, fmt.Errorf("check %s: %w", rc, err)
			}
			if edited {
				diff, _ := DiffBlock(rc, expectedBlock)
				return nil, &ErrBlockEdited{
					FilePath: rc,
					Diff:     diff.Diff,
				}
			}
		}
	}

	// Remove blocks
	var removedRCFiles []string
	for _, rc := range rcFiles {
		if _, err := os.Stat(rc); os.IsNotExist(err) {
			continue
		}
		block, _ := ReadBlock(rc)
		if block == "" {
			continue // no block, skip
		}
		if err := RemoveBlock(rc); err != nil {
			return nil, fmt.Errorf("remove block from %s: %w", rc, err)
		}
		removedRCFiles = append(removedRCFiles, rc)
	}

	return &UninstallResult{
		RemovedRCFiles: removedRCFiles,
	}, nil
}
