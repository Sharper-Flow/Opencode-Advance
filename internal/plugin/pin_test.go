package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// TestCapturePin_ReturnsSHA verifies CapturePin returns a 40-char SHA
// from a real git checkout.
func TestCapturePin_ReturnsSHA(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	remote, _ := prepareIntegrationSetup(t)
	tmp := t.TempDir()
	checkout := filepath.Join(tmp, "advance")

	p := config.Plugin{
		Source:   remote,
		Ref:      "master",
		Checkout: checkout,
	}
	ctx := context.Background()
	if err := Prepare(ctx, p); err != nil {
		t.Fatalf("setup: %v", err)
	}

	sha, err := CapturePin(ctx, p)
	if err != nil {
		t.Fatalf("CapturePin: %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("SHA length = %d, want 40: %q", len(sha), sha)
	}
}

// TestCapturePin_SkipsNPM verifies CapturePin returns empty string
// with no error for npm-source plugins.
func TestCapturePin_SkipsNPM(t *testing.T) {
	p := config.Plugin{
		Source: "npm:some-pkg@1.0.0",
	}
	sha, err := CapturePin(context.Background(), p)
	if err != nil {
		t.Fatalf("CapturePin for npm: %v", err)
	}
	if sha != "" {
		t.Errorf("expected empty SHA for npm plugin, got %q", sha)
	}
}

// TestWritePin_UpdatesTOMLRef verifies WritePin rewrites [plugins.<name>].ref
// in a TOML file and returns true (changed).
func TestWritePin_UpdatesTOMLRef(t *testing.T) {
	tmp := t.TempDir()
	tomlFile := filepath.Join(tmp, "stack.toml")
	content := `[meta]
version = "1"

[plugins.advance]
source = "https://github.com/example/advance.git"
ref = "trunk"
checkout = "~/dev/oc-plugins/advance"
`
	if err := os.WriteFile(tomlFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := config.Plugin{
		Source:   "https://github.com/example/advance.git",
		Ref:      "trunk",
		Checkout: "~/dev/oc-plugins/advance",
	}

	changed, err := WritePin(tomlFile, "advance", p, "abc123def456")
	if err != nil {
		t.Fatalf("WritePin: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when ref differs")
	}

	// Verify the file now contains the pinned SHA.
	data, err := os.ReadFile(tomlFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `ref = "abc123def456"`) {
		t.Errorf("TOML file should contain pinned ref, got:\n%s", string(data))
	}
	// Verify other fields preserved.
	if !strings.Contains(string(data), `source = "https://github.com/example/advance.git"`) {
		t.Error("TOML file should preserve source field")
	}
}

// TestWritePin_NoOpWhenRefMatches verifies WritePin returns false (no change)
// when the current ref already matches the captured SHA.
func TestWritePin_NoOpWhenRefMatches(t *testing.T) {
	tmp := t.TempDir()
	tomlFile := filepath.Join(tmp, "stack.toml")
	content := `[plugins.advance]
source = "https://github.com/example/advance.git"
ref = "abc123"
`
	if err := os.WriteFile(tomlFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := config.Plugin{
		Source: "https://github.com/example/advance.git",
		Ref:    "abc123",
	}

	changed, err := WritePin(tomlFile, "advance", p, "abc123")
	if err != nil {
		t.Fatalf("WritePin: %v", err)
	}
	if changed {
		t.Error("expected changed=false when ref already matches")
	}
}

// TestWritePin_SkipsNPM verifies WritePin returns false (no-op) for
// npm-source plugins.
func TestWritePin_SkipsNPM(t *testing.T) {
	tmp := t.TempDir()
	tomlFile := filepath.Join(tmp, "stack.toml")
	content := `[plugins.morph]
source = "npm:opencode-morph@1.0.0"
`
	if err := os.WriteFile(tomlFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := config.Plugin{
		Source: "npm:opencode-morph@1.0.0",
	}

	changed, err := WritePin(tomlFile, "morph", p, "deadbeef")
	if err != nil {
		t.Fatalf("WritePin: %v", err)
	}
	if changed {
		t.Error("expected changed=false for npm plugin")
	}
}

// TestWritePin_PreservesOtherPlugins verifies that WritePin only changes
// the targeted plugin and leaves others untouched.
func TestWritePin_PreservesOtherPlugins(t *testing.T) {
	tmp := t.TempDir()
	tomlFile := filepath.Join(tmp, "stack.toml")
	content := `[plugins.advance]
source = "https://github.com/example/advance.git"
ref = "trunk"

[plugins.other]
source = "https://github.com/example/other.git"
ref = "v1.0"
`
	if err := os.WriteFile(tomlFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := config.Plugin{
		Source: "https://github.com/example/advance.git",
		Ref:    "trunk",
	}

	changed, err := WritePin(tomlFile, "advance", p, "newsha123")
	if err != nil {
		t.Fatalf("WritePin: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}

	data, err := os.ReadFile(tomlFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `ref = "newsha123"`) {
		t.Error("advance ref should be updated")
	}
	if !strings.Contains(s, `ref = "v1.0"`) {
		t.Error("other plugin ref should be preserved")
	}
}
