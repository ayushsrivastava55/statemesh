package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/pkg/types"
	"go.uber.org/zap"
)

// Manager manages database connections and operations
type Manager struct {
	postgres   *PostgresStore
	clickhouse *ClickHouseStore
	logger     *zap.Logger
}

// NewManager creates a new storage manager
func NewManager(cfg config.DatabaseConfig) (*Manager, error) {
	logger := zap.L().Named("storage")

	// Initialize PostgreSQL
	pgStore, err := NewPostgresStore(cfg.Postgres.DSN(), logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Initialize ClickHouse (optional)
	var clickhouse *ClickHouseStore
	if cfg.ClickHouse.Enabled {
		clickhouse, err = NewClickHouseStore(cfg.ClickHouse)
		if err != nil {
			logger.Warn("Failed to initialize ClickHouse, continuing without analytics", zap.Error(err))
		}
	}

	return &Manager{
		postgres:   pgStore,
		clickhouse: clickhouse,
		logger:     logger,
	}, nil
}

// Postgres returns the PostgreSQL store
func (m *Manager) Postgres() *PostgresStore {
	return m.postgres
}

// ClickHouse returns the ClickHouse store (may be nil)
func (m *Manager) ClickHouse() *ClickHouseStore {
	return m.clickhouse
}

// Ping tests connectivity to all databases
func (m *Manager) Ping(ctx context.Context) error {
	// Test PostgreSQL
	if err := m.postgres.Ping(ctx); err != nil {
		return fmt.Errorf("PostgreSQL ping failed: %w", err)
	}

	// Test ClickHouse if enabled
	if m.clickhouse != nil {
		if err := m.clickhouse.Ping(ctx); err != nil {
			m.logger.Warn("ClickHouse ping failed", zap.Error(err))
		}
	}

	return nil
}

// Close closes all database connections
func (m *Manager) Close() error {
	var errs []error

	if err := m.postgres.Close(); err != nil {
		errs = append(errs, fmt.Errorf("PostgreSQL close error: %w", err))
	}

	if m.clickhouse != nil {
		if err := m.clickhouse.Close(); err != nil {
			errs = append(errs, fmt.Errorf("ClickHouse close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("storage close errors: %v", errs)
	}

	return nil
}

// BeginTx starts a new transaction
func (m *Manager) BeginTx(ctx context.Context) (*Tx, error) {
	pgTx, err := m.postgres.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin PostgreSQL transaction: %w", err)
	}

	return &Tx{
		postgres: pgTx,
		logger:   m.logger,
	}, nil
}


// GetBalances returns balances for an address on a chain (Bank module)
func (m *Manager) GetBalances(ctx context.Context, address, chain string) ([]*types.Balance, error) {
	balances, err := m.postgres.GetBalances(ctx, chain, address)
	if err != nil {
		return nil, err
	}
	
	// Convert slice to pointer slice
	result := make([]*types.Balance, len(balances))
	for i := range balances {
		result[i] = &balances[i]
	}
	return result, nil
}

// GetDelegations returns delegations for an address on a chain (Staking module)
func (m *Manager) GetDelegations(ctx context.Context, address, chain string) ([]*types.Delegation, error) {
	delegations, err := m.postgres.GetDelegations(ctx, chain, address)
	if err != nil {
		return nil, err
	}
	
	// Convert slice to pointer slice
	result := make([]*types.Delegation, len(delegations))
	for i := range delegations {
		result[i] = &delegations[i]
	}
	return result, nil
}

// GetChains returns all configured chains
func (m *Manager) GetChains(ctx context.Context) ([]*types.ChainInfo, error) {
	// For now, return demo data based on PRD requirements
	return []*types.ChainInfo{
		{
			Name:         "cosmoshub",
			ChainID:      "cosmoshub-4", 
			Status:       "active",
			LatestHeight: 12345678,
			LatestTime:   time.Now(),
			UpdatedAt:    time.Now(),
		},
	}, nil
}

// GetChain returns a specific chain by name
func (m *Manager) GetChain(ctx context.Context, name string) (*types.ChainInfo, error) {
	chains, err := m.GetChains(ctx)
	if err != nil {
		return nil, err
	}
	
	for _, chain := range chains {
		if chain.Name == name {
			return chain, nil
		}
	}
	
	return nil, fmt.Errorf("chain not found: %s", name)
}

// Tx represents a database transaction
type Tx struct {
	postgres *PostgresTx
	logger   *zap.Logger
}

// Commit commits the transaction
func (tx *Tx) Commit() error {
	return tx.postgres.Commit()
}

// Rollback rolls back the transaction
func (tx *Tx) Rollback() error {
	return tx.postgres.Rollback()
}

// Postgres returns the PostgreSQL transaction
func (tx *Tx) Postgres() *PostgresTx {
	return tx.postgres
}

// ClickHouse returns the ClickHouse store
func (tx *Tx) ClickHouse() *ClickHouseStore {
	return nil // Transactions don't support ClickHouse operations
}
