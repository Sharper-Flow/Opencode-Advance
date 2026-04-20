package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRunBuild_Success(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found")
	}

	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "checkout")
	os.MkdirAll(checkout, 0755)

	// Build command creates a marker file.
	p := config.Plugin{
		Checkout: checkout,
		Build:    []string{"touch marker.txt"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := RunBuild(ctx, p)
	if err != nil {
		t.Fatalf("RunBuild failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(checkout, "marker.txt")); err != nil {
		t.Errorf("marker.txt not created by build: %v", err)
	}
}

func TestRunBuild_SetsCIEnv(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found")
	}

	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "checkout")
	os.MkdirAll(checkout, 0755)

	// Build command writes CI env var to a file.
	p := config.Plugin{
		Checkout: checkout,
		Build:    []string{fmt.Sprintf("printf $CI > %s/ci.txt", checkout)},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := RunBuild(ctx, p)
	if err != nil {
		t.Fatalf("RunBuild failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(checkout, "ci.txt"))
	if err != nil {
		t.Fatalf("read ci.txt: %v", err)
	}
	if string(data) != "true" {
		t.Errorf("CI = %q, want true", string(data))
	}
}

func TestRunBuild_NonZeroExitReturnsOutput(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found")
	}

	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "checkout")
	os.MkdirAll(checkout, 0755)

	p := config.Plugin{
		Checkout: checkout,
		Build:    []string{"echo 'build failed' && exit 1"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := RunBuild(ctx, p)
	if err == nil {
		t.Fatal("expected error on non-zero exit")
	}
	// Error should contain the captured output.
	if !contains(err.Error(), "build failed") {
		t.Errorf("error should contain build output: %v", err)
	}
}

func TestRunBuild_NoBuildCommands_IsNoOp(t *testing.T) {
	p := config.Plugin{
		Checkout: "/nonexistent",
		Build:    nil,
	}
	ctx := context.Background()
	err := RunBuild(ctx, p)
	if err != nil {
		t.Errorf("RunBuild with no commands should be no-op: %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
