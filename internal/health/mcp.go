package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
)

type Options struct {
	VisionAdminURL string
	Timeout        time.Duration
	HTTPClient     *http.Client
}

type versionResponse struct {
	Version string          `json:"version"`
	Build   string          `json:"build"`
	API     map[string]bool `json:"api"`
}

type serversResponse struct {
	Servers []serverStatus `json:"servers"`
}

type serverStatus struct {
	Name         string `json:"name"`
	State        string `json:"state"`
	Port         int    `json:"port"`
	Transport    string `json:"transport"`
	Autostart    bool   `json:"autostart"`
	Required     bool   `json:"required"`
	LastError    string `json:"last_error"`
	RestartCount int    `json:"restart_count"`
}

func CheckMCP(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	base := opts.VisionAdminURL
	if base == "" {
		base = "http://127.0.0.1:6275"
	}
	client := opts.HTTPClient
	if client == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 5 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	checks := []Check{}

	start := time.Now()
	vr, versionErr := fetchVersion(ctx, client, base+"/version")
	if versionErr != nil {
		checks = append(checks, Check{
			Name:    "vision.version",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Vision unreachable or missing /version: %v", versionErr),
			Hint:    "run vision start or upgrade Vision to a build that exposes GET /version and GET /v1/servers",
			Elapsed: time.Since(start),
		})
		checks = append(checks, envFileChecks(stack)...)
		checks = append(checks, commandPathChecks(stack)...)
		return checks, nil
	}
	if !vr.API["v1_servers"] {
		checks = append(checks, Check{
			Name:    "vision.version",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Vision %s lacks api.v1_servers capability", vr.Version),
			Hint:    "upgrade Vision to a build exposing GET /v1/servers",
			Elapsed: time.Since(start),
		})
		checks = append(checks, envFileChecks(stack)...)
		checks = append(checks, commandPathChecks(stack)...)
		return checks, nil
	}
	checks = append(checks, Check{Name: "vision.version", Status: StatusPass, Message: fmt.Sprintf("Vision %s supports /v1/servers", vr.Version), Elapsed: time.Since(start)})

	start = time.Now()
	sr, err := fetchServers(ctx, client, base+"/v1/servers")
	if err != nil {
		checks = append(checks, Check{
			Name:    "vision.v1_servers",
			Status:  StatusWarn,
			Message: fmt.Sprintf("failed to query /v1/servers: %v", err),
			Hint:    "ensure Vision is running and bound to the expected admin port",
			Elapsed: time.Since(start),
		})
		checks = append(checks, envFileChecks(stack)...)
		checks = append(checks, commandPathChecks(stack)...)
		return checks, nil
	}
	statusByName := map[string]serverStatus{}
	for _, s := range sr.Servers {
		statusByName[s.Name] = s
	}
	for name, declared := range stack.MCP.Servers {
		if !declared.IsEnabled() {
			checks = append(checks, Check{Name: "mcp." + name, Status: StatusPass, Message: "disabled in stack.toml", Elapsed: 0})
			continue
		}
		ss, ok := statusByName[name]
		if !ok {
			st := StatusWarn
			if declared.Required {
				st = StatusFail
			}
			checks = append(checks, Check{Name: "mcp." + name, Status: st, Message: "declared but not registered in Vision", Hint: "run oca apply --target mcp or ensure Vision reloaded its config"})
			continue
		}
		switch ss.State {
		case "running", "starting":
			checks = append(checks, Check{Name: "mcp." + name, Status: StatusPass, Message: fmt.Sprintf("%s on :%d", ss.State, ss.Port)})
		case "stopped":
			st := StatusWarn
			if ss.Required || declared.Required {
				st = StatusFail
			}
			checks = append(checks, Check{Name: "mcp." + name, Status: st, Message: "stopped", Hint: "start Vision or enable autostart for this server"})
		case "failed", "crashed":
			st := StatusWarn
			if ss.Required || declared.Required {
				st = StatusFail
			}
			redactedErr := string(render.Redact([]byte(ss.LastError)))
			checks = append(checks, Check{Name: "mcp." + name, Status: st, Message: fmt.Sprintf("%s: %s", ss.State, redactedErr), Hint: "inspect Vision logs or the server configuration"})
		default:
			checks = append(checks, Check{Name: "mcp." + name, Status: StatusWarn, Message: fmt.Sprintf("unknown state %q", ss.State)})
		}
	}
	checks = append(checks, envFileChecks(stack)...)
	checks = append(checks, commandPathChecks(stack)...)
	return checks, nil
}

func fetchVersion(ctx context.Context, client *http.Client, url string) (*versionResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var vr versionResponse
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		return nil, err
	}
	return &vr, nil
}

func fetchServers(ctx context.Context, client *http.Client, url string) (*serversResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var sr serversResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

func envFileChecks(stack *cfg.Stack) []Check {
	checks := []Check{}
	for name, s := range stack.MCP.Servers {
		if s.EnvFile == "" {
			continue
		}
		if _, err := os.Stat(s.EnvFile); err != nil {
			checks = append(checks, Check{
				Name:    "mcp." + name + ".env_file",
				Status:  StatusWarn,
				Message: "env_file missing: " + s.EnvFile,
				Hint:    "create the file or remove the env_file reference; OCA never reads secret contents",
			})
		} else {
			checks = append(checks, Check{Name: "mcp." + name + ".env_file", Status: StatusPass, Message: "env_file present"})
		}
	}
	return checks
}

// commandPathChecks warns when an enabled stdio server declares a non-absolute
// command. Vision resolves such commands via $PATH at spawn time, which makes
// the stack non-hermetic and can cause surprises across hosts. OCA never
// reads the command value beyond this advisory path check.
func commandPathChecks(stack *cfg.Stack) []Check {
	checks := []Check{}
	for name, s := range stack.MCP.Servers {
		if !s.IsEnabled() {
			continue
		}
		if s.Command == "" {
			continue
		}
		if filepath.IsAbs(s.Command) {
			continue
		}
		checks = append(checks, Check{
			Name:    "mcp." + name + ".command_path",
			Status:  StatusWarn,
			Message: "command is not an absolute path: " + s.Command,
			Hint:    "prefer an absolute path (e.g. /usr/local/bin/" + s.Command + ") so spawns do not depend on $PATH resolution",
		})
	}
	return checks
}

// HasFailures reports whether any check has StatusFail. Used by doctor
// and apply to determine whether to exit non-zero after a health pass.
func HasFailures(checks []Check) bool {
	for _, c := range checks {
		if c.Status == StatusFail {
			return true
		}
	}
	return false
}

// HasWarnings reports whether any check has StatusWarn. Callers use this
// to surface a non-fatal advisory summary without forcing a failure exit.
func HasWarnings(checks []Check) bool {
	for _, c := range checks {
		if c.Status == StatusWarn {
			return true
		}
	}
	return false
}

// Summary returns a compact "N pass, M warn, K fail" string suitable for
// a single-line status report. Zero-count buckets are omitted.
func Summary(checks []Check) string {
	pass := 0
	warn := 0
	fail := 0
	for _, c := range checks {
		switch c.Status {
		case StatusPass:
			pass++
		case StatusWarn:
			warn++
		case StatusFail:
			fail++
		}
	}
	parts := []string{}
	if pass > 0 {
		parts = append(parts, fmt.Sprintf("%d pass", pass))
	}
	if warn > 0 {
		parts = append(parts, fmt.Sprintf("%d warn", warn))
	}
	if fail > 0 {
		parts = append(parts, fmt.Sprintf("%d fail", fail))
	}
	return strings.Join(parts, ", ")
}
