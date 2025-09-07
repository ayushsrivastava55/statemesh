package cosmos

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Client represents a Cosmos SDK gRPC client
type Client struct {
	conn     *grpc.ClientConn
	chainName string
	logger   *zap.Logger
	
	// Module clients
	bankClient   banktypes.QueryClient
	stakingClient stakingtypes.QueryClient
	distrClient  distrtypes.QueryClient
	govClient    govtypes.QueryClient
}

// NewClient creates a new Cosmos SDK client
func NewClient(chainName, grpcEndpoint string) (*Client, error) {
	logger := zap.L().Named("cosmos-client").With(zap.String("chain", chainName))
	
	// Create gRPC connection
	conn, err := grpc.Dial(grpcEndpoint, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)), // 16MB
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC endpoint %s: %w", grpcEndpoint, err)
	}

	client := &Client{
		conn:      conn,
		chainName: chainName,
		logger:    logger,
		
		// Initialize module clients
		bankClient:   banktypes.NewQueryClient(conn),
		stakingClient: stakingtypes.NewQueryClient(conn),
		distrClient:  distrtypes.NewQueryClient(conn),
		govClient:    govtypes.NewQueryClient(conn),
	}

	return client, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// ChainName returns the chain name
func (c *Client) ChainName() string {
	return c.chainName
}

// Bank module methods

// GetBalance gets the balance for a specific address and denom
func (c *Client) GetBalance(ctx context.Context, address, denom string) (sdk.Coin, error) {
	req := &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	}

	resp, err := c.bankClient.Balance(ctx, req)
	if err != nil {
		return sdk.Coin{}, fmt.Errorf("failed to get balance: %w", err)
	}

	return resp.Balance, nil
}

// GetAllBalances gets all balances for a specific address
func (c *Client) GetAllBalances(ctx context.Context, address string) ([]sdk.Coin, error) {
	req := &banktypes.QueryAllBalancesRequest{
		Address: address,
	}

	resp, err := c.bankClient.AllBalances(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get all balances: %w", err)
	}

	return resp.Balances, nil
}

// GetSupplyOf gets the total supply of a specific denom
func (c *Client) GetSupplyOf(ctx context.Context, denom string) (*banktypes.Coin, error) {
	req := &banktypes.QuerySupplyOfRequest{
		Denom: denom,
	}

	resp, err := c.bankClient.SupplyOf(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get supply: %w", err)
	}

	return &resp.Amount, nil
}

// GetTotalSupply gets the total supply of all denoms
func (c *Client) GetTotalSupply(ctx context.Context) ([]banktypes.Coin, error) {
	req := &banktypes.QueryTotalSupplyRequest{
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	}

	resp, err := c.bankClient.TotalSupply(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get total supply: %w", err)
	}

	return resp.Supply, nil
}

// Staking module methods

// GetDelegation gets a specific delegation
func (c *Client) GetDelegation(ctx context.Context, delegatorAddr, validatorAddr string) (*stakingtypes.DelegationResponse, error) {
	req := &stakingtypes.QueryDelegationRequest{
		DelegatorAddr: delegatorAddr,
		ValidatorAddr: validatorAddr,
	}

	resp, err := c.stakingClient.Delegation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get delegation: %w", err)
	}

	return resp.DelegationResponse, nil
}

// GetDelegatorDelegations gets all delegations for a delegator
func (c *Client) GetDelegatorDelegations(ctx context.Context, delegatorAddr string) ([]stakingtypes.DelegationResponse, error) {
	req := &stakingtypes.QueryDelegatorDelegationsRequest{
		DelegatorAddr: delegatorAddr,
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	}

	resp, err := c.stakingClient.DelegatorDelegations(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get delegator delegations: %w", err)
	}

	return resp.DelegationResponses, nil
}

// GetValidator gets a specific validator
func (c *Client) GetValidator(ctx context.Context, validatorAddr string) (*stakingtypes.Validator, error) {
	req := &stakingtypes.QueryValidatorRequest{
		ValidatorAddr: validatorAddr,
	}

	resp, err := c.stakingClient.Validator(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator: %w", err)
	}

	return &resp.Validator, nil
}

// GetValidators gets all validators
func (c *Client) GetValidators(ctx context.Context, status string) ([]stakingtypes.Validator, error) {
	req := &stakingtypes.QueryValidatorsRequest{
		Status: status,
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	}

	resp, err := c.stakingClient.Validators(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	return resp.Validators, nil
}

// GetUnbondingDelegation gets a specific unbonding delegation
func (c *Client) GetUnbondingDelegation(ctx context.Context, delegatorAddr, validatorAddr string) (*stakingtypes.UnbondingDelegation, error) {
	req := &stakingtypes.QueryUnbondingDelegationRequest{
		DelegatorAddr: delegatorAddr,
		ValidatorAddr: validatorAddr,
	}

	resp, err := c.stakingClient.UnbondingDelegation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get unbonding delegation: %w", err)
	}

	return &resp.Unbond, nil
}

// GetDelegatorUnbondingDelegations gets all unbonding delegations for a delegator
func (c *Client) GetDelegatorUnbondingDelegations(ctx context.Context, delegatorAddr string) ([]stakingtypes.UnbondingDelegation, error) {
	req := &stakingtypes.QueryDelegatorUnbondingDelegationsRequest{
		DelegatorAddr: delegatorAddr,
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	}

	resp, err := c.stakingClient.DelegatorUnbondingDelegations(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get delegator unbonding delegations: %w", err)
	}

	return resp.UnbondingResponses, nil
}

// Distribution module methods

// GetDelegationRewards gets delegation rewards for a specific delegation
func (c *Client) GetDelegationRewards(ctx context.Context, delegatorAddr, validatorAddr string) ([]banktypes.DecCoin, error) {
	req := &distrtypes.QueryDelegationRewardsRequest{
		DelegatorAddress: delegatorAddr,
		ValidatorAddress: validatorAddr,
	}

	resp, err := c.distrClient.DelegationRewards(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get delegation rewards: %w", err)
	}

	return resp.Rewards, nil
}

// GetDelegationTotalRewards gets total delegation rewards for a delegator
func (c *Client) GetDelegationTotalRewards(ctx context.Context, delegatorAddr string) (*distrtypes.QueryDelegationTotalRewardsResponse, error) {
	req := &distrtypes.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: delegatorAddr,
	}

	resp, err := c.distrClient.DelegationTotalRewards(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get delegation total rewards: %w", err)
	}

	return resp, nil
}

// GetValidatorCommission gets validator commission
func (c *Client) GetValidatorCommission(ctx context.Context, validatorAddr string) ([]banktypes.DecCoin, error) {
	req := &distrtypes.QueryValidatorCommissionRequest{
		ValidatorAddress: validatorAddr,
	}

	resp, err := c.distrClient.ValidatorCommission(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator commission: %w", err)
	}

	return resp.Commission.Commission, nil
}

// Governance module methods

// GetProposal gets a specific proposal
func (c *Client) GetProposal(ctx context.Context, proposalID uint64) (*govtypes.Proposal, error) {
	req := &govtypes.QueryProposalRequest{
		ProposalId: proposalID,
	}

	resp, err := c.govClient.Proposal(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposal: %w", err)
	}

	return resp.Proposal, nil
}

// GetProposals gets all proposals
func (c *Client) GetProposals(ctx context.Context, status govtypes.ProposalStatus) ([]govtypes.Proposal, error) {
	req := &govtypes.QueryProposalsRequest{
		ProposalStatus: status,
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	}

	resp, err := c.govClient.Proposals(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposals: %w", err)
	}

	return resp.Proposals, nil
}

// GetVote gets a specific vote
func (c *Client) GetVote(ctx context.Context, proposalID uint64, voter string) (*govtypes.Vote, error) {
	req := &govtypes.QueryVoteRequest{
		ProposalId: proposalID,
		Voter:      voter,
	}

	resp, err := c.govClient.Vote(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get vote: %w", err)
	}

	return resp.Vote, nil
}

// GetVotes gets all votes for a proposal
func (c *Client) GetVotes(ctx context.Context, proposalID uint64) ([]govtypes.Vote, error) {
	req := &govtypes.QueryVotesRequest{
		ProposalId: proposalID,
		Pagination: &query.PageRequest{
			Limit: 10000,
		},
	}

	resp, err := c.govClient.Votes(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes: %w", err)
	}

	return resp.Votes, nil
}

// Health check methods

// Ping tests the connection to the chain
func (c *Client) Ping(ctx context.Context) error {
	// Use a simple query to test connectivity
	_, err := c.GetTotalSupply(ctx)
	if err != nil {
		return fmt.Errorf("chain ping failed: %w", err)
	}
	return nil
}

// GetLatestHeight gets the latest block height
func (c *Client) GetLatestHeight(ctx context.Context) (int64, error) {
	// Get a validator to determine latest height
	validators, err := c.GetValidators(ctx, "")
	if err != nil {
		return 0, fmt.Errorf("failed to get latest height: %w", err)
	}
	
	if len(validators) == 0 {
		return 0, fmt.Errorf("no validators found")
	}
	
	// Return the unbonding height of the first validator as a proxy for latest height
	// In a real implementation, you'd query the consensus module or use Tendermint RPC
	return validators[0].UnbondingHeight, nil
}

// Utility methods

// WaitForHeight waits for the chain to reach a specific height
func (c *Client) WaitForHeight(ctx context.Context, targetHeight int64, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			height, err := c.GetLatestHeight(ctx)
			if err != nil {
				c.logger.Warn("Failed to get latest height", zap.Error(err))
				continue
			}
			
			if height >= targetHeight {
				return nil
			}
		}
	}
}
