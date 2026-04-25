package health

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
	"github.com/Sharper-Flow/Opencode-Advance/internal/subprocess"
)

func init() {
	registerBuiltin("temporal", CheckTemporal)
}

// runSubprocess is swappable for testing.
var runSubprocess = subprocess.Run

func CheckTemporal(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	if !stack.Temporal.IsEnabled() {
		return []Check{{Name: "temporal.enabled", Status: StatusPass, Message: "temporal disabled or not configured"}}, nil
	}

	addr := stack.Temporal.Address
	if addr == "" {
		addr = "127.0.0.1:7233"
	}
	ns := stack.Temporal.Namespace
	if ns == "" {
		ns = "default"
	}

	checks := []Check{}

	// Step 1: TCP reachability
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		checks = append(checks, Check{
			Name:    "temporal.reachable",
			Status:  StatusWarn,
			Message: fmt.Sprintf("temporal server unreachable at %s: %v", addr, err),
			Hint:    "ensure temporal server is running and address is correct",
			Elapsed: time.Since(start),
		})
		return checks, nil
	}
	conn.Close()
	checks = append(checks, Check{
		Name:    "temporal.reachable",
		Status:  StatusPass,
		Message: fmt.Sprintf("temporal server reachable at %s", addr),
		Elapsed: time.Since(start),
	})

	// Step 2: namespace check via CLI
	start = time.Now()
	cmd := subprocess.Cmd{
		Name:    "temporal",
		Args:    []string{"operator", "namespace", "describe", ns, "--address", addr},
		Timeout: 5 * time.Second,
	}
	res, _ := runSubprocess(ctx, cmd)
	elapsed := time.Since(start)

	output := string(res.Output)
	redactedOutput := string(render.Redact(res.Output))

	switch res.ExitClass {
	case subprocess.ExitSuccess:
		checks = append(checks, Check{
			Name:    "temporal.namespace",
			Status:  StatusPass,
			Message: fmt.Sprintf("namespace %q exists on %s", ns, addr),
			Elapsed: elapsed,
		})
	case subprocess.ExitTimeout:
		checks = append(checks, Check{
			Name:    "temporal.namespace",
			Status:  StatusWarn,
			Message: "namespace check timed out after 5s",
			Hint:    "temporal server may be under load or unreachable",
			Elapsed: elapsed,
		})
	case subprocess.ExitNonZero:
		if strings.Contains(output, "Namespace") && strings.Contains(output, "not found") {
			checks = append(checks, Check{
				Name:    "temporal.namespace",
				Status:  StatusWarn,
				Message: fmt.Sprintf("namespace %q not found on %s", ns, addr),
				Hint:    fmt.Sprintf("create the namespace with: temporal operator namespace create %s --address %s", ns, addr),
				Elapsed: elapsed,
			})
		} else {
			checks = append(checks, Check{
				Name:    "temporal.namespace",
				Status:  StatusWarn,
				Message: fmt.Sprintf("namespace check failed: %s", redactedOutput),
				Elapsed: elapsed,
			})
		}
	default:
		// ExitSignal or other unexpected
		checks = append(checks, Check{
			Name:    "temporal.namespace",
			Status:  StatusWarn,
			Message: fmt.Sprintf("namespace check failed: %s", redactedOutput),
			Elapsed: elapsed,
		})
	}

	return checks, nil
}
