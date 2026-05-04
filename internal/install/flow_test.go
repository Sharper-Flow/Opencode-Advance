package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/Sharper-Flow/Opencode-Advance/internal/health"
)

// --- Install tests ---

func TestInstall(t *testing.T) {
	t.Run("fresh install creates dirs and writes rc files", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("OCA_OPENCODE_CONFIG_DIR", filepath.Join(home, ".config", "opencode"))
		t.Setenv("OCA_VISION_CONFIG_DIR", filepath.Join(home, ".config", "vision"))
		t.Setenv("OCA_CACHE_DIR", filepath.Join(home, ".cache", "oca"))
		mockedHome := func() (string, error) { return home, nil }
		mockedBin := func() (string, error) { return home + "/bin/oca", nil }
		mockedPrereqs := func(ctx context.Context, _ interface{}, _ health.Options) ([]health.Check, error) {
			return []health.Check{
				{Name: "prerequisites.git", Status: health.StatusPass},
				{Name: "prerequisites.tmux", Status: health.StatusPass},
				{Name: "prerequisites.opencode", Status: health.StatusPass},
				{Name: "prerequisites.vision", Status: health.StatusPass},
				{Name: "prerequisites.temporal", Status: health.StatusPass},
			}, nil
		}

		origHome, origBin, origPrereqs := userHomeDir, osExecutable, checkPrereqs
		defer func() { userHomeDir, osExecutable, checkPrereqs = origHome, origBin, origPrereqs }()
		userHomeDir, osExecutable, checkPrereqs = mockedHome, mockedBin, mockedPrereqs

		bashrc := filepath.Join(home, ".bashrc")
		zshrc := filepath.Join(home, ".zshrc")

		result, err := Install(context.Background(), InstallOptions{})
		if err != nil {
			t.Fatal(err)
		}

		if result.BinaryPath != home+"/bin/oca" {
			t.Errorf("BinaryPath = %q, want %q", result.BinaryPath, home+"/bin/oca")
		}
		if len(result.CreatedDirs) == 0 {
			t.Error("should have created directories")
		}
		if len(result.WroteRCFiles) != 2 {
			t.Errorf("WroteRCFiles = %d, want 2", len(result.WroteRCFiles))
		}

		// Verify rc files have managed blocks
		for _, rc := range []string{bashrc, zshrc} {
			content, err := os.ReadFile(rc)
			if err != nil {
				t.Errorf("read %s: %v", rc, err)
				continue
			}
			if !contains(string(content), SentinelStart) {
				t.Errorf("%s missing start sentinel", rc)
			}
		}

		paths := config.ResolvePaths()
		if _, err := os.Stat(paths.OCAEnvPath()); err != nil {
			t.Fatalf("install should write shell env: %v", err)
		}
		if _, err := os.Stat(paths.EnvStampPath()); err != nil {
			t.Fatalf("install should write env stamp: %v", err)
		}
	})

	t.Run("idempotent install produces same output", func(t *testing.T) {
		home := t.TempDir()
		mockedHome := func() (string, error) { return home, nil }
		mockedBin := func() (string, error) { return "/usr/local/bin/oca", nil }
		mockedPrereqs := allPassPrereqs

		origHome, origBin, origPrereqs := userHomeDir, osExecutable, checkPrereqs
		defer func() { userHomeDir, osExecutable, checkPrereqs = origHome, origBin, origPrereqs }()
		userHomeDir, osExecutable, checkPrereqs = mockedHome, mockedBin, mockedPrereqs

		_, err := Install(context.Background(), InstallOptions{})
		if err != nil {
			t.Fatal(err)
		}

		bashrc := filepath.Join(home, ".bashrc")
		first, _ := os.ReadFile(bashrc)

		_, err = Install(context.Background(), InstallOptions{})
		if err != nil {
			t.Fatal(err)
		}
		second, _ := os.ReadFile(bashrc)

		if string(first) != string(second) {
			t.Error("idempotent install should produce identical rc file")
		}
	})

	t.Run("missing required prereq returns error", func(t *testing.T) {
		home := t.TempDir()
		mockedHome := func() (string, error) { return home, nil }
		mockedBin := func() (string, error) { return "/usr/local/bin/oca", nil }
		mockedPrereqs := func(ctx context.Context, _ interface{}, _ health.Options) ([]health.Check, error) {
			return []health.Check{
				{Name: "prerequisites.git", Status: health.StatusFail, Message: "git not found"},
				{Name: "prerequisites.tmux", Status: health.StatusPass},
				{Name: "prerequisites.opencode", Status: health.StatusPass},
			}, nil
		}

		origHome, origBin, origPrereqs := userHomeDir, osExecutable, checkPrereqs
		defer func() { userHomeDir, osExecutable, checkPrereqs = origHome, origBin, origPrereqs }()
		userHomeDir, osExecutable, checkPrereqs = mockedHome, mockedBin, mockedPrereqs

		_, err := Install(context.Background(), InstallOptions{})
		if err == nil {
			t.Fatal("expected error for missing prereq")
		}

		var pe *ErrPrereqFailed
		if !errors.As(err, &pe) {
			t.Errorf("expected ErrPrereqFailed, got %T: %v", err, err)
		}
	})
}

// --- Uninstall tests ---

func TestUninstall(t *testing.T) {
	t.Run("clean uninstall removes blocks", func(t *testing.T) {
		home := t.TempDir()
		mockedHome := func() (string, error) { return home, nil }
		mockedBin := func() (string, error) { return "/usr/local/bin/oca", nil }

		origHome, origBin := userHomeDir, osExecutable
		defer func() { userHomeDir, osExecutable = origHome, origBin }()
		userHomeDir, osExecutable = mockedHome, mockedBin

		bashrc := filepath.Join(home, ".bashrc")
		zshrc := filepath.Join(home, ".zshrc")

		// Write blocks matching what Install would produce
		expectedBlock := RenderShellProfile("/usr/local/bin/oca")
		WriteBlock(bashrc, expectedBlock)
		WriteBlock(zshrc, expectedBlock)

		result, err := Uninstall(UninstallOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.RemovedRCFiles) != 2 {
			t.Errorf("RemovedRCFiles = %d, want 2", len(result.RemovedRCFiles))
		}

		// Verify blocks removed
		for _, rc := range []string{bashrc, zshrc} {
			content, _ := os.ReadFile(rc)
			if contains(string(content), SentinelStart) {
				t.Errorf("%s still has managed block after uninstall", rc)
			}
		}
	})

	t.Run("edited block without force returns error", func(t *testing.T) {
		home := t.TempDir()
		mockedHome := func() (string, error) { return home, nil }
		mockedBin := func() (string, error) { return "/usr/local/bin/oca", nil }

		origHome, origBin := userHomeDir, osExecutable
		defer func() { userHomeDir, osExecutable = origHome, origBin }()
		userHomeDir, osExecutable = mockedHome, mockedBin

		bashrc := filepath.Join(home, ".bashrc")
		// Write a block with arbitrary content (will be "edited" relative to expected template)
		WriteBlock(bashrc, "original content")
		WriteBlock(bashrc, "original content")

		// Now manually edit the block content
		data, _ := os.ReadFile(bashrc)
		edited := replaceString(string(data), "original content", "EDITED content")
		os.WriteFile(bashrc, []byte(edited), 0o644)

		_, err := Uninstall(UninstallOptions{Force: false})
		if err == nil {
			t.Fatal("expected error for edited block without force")
		}

		var be *ErrBlockEdited
		if !errors.As(err, &be) {
			t.Errorf("expected ErrBlockEdited, got %T: %v", err, err)
		}
	})

	t.Run("edited block with force succeeds", func(t *testing.T) {
		home := t.TempDir()
		mockedHome := func() (string, error) { return home, nil }
		mockedBin := func() (string, error) { return "/usr/local/bin/oca", nil }

		origHome, origBin := userHomeDir, osExecutable
		defer func() { userHomeDir, osExecutable = origHome, origBin }()
		userHomeDir, osExecutable = mockedHome, mockedBin

		bashrc := filepath.Join(home, ".bashrc")
		WriteBlock(bashrc, "original content")

		// Edit the block
		data, _ := os.ReadFile(bashrc)
		edited := replaceString(string(data), "original content", "EDITED content")
		os.WriteFile(bashrc, []byte(edited), 0o644)

		_, err := Uninstall(UninstallOptions{Force: true})
		if err != nil {
			t.Fatal(err)
		}

		content, _ := os.ReadFile(bashrc)
		if contains(string(content), SentinelStart) {
			t.Error("block should be removed with --force")
		}
	})

	t.Run("no blocks is a no-op", func(t *testing.T) {
		home := t.TempDir()
		mockedHome := func() (string, error) { return home, nil }
		mockedBin := func() (string, error) { return "/usr/local/bin/oca", nil }

		origHome, origBin := userHomeDir, osExecutable
		defer func() { userHomeDir, osExecutable = origHome, origBin }()
		userHomeDir, osExecutable = mockedHome, mockedBin

		_, err := Uninstall(UninstallOptions{})
		if err != nil {
			t.Errorf("uninstall with no blocks should not error: %v", err)
		}
	})
}

// --- helpers ---

func allPassPrereqs(ctx context.Context, _ interface{}, _ health.Options) ([]health.Check, error) {
	return []health.Check{
		{Name: "prerequisites.git", Status: health.StatusPass},
		{Name: "prerequisites.tmux", Status: health.StatusPass},
		{Name: "prerequisites.opencode", Status: health.StatusPass},
		{Name: "prerequisites.vision", Status: health.StatusPass},
		{Name: "prerequisites.temporal", Status: health.StatusPass},
	}, nil
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func replaceString(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

// Stub for types that flow.go will define — this lets the test compile.
// The actual types are in flow.go.
var _ = fmt.Sprintf
