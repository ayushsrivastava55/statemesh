package storage

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/pkg/types"
	"go.uber.org/zap"
)

// ClickHouseStore handles ClickHouse operations for analytics
type ClickHouseStore struct {
	conn   driver.Conn
	logger *zap.Logger
}

// NewClickHouseStore creates a new ClickHouse store
func NewClickHouseStore(cfg config.ClickHouseConfig) (*ClickHouseStore, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 30,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	return &ClickHouseStore{
		conn:   conn,
		logger: zap.L().Named("clickhouse"),
	}, nil
}

// Ping tests the ClickHouse connection
func (s *ClickHouseStore) Ping(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

// Close closes the ClickHouse connection
func (s *ClickHouseStore) Close() error {
	return s.conn.Close()
}

// InsertBalanceEvents inserts balance change events for analytics
func (s *ClickHouseStore) InsertBalanceEvents(ctx context.Context, events []types.BalanceEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := s.conn.PrepareBatch(ctx, `
		INSERT INTO balance_events (
			timestamp, chain_name, address, denom, amount, 
			previous_amount, change_type, height, tx_hash
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare balance events batch: %w", err)
	}

	for _, event := range events {
		err := batch.Append(
			event.Timestamp,
			event.ChainName,
			event.Address,
			event.Denom,
			event.Amount,
			event.PreviousAmount,
			event.ChangeType,
			event.Height,
			event.TxHash,
		)
		if err != nil {
			return fmt.Errorf("failed to append balance event: %w", err)
		}
	}

	return batch.Send()
}

// InsertDelegationEvents inserts delegation change events for analytics
func (s *ClickHouseStore) InsertDelegationEvents(ctx context.Context, events []types.DelegationEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := s.conn.PrepareBatch(ctx, `
		INSERT INTO delegation_events (
			timestamp, chain_name, delegator_address, validator_address, 
			shares, previous_shares, change_type, height, tx_hash
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare delegation events batch: %w", err)
	}

	for _, event := range events {
		err := batch.Append(
			event.Timestamp,
			event.ChainName,
			event.DelegatorAddress,
			event.ValidatorAddress,
			event.Shares,
			event.PreviousShares,
			event.ChangeType,
			event.Height,
			event.TxHash,
		)
		if err != nil {
			return fmt.Errorf("failed to append delegation event: %w", err)
		}
	}

	return batch.Send()
}

// GetBalanceHistory returns balance history for analytics
func (s *ClickHouseStore) GetBalanceHistory(ctx context.Context, chainName, address, denom string, limit int) ([]types.BalanceEvent, error) {
	query := `
		SELECT timestamp, chain_name, address, denom, amount, 
		       previous_amount, change_type, height, tx_hash
		FROM balance_events
		WHERE chain_name = ? AND address = ? AND denom = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := s.conn.Query(ctx, query, chainName, address, denom, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query balance history: %w", err)
	}
	defer rows.Close()

	var events []types.BalanceEvent
	for rows.Next() {
		var event types.BalanceEvent
		err := rows.Scan(
			&event.Timestamp,
			&event.ChainName,
			&event.Address,
			&event.Denom,
			&event.Amount,
			&event.PreviousAmount,
			&event.ChangeType,
			&event.Height,
			&event.TxHash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetDelegationHistory returns delegation history for analytics
func (s *ClickHouseStore) GetDelegationHistory(ctx context.Context, chainName, delegatorAddress string, limit int) ([]types.DelegationEvent, error) {
	query := `
		SELECT timestamp, chain_name, delegator_address, validator_address, 
		       shares, previous_shares, change_type, height, tx_hash
		FROM delegation_events
		WHERE chain_name = ? AND delegator_address = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := s.conn.Query(ctx, query, chainName, delegatorAddress, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query delegation history: %w", err)
	}
	defer rows.Close()

	var events []types.DelegationEvent
	for rows.Next() {
		var event types.DelegationEvent
		err := rows.Scan(
			&event.Timestamp,
			&event.ChainName,
			&event.DelegatorAddress,
			&event.ValidatorAddress,
			&event.Shares,
			&event.PreviousShares,
			&event.ChangeType,
			&event.Height,
			&event.TxHash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delegation event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetChainStats returns aggregated chain statistics
func (s *ClickHouseStore) GetChainStats(ctx context.Context, chainName string) (*types.ChainStats, error) {
	query := `
		SELECT 
			chain_name,
			count(DISTINCT address) as total_accounts,
			sum(amount) as total_supply,
			count(DISTINCT validator_address) as total_validators,
			max(height) as latest_height
		FROM (
			SELECT chain_name, address, amount, 0 as validator_address, height
			FROM balance_events
			WHERE chain_name = ? AND change_type = 'current'
			UNION ALL
			SELECT chain_name, delegator_address as address, 0 as amount, validator_address, height
			FROM delegation_events
			WHERE chain_name = ? AND change_type = 'current'
		)
		GROUP BY chain_name
	`

	var stats types.ChainStats
	err := s.conn.QueryRow(ctx, query, chainName, chainName).Scan(
		&stats.ChainName,
		&stats.TotalValidators,
		&stats.ActiveValidators,
		&stats.TotalDelegated,
		&stats.TotalSupply,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get chain stats: %w", err)
	}

	return &stats, nil
}

// GetTopHolders returns top token holders for a specific denom
func (s *ClickHouseStore) GetTopHolders(ctx context.Context, chainName, denom string, limit int) ([]types.TokenHolder, error) {
	query := `
		SELECT address, amount
		FROM (
			SELECT address, argMax(amount, timestamp) as amount
			FROM balance_events
			WHERE chain_name = ? AND denom = ?
			GROUP BY address
		)
		WHERE amount > 0
		ORDER BY amount DESC
		LIMIT ?
	`

	rows, err := s.conn.Query(ctx, query, chainName, denom, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top holders: %w", err)
	}
	defer rows.Close()

	var holders []types.TokenHolder
	for rows.Next() {
		var holder types.TokenHolder
		err := rows.Scan(&holder.Address, &holder.Amount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token holder: %w", err)
		}
		holder.ChainName = chainName
		holder.Denom = denom
		holders = append(holders, holder)
	}

	return holders, rows.Err()
}
