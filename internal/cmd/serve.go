package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cosmos/state-mesh/internal/api"
	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the State Mesh API server",
	Long: `Start the State Mesh API server that provides GraphQL and REST endpoints
for querying unified Cosmos state data.

The server exposes:
- GraphQL API on the configured port (default: 8080)
- REST API on the configured port (default: 8081)
- Metrics endpoint on /metrics
- Health check endpoint on /health`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Server-specific flags
	serveCmd.Flags().Int("graphql-port", 8080, "GraphQL API server port")
	serveCmd.Flags().Int("rest-port", 8081, "REST API server port")
	serveCmd.Flags().Int("metrics-port", 9090, "Metrics server port")
	serveCmd.Flags().Bool("enable-playground", true, "Enable GraphQL playground")
	serveCmd.Flags().Bool("enable-cors", true, "Enable CORS headers")

	// Bind flags to viper
	viper.BindPFlag("api.graphql.port", serveCmd.Flags().Lookup("graphql-port"))
	viper.BindPFlag("api.rest.port", serveCmd.Flags().Lookup("rest-port"))
	viper.BindPFlag("api.metrics.port", serveCmd.Flags().Lookup("metrics-port"))
	viper.BindPFlag("api.graphql.playground", serveCmd.Flags().Lookup("enable-playground"))
	viper.BindPFlag("api.cors.enabled", serveCmd.Flags().Lookup("enable-cors"))
}

func runServe(cmd *cobra.Command, args []string) error {
	logger := GetLogger()
	logger.Info("Starting State Mesh API server")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize storage
	storageManager, err := storage.NewManager(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer storageManager.Close()

	// Test database connections
	if err := storageManager.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to connect to databases: %w", err)
	}

	logger.Info("Database connections established")

	// Initialize API server
	apiServer, err := api.NewServer(cfg.API, storageManager, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize API server: %w", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start servers
	errChan := make(chan error, 3)

	// Start GraphQL server
	go func() {
		logger.Info("Starting GraphQL server", zap.Int("port", cfg.API.GraphQL.Port))
		if err := apiServer.StartGraphQL(ctx); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("GraphQL server error: %w", err)
		}
	}()

	// Start REST server
	go func() {
		logger.Info("Starting REST server", zap.Int("port", cfg.API.REST.Port))
		if err := apiServer.StartREST(ctx); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("REST server error: %w", err)
		}
	}()

	// Start metrics server
	go func() {
		logger.Info("Starting metrics server", zap.Int("port", cfg.API.Metrics.Port))
		if err := apiServer.StartMetrics(ctx); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("State Mesh API server started successfully")
	logger.Info("GraphQL endpoint", zap.String("url", fmt.Sprintf("http://localhost:%d/graphql", cfg.API.GraphQL.Port)))
	logger.Info("REST endpoint", zap.String("url", fmt.Sprintf("http://localhost:%d/api/v1", cfg.API.REST.Port)))
	logger.Info("Metrics endpoint", zap.String("url", fmt.Sprintf("http://localhost:%d/metrics", cfg.API.Metrics.Port)))

	if cfg.API.GraphQL.Playground {
		logger.Info("GraphQL Playground", zap.String("url", fmt.Sprintf("http://localhost:%d/playground", cfg.API.GraphQL.Port)))
	}

	select {
	case err := <-errChan:
		logger.Error("Server error", zap.Error(err))
		return err
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	logger.Info("Shutting down servers...")
	cancel()

	// Give servers time to shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
		return err
	}

	logger.Info("State Mesh API server stopped")
	return nil
}
