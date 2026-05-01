package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/Sharper-Flow/Opencode-Advance/internal/dashboard"
	"github.com/spf13/cobra"
)

func newDashboardCmd(state *commandState) *cobra.Command {
	var (
		bind   string
		port   int
		noOpen bool
	)

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch OCA operator dashboard",
		Long: `Launch a read-only web dashboard showing cross-project ADV changes,
tmux sessions, and Temporal/worker health. Auto-opens in your default browser
unless --no-open is set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf("%s:%d", bind, port)
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				return newCLIError(1, "failed to listen on %s: %w", addr, err)
			}

			logger := state.logger
			if logger == nil {
				logger = slog.Default()
			}

			cfg := dashboard.Config{
				Listener: listener,
				Logger:   logger,
			}

			srv, err := dashboard.NewServer(cfg)
			if err != nil {
				return newCLIError(1, "failed to create dashboard server: %w", err)
			}

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// Open browser before blocking
			if !noOpen {
				url := fmt.Sprintf("http://%s", listener.Addr())
				go openBrowser(url, logger)
			}

			logger.Info("dashboard starting", "addr", listener.Addr())

			// Wait for interrupt signal
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigCh
				logger.Info("shutdown signal received")
				cancel()
			}()

			if err := srv.Run(ctx); err != nil {
				return newCLIError(1, "dashboard server error: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&bind, "bind", "127.0.0.1", "Bind address for the dashboard HTTP server")
	cmd.Flags().IntVar(&port, "port", 9191, "Port for the dashboard HTTP server")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Don't auto-open the dashboard in a browser")

	return cmd
}

// openBrowser attempts to open the given URL in the user's default browser.
func openBrowser(url string, logger *slog.Logger) {
	var cmd string
	switch {
	case pathExists("/usr/bin/xdg-open"):
		cmd = "/usr/bin/xdg-open"
	case pathExists("/usr/bin/sensible-browser"):
		cmd = "/usr/bin/sensible-browser"
	case pathExists("/usr/bin/open"):
		cmd = "/usr/bin/open"
	default:
		logger.Warn("no browser launcher found, skip auto-open", "url", url)
		return
	}
	if err := exec.Command(cmd, url).Start(); err != nil {
		logger.Warn("failed to open browser", "error", err, "url", url)
	}
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
