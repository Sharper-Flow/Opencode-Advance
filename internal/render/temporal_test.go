package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func TestRenderTemporalEnv_AllFields(t *testing.T) {
	stack := &config.Stack{
		Temporal: &config.TemporalSection{
			Enabled:     boolPtr(true),
			Address:     "127.0.0.1:7233",
			Namespace:   "default",
			AllowRemote: boolPtr(false),
			NodePath:    "/usr/local/bin/node",
		},
	}
	got, err := RenderTemporalEnv(stack)
	if err != nil {
		t.Fatalf("RenderTemporalEnv: %v", err)
	}
	want := loadGolden(t, filepath.Join("testdata", "temporal-all-fields", "temporal.env.golden"))
	compareGolden(t, got, want)
}

func TestRenderTemporalEnv_Minimal(t *testing.T) {
	stack := &config.Stack{
		Temporal: &config.TemporalSection{
			Enabled:   boolPtr(true),
			Address:   "127.0.0.1:7233",
			Namespace: "default",
		},
	}
	got, err := RenderTemporalEnv(stack)
	if err != nil {
		t.Fatalf("RenderTemporalEnv: %v", err)
	}
	want := loadGolden(t, filepath.Join("testdata", "temporal-minimal", "temporal.env.golden"))
	compareGolden(t, got, want)
}

func TestRenderTemporalEnv_Disabled(t *testing.T) {
	stack := &config.Stack{
		Temporal: &config.TemporalSection{
			Enabled: boolPtr(false),
		},
	}
	got, err := RenderTemporalEnv(stack)
	if err != nil {
		t.Fatalf("RenderTemporalEnv: %v", err)
	}
	if got != "" {
		t.Errorf("disabled temporal should produce empty string, got %q", got)
	}
}

func TestRenderTemporalEnv_Nil(t *testing.T) {
	stack := &config.Stack{}
	got, err := RenderTemporalEnv(stack)
	if err != nil {
		t.Fatalf("RenderTemporalEnv: %v", err)
	}
	if got != "" {
		t.Errorf("nil temporal should produce empty string, got %q", got)
	}
}

func TestRenderTemporalEnv_CustomNamespace(t *testing.T) {
	stack := &config.Stack{
		Temporal: &config.TemporalSection{
			Enabled:     boolPtr(true),
			Address:     "127.0.0.1:7233",
			Namespace:   "production",
			AllowRemote: boolPtr(true),
		},
	}
	got, err := RenderTemporalEnv(stack)
	if err != nil {
		t.Fatalf("RenderTemporalEnv: %v", err)
	}
	want := loadGolden(t, filepath.Join("testdata", "temporal-custom-namespace", "temporal.env.golden"))
	compareGolden(t, got, want)
}

func TestCacheDir_Explicit(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", "/custom/cache")
	got := CacheDir()
	if got != "/custom/cache" {
		t.Errorf("CacheDir() = %q, want /custom/cache", got)
	}
}

func TestCacheDir_XDGFallback(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", "")
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	got := CacheDir()
	want := "/run/user/1000/opencode-advance"
	if got != want {
		t.Errorf("CacheDir() = %q, want %q", got, want)
	}
}

func TestCacheDir_TMPDIRFallback(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("TMPDIR", "/var/tmp")
	got := CacheDir()
	want := "/var/tmp/opencode-advance"
	if got != want {
		t.Errorf("CacheDir() = %q, want %q", got, want)
	}
}

func TestCacheDir_DefaultFallback(t *testing.T) {
	t.Setenv("OCA_CACHE_DIR", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("TMPDIR", "")
	got := CacheDir()
	want := "/tmp/opencode-advance"
	if got != want {
		t.Errorf("CacheDir() = %q, want %q", got, want)
	}
}

// loadGolden reads a golden file.
func loadGolden(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	return string(data)
}

// compareGolden compares rendered output to golden content.
func compareGolden(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
