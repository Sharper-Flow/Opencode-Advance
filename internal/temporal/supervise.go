package temporal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

var (
	ErrUnmanagedServer = errors.New("temporal server reachable but not OCA-managed")
	ErrRemoteStart     = errors.New("refusing to start Temporal dev server on non-loopback address")
)

// State is the canonical Temporal supervisor status enum.
type State string

const (
	StateDisabled  State = "disabled"
	StateStopped   State = "stopped"
	StateStarting  State = "starting"
	StateRunning   State = "running"
	StateUnhealthy State = "unhealthy"
	StateUnmanaged State = "unmanaged"
	StateStale     State = "stale"
	StateError     State = "error"
)

// RuntimePaths are OCA-owned files for local Temporal dev-server supervision.
type RuntimePaths struct {
	Root     string
	Metadata string
	Log      string
	DB       string
	Status   string
}

// Metadata is the atomic JSON record proving OCA owns a Temporal process.
type Metadata struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	Address   string    `json:"address"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Namespace string    `json:"namespace"`
	DBPath    string    `json:"db_path"`
	LogPath   string    `json:"log_path"`
	CLIPath   string    `json:"cli_path"`
	Args      []string  `json:"args"`
	Version   string    `json:"version"`
}

// Status is the stable machine-readable status returned by `oca temporal status`.
type Status struct {
	Configured bool   `json:"configured"`
	Enabled    bool   `json:"enabled"`
	Address    string `json:"address"`
	Namespace  string `json:"namespace"`
	PID        int    `json:"pid,omitempty"`
	Managed    bool   `json:"managed"`
	Running    bool   `json:"running"`
	Reachable  bool   `json:"reachable"`
	Healthy    bool   `json:"healthy"`
	State      State  `json:"state"`
	LogPath    string `json:"log_path,omitempty"`
	DBPath     string `json:"db_path,omitempty"`
}

// ProbeFuncs allows tests and CLI code to provide cheap reachability/health probes.
type ProbeFuncs struct {
	Reachable func(context.Context, string) bool
	Healthy   func(context.Context, *cfg.Stack) bool
}

// Supervisor coordinates OCA-owned Temporal dev-server lifecycle operations.
type Supervisor struct {
	Paths              RuntimePaths
	DetectCLI          func(context.Context) (DetectResult, error)
	StartBackground    func(context.Context, subprocess.Cmd, subprocess.StartOptions) (subprocess.BackgroundProcess, error)
	Reachable          func(context.Context, string) bool
	Healthy            func(context.Context, *cfg.Stack) bool
	Now                func() time.Time
	PIDAlive           func(int) bool
	SignalProcessGroup func(int, syscall.Signal) error
	ReadinessWait      time.Duration
	ReadinessTick      time.Duration
	StopWait           time.Duration
	StopTick           time.Duration
}

// Start launches the OCA-managed local Temporal dev server or returns existing state.
func (s Supervisor) Start(ctx context.Context, stack *cfg.Stack) (Status, error) {
	paths := s.paths()
	probes := s.probes()
	status, err := EvaluateStatus(ctx, stack, paths, probes)
	if err != nil {
		return status, err
	}
	if !status.Enabled {
		return status, nil
	}
	if status.Managed && status.Running {
		return status, nil
	}
	if status.State == StateStale {
		if err := RemoveMetadata(paths); err != nil {
			return status, err
		}
	} else if status.Reachable && !status.Managed {
		return status, ErrUnmanagedServer
	}

	host, port, err := splitHostPort(status.Address)
	if err != nil {
		status.State = StateError
		return status, err
	}
	if !isLoopbackHost(host) {
		status.State = StateError
		return status, ErrRemoteStart
	}

	detect := s.DetectCLI
	if detect == nil {
		detect = DetectCLI
	}
	detected, err := detect(ctx)
	if err != nil {
		status.State = StateError
		return status, err
	}
	starter := s.StartBackground
	if starter == nil {
		starter = subprocess.StartBackground
	}
	args := []string{"server", "start-dev", "--ip", host, "--port", strconv.Itoa(port), "--namespace", status.Namespace, "--db-filename", paths.DB, "--log-level", "warn", "--headless"}
	proc, err := starter(ctx, subprocess.Cmd{Name: detected.Path, Args: args}, subprocess.StartOptions{OutputPath: paths.Log, SetProcessGroup: true})
	if err != nil {
		status.State = StateError
		return status, err
	}
	if !s.waitReachable(ctx, status.Address) {
		_ = killProcessGroup(proc.ProcessGroupID, syscall.SIGTERM)
		status.State = StateError
		return status, fmt.Errorf("temporal dev server not reachable at %s", status.Address)
	}
	meta := Metadata{
		PID:       proc.PID,
		StartedAt: s.now(),
		Address:   status.Address,
		Host:      host,
		Port:      port,
		Namespace: status.Namespace,
		DBPath:    paths.DB,
		LogPath:   paths.Log,
		CLIPath:   detected.Path,
		Args:      args,
		Version:   detected.Version,
	}
	if err := WriteMetadata(paths, meta); err != nil {
		status.State = StateError
		return status, err
	}
	return EvaluateStatus(ctx, stack, paths, probes)
}

// Status returns current Temporal supervisor status.
func (s Supervisor) Status(ctx context.Context, stack *cfg.Stack) (Status, error) {
	return EvaluateStatus(ctx, stack, s.paths(), s.probes())
}

// Stop terminates only the OCA-managed Temporal process group.
func (s Supervisor) Stop(ctx context.Context, stack *cfg.Stack) (Status, error) {
	paths := s.paths()
	probes := s.probes()
	status, err := EvaluateStatus(ctx, stack, paths, probes)
	if err != nil {
		return status, err
	}
	if !status.Enabled {
		return status, nil
	}
	if status.Reachable && !status.Managed {
		return status, ErrUnmanagedServer
	}
	meta, hasMeta, err := ReadMetadata(paths)
	if err != nil {
		status.State = StateError
		return status, err
	}
	if !hasMeta {
		status.State = StateStopped
		return status, nil
	}
	if !s.pidAlive(meta.PID) {
		if err := RemoveMetadata(paths); err != nil {
			status.State = StateError
			return status, err
		}
		status.Managed = false
		status.Running = false
		status.State = StateStopped
		return status, nil
	}
	if err := s.signalProcessGroup(meta.PID, syscall.SIGTERM); err != nil {
		status.State = StateError
		return status, err
	}
	if !s.waitStopped(ctx, meta.PID) {
		if err := s.signalProcessGroup(meta.PID, syscall.SIGKILL); err != nil {
			status.State = StateError
			return status, err
		}
		_ = s.waitStopped(ctx, meta.PID)
	}
	if err := RemoveMetadata(paths); err != nil {
		status.State = StateError
		return status, err
	}
	status.Managed = false
	status.Running = false
	status.PID = 0
	status.State = StateStopped
	return status, nil
}

// Restart composes Stop then Start while preserving the persistent DB path.
func (s Supervisor) Restart(ctx context.Context, stack *cfg.Stack) (Status, error) {
	if _, err := s.Stop(ctx, stack); err != nil {
		return Status{}, err
	}
	return s.Start(ctx, stack)
}

func (s Supervisor) paths() RuntimePaths {
	if s.Paths.Root == "" {
		return RuntimePathsFromEnv()
	}
	return s.Paths
}

func (s Supervisor) probes() ProbeFuncs {
	return ProbeFuncs{Reachable: s.Reachable, Healthy: s.Healthy}
}

func (s Supervisor) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s Supervisor) pidAlive(pid int) bool {
	if s.PIDAlive != nil {
		return s.PIDAlive(pid)
	}
	return pidAlive(pid)
}

func (s Supervisor) signalProcessGroup(pgid int, signal syscall.Signal) error {
	if s.SignalProcessGroup != nil {
		return s.SignalProcessGroup(pgid, signal)
	}
	return killProcessGroup(pgid, signal)
}

func (s Supervisor) waitReachable(ctx context.Context, address string) bool {
	wait := s.ReadinessWait
	if wait == 0 {
		wait = 30 * time.Second
	}
	tick := s.ReadinessTick
	if tick == 0 {
		tick = 500 * time.Millisecond
	}
	deadline := time.Now().Add(wait)
	for {
		if s.probes().Reachable != nil {
			if s.probes().Reachable(ctx, address) {
				return true
			}
		} else if tcpReachable(ctx, address, time.Second) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(tick):
		}
	}
}

func (s Supervisor) waitStopped(ctx context.Context, pid int) bool {
	wait := s.StopWait
	if wait == 0 {
		wait = 10 * time.Second
	}
	tick := s.StopTick
	if tick == 0 {
		tick = 250 * time.Millisecond
	}
	deadline := time.Now().Add(wait)
	for {
		if !s.pidAlive(pid) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(tick):
		}
	}
}

// RuntimePathsFromEnv resolves the runtime dir from the same OCA path policy as config.ResolvePaths.
func RuntimePathsFromEnv() RuntimePaths {
	root := filepath.Join(cfg.ResolvePaths().CacheDir, "temporal")
	return RuntimePaths{
		Root:     root,
		Metadata: filepath.Join(root, "temporal.pid.json"),
		Log:      filepath.Join(root, "temporal.log"),
		DB:       filepath.Join(root, "temporal.db"),
		Status:   filepath.Join(root, "status.json"),
	}
}

// ReadMetadata loads OCA's supervisor metadata. ok=false means no metadata exists.
func ReadMetadata(paths RuntimePaths) (Metadata, bool, error) {
	b, err := os.ReadFile(paths.Metadata)
	if errors.Is(err, os.ErrNotExist) {
		return Metadata{}, false, nil
	}
	if err != nil {
		return Metadata{}, false, err
	}
	var meta Metadata
	if err := json.Unmarshal(b, &meta); err != nil {
		return Metadata{}, false, err
	}
	return meta, true, nil
}

// WriteMetadata writes OCA metadata atomically so status readers never see partial JSON.
func WriteMetadata(paths RuntimePaths, meta Metadata) error {
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return writeAtomic(paths.Metadata, b, 0o600)
}

// RemoveMetadata deletes OCA's supervisor metadata.
func RemoveMetadata(paths RuntimePaths) error {
	if err := os.Remove(paths.Metadata); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// CleanupStaleMetadata removes metadata whose PID is no longer alive.
func CleanupStaleMetadata(paths RuntimePaths) (bool, error) {
	meta, ok, err := ReadMetadata(paths)
	if err != nil || !ok {
		return false, err
	}
	if pidAlive(meta.PID) {
		return false, nil
	}
	return true, RemoveMetadata(paths)
}

// EvaluateStatus combines config, metadata, PID liveness, reachability, and health.
func EvaluateStatus(ctx context.Context, stack *cfg.Stack, paths RuntimePaths, probes ProbeFuncs) (Status, error) {
	status := Status{State: StateDisabled, LogPath: paths.Log, DBPath: paths.DB}
	if stack == nil || stack.Temporal == nil {
		return status, nil
	}
	status.Configured = true
	status.Enabled = stack.Temporal.IsEnabled()
	status.Address = defaultAddress(stack.Temporal.Address)
	status.Namespace = defaultNamespace(stack.Temporal.Namespace)
	if !status.Enabled {
		return status, nil
	}

	meta, hasMeta, err := ReadMetadata(paths)
	if err != nil {
		status.State = StateError
		return status, err
	}
	if hasMeta {
		status.PID = meta.PID
		status.LogPath = meta.LogPath
		status.DBPath = meta.DBPath
		status.Managed = true
		status.Running = pidAlive(meta.PID)
		if !status.Running {
			status.State = StateStale
			return status, nil
		}
	}

	if probes.Reachable != nil {
		status.Reachable = probes.Reachable(ctx, status.Address)
	} else {
		status.Reachable = tcpReachable(ctx, status.Address, time.Second)
	}
	if probes.Healthy != nil {
		status.Healthy = probes.Healthy(ctx, stack)
	} else {
		status.Healthy = status.Reachable
	}

	switch {
	case status.Managed && status.Running && status.Healthy:
		status.State = StateRunning
	case status.Managed && status.Running:
		status.State = StateUnhealthy
	case status.Reachable:
		status.State = StateUnmanaged
	default:
		status.State = StateStopped
	}
	return status, nil
}

func defaultAddress(address string) string {
	if address == "" {
		return "127.0.0.1:7233"
	}
	return address
}

func defaultNamespace(namespace string) string {
	if namespace == "" {
		return "default"
	}
	return namespace
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func tcpReachable(ctx context.Context, address string, timeout time.Duration) bool {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func isLoopbackHost(host string) bool {
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func killProcessGroup(pgid int, signal syscall.Signal) error {
	if pgid <= 0 {
		return nil
	}
	return syscall.Kill(-pgid, signal)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".oca-temporal-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func splitHostPort(address string) (string, int, error) {
	host, portRaw, err := net.SplitHostPort(address)
	if err != nil {
		if strings.Count(address, ":") == 1 {
			parts := strings.Split(address, ":")
			host, portRaw = parts[0], parts[1]
		} else {
			return "", 0, err
		}
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}
