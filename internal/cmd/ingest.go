package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/internal/ingester"
	"github.com/cosmos/state-mesh/internal/storage"
	"github.com/cosmos/state-mesh/internal/streaming"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ingestCmd represents the ingest command
var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Start the State Mesh ingester",
	Long: `Start the State Mesh ingester that connects to Cosmos SDK chains
and streams state changes in real-time using ADR-038 State Listening.

The ingester:
- Connects to configured chain gRPC endpoints
- Subscribes to module state changes (bank, staking, distribution, etc.)
- Normalizes and stores state data in PostgreSQL
- Streams analytics data to ClickHouse
- Publishes state changes to Kafka for real-time processing`,
	RunE: runIngest,
}

func init() {
	rootCmd.AddCommand(ingestCmd)

	// Ingester-specific flags
	ingestCmd.Flags().StringSlice("chains", []string{}, "Specific chains to ingest (default: all configured chains)")
	ingestCmd.Flags().StringSlice("modules", []string{}, "Specific modules to ingest (default: all configured modules)")
	ingestCmd.Flags().Bool("enable-streaming", true, "Enable Kafka streaming")
	ingestCmd.Flags().Bool("enable-analytics", true, "Enable ClickHouse analytics storage")
	ingestCmd.Flags().Int("batch-size", 1000, "Batch size for database operations")
	ingestCmd.Flags().Duration("flush-interval", 0, "Flush interval for batched operations (0 = auto)")

	// Bind flags to viper
	viper.BindPFlag("ingester.chains", ingestCmd.Flags().Lookup("chains"))
	viper.BindPFlag("ingester.modules", ingestCmd.Flags().Lookup("modules"))
	viper.BindPFlag("ingester.streaming.enabled", ingestCmd.Flags().Lookup("enable-streaming"))
	viper.BindPFlag("ingester.analytics.enabled", ingestCmd.Flags().Lookup("enable-analytics"))
	viper.BindPFlag("ingester.batch_size", ingestCmd.Flags().Lookup("batch-size"))
	viper.BindPFlag("ingester.flush_interval", ingestCmd.Flags().Lookup("flush-interval"))
}

func runIngest(cmd *cobra.Command, args []string) error {
	logger := GetLogger()
	logger.Info("Starting State Mesh ingester")

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

	// Initialize streaming (optional)
	var streamingManager *streaming.Manager
	if cfg.Streaming.Enabled {
		streamingManager, err = streaming.NewManager(cfg.Streaming, logger)
		if err != nil {
			logger.Warn("Failed to initialize streaming, continuing without it", zap.Error(err))
		} else {
			defer streamingManager.Close()
			logger.Info("Streaming manager initialized")
		}
	}

	// Filter chains and modules based on flags
	chains := viper.GetStringSlice("ingester.chains")
	modules := viper.GetStringSlice("ingester.modules")

	// Initialize ingester
	ing, err := ingester.New(cfg.Ingester, cfg.Chains, storageManager, streamingManager, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize ingester: %w", err)
	}

	// Apply filters
	if len(chains) > 0 {
		ing.FilterChains(chains)
	}
	if len(modules) > 0 {
		ing.FilterModules(modules)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start ingester
	errChan := make(chan error, 1)
	go func() {
		if err := ing.Start(ctx); err != nil {
			errChan <- fmt.Errorf("ingester error: %w", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("State Mesh ingester started successfully")
	
	// Log configured chains and modules
	for _, chain := range cfg.Chains {
		logger.Info("Monitoring chain", 
			zap.String("name", chain.Name),
			zap.String("endpoint", chain.GRPCEndpoint),
			zap.Strings("modules", chain.Modules))
	}

	select {
	case err := <-errChan:
		logger.Error("Ingester error", zap.Error(err))
		return err
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	logger.Info("Shutting down ingester...")
	cancel()

	// Wait for ingester to stop
	if err := ing.Stop(context.Background()); err != nil {
		logger.Error("Error during ingester shutdown", zap.Error(err))
		return err
	}

	logger.Info("State Mesh ingester stopped")
	return nil
}
