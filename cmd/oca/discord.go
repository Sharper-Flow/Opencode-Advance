package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	ocadiscord "github.com/Sharper-Flow/Opencode-Advance/internal/discord"
	"github.com/spf13/cobra"
)

const discordDefaultAppID = "1476685752363516135"

type discordPaths struct {
	Root         string
	Wrapper      string
	Status       string
	LastIndex    string
	LastUpdate   string
	Taglines     string
	SessionStart string
}

type discordStatus struct {
	Enabled    bool      `json:"enabled"`
	Mode       string    `json:"mode"`
	AppID      string    `json:"app_id"`
	Wrapper    string    `json:"wrapper"`
	LastUpdate time.Time `json:"last_update,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
}

func discordRuntimePaths() discordPaths {
	root := filepath.Join(cfg.ResolvePaths().CacheDir, "discord")
	return discordPaths{
		Root:         root,
		Wrapper:      filepath.Join(root, "discord-update.sh"),
		Status:       filepath.Join(root, "status.json"),
		LastIndex:    filepath.Join(root, "last-tagline"),
		LastUpdate:   filepath.Join(root, "last-update"),
		Taglines:     filepath.Join(root, "taglines.toml"),
		SessionStart: filepath.Join(root, "session-start"),
	}
}

func newDiscordCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discord",
		Short: "Manage Discord Rich Presence",
	}
	cmd.AddCommand(newDiscordEnableCmd(state))
	cmd.AddCommand(newDiscordDisableCmd(state))
	cmd.AddCommand(newDiscordStatusCmd(state))
	cmd.AddCommand(newDiscordUpdateCmd(state))
	return cmd
}

func newDiscordEnableCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable Discord Rich Presence updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			paths := discordRuntimePaths()
			appID, mode := discordAppID(stack)
			if err := os.MkdirAll(paths.Root, 0o700); err != nil {
				return newCLIError(3, "discord enable: %w", err)
			}
			if err := writeDiscordWrapper(paths.Wrapper, state.configPath); err != nil {
				return newCLIError(3, "discord enable: %w", err)
			}
			if err := writeDiscordStatus(paths.Status, discordStatus{Enabled: true, Mode: mode, AppID: appID, Wrapper: paths.Wrapper}); err != nil {
				return newCLIError(3, "discord enable: %w", err)
			}
			_, err = fmt.Fprintf(state.opts.Stdout, "Discord Rich Presence enabled\nwrapper: %s\n", paths.Wrapper)
			return err
		},
	}
}

func newDiscordDisableCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable Discord Rich Presence updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if _, err := loadStack(state); err != nil {
				return err
			}
			paths := discordRuntimePaths()
			if err := os.Remove(paths.Wrapper); err != nil && !errors.Is(err, os.ErrNotExist) {
				return newCLIError(3, "discord disable: %w", err)
			}
			if err := writeDiscordStatus(paths.Status, discordStatus{Enabled: false, Mode: "disabled", Wrapper: paths.Wrapper}); err != nil {
				return newCLIError(3, "discord disable: %w", err)
			}
			_, err := fmt.Fprintln(state.opts.Stdout, "Discord Rich Presence disabled")
			return err
		},
	}
}

func newDiscordStatusCmd(state *commandState) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Discord Rich Presence status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputMode(state.output); err != nil {
				return err
			}
			if _, err := loadStack(state); err != nil {
				return err
			}
			status, ok, err := readDiscordStatus(discordRuntimePaths().Status)
			if err != nil {
				return newCLIError(3, "discord status: %w", err)
			}
			if !ok {
				status = discordStatus{Enabled: false, Mode: "disabled"}
			}
			if state.output == "json" {
				return printJSON(state.opts.Stdout, status)
			}
			stateText := "disabled"
			if status.Enabled {
				stateText = "enabled"
			}
			_, err = fmt.Fprintf(state.opts.Stdout, "Discord Rich Presence: %s (%s)\n", stateText, status.Mode)
			return err
		},
	}
}

func newDiscordUpdateCmd(state *commandState) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "update",
		Short:  "Update Discord Rich Presence once",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := loadStack(state)
			if err != nil {
				return err
			}
			paths := discordRuntimePaths()
			appID, mode := discordAppID(stack)
			manager := ocadiscord.PresenceManager{
				Client:        ocadiscord.RichGoClient{},
				AppID:         appID,
				Details:       ocadiscord.DefaultDetails,
				StartedAt:     readDiscordSessionStart(paths.SessionStart),
				TaglinePath:   paths.Taglines,
				LastIndexPath: paths.LastIndex,
				RateLimitPath: paths.LastUpdate,
				RateLimit:     15 * time.Second,
			}
			presence, err := manager.Update()
			status := discordStatus{Enabled: true, Mode: mode, AppID: appID, Wrapper: paths.Wrapper, LastUpdate: presence.Updated}
			if err != nil {
				status.LastError = err.Error()
				_ = writeDiscordStatus(paths.Status, status)
				return err
			}
			return writeDiscordStatus(paths.Status, status)
		},
	}
	return cmd
}

func discordAppID(stack *cfg.Stack) (string, string) {
	if stack != nil && stack.Discord != nil && stack.Discord.Mode == "custom" && stack.Discord.AppID != "" {
		return stack.Discord.AppID, "custom"
	}
	return discordDefaultAppID, "builtin"
}

func writeDiscordWrapper(path, configPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	script := fmt.Sprintf("#!/usr/bin/env sh\nexec %q --config %q discord update >/dev/null 2>&1\n", exe, configPath)
	return os.WriteFile(path, []byte(script), 0o700)
}

func writeDiscordStatus(path string, status discordStatus) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func readDiscordStatus(path string) (discordStatus, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return discordStatus{}, false, nil
	}
	if err != nil {
		return discordStatus{}, false, err
	}
	var status discordStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return discordStatus{}, false, err
	}
	return status, true, nil
}

func readDiscordSessionStart(path string) time.Time {
	data, err := os.ReadFile(path)
	if err == nil {
		if ts, parseErr := time.Parse(time.RFC3339Nano, string(data)); parseErr == nil {
			return ts
		}
	}
	now := time.Now()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte(now.Format(time.RFC3339Nano)), 0o600)
	return now
}
