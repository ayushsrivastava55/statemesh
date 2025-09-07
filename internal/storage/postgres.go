package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/pkg/types"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// PostgresStore handles PostgreSQL operations
type PostgresStore struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(cfg config.PostgresConfig) (*PostgresStore, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MinConns)

	return &PostgresStore{
		db:     db,
		logger: zap.L().Named("postgres"),
	}, nil
}

// Ping tests the database connection
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the database connection
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// BeginTx starts a new transaction
func (s *PostgresStore) BeginTx(ctx context.Context) (*PostgresTx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &PostgresTx{
		tx:     tx,
		logger: s.logger,
	}, nil
}

// Account operations
func (s *PostgresStore) GetAccount(ctx context.Context, chainName, address string) (*types.Account, error) {
	query := `
		SELECT chain_name, address, created_at, updated_at
		FROM accounts
		WHERE chain_name = $1 AND address = $2
	`

	var account types.Account
	err := s.db.QueryRowContext(ctx, query, chainName, address).Scan(
		&account.ChainName,
		&account.Address,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// Balance operations
func (s *PostgresStore) GetBalances(ctx context.Context, chainName, address string) ([]types.Balance, error) {
	query := `
		SELECT chain_name, address, denom, amount, height, updated_at
		FROM balances
		WHERE chain_name = $1 AND address = $2
		ORDER BY denom
	`

	rows, err := s.db.QueryContext(ctx, query, chainName, address)
	if err != nil {
		return nil, fmt.Errorf("failed to query balances: %w", err)
	}
	defer rows.Close()

	var balances []types.Balance
	for rows.Next() {
		var balance types.Balance
		err := rows.Scan(
			&balance.ChainName,
			&balance.Address,
			&balance.Denom,
			&balance.Amount,
			&balance.Height,
			&balance.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}
		balances = append(balances, balance)
	}

	return balances, rows.Err()
}

// Delegation operations
func (s *PostgresStore) GetDelegations(ctx context.Context, chainName, delegatorAddress string) ([]types.Delegation, error) {
	query := `
		SELECT chain_name, delegator_address, validator_address, shares, height, updated_at
		FROM delegations
		WHERE chain_name = $1 AND delegator_address = $2
		ORDER BY validator_address
	`

	rows, err := s.db.QueryContext(ctx, query, chainName, delegatorAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to query delegations: %w", err)
	}
	defer rows.Close()

	var delegations []types.Delegation
	for rows.Next() {
		var delegation types.Delegation
		err := rows.Scan(
			&delegation.ChainName,
			&delegation.DelegatorAddress,
			&delegation.ValidatorAddress,
			&delegation.Shares,
			&delegation.Height,
			&delegation.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delegation: %w", err)
		}
		delegations = append(delegations, delegation)
	}

	return delegations, rows.Err()
}

// Validator operations
func (s *PostgresStore) GetValidators(ctx context.Context, chainName string) ([]types.Validator, error) {
	query := `
		SELECT chain_name, operator_address, consensus_pubkey, jailed, status, tokens, 
		       delegator_shares, description_moniker, description_identity, description_website,
		       description_security_contact, description_details, unbonding_height, unbonding_time,
		       commission_rate, commission_max_rate, commission_max_change_rate, min_self_delegation,
		       height, updated_at
		FROM validators
		WHERE chain_name = $1
		ORDER BY tokens DESC
	`

	rows, err := s.db.QueryContext(ctx, query, chainName)
	if err != nil {
		return nil, fmt.Errorf("failed to query validators: %w", err)
	}
	defer rows.Close()

	var validators []types.Validator
	for rows.Next() {
		var validator types.Validator
		err := rows.Scan(
			&validator.ChainName,
			&validator.OperatorAddress,
			&validator.ConsensusPubkey,
			&validator.Jailed,
			&validator.Status,
			&validator.Tokens,
			&validator.DelegatorShares,
			&validator.Description.Moniker,
			&validator.Description.Identity,
			&validator.Description.Website,
			&validator.Description.SecurityContact,
			&validator.Description.Details,
			&validator.UnbondingHeight,
			&validator.UnbondingTime,
			&validator.Commission.Rate,
			&validator.Commission.MaxRate,
			&validator.Commission.MaxChangeRate,
			&validator.MinSelfDelegation,
			&validator.Height,
			&validator.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan validator: %w", err)
		}
		validators = append(validators, validator)
	}

	return validators, rows.Err()
}

// PostgresTx represents a PostgreSQL transaction
type PostgresTx struct {
	tx     *sql.Tx
	logger *zap.Logger
}

// Commit commits the transaction
func (tx *PostgresTx) Commit() error {
	return tx.tx.Commit()
}

// Rollback rolls back the transaction
func (tx *PostgresTx) Rollback() error {
	return tx.tx.Rollback()
}

// UpsertAccount inserts or updates an account
func (tx *PostgresTx) UpsertAccount(ctx context.Context, account *types.Account) error {
	query := `
		INSERT INTO accounts (chain_name, address, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (chain_name, address)
		DO UPDATE SET updated_at = EXCLUDED.updated_at
	`

	_, err := tx.tx.ExecContext(ctx, query,
		account.ChainName,
		account.Address,
		account.CreatedAt,
		account.UpdatedAt,
	)

	return err
}

// UpsertBalance inserts or updates a balance
func (tx *PostgresTx) UpsertBalance(ctx context.Context, balance *types.Balance) error {
	query := `
		INSERT INTO balances (chain_name, address, denom, amount, height, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chain_name, address, denom)
		DO UPDATE SET 
			amount = EXCLUDED.amount,
			height = EXCLUDED.height,
			updated_at = EXCLUDED.updated_at
	`

	_, err := tx.tx.ExecContext(ctx, query,
		balance.ChainName,
		balance.Address,
		balance.Denom,
		balance.Amount,
		balance.Height,
		balance.UpdatedAt,
	)

	return err
}

// UpsertBalances inserts or updates multiple balances in a batch
func (tx *PostgresTx) UpsertBalances(ctx context.Context, balances []types.Balance) error {
	if len(balances) == 0 {
		return nil
	}

	stmt, err := tx.tx.PrepareContext(ctx, `
		INSERT INTO balances (chain_name, address, denom, amount, height, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chain_name, address, denom)
		DO UPDATE SET 
			amount = EXCLUDED.amount,
			height = EXCLUDED.height,
			updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare balance upsert statement: %w", err)
	}
	defer stmt.Close()

	for _, balance := range balances {
		_, err := stmt.ExecContext(ctx,
			balance.ChainName,
			balance.Address,
			balance.Denom,
			balance.Amount,
			balance.Height,
			balance.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert balance: %w", err)
		}
	}

	return nil
}

// UpsertDelegation inserts or updates a delegation
func (tx *PostgresTx) UpsertDelegation(ctx context.Context, delegation *types.Delegation) error {
	query := `
		INSERT INTO delegations (chain_name, delegator_address, validator_address, shares, height, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chain_name, delegator_address, validator_address)
		DO UPDATE SET 
			shares = EXCLUDED.shares,
			height = EXCLUDED.height,
			updated_at = EXCLUDED.updated_at
	`

	_, err := tx.tx.ExecContext(ctx, query,
		delegation.ChainName,
		delegation.DelegatorAddress,
		delegation.ValidatorAddress,
		delegation.Shares,
		delegation.Height,
		delegation.UpdatedAt,
	)

	return err
}

// UpsertValidator inserts or updates a validator
func (tx *PostgresTx) UpsertValidator(ctx context.Context, validator *types.Validator) error {
	query := `
		INSERT INTO validators (
			chain_name, operator_address, consensus_pubkey, jailed, status, tokens, 
			delegator_shares, description_moniker, description_identity, description_website,
			description_security_contact, description_details, unbonding_height, unbonding_time,
			commission_rate, commission_max_rate, commission_max_change_rate, min_self_delegation,
			height, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (chain_name, operator_address)
		DO UPDATE SET 
			consensus_pubkey = EXCLUDED.consensus_pubkey,
			jailed = EXCLUDED.jailed,
			status = EXCLUDED.status,
			tokens = EXCLUDED.tokens,
			delegator_shares = EXCLUDED.delegator_shares,
			description_moniker = EXCLUDED.description_moniker,
			description_identity = EXCLUDED.description_identity,
			description_website = EXCLUDED.description_website,
			description_security_contact = EXCLUDED.description_security_contact,
			description_details = EXCLUDED.description_details,
			unbonding_height = EXCLUDED.unbonding_height,
			unbonding_time = EXCLUDED.unbonding_time,
			commission_rate = EXCLUDED.commission_rate,
			commission_max_rate = EXCLUDED.commission_max_rate,
			commission_max_change_rate = EXCLUDED.commission_max_change_rate,
			min_self_delegation = EXCLUDED.min_self_delegation,
			height = EXCLUDED.height,
			updated_at = EXCLUDED.updated_at
	`

	_, err := tx.tx.ExecContext(ctx, query,
		validator.ChainName,
		validator.OperatorAddress,
		validator.ConsensusPubkey,
		validator.Jailed,
		validator.Status,
		validator.Tokens,
		validator.DelegatorShares,
		validator.Description.Moniker,
		validator.Description.Identity,
		validator.Description.Website,
		validator.Description.SecurityContact,
		validator.Description.Details,
		validator.UnbondingHeight,
		validator.UnbondingTime,
		validator.Commission.Rate,
		validator.Commission.MaxRate,
		validator.Commission.MaxChangeRate,
		validator.MinSelfDelegation,
		validator.Height,
		validator.UpdatedAt,
	)

	return err
}
