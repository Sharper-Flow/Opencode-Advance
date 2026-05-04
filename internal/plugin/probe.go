package plugin

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

type DriftStatus string

const (
	DriftUpToDate        DriftStatus = "up_to_date"
	DriftUpdateAvailable DriftStatus = "update_available"
	DriftPinned          DriftStatus = "pinned"
	DriftUnknown         DriftStatus = "unknown"
)

type DriftResult struct {
	Name      string      `json:"name"`
	Ref       string      `json:"ref"`
	LocalSHA  string      `json:"local_sha,omitempty"`
	RemoteSHA string      `json:"remote_sha,omitempty"`
	Status    DriftStatus `json:"status"`
	Error     string      `json:"error,omitempty"`
}

type ProbeOptions struct {
	TimeoutPerPlugin time.Duration
	TimeoutGlobal    time.Duration
	Parallelism      int
}

var fullSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

func normalizeProbeOptions(opts ProbeOptions) ProbeOptions {
	if opts.TimeoutPerPlugin <= 0 {
		opts.TimeoutPerPlugin = gitTimeout
	}
	if opts.TimeoutGlobal <= 0 {
		opts.TimeoutGlobal = 30 * time.Second
	}
	if opts.Parallelism <= 0 {
		opts.Parallelism = 4
	}
	return opts
}

func Probe(ctx context.Context, name string, p config.Plugin, opts ProbeOptions) (DriftResult, error) {
	opts = normalizeProbeOptions(opts)
	ref := p.Ref
	if ref == "" {
		ref = "trunk"
	}
	res := DriftResult{Name: name, Ref: ref}

	local, err := RevParseHEAD(ctx, p.Checkout)
	if err != nil {
		res.Status = DriftUnknown
		res.Error = err.Error()
		return res, err
	}
	res.LocalSHA = local

	if fullSHA.MatchString(ref) {
		res.RemoteSHA = ref
		res.Status = DriftPinned
		return res, nil
	}

	remote, err := lsRemote(ctx, p.Source, ref, opts.TimeoutPerPlugin)
	if err != nil {
		res.Status = DriftUnknown
		res.Error = err.Error()
		return res, err
	}
	res.RemoteSHA = remote
	if local == remote {
		res.Status = DriftUpToDate
	} else {
		res.Status = DriftUpdateAvailable
	}
	return res, nil
}

func ProbeAll(ctx context.Context, plugins config.PluginsSection, selected map[string]bool, opts ProbeOptions) ([]DriftResult, error) {
	opts = normalizeProbeOptions(opts)
	ctx, cancel := context.WithTimeout(ctx, opts.TimeoutGlobal)
	defer cancel()

	names := make([]string, 0, len(plugins))
	for name, p := range plugins {
		if len(selected) > 0 && !selected[name] {
			continue
		}
		if !p.IsEnabled() || p.IsNPMSource() {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]DriftResult, len(names))
	sem := make(chan struct{}, opts.Parallelism)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, name := range names {
		i, name := i, name
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				if firstErr == nil {
					firstErr = ctx.Err()
				}
				mu.Unlock()
				return
			}
			res, err := Probe(ctx, name, plugins[name], opts)
			results[i] = res
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return results, firstErr
}

func lsRemote(ctx context.Context, source, ref string, timeout time.Duration) (string, error) {
	if err := validateGitRef(ref); err != nil {
		return "", fmt.Errorf("git ls-remote: %w", err)
	}
	res, err := subprocess.Run(ctx, subprocess.Cmd{
		Name:    "git",
		Args:    prependHardening([]string{"ls-remote", "--", source, ref}),
		Timeout: timeout,
		Env: map[string]string{
			"GIT_TERMINAL_PROMPT": "0",
		},
	})
	if err != nil {
		return "", fmt.Errorf("git ls-remote %s %s: %w", source, ref, err)
	}
	line := strings.TrimSpace(string(res.Output))
	if line == "" {
		return "", fmt.Errorf("git ls-remote %s %s: no matching ref", source, ref)
	}
	fields := strings.Fields(line)
	if len(fields) == 0 || !fullSHA.MatchString(fields[0]) {
		return "", fmt.Errorf("git ls-remote %s %s: malformed output %q", source, ref, line)
	}
	return fields[0], nil
}
