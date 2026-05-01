package main

import (
	"fmt"
	"os"

	"github.com/Sharper-Flow/Opencode-Advance/internal/migrate"
	"github.com/spf13/cobra"
)

func newMigrateCmd(state *commandState) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from open-chad or create a starter stack",
		Long:  "Migration commands for converting existing open-chad state to stack.toml or creating a new starter stack.",
	}

	// from-open-chad subcommand
	fromOpenChadCmd := &cobra.Command{
		Use:   "from-open-chad",
		Short: "Read open-chad state and emit stack.toml",
		Long: `Reads the current open-chad managed state from the filesystem
(opencode.json, vision/servers.yaml, plugin checkouts, etc.) and emits
a stack.toml representation. The output is written to stdout or --output.

Always review the generated stack.toml before running 'oca apply'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			state.logger.Debug("reading open-chad state")

			cfg := migrate.DefaultReaderConfig()
			ocState, err := migrate.ReadOpenChadState(cfg)
			if err != nil {
				return fmt.Errorf("read open-chad state: %w", err)
			}

			if cmd.Flags().Changed("dry-run") {
				// In dry-run mode, just print summary
				fmt.Fprintf(cmd.OutOrStdout(), "# Dry run: would read from:\n")
				fmt.Fprintf(cmd.OutOrStdout(), "#   open-chad repo: %s\n", cfg.OpenChadRepo)
				fmt.Fprintf(cmd.OutOrStdout(), "#   opencode config: %s\n", cfg.OpenCodeConfigDir)
				fmt.Fprintf(cmd.OutOrStdout(), "#   vision config: %s\n", cfg.VisionConfigDir)
				fmt.Fprintf(cmd.OutOrStdout(), "#   plugins dir: %s\n", cfg.OcPluginsDir)
				fmt.Fprintf(cmd.OutOrStdout(), "#\n")
				fmt.Fprintf(cmd.OutOrStdout(), "# Discovered:\n")
				fmt.Fprintf(cmd.OutOrStdout(), "#   MCP servers: %d\n", len(ocState.MCPServers))
				fmt.Fprintf(cmd.OutOrStdout(), "#   Plugins: %d\n", len(ocState.Plugins))
				fmt.Fprintf(cmd.OutOrStdout(), "#   Instructions: %d\n", len(ocState.Instructions))
				fmt.Fprintf(cmd.OutOrStdout(), "#   Providers: %d\n", len(ocState.Providers))
				fmt.Fprintf(cmd.OutOrStdout(), "#   Warnings: %d\n", len(ocState.Warnings))
				return nil
			}

			output, err := migrate.EmitTOML(ocState)
			if err != nil {
				return fmt.Errorf("emit TOML: %w", err)
			}

			if outputPath != "" {
				if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
					return fmt.Errorf("write output file: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "stack.toml written to %s\n", outputPath)
			} else {
				fmt.Fprint(cmd.OutOrStdout(), output)
			}

			return nil
		},
	}
	fromOpenChadCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Write output to file instead of stdout")
	fromOpenChadCmd.Flags().Bool("dry-run", false, "Print what would be migrated without writing")

	// init subcommand
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a minimal starter stack.toml",
		Long: `Generates a minimal working stack.toml with sensible defaults:
vision daemon, context7 MCP, advance plugin, and a default provider.

Edit the generated file to match your environment, then run 'oca apply'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := migrate.EmitInit()
			if err != nil {
				return fmt.Errorf("emit init TOML: %w", err)
			}

			if outputPath != "" {
				if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
					return fmt.Errorf("write output file: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "stack.toml written to %s\n", outputPath)
			} else {
				fmt.Fprint(cmd.OutOrStdout(), output)
			}

			return nil
		},
	}
	initCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Write output to file instead of stdout")

	cmd.AddCommand(fromOpenChadCmd)
	cmd.AddCommand(initCmd)

	return cmd
}
