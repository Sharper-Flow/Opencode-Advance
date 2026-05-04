package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

// RenderShellEnv renders the OCA-owned shell environment file. Shell hooks
// source this file instead of re-sourcing user-owned rc files.
func RenderShellEnv(paths cfg.Paths) []byte {
	vars := []struct {
		key   string
		value string
	}{
		{key: "OCA_OPENCODE_CONFIG_DIR", value: paths.OpencodeConfigDir},
		{key: "OCA_VISION_CONFIG_DIR", value: paths.VisionConfigDir},
		{key: "OCA_PLUGIN_CHECKOUT_ROOT", value: paths.PluginCheckoutRoot()},
		{key: "OCA_CACHE_DIR", value: paths.CacheDir},
	}

	var b bytes.Buffer
	b.WriteString("# This file is managed by OpenCode Advance (oca).\n")
	b.WriteString("# Do not edit manually — changes will be overwritten on next `oca apply`.\n\n")
	for _, v := range vars {
		fmt.Fprintf(&b, "export %s=%s\n", v.key, shellQuote(v.value))
	}
	return b.Bytes()
}

// RenderShellEnvWriteFile writes the rendered OCA env file atomically.
func RenderShellEnvWriteFile(path string, paths cfg.Paths) error {
	if _, err := WriteAtomic(path, RenderShellEnv(paths), 0o644, 3); err != nil {
		return fmt.Errorf("write shell env %s: %w", path, err)
	}
	return nil
}

// WriteStamp touches path, creating parent directories as needed.
func WriteStamp(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create stamp dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open stamp %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close stamp %s: %w", path, err)
	}
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		return fmt.Errorf("touch stamp %s: %w", path, err)
	}
	return nil
}

// WriteShellEnvAndStamp writes env.sh first, then touches env.stamp. This
// order is the TOCTOU guard consumed by shell hooks.
func WriteShellEnvAndStamp(paths cfg.Paths) error {
	if err := RenderShellEnvWriteFile(paths.OCAEnvPath(), paths); err != nil {
		return err
	}
	return WriteStamp(paths.EnvStampPath())
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
