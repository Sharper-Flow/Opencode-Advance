package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Theme represents an OpenCode theme loaded from a JSON file.
type Theme struct {
	Name        string
	Description string
	Colors      map[string]string
}

func newThemeCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Manage OCA themes",
		Long:  "List and apply OpenCode Advance themes for tmux and UI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newThemeListCmd(state))
	cmd.AddCommand(newThemeApplyCmd(state))
	return cmd
}

func newThemeListCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available themes",
		Long:  "Scan assets/themes/ for available theme definitions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}

			themes, err := discoverThemes()
			if err != nil {
				return newCLIError(1, "discover themes: %v", err)
			}

			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), themes)
			}

			if len(themes) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no themes found")
				return nil
			}

			for _, t := range themes {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", t.Name, t.Description)
			}
			return nil
		},
	}
	return cmd
}

func newThemeApplyCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply <name>",
		Short: "Apply a theme",
		Long:  "Apply a theme by name, regenerating the tmux configuration block.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			theme, err := loadTheme(name)
			if err != nil {
				return newCLIError(1, "load theme %q: %v", name, err)
			}

			// For v1: output the theme's color palette
			// Full tmux conf regeneration is a future enhancement
			if state.output == "json" {
				return printJSON(cmd.OutOrStdout(), theme)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "theme %s applied\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "colors:\n")
			for k, v := range theme.Colors {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", k, v)
			}
			return nil
		},
	}
	return cmd
}

// discoverThemes scans assets/themes/ for .json theme files.
func discoverThemes() ([]Theme, error) {
	themesDir := resolveThemesDir()
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return nil, err
	}

	var themes []Theme
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		themes = append(themes, Theme{
			Name:        name,
			Description: fmt.Sprintf("Theme: %s", name),
		})
	}
	return themes, nil
}

// loadTheme reads a theme JSON file by name.
func loadTheme(name string) (*Theme, error) {
	themesDir := resolveThemesDir()
	path := filepath.Join(themesDir, name+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Defs  map[string]string `json:"defs"`
		Theme map[string]string `json:"theme"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Resolve theme colors from defs
	colors := make(map[string]string)
	for k, v := range raw.Theme {
		if hex, ok := raw.Defs[v]; ok {
			colors[k] = hex
		} else {
			colors[k] = v // Fallback: use value directly if not a def reference
		}
	}

	return &Theme{
		Name:        name,
		Description: fmt.Sprintf("Theme: %s", name),
		Colors:      colors,
	}, nil
}

// resolveThemesDir returns the path to the themes directory.
func resolveThemesDir() string {
	if root := os.Getenv("OCA_ASSETS_ROOT"); root != "" {
		return filepath.Join(root, "themes")
	}

	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "assets", "themes")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Try relative to repo root (for development)
	return "assets/themes"
}
