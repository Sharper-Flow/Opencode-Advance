package sync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestInvokeAdvance_RunsSyncCommandInCheckoutAndRedactsOutput(t *testing.T) {
	checkout := t.TempDir()
	script := filepath.Join(checkout, "sync.sh")
	outFile := filepath.Join(checkout, "ran.txt")
	content := "#!/bin/sh\npwd > ran.txt\necho 'authorization: Bearer secret-token'\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	plugin := cfg.Plugin{Checkout: checkout, Sync: script}
	res, err := InvokeAdvance(context.Background(), plugin)
	if err != nil {
		t.Fatalf("InvokeAdvance: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code=%d want 0", res.ExitCode)
	}
	runDir, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read ran.txt: %v", err)
	}
	if strings.TrimSpace(string(runDir)) != checkout {
		t.Fatalf("script ran in %q want %q", strings.TrimSpace(string(runDir)), checkout)
	}
	if strings.Contains(string(res.Output), "secret-token") {
		t.Fatalf("output not redacted: %s", string(res.Output))
	}
	if !strings.Contains(string(res.Output), "***REDACTED***") {
		t.Fatalf("redaction marker missing: %s", string(res.Output))
	}
}

func TestInvokeAdvance_ReturnsNonZeroWithRedactedOutput(t *testing.T) {
	checkout := t.TempDir()
	script := filepath.Join(checkout, "sync.sh")
	content := "#!/bin/sh\necho 'API_KEY=topsecret'\nexit 7\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	plugin := cfg.Plugin{Checkout: checkout, Sync: script + " --fix"}
	res, err := InvokeAdvance(context.Background(), plugin)
	if err == nil {
		t.Fatal("expected non-zero error")
	}
	if res.ExitCode != 7 {
		t.Fatalf("exit code=%d want 7", res.ExitCode)
	}
	if strings.Contains(string(res.Output), "topsecret") {
		t.Fatalf("output not redacted: %s", string(res.Output))
	}
}
