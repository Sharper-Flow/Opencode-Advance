package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/brand"
)

func TestDiscordEnableWritesWrapperAndStatus(t *testing.T) {
	tmp := t.TempDir()
	stackPath := writeDiscordTestStack(t, tmp)
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))
	t.Setenv("OCA_ASSETS_ROOT", filepath.Join(tmp, "assets"))
	assetPath := filepath.Join(tmp, "assets", "discord", "taglines.toml")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, []byte("taglines = [\"Focused delivery\", \"Spec-driven work\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stdout, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--config", stackPath, "discord", "enable"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("discord enable failed: %v\n%s", err, stdout.String())
	}

	paths := discordRuntimePaths()
	if _, err := os.Stat(paths.Wrapper); err != nil {
		t.Fatalf("wrapper not written: %v", err)
	}
	status := readDiscordStatusForTest(t, paths.Status)
	if !status.Enabled {
		t.Fatal("status should be enabled")
	}
	if status.Mode != "builtin" {
		t.Fatalf("mode = %q, want builtin", status.Mode)
	}
	taglines, err := os.ReadFile(paths.Taglines)
	if err != nil {
		t.Fatalf("taglines not installed: %v", err)
	}
	if !strings.Contains(string(taglines), "Focused delivery") {
		t.Fatalf("taglines content = %q", string(taglines))
	}
	if !strings.Contains(stdout.String(), "Discord Rich Presence enabled") {
		t.Fatalf("stdout missing enabled message: %q", stdout.String())
	}
}

func TestDiscordDisableRemovesWrapperAndStatus(t *testing.T) {
	tmp := t.TempDir()
	stackPath := writeDiscordTestStack(t, tmp)
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))
	paths := discordRuntimePaths()
	if err := os.MkdirAll(paths.Root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Wrapper, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stdout, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--config", stackPath, "discord", "disable"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("discord disable failed: %v", err)
	}
	if _, err := os.Stat(paths.Wrapper); !os.IsNotExist(err) {
		t.Fatalf("wrapper should be removed, stat err=%v", err)
	}
	status := readDiscordStatusForTest(t, paths.Status)
	if status.Enabled {
		t.Fatal("status should be disabled")
	}
}

func TestDiscordStatusReportsRuntimeState(t *testing.T) {
	tmp := t.TempDir()
	stackPath := writeDiscordTestStack(t, tmp)
	t.Setenv("OCA_CACHE_DIR", filepath.Join(tmp, "cache"))
	paths := discordRuntimePaths()
	if err := writeDiscordStatus(paths.Status, discordStatus{Enabled: true, Mode: "builtin", LastUpdate: time.Date(2026, 5, 2, 4, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd(commandOptions{Stdout: &stdout, Stderr: &stdout, Environment: brand.Environment{IsTTY: false}})
	cmd.SetArgs([]string{"--config", stackPath, "discord", "status"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("discord status failed: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "enabled") || !strings.Contains(got, "builtin") {
		t.Fatalf("status output = %q", got)
	}
}

func writeDiscordTestStack(t *testing.T, tmp string) string {
	t.Helper()
	path := filepath.Join(tmp, "stack.toml")
	content := `[meta]
version = "1.0.0"

[discord]
enabled = false
mode = "builtin"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readDiscordStatusForTest(t *testing.T, path string) discordStatus {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var status discordStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	return status
}
