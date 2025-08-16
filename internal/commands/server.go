package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grumpyguvner/gomail/internal/api"
	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewServerCommand() *cobra.Command {
	var (
		port        int
		mode        string
		dataDir     string
		bearerToken string
	)

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the mail API server",
		Long:  `Start the mail API server that receives emails from Postfix and forwards them via webhook.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration from viper first
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Override with command-line flags if provided
			if cmd.Flags().Changed("port") {
				cfg.Port = port
			}
			if cmd.Flags().Changed("mode") {
				cfg.Mode = mode
			}
			if cmd.Flags().Changed("data-dir") {
				cfg.DataDir = dataDir
			}
			if cmd.Flags().Changed("token") {
				cfg.BearerToken = bearerToken
			}

			// Override with environment variables if set
			if token := os.Getenv("API_BEARER_TOKEN"); token != "" {
				cfg.BearerToken = token
			}

			// Validate configuration
			if cfg.BearerToken == "" || cfg.BearerToken == "change-this-token" {
				return fmt.Errorf("bearer token must be configured (set API_BEARER_TOKEN environment variable)")
			}

			// Create and start server
			server, err := api.NewServer(cfg)
			if err != nil {
				return fmt.Errorf("failed to create server: %w", err)
			}

			// Setup graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigChan
				logging.Get().Info("Shutdown signal received, stopping server...")
				cancel()
			}()

			// Start server
			logging.Get().Infof("Starting mail API server on port %d (mode: %s)", cfg.Port, cfg.Mode)
			if err := server.Start(ctx); err != nil {
				return fmt.Errorf("server error: %w", err)
			}

			// Wait for shutdown
			<-ctx.Done()

			// Graceful shutdown with 30-second timeout
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()

			logging.Get().Info("Gracefully shutting down server (timeout: 30s)...")
			if err := server.Shutdown(shutdownCtx); err != nil {
				if err == context.DeadlineExceeded {
					logging.Get().Error("Graceful shutdown timed out after 30 seconds, forcing shutdown")
				}
				return fmt.Errorf("shutdown error: %w", err)
			}

			logging.Get().Info("Server stopped gracefully")
			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", viper.GetInt("port"), "server port")
	cmd.Flags().StringVarP(&mode, "mode", "m", "simple", "operation mode (simple, socket)")
	cmd.Flags().StringVarP(&dataDir, "data-dir", "d", viper.GetString("data_dir"), "data storage directory")
	cmd.Flags().StringVarP(&bearerToken, "token", "t", viper.GetString("bearer_token"), "API bearer token")

	return cmd
}
