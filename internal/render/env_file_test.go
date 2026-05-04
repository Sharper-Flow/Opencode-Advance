package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func TestRenderShellEnvQuotesResolvedPaths(t *testing.T) {
	paths := config.Paths{
		OpencodeConfigDir:   filepath.Join(t.TempDir(), "open code"),
		VisionConfigDir:     filepath.Join(t.TempDir(), "vision's dir"),
		CacheDir:            filepath.Join(t.TempDir(), "cache"),
		PluginCheckoutRoot_: filepath.Join(t.TempDir(), "plugins"),
	}

	got := string(RenderShellEnv(paths))
	for _, want := range []string{
		"export OCA_OPENCODE_CONFIG_DIR='" + paths.OpencodeConfigDir + "'",
		"export OCA_VISION_CONFIG_DIR='" + strings.ReplaceAll(paths.VisionConfigDir, "'", "'\\''") + "'",
		"export OCA_PLUGIN_CHECKOUT_ROOT='" + paths.PluginCheckoutRoot() + "'",
		"export OCA_CACHE_DIR='" + paths.CacheDir + "'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderShellEnv() missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteShellEnvAndStampWritesEnvBeforeStamp(t *testing.T) {
	root := t.TempDir()
	paths := config.Paths{
		OpencodeConfigDir:   filepath.Join(root, "opencode"),
		VisionConfigDir:     filepath.Join(root, "vision"),
		CacheDir:            filepath.Join(root, "cache"),
		PluginCheckoutRoot_: filepath.Join(root, "plugins"),
	}

	if err := WriteShellEnvAndStamp(paths); err != nil {
		t.Fatalf("WriteShellEnvAndStamp() = %v", err)
	}
	envInfo, err := os.Stat(paths.OCAEnvPath())
	if err != nil {
		t.Fatalf("stat env: %v", err)
	}
	stampInfo, err := os.Stat(paths.EnvStampPath())
	if err != nil {
		t.Fatalf("stat stamp: %v", err)
	}
	if stampInfo.ModTime().Before(envInfo.ModTime()) {
		t.Fatalf("stamp mtime %s is before env mtime %s", stampInfo.ModTime(), envInfo.ModTime())
	}
}

func TestWriteShellEnvAndStampDoesNotStampOnEnvWriteFailure(t *testing.T) {
	root := t.TempDir()
	blockedParent := filepath.Join(root, "not-a-dir")
	if err := os.WriteFile(blockedParent, []byte("file blocks mkdir"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	paths := config.Paths{
		OpencodeConfigDir:   filepath.Join(blockedParent, "opencode"),
		VisionConfigDir:     filepath.Join(root, "vision"),
		CacheDir:            filepath.Join(root, "cache"),
		PluginCheckoutRoot_: filepath.Join(root, "plugins"),
	}

	if err := WriteShellEnvAndStamp(paths); err == nil {
		t.Fatal("WriteShellEnvAndStamp() = nil, want env write error")
	}
	if _, err := os.Stat(paths.EnvStampPath()); err == nil || !os.IsNotExist(err) {
		t.Fatalf("stamp should not exist after env write failure, stat err=%v", err)
	}
}

func TestWriteStampTouchesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env.stamp")
	if err := WriteStamp(path); err != nil {
		t.Fatalf("initial WriteStamp() = %v", err)
	}
	first, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat first: %v", err)
	}
	time.Sleep(time.Millisecond)
	if err := WriteStamp(path); err != nil {
		t.Fatalf("second WriteStamp() = %v", err)
	}
	second, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat second: %v", err)
	}
	if !second.ModTime().After(first.ModTime()) {
		t.Fatalf("WriteStamp did not advance mtime: first=%s second=%s", first.ModTime(), second.ModTime())
	}
}
