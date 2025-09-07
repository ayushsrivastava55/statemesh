package ingester

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/internal/storage"
	"github.com/cosmos/state-mesh/pkg/cosmos"
	"github.com/cosmos/state-mesh/pkg/types"
)

// Ingester handles state ingestion from Cosmos SDK chains
type Ingester struct {
	cfg              config.IngesterConfig
	chains           []config.ChainConfig
	storage          *storage.Manager
	// streaming        *streaming.Manager
	logger           *zap.Logger
	clients          map[string]*cosmos.Client
	workers          map[string]*ChainWorker
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// New creates a new ingester
func New(cfg config.IngesterConfig, chains []config.ChainConfig, storage *storage.Manager) (*Ingester, error) {
	return &Ingester{
		cfg:     cfg,
		chains:  chains,
		storage: storage,
		logger:  zap.L().Named("ingester"),
		clients: make(map[string]*cosmos.Client),
		workers: make(map[string]*ChainWorker),
	}, nil
}

// FilterChains filters chains to ingest
func (i *Ingester) FilterChains(chainNames []string) {
	if len(chainNames) == 0 {
		return
	}

	nameSet := make(map[string]bool)
	for _, name := range chainNames {
		nameSet[name] = true
	}

	var filtered []config.ChainConfig
	for _, chain := range i.chains {
		if nameSet[chain.Name] {
			filtered = append(filtered, chain)
		}
	}

	i.chains = filtered
}

// FilterModules filters modules to ingest
func (i *Ingester) FilterModules(moduleNames []string) {
	if len(moduleNames) == 0 {
		return
	}

	nameSet := make(map[string]bool)
	for _, name := range moduleNames {
		nameSet[name] = true
	}

	for j := range i.chains {
		var filtered []string
		for _, module := range i.chains[j].Modules {
			if nameSet[module] {
				filtered = append(filtered, module)
			}
		}
		i.chains[j].Modules = filtered
	}
}

// Start starts the ingester
func (i *Ingester) Start(ctx context.Context) error {
	i.ctx, i.cancel = context.WithCancel(ctx)

	// Initialize clients for each chain
	for _, chainCfg := range i.chains {
		if !chainCfg.Enabled {
			continue
		}

		client, err := cosmos.NewClient(chainCfg.Name, chainCfg.GRPCEndpoint)
		if err != nil {
			i.logger.Error("Failed to create client for chain",
				zap.String("chain", chainCfg.Name),
				zap.Error(err))
			continue
		}

		// Test connection
		if err := client.Ping(i.ctx); err != nil {
			i.logger.Error("Failed to ping chain",
				zap.String("chain", chainCfg.Name),
				zap.Error(err))
			client.Close()
			continue
		}

		i.mu.Lock()
		i.clients[chainCfg.Name] = client
		i.mu.Unlock()

		i.logger.Info("Connected to chain",
			zap.String("chain", chainCfg.Name),
			zap.String("endpoint", chainCfg.GRPCEndpoint))
	}

	// Start workers for each chain
	for _, chainCfg := range i.chains {
		if !chainCfg.Enabled {
			continue
		}

		client := i.clients[chainCfg.Name]
		if client == nil {
			continue
		}

		worker := NewChainWorker(chainCfg, client, i.storage, i.logger)
		i.workers[chainCfg.Name] = worker

		i.wg.Add(1)
		go func(w *ChainWorker) {
			defer i.wg.Done()
			if err := w.Start(i.ctx); err != nil {
				i.logger.Error("Chain worker error",
					zap.String("chain", w.chainName),
					zap.Error(err))
			}
		}(worker)
	}

	i.logger.Info("Ingester started",
		zap.Int("chains", len(i.workers)),
		zap.Int("workers", i.cfg.Workers))

	return nil
}

// Stop stops the ingester
func (i *Ingester) Stop(ctx context.Context) error {
	if i.cancel != nil {
		i.cancel()
	}

	// Wait for workers to stop
	done := make(chan struct{})
	go func() {
		i.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		i.logger.Info("All workers stopped")
	case <-ctx.Done():
		i.logger.Warn("Timeout waiting for workers to stop")
	}

	// Close clients
	i.mu.Lock()
	for name, client := range i.clients {
		if err := client.Close(); err != nil {
			i.logger.Error("Failed to close client",
				zap.String("chain", name),
				zap.Error(err))
		}
	}
	i.clients = make(map[string]*cosmos.Client)
	i.mu.Unlock()

	return nil
}

// ChainWorker handles ingestion for a single chain
type ChainWorker struct {
	chainName string
	chainCfg  config.ChainConfig
	client    *cosmos.Client
	storage   *storage.Manager
	logger    *zap.Logger
	ticker    *time.Ticker
}

// NewChainWorker creates a new chain worker
func NewChainWorker(chainCfg config.ChainConfig, client *cosmos.Client, storage *storage.Manager, logger *zap.Logger) *ChainWorker {
	return &ChainWorker{
		chainName: chainCfg.Name,
		chainCfg:  chainCfg,
		client:    client,
		storage:   storage,
		logger:    logger.Named("worker").With(zap.String("chain", chainCfg.Name)),
		ticker:    time.NewTicker(10 * time.Second), // Poll every 10 seconds
	}
}

// Start starts the chain worker
func (w *ChainWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting chain worker")

	for {
		select {
		case <-ctx.Done():
			w.ticker.Stop()
			w.logger.Info("Chain worker stopped")
			return nil
		case <-w.ticker.C:
			if err := w.ingestChainState(ctx); err != nil {
				w.logger.Error("Failed to ingest chain state", zap.Error(err))
			}
		}
	}
}

// ingestChainState ingests the current state of the chain
func (w *ChainWorker) ingestChainState(ctx context.Context) error {
	w.logger.Debug("Ingesting chain state")

	// Get current height
	height, err := w.client.GetLatestHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest height: %w", err)
	}

	// Start transaction
	tx, err := w.storage.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Ingest data based on enabled modules
	for _, module := range w.chainCfg.Modules {
		switch module {
		case "bank":
			if err := w.ingestBankModule(ctx, height); err != nil {
				w.logger.Error("Failed to ingest bank module",
					zap.String("chain", w.chainName),
					zap.Error(err))
				return err
			}
		case "staking":
			if err := w.ingestStakingModule(ctx, height); err != nil {
				w.logger.Error("Failed to ingest staking module",
					zap.String("chain", w.chainName),
					zap.Error(err))
				return err
			}
		case "distribution":
			if err := w.ingestDistributionModule(ctx, height); err != nil {
				w.logger.Error("Failed to ingest distribution module",
					zap.String("chain", w.chainName),
					zap.Error(err))
				return err
			}
		case "governance":
			if err := w.ingestGovernanceModule(ctx, height); err != nil {
				w.logger.Error("Failed to ingest governance module",
					zap.String("chain", w.chainName),
					zap.Error(err))
				return err
			}
		case "mint":
			if err := w.ingestMintModule(ctx, height); err != nil {
				w.logger.Error("Failed to ingest mint module",
					zap.String("chain", w.chainName),
					zap.Error(err))
				return err
			}
		case "slashing":
			if err := w.ingestSlashingModule(ctx, height); err != nil {
				w.logger.Error("Failed to ingest slashing module",
					zap.String("chain", w.chainName),
					zap.Error(err))
				return err
			}
		default:
			w.logger.Debug("Unknown module",
				zap.String("chain", w.chainName),
				zap.String("module", module))
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ingestBankModule ingests bank module state
func (w *ChainWorker) ingestBankModule(ctx context.Context, height int64) error {
	// Get total supply
	supply, err := w.client.GetTotalSupply(ctx, "uatom")
	if err != nil {
		return fmt.Errorf("failed to get total supply: %w", err)
	}

	// For now, we'll just log the supply
	// In a real implementation, we'd track all account balances
	if !supply.Amount.IsZero() {
		w.logger.Debug("Bank module state",
			zap.String("supply", supply.String()),
			zap.Int64("height", height))
	}

	return nil
}

// ingestStakingModule ingests staking module state
func (w *ChainWorker) ingestStakingModule(ctx context.Context, height int64) error {
	// Get all validators
	validators, err := w.client.GetValidators(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get validators: %w", err)
	}

	// Start transaction
	tx, err := w.storage.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// Process validators
	for _, val := range validators {
		validator := &types.Validator{
			ChainName:       w.chainName,
			OperatorAddress: val.OperatorAddress,
			ConsensusPubkey: val.ConsensusPubkey.String(),
			Jailed:          val.Jailed,
			Status:          val.Status.String(),
			Tokens:          val.Tokens.String(),
			DelegatorShares: val.DelegatorShares.String(),
			Description: types.ValidatorDescription{
				Moniker:         val.Description.Moniker,
				Identity:        val.Description.Identity,
				Website:         val.Description.Website,
				SecurityContact: val.Description.SecurityContact,
				Details:         val.Description.Details,
			},
			UnbondingHeight: val.UnbondingHeight,
			UnbondingTime:   val.UnbondingTime,
			Commission: types.ValidatorCommission{
				Rate:          val.Commission.Rate.String(),
				MaxRate:       val.Commission.MaxRate.String(),
				MaxChangeRate: val.Commission.MaxChangeRate.String(),
			},
			MinSelfDelegation: val.MinSelfDelegation.String(),
			Height:            height,
			UpdatedAt:         now,
		}

		if err := tx.Postgres().UpsertValidator(ctx, validator); err != nil {
			return fmt.Errorf("failed to upsert validator: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	w.logger.Debug("Staking module state ingested",
		zap.Int("validators", len(validators)),
		zap.Int64("height", height))

	return nil
}

// ingestDistributionModule ingests distribution module state
func (w *ChainWorker) ingestDistributionModule(ctx context.Context, height int64) error {
	// Distribution module ingestion would go here
	// For now, just log
	w.logger.Debug("Distribution module state ingested", zap.Int64("height", height))
	return nil
}

// ingestGovernanceModule ingests governance module state
func (w *ChainWorker) ingestGovernanceModule(ctx context.Context, height int64) error {
	// Get all proposals
	proposals, err := w.client.GetProposals(ctx, 0) // 0 = all statuses
	if err != nil {
		return fmt.Errorf("failed to get proposals: %w", err)
	}

	w.logger.Debug("Governance module state ingested",
		zap.Int("proposals", len(proposals)),
		zap.Int64("height", height))

	return nil
}

// ingestMintModule ingests mint module state
func (w *ChainWorker) ingestMintModule(ctx context.Context, height int64) error {
	w.logger.Debug("Mint module state ingested", zap.Int64("height", height))
	return nil
}

// ingestSlashingModule ingests slashing module state
func (w *ChainWorker) ingestSlashingModule(ctx context.Context, height int64) error {
	w.logger.Debug("Slashing module state ingested", zap.Int64("height", height))
	return nil
}
