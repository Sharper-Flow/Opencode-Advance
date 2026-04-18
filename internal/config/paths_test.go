package config

import (
	"path/filepath"
	"testing"
)

func TestResolvePaths_HonorsEnvOverrides(t *testing.T) {
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", "/tmp/test-opencode")
	t.Setenv("OCA_VISION_CONFIG_DIR", "/tmp/test-vision")
	t.Setenv("OCA_CACHE_DIR", "/tmp/test-cache")

	p := ResolvePaths()
	if p.OpencodeConfigDir != "/tmp/test-opencode" {
		t.Errorf("OpencodeConfigDir = %q, want /tmp/test-opencode", p.OpencodeConfigDir)
	}
	if p.VisionConfigDir != "/tmp/test-vision" {
		t.Errorf("VisionConfigDir = %q, want /tmp/test-vision", p.VisionConfigDir)
	}
	if p.CacheDir != "/tmp/test-cache" {
		t.Errorf("CacheDir = %q, want /tmp/test-cache", p.CacheDir)
	}
	if p.OpencodeJSON() != filepath.Join("/tmp/test-opencode", "opencode.json") {
		t.Errorf("OpencodeJSON() = %q, mismatched", p.OpencodeJSON())
	}
	if p.VisionServersYAML() != filepath.Join("/tmp/test-vision", "servers.yaml") {
		t.Errorf("VisionServersYAML() = %q, mismatched", p.VisionServersYAML())
	}
}

func TestResolvePaths_FallsBackToDefaults(t *testing.T) {
	t.Setenv("OCA_OPENCODE_CONFIG_DIR", "")
	t.Setenv("OCA_VISION_CONFIG_DIR", "")
	t.Setenv("OCA_CACHE_DIR", "")
	t.Setenv("XDG_RUNTIME_DIR", "")

	p := ResolvePaths()
	if p.OpencodeConfigDir == "" || filepath.Base(p.OpencodeConfigDir) != "opencode" {
		t.Errorf("OpencodeConfigDir fallback unexpected: %q", p.OpencodeConfigDir)
	}
	if p.VisionConfigDir == "" || filepath.Base(p.VisionConfigDir) != "vision" {
		t.Errorf("VisionConfigDir fallback unexpected: %q", p.VisionConfigDir)
	}
	if p.CacheDir == "" {
		t.Error("CacheDir fallback is empty")
	}
}

func TestServer_IsEnabled_NilDefaultsTrue(t *testing.T) {
	var s Server
	if !s.IsEnabled() {
		t.Error("zero-value Server.IsEnabled() = false, want true")
	}
	f := false
	s.Enabled = &f
	if s.IsEnabled() {
		t.Error("Enabled=&false should be disabled")
	}
	tr := true
	s.Enabled = &tr
	if !s.IsEnabled() {
		t.Error("Enabled=&true should be enabled")
	}
}
