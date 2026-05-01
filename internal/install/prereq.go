package install

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/Sharper-Flow/Opencode-Advance/internal/health"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
	"time"
)

const detectTimeout = 2 * time.Second

// runDetect is the subprocess runner used for binary detection.
// Overridden in tests for mocking.
var runDetect = subprocess.Run

// prereqDef defines a single prerequisite check.
type prereqDef struct {
	name     string                    // check name (e.g., "prerequisites.git")
	binary   string                    // binary name to detect
	args     []string                  // args for version detection
	required bool                      // true = StatusFail on missing/old, false = StatusWarn
	minMajor int                       // minimum major version (0 = no version check)
	parseFn  func(string) (int, error) // custom version parser (nil = use default)
}

var prereqDefs = []prereqDef{
	{
		name:     "prerequisites.git",
		binary:   "git",
		args:     []string{"--version"},
		required: true,
		minMajor: 2,
	},
	{
		name:     "prerequisites.tmux",
		binary:   "tmux",
		args:     []string{"-V"},
		required: true,
		minMajor: 3,
	},
	{
		name:     "prerequisites.opencode",
		binary:   "opencode",
		args:     []string{"--version"},
		required: true,
		minMajor: 0, // any version OK
	},
	{
		name:     "prerequisites.vision",
		binary:   "vision",
		args:     []string{"--version"},
		required: false,
		minMajor: 0,
	},
	{
		name:     "prerequisites.temporal",
		binary:   "temporal",
		args:     []string{"--version"},
		required: false,
		minMajor: 0,
	},
}

// CheckPrerequisites probes for required and optional tool installations.
// Required tools (git, tmux, opencode) produce StatusFail when missing or too old.
// Optional tools (vision, temporal) produce StatusWarn when missing.
func CheckPrerequisites(ctx context.Context, _ interface{}, _ health.Options) ([]health.Check, error) {
	checks := make([]health.Check, 0, len(prereqDefs))

	for _, def := range prereqDefs {
		checks = append(checks, checkBinary(ctx, def))
	}

	return checks, nil
}

func checkBinary(ctx context.Context, def prereqDef) health.Check {
	res, err := runDetect(ctx, subprocess.Cmd{
		Name:    def.binary,
		Args:    def.args,
		Timeout: detectTimeout,
	})

	if err != nil {
		status := health.StatusWarn
		hint := fmt.Sprintf("install %s", def.binary)
		if def.required {
			status = health.StatusFail
		}
		return health.Check{
			Name:    def.name,
			Status:  status,
			Message: fmt.Sprintf("%s not found: %v", def.binary, err),
			Hint:    hint,
		}
	}

	if res.ExitClass != subprocess.ExitSuccess {
		status := health.StatusWarn
		if def.required {
			status = health.StatusFail
		}
		return health.Check{
			Name:    def.name,
			Status:  status,
			Message: fmt.Sprintf("%s: %s", def.binary, string(res.Output)),
		}
	}

	version := string(res.Output)

	// Version check
	if def.minMajor > 0 {
		major, err := parseMajorVersion(version)
		if err != nil {
			return health.Check{
				Name:    def.name,
				Status:  health.StatusWarn,
				Message: fmt.Sprintf("%s: parse version: %v", def.binary, err),
			}
		}
		if major < def.minMajor {
			status := health.StatusWarn
			hint := ""
			if def.required {
				status = health.StatusFail
				hint = fmt.Sprintf("upgrade %s to version %d+", def.binary, def.minMajor)
			}
			return health.Check{
				Name:    def.name,
				Status:  status,
				Message: fmt.Sprintf("%s version %d (need ≥ %d)", def.binary, major, def.minMajor),
				Hint:    hint,
			}
		}
	}

	return health.Check{
		Name:    def.name,
		Status:  health.StatusPass,
		Message: fmt.Sprintf("%s %s", def.binary, version),
	}
}

// versionRe matches the first sequence of digits in a string.
var versionRe = regexp.MustCompile(`(\d+)`)

// parseMajorVersion extracts the major version number from a version string.
// Handles formats like "git version 2.43.0", "tmux 3.4", "v0.5.0", "1.2.3".
func parseMajorVersion(s string) (int, error) {
	matches := versionRe.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0, fmt.Errorf("no version number found in %q", s)
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("parse version %q: %w", s, err)
	}
	return major, nil
}
