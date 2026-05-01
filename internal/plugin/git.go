// Package plugin provides lifecycle operations for OpenCode plugins:
// clone/fetch/checkout (git), build commands, and pin management.
// All external commands go through internal/subprocess.Run.
package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

// gitTimeout is the default wall-clock budget for individual git operations.
const gitTimeout = 60 * time.Second

// gitHardeningArgs are prepended to every `git` invocation to disable
// dangerous protocols and reduce attack surface from a malicious remote
// or hostile environment.
//
//   - protocol.file.allow=user blocks `git clone file://...` from
//     submodules / refs we did not author.
//   - protocol.ext.allow=false disables the `ext::` transport which can
//     execute arbitrary commands.
//
// These mitigate CVE-2022-39253, CVE-2017-1000117, and similar
// transport-confusion classes. Apply via prependHardening() before
// composing the per-command args.
var gitHardeningArgs = []string{
	"-c", "protocol.file.allow=user",
	"-c", "protocol.ext.allow=false",
}

func prependHardening(args []string) []string {
	out := make([]string, 0, len(gitHardeningArgs)+len(args))
	out = append(out, gitHardeningArgs...)
	out = append(out, args...)
	return out
}

// gitRefPattern is a conservative whitelist for branch names, tags, and
// short/long SHAs. It rejects characters that could be option-confused
// (`-foo`), pathname-traversed (`..`), or shell-metacharacter-injected
// even though we already use exec without a shell.
var gitRefPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// validateGitRef rejects refs that could be misinterpreted as command-line
// options or contain pathname-traversal / control sequences. Empty refs
// are rejected; callers that allow "use default branch" must check for
// emptiness before calling.
func validateGitRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("git ref must not be empty")
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("git ref %q must not start with '-'", ref)
	}
	if strings.Contains(ref, "..") {
		return fmt.Errorf("git ref %q must not contain '..'", ref)
	}
	if !gitRefPattern.MatchString(ref) {
		return fmt.Errorf("git ref %q contains disallowed characters (allowed: A-Z a-z 0-9 . _ / -)", ref)
	}
	return nil
}

// Clone runs git clone {repo} into {dir}. If branch is non-empty, passes
// --branch {branch} --single-branch. Returns an error if the subprocess
// fails, the directory already exists, or the branch ref fails validation.
func Clone(ctx context.Context, repo, dir, branch string) error {
	args := []string{"clone"}
	if branch != "" {
		if err := validateGitRef(branch); err != nil {
			return fmt.Errorf("git clone %s: %w", repo, err)
		}
		args = append(args, "--branch", branch, "--single-branch")
	}
	args = append(args, "--", repo, dir)

	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "git",
		Args:    prependHardening(args),
		Timeout: gitTimeout,
	})
	if err != nil {
		return fmt.Errorf("git clone %s: %w", repo, err)
	}
	return nil
}

// Fetch runs git fetch in the checkout directory. Fetches all remotes.
func Fetch(ctx context.Context, checkout string) error {
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "git",
		Args:    prependHardening([]string{"fetch", "--all"}),
		Dir:     checkout,
		Timeout: gitTimeout,
	})
	if err != nil {
		return fmt.Errorf("git fetch: %w", err)
	}
	return nil
}

// Checkout runs git checkout {ref} in the checkout directory. The ref is
// validated before invocation; invalid refs return an error without
// touching the working tree.
func Checkout(ctx context.Context, checkout, ref string) error {
	if err := validateGitRef(ref); err != nil {
		return fmt.Errorf("git checkout: %w", err)
	}
	// No `--` separator: in `git checkout` the `--` marks the start of
	// pathspecs, and adding it would cause git to interpret the ref as a
	// file to check out rather than a branch/tag/SHA. Safety comes from
	// validateGitRef above, which has already rejected option-like inputs.
	_, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "git",
		Args:    prependHardening([]string{"checkout", ref}),
		Dir:     checkout,
		Timeout: gitTimeout,
	})
	if err != nil {
		return fmt.Errorf("git checkout %s: %w", ref, err)
	}
	return nil
}

// RevParseHEAD returns the full SHA of HEAD in the checkout directory.
func RevParseHEAD(ctx context.Context, checkout string) (string, error) {
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "git",
		Args:    prependHardening([]string{"rev-parse", "HEAD"}),
		Dir:     checkout,
		Timeout: gitTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(res.Output)), nil
}

// StatusClean reports whether the working tree has no uncommitted changes.
// Returns true when the checkout is clean, false when dirty.
func StatusClean(ctx context.Context, checkout string) (bool, error) {
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "git",
		Args:    prependHardening([]string{"status", "--porcelain"}),
		Dir:     checkout,
		Timeout: gitTimeout,
	})
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return len(res.Output) == 0, nil
}

// RemoteOriginURL returns the configured `remote.origin.url` for the
// checkout. Used to detect drift between stack.toml `source` and the
// already-cloned remote so users do not silently keep an old remote when
// they edit the source URL in stack.toml.
func RemoteOriginURL(ctx context.Context, checkout string) (string, error) {
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "git",
		Args:    prependHardening([]string{"config", "--get", "remote.origin.url"}),
		Dir:     checkout,
		Timeout: gitTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("git config remote.origin.url: %w", err)
	}
	return strings.TrimSpace(string(res.Output)), nil
}

// execCmd abstracts the subset of exec.Cmd used by test helpers.
type execCmd interface {
	CombinedOutput() ([]byte, error)
}

// execLookPathStd and defaultExecCmdContext are the real implementations
// used by production code and test helpers. They are wrapped by package-level
// function variables so integration tests can call them without import cycles.
var (
	execLookPathStd       = exec.LookPath
	defaultExecCmdContext = func(ctx context.Context, name string, args ...string) execCmd {
		return exec.CommandContext(ctx, name, args...)
	}
)
