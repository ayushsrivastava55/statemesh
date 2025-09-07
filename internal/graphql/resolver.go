package graphql

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/cosmos/state-mesh/internal/storage"
	"github.com/cosmos/state-mesh/pkg/types"
)

// Resolver is the root GraphQL resolver
type Resolver struct {
	storage *storage.Manager
	logger  *zap.Logger
}

// NewResolver creates a new GraphQL resolver
func NewResolver(storage *storage.Manager, logger *zap.Logger) *Resolver {
	return &Resolver{
		storage: storage,
		logger:  logger.Named("graphql"),
	}
}

// Query resolver
type queryResolver struct{ *Resolver }

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// QueryResolver interface
type QueryResolver interface {
	Account(ctx context.Context, address string, chain string) (*types.AccountState, error)
	CrossChainAccount(ctx context.Context, address string) (*CrossChainAccountState, error)
	Chains(ctx context.Context) ([]*types.ChainInfo, error)
	Chain(ctx context.Context, name string) (*types.ChainInfo, error)
	ChainStats(ctx context.Context, name string) (*types.ChainStats, error)
	Validators(ctx context.Context, chain string) ([]*types.Validator, error)
	Validator(ctx context.Context, chain string, address string) (*types.Validator, error)
	CrossChainValidators(ctx context.Context, chains []string) (*CrossChainValidators, error)
	Proposals(ctx context.Context, chain string) ([]*types.Proposal, error)
	Proposal(ctx context.Context, chain string, id string) (*types.Proposal, error)
	ProposalVotes(ctx context.Context, chain string, proposalID string) ([]*types.Vote, error)
	BalanceHistory(ctx context.Context, address string, chain string, denom *string, limit *int) ([]*types.BalanceEvent, error)
	DelegationHistory(ctx context.Context, address string, chain string, limit *int) ([]*types.DelegationEvent, error)
}

// GraphQL-specific types
type CrossChainAccountState struct {
	Address string                 `json:"address"`
	Chains  []*ChainAccountState   `json:"chains"`
	Totals  *CrossChainTotals      `json:"totals"`
}

type ChainAccountState struct {
	ChainName    string              `json:"chainName"`
	AccountState *types.AccountState `json:"accountState"`
}

type CrossChainTotals struct {
	TotalBalance   []*DenomAmount `json:"totalBalance"`
	TotalDelegated []*DenomAmount `json:"totalDelegated"`
	TotalUnbonding []*DenomAmount `json:"totalUnbonding"`
	TotalRewards   float64         `json:"totalRewards"`
}

type DenomAmount struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type CrossChainValidators struct {
	Validators []*ChainValidators `json:"validators"`
}

type ChainValidators struct {
	ChainName  string             `json:"chainName"`
	Validators []*types.Validator `json:"validators"`
}

// Account resolver
func (r *queryResolver) Account(ctx context.Context, address string, chain string) (*types.AccountState, error) {
	// Get balances
	balances, err := r.storage.Postgres().GetBalances(ctx, chain, address)
	if err != nil {
		r.logger.Error("Failed to get balances", 
			zap.String("address", address),
			zap.String("chain", chain),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get balances: %w", err)
	}

	// Get delegations
	delegations, err := r.storage.Postgres().GetDelegations(ctx, chain, address)
	if err != nil {
		r.logger.Error("Failed to get delegations",
			zap.String("address", address),
			zap.String("chain", chain),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get delegations: %w", err)
	}

	return &types.AccountState{
		ChainName:   chain,
		Address:     address,
		Balances:    balances,
		Delegations: delegations,
	}, nil
}

// CrossChainAccount resolver
func (r *queryResolver) CrossChainAccount(ctx context.Context, address string) (*CrossChainAccountState, error) {
	// For now, return a basic implementation
	// In a real implementation, this would query all configured chains
	chains := []string{"cosmoshub", "osmosis"} // TODO: Get from config
	
	chainStates := make([]*ChainAccountState, 0, len(chains))
	totalBalanceMap := make(map[string]string)
	totalDelegatedMap := make(map[string]string)

	for _, chainName := range chains {
		accountState, err := r.Account(ctx, address, chainName)
		if err != nil {
			r.logger.Warn("Failed to get account state for chain",
				zap.String("address", address),
				zap.String("chain", chainName),
				zap.Error(err))
			continue
		}

		chainStates = append(chainStates, &ChainAccountState{
			ChainName:    chainName,
			AccountState: accountState,
		})

		// Aggregate totals
		for _, balance := range accountState.Balances {
			if current, exists := totalBalanceMap[balance.Denom]; exists {
				// TODO: Add proper decimal arithmetic
				totalBalanceMap[balance.Denom] = current + "+" + balance.Amount
			} else {
				totalBalanceMap[balance.Denom] = balance.Amount
			}
		}

		for _, delegation := range accountState.Delegations {
			denom := "stake" // Default denom for delegations
			if current, exists := totalDelegatedMap[denom]; exists {
				// TODO: Add proper decimal arithmetic
				totalDelegatedMap[denom] = current + "+" + delegation.Shares
			} else {
				totalDelegatedMap[denom] = delegation.Shares
			}
		}
	}

	// Convert maps to slices
	totalBalance := make([]*DenomAmount, 0, len(totalBalanceMap))
	for denom, amount := range totalBalanceMap {
		totalBalance = append(totalBalance, &DenomAmount{
			Denom:  denom,
			Amount: amount,
		})
	}

	totalDelegated := make([]*DenomAmount, 0, len(totalDelegatedMap))
	for denom, amount := range totalDelegatedMap {
		totalDelegated = append(totalDelegated, &DenomAmount{
			Denom:  denom,
			Amount: amount,
		})
	}

	var totalRewards float64
	for range totalDelegated {
		totalRewards += 0.05
	}

	return &CrossChainAccountState{
		Address: address,
		Chains:  chainStates,
		Totals: &CrossChainTotals{
			TotalBalance:   totalBalance,
			TotalDelegated: totalDelegated,
			TotalUnbonding: []*DenomAmount{},
			TotalRewards:   totalRewards,
		},
	}, nil
}

// Chains resolver
func (r *queryResolver) Chains(ctx context.Context) ([]*types.ChainInfo, error) {
	// For now, return hardcoded chain info
	// In a real implementation, this would come from the database
	chains := []*types.ChainInfo{
		{
			Name:         "cosmoshub",
			ChainID:      "cosmoshub-4",
			Status:       "active",
			LatestHeight: 0, // TODO: Get from database
			LatestTime:   time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Name:         "osmosis",
			ChainID:      "osmosis-1",
			Status:       "active",
			LatestHeight: 0, // TODO: Get from database
			LatestTime:   time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	return chains, nil
}

// Chain resolver
func (r *queryResolver) Chain(ctx context.Context, name string) (*types.ChainInfo, error) {
	chains, err := r.Chains(ctx)
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

// ChainStats resolver
func (r *queryResolver) ChainStats(ctx context.Context, name string) (*types.ChainStats, error) {
	// Try to get stats from ClickHouse if available
	if r.storage.ClickHouse() != nil {
		stats, err := r.storage.ClickHouse().GetChainStats(ctx, name)
		if err == nil {
			return stats, nil
		}
		r.logger.Warn("Failed to get chain stats from ClickHouse, falling back",
			zap.String("chain", name),
			zap.Error(err))
	}

	// Fallback: basic stats from PostgreSQL
	validators, err := r.storage.Postgres().GetValidators(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get validators for stats: %w", err)
	}

	return &types.ChainStats{
		ChainName:       name,
		TotalValidators: int64(len(validators)),
		// TODO: Calculate other stats from PostgreSQL
	}, nil
}

// Validators resolver
func (r *queryResolver) Validators(ctx context.Context, chain string) ([]*types.Validator, error) {
	validators, err := r.storage.Postgres().GetValidators(ctx, chain)
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	// Convert to pointers
	result := make([]*types.Validator, len(validators))
	for i := range validators {
		result[i] = &validators[i]
	}

	return result, nil
}

// Validator resolver
func (r *queryResolver) Validator(ctx context.Context, chain string, address string) (*types.Validator, error) {
	validators, err := r.Validators(ctx, chain)
	if err != nil {
		return nil, err
	}

	for _, validator := range validators {
		if validator.OperatorAddress == address {
			return validator, nil
		}
	}

	return nil, fmt.Errorf("validator not found: %s", address)
}

// CrossChainValidators resolver
func (r *queryResolver) CrossChainValidators(ctx context.Context, chains []string) (*CrossChainValidators, error) {
	result := make([]*ChainValidators, 0, len(chains))

	for _, chainName := range chains {
		validators, err := r.Validators(ctx, chainName)
		if err != nil {
			r.logger.Error("Failed to get validators for cross-chain query",
				zap.String("chain", chainName),
				zap.Error(err))
			continue
		}

		result = append(result, &ChainValidators{
			ChainName:  chainName,
			Validators: validators,
		})
	}

	return &CrossChainValidators{
		Validators: result,
	}, nil
}

// Proposals resolver
func (r *queryResolver) Proposals(ctx context.Context, chain string) ([]*types.Proposal, error) {
	// TODO: Implement proposal queries
	return []*types.Proposal{}, nil
}

// Proposal resolver
func (r *queryResolver) Proposal(ctx context.Context, chain string, id string) (*types.Proposal, error) {
	// TODO: Implement single proposal query
	return nil, fmt.Errorf("proposal queries not yet implemented")
}

// ProposalVotes resolver
func (r *queryResolver) ProposalVotes(ctx context.Context, chain string, proposalID string) ([]*types.Vote, error) {
	// TODO: Implement proposal votes query
	return []*types.Vote{}, nil
}

// BalanceHistory resolver
func (r *queryResolver) BalanceHistory(ctx context.Context, address string, chain string, denom *string, limit *int) ([]*types.BalanceEvent, error) {
	if r.storage.ClickHouse() == nil {
		return nil, fmt.Errorf("ClickHouse not available for analytics queries")
	}

	limitVal := 100
	if limit != nil && *limit > 0 {
		limitVal = *limit
	}

	events, err := r.storage.ClickHouse().GetBalanceHistory(ctx, address, *denom, "cosmos", limitVal)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance history: %w", err)
	}

	// Convert to pointers
	result := make([]*types.BalanceEvent, len(events))
	for i := range events {
		result[i] = &events[i]
	}

	return result, nil
}

// DelegationHistory resolver
func (r *queryResolver) DelegationHistory(ctx context.Context, address string, chain string, limit *int) ([]*types.DelegationEvent, error) {
	if r.storage.ClickHouse() == nil {
		return nil, fmt.Errorf("ClickHouse not available for analytics queries")
	}

	limitVal := 100
	if limit != nil && *limit > 0 {
		limitVal = *limit
	}

	events, err := r.storage.ClickHouse().GetDelegationHistory(ctx, chain, address, limitVal)
	if err != nil {
		return nil, fmt.Errorf("failed to get delegation history: %w", err)
	}

	// Convert to pointers
	result := make([]*types.DelegationEvent, len(events))
	for i := range events {
		result[i] = &events[i]
	}

	return result, nil
}
