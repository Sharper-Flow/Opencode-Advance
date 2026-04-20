package plugin

import (
	"context"
	"fmt"
	"os"

	"github.com/Sharper-Flow/Opencode-Advance/internal/config"
	"github.com/BurntSushi/toml"
)

// CapturePin returns the current HEAD SHA of a git-sourced plugin's checkout.
// Returns ("", nil) for npm-source plugins (no capture needed).
// The returned SHA is a 40-character hex string.
func CapturePin(ctx context.Context, p config.Plugin) (string, error) {
	if p.IsNPMSource() {
		return "", nil
	}

	sha, err := RevParseHEAD(ctx, p.Checkout)
	if err != nil {
		return "", fmt.Errorf("capture pin: %w", err)
	}
	return sha, nil
}

// WritePin updates [plugins.<name>].ref in the TOML file to the given SHA.
// It reads the file as a generic map, updates the nested key, and writes
// the whole file back (TOML round-trip).
//
// Returns (false, nil) when:
//   - the plugin is npm-sourced (no-op)
//   - the current ref already matches sha (idempotent no-op)
//
// Returns (true, nil) when the file was rewritten with the new ref.
func WritePin(tomlFile, pluginName string, p config.Plugin, sha string) (bool, error) {
	if p.IsNPMSource() {
		return false, nil
	}

	// Idempotent: skip when already pinned.
	if p.Ref == sha {
		return false, nil
	}

	// Read TOML as generic map.
	data, err := os.ReadFile(tomlFile)
	if err != nil {
		return false, fmt.Errorf("write pin: read %s: %w", tomlFile, err)
	}

	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return false, fmt.Errorf("write pin: parse %s: %w", tomlFile, err)
	}

	// Navigate to [plugins.<name>].
	plugins, ok := root["plugins"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("write pin: [plugins] section missing or not a table")
	}
	entry, ok := plugins[pluginName].(map[string]any)
	if !ok {
		return false, fmt.Errorf("write pin: [plugins.%s] section missing or not a table", pluginName)
	}

	// Update ref.
	entry["ref"] = sha

	// Encode back to TOML.
	out, err := toml.Marshal(root)
	if err != nil {
		return false, fmt.Errorf("write pin: encode: %w", err)
	}

	if err := os.WriteFile(tomlFile, out, 0644); err != nil {
		return false, fmt.Errorf("write pin: write %s: %w", tomlFile, err)
	}

	return true, nil
}
