package subprocess

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// StartOptions controls long-running background process startup.
type StartOptions struct {
	// OutputPath receives combined stdout/stderr in append mode when set.
	OutputPath string
	// SetProcessGroup starts the child in its own process group.
	SetProcessGroup bool
}

// BackgroundProcess identifies a started long-running subprocess.
type BackgroundProcess struct {
	PID            int
	ProcessGroupID int
}

// StartBackground starts c without shell interpretation and returns immediately.
func StartBackground(ctx context.Context, c Cmd, opts StartOptions) (BackgroundProcess, error) {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	if c.Dir != "" {
		cmd.Dir = c.Dir
	}
	cmd.Env = mergeEnv(os.Environ(), c.Env)
	if opts.SetProcessGroup {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	var output *os.File
	if opts.OutputPath != "" {
		if err := os.MkdirAll(filepath.Dir(opts.OutputPath), 0o755); err != nil {
			return BackgroundProcess{}, err
		}
		f, err := os.OpenFile(opts.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return BackgroundProcess{}, err
		}
		defer f.Close()
		output = f
		cmd.Stdout = output
		cmd.Stderr = output
	}

	if err := cmd.Start(); err != nil {
		return BackgroundProcess{}, fmt.Errorf("start background %s: %w", c.Name, err)
	}
	pid := cmd.Process.Pid
	pgid := pid
	if !opts.SetProcessGroup {
		if got, err := syscall.Getpgid(pid); err == nil {
			pgid = got
		}
	}
	go func() { _ = cmd.Wait() }()
	return BackgroundProcess{PID: pid, ProcessGroupID: pgid}, nil
}
