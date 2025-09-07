package types

import (
	"time"
)

// Account represents a blockchain account
type Account struct {
	ChainName string    `json:"chain_name" db:"chain_name"`
	Address   string    `json:"address" db:"address"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Balance represents an account balance
type Balance struct {
	ChainName string    `json:"chain_name" db:"chain_name"`
	Address   string    `json:"address" db:"address"`
	Denom     string    `json:"denom" db:"denom"`
	Amount    string    `json:"amount" db:"amount"`
	Height    int64     `json:"height" db:"height"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Delegation represents a staking delegation
type Delegation struct {
	ChainName        string    `json:"chain_name" db:"chain_name"`
	DelegatorAddress string    `json:"delegator_address" db:"delegator_address"`
	ValidatorAddress string    `json:"validator_address" db:"validator_address"`
	Shares           string    `json:"shares" db:"shares"`
	Height           int64     `json:"height" db:"height"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Validator represents a validator
type Validator struct {
	ChainName          string              `json:"chain_name" db:"chain_name"`
	OperatorAddress    string              `json:"operator_address" db:"operator_address"`
	ConsensusPubkey    string              `json:"consensus_pubkey" db:"consensus_pubkey"`
	Jailed             bool                `json:"jailed" db:"jailed"`
	Status             string              `json:"status" db:"status"`
	Tokens             string              `json:"tokens" db:"tokens"`
	DelegatorShares    string              `json:"delegator_shares" db:"delegator_shares"`
	Description        ValidatorDescription `json:"description"`
	UnbondingHeight    int64               `json:"unbonding_height" db:"unbonding_height"`
	UnbondingTime      time.Time           `json:"unbonding_time" db:"unbonding_time"`
	Commission         ValidatorCommission `json:"commission"`
	MinSelfDelegation  string              `json:"min_self_delegation" db:"min_self_delegation"`
	Height             int64               `json:"height" db:"height"`
	UpdatedAt          time.Time           `json:"updated_at" db:"updated_at"`
}

// ValidatorDescription represents validator description
type ValidatorDescription struct {
	Moniker         string `json:"moniker" db:"description_moniker"`
	Identity        string `json:"identity" db:"description_identity"`
	Website         string `json:"website" db:"description_website"`
	SecurityContact string `json:"security_contact" db:"description_security_contact"`
	Details         string `json:"details" db:"description_details"`
}

// ValidatorCommission represents validator commission
type ValidatorCommission struct {
	Rate          string `json:"rate" db:"commission_rate"`
	MaxRate       string `json:"max_rate" db:"commission_max_rate"`
	MaxChangeRate string `json:"max_change_rate" db:"commission_max_change_rate"`
}

// UnbondingDelegation represents an unbonding delegation
type UnbondingDelegation struct {
	ChainName        string                   `json:"chain_name" db:"chain_name"`
	DelegatorAddress string                   `json:"delegator_address" db:"delegator_address"`
	ValidatorAddress string                   `json:"validator_address" db:"validator_address"`
	Entries          []UnbondingDelegationEntry `json:"entries"`
	Height           int64                    `json:"height" db:"height"`
	UpdatedAt        time.Time                `json:"updated_at" db:"updated_at"`
}

// UnbondingDelegationEntry represents an unbonding delegation entry
type UnbondingDelegationEntry struct {
	CreationHeight int64     `json:"creation_height"`
	CompletionTime time.Time `json:"completion_time"`
	InitialBalance string    `json:"initial_balance"`
	Balance        string    `json:"balance"`
}

// Redelegation represents a redelegation
type Redelegation struct {
	ChainName             string              `json:"chain_name" db:"chain_name"`
	DelegatorAddress      string              `json:"delegator_address" db:"delegator_address"`
	ValidatorSrcAddress   string              `json:"validator_src_address" db:"validator_src_address"`
	ValidatorDstAddress   string              `json:"validator_dst_address" db:"validator_dst_address"`
	Entries               []RedelegationEntry `json:"entries"`
	Height                int64               `json:"height" db:"height"`
	UpdatedAt             time.Time           `json:"updated_at" db:"updated_at"`
}

// RedelegationEntry represents a redelegation entry
type RedelegationEntry struct {
	CreationHeight int64     `json:"creation_height"`
	CompletionTime time.Time `json:"completion_time"`
	InitialBalance string    `json:"initial_balance"`
	SharesDst      string    `json:"shares_dst"`
}

// Reward represents staking rewards
type Reward struct {
	ChainName        string    `json:"chain_name" db:"chain_name"`
	DelegatorAddress string    `json:"delegator_address" db:"delegator_address"`
	ValidatorAddress string    `json:"validator_address" db:"validator_address"`
	Reward           []Coin    `json:"reward"`
	Height           int64     `json:"height" db:"height"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Coin represents a coin amount
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// Proposal represents a governance proposal
type Proposal struct {
	ChainName      string           `json:"chain_name" db:"chain_name"`
	ProposalID     uint64           `json:"proposal_id" db:"proposal_id"`
	Content        ProposalContent  `json:"content"`
	Status         string           `json:"status" db:"status"`
	FinalTallyResult TallyResult    `json:"final_tally_result"`
	SubmitTime     time.Time        `json:"submit_time" db:"submit_time"`
	DepositEndTime time.Time        `json:"deposit_end_time" db:"deposit_end_time"`
	TotalDeposit   []Coin           `json:"total_deposit"`
	VotingStartTime time.Time       `json:"voting_start_time" db:"voting_start_time"`
	VotingEndTime  time.Time        `json:"voting_end_time" db:"voting_end_time"`
	Height         int64            `json:"height" db:"height"`
	UpdatedAt      time.Time        `json:"updated_at" db:"updated_at"`
}

// ProposalContent represents proposal content
type ProposalContent struct {
	Type        string `json:"@type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// TallyResult represents proposal tally result
type TallyResult struct {
	Yes        string `json:"yes"`
	Abstain    string `json:"abstain"`
	No         string `json:"no"`
	NoWithVeto string `json:"no_with_veto"`
}

// Vote represents a governance vote
type Vote struct {
	ChainName  string    `json:"chain_name" db:"chain_name"`
	ProposalID uint64    `json:"proposal_id" db:"proposal_id"`
	Voter      string    `json:"voter" db:"voter"`
	Option     string    `json:"option" db:"option"`
	Height     int64     `json:"height" db:"height"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// Analytics types for ClickHouse

// BalanceEvent represents a balance change event
type BalanceEvent struct {
	Timestamp      time.Time `json:"timestamp"`
	ChainName      string    `json:"chain_name"`
	Address        string    `json:"address"`
	Denom          string    `json:"denom"`
	Amount         string    `json:"amount"`
	PreviousAmount string    `json:"previous_amount"`
	ChangeType     string    `json:"change_type"` // "increase", "decrease", "current"
	Height         int64     `json:"height"`
	TxHash         string    `json:"tx_hash"`
}

// DelegationEvent represents a delegation change event
type DelegationEvent struct {
	Timestamp       time.Time `json:"timestamp"`
	ChainName       string    `json:"chain_name"`
	DelegatorAddress string   `json:"delegator_address"`
	ValidatorAddress string   `json:"validator_address"`
	Shares          string    `json:"shares"`
	PreviousShares  string    `json:"previous_shares"`
	ChangeType      string    `json:"change_type"` // "delegate", "undelegate", "redelegate", "current"
	Height          int64     `json:"height"`
	TxHash          string    `json:"tx_hash"`
}

// ChainStats represents aggregated chain statistics
type ChainStats struct {
	ChainName       string `json:"chain_name"`
	TotalValidators int64  `json:"total_validators"`
	ActiveValidators int64 `json:"active_validators"`
	TotalDelegated  string `json:"total_delegated"`
	TotalSupply     string `json:"total_supply"`
	InflationRate   string `json:"inflation_rate"`
}

// TokenHolder represents a token holder for analytics
type TokenHolder struct {
	ChainName string `json:"chain_name"`
	Address   string `json:"address"`
	Denom     string `json:"denom"`
	Amount    string `json:"amount"`
}

// StateChange represents a generic state change from ADR-038
type StateChange struct {
	ChainName string    `json:"chain_name"`
	StoreKey  string    `json:"store_key"`
	Key       []byte    `json:"key"`
	Value     []byte    `json:"value"`
	Delete    bool      `json:"delete"`
	Height    int64     `json:"height"`
	Timestamp time.Time `json:"timestamp"`
}

// AccountState represents unified account state across modules
type AccountState struct {
	ChainName    string                `json:"chain_name"`
	Address      string                `json:"address"`
	Balances     []Balance             `json:"balances"`
	Delegations  []Delegation          `json:"delegations"`
	Unbonding    []UnbondingDelegation `json:"unbonding"`
	Redelegations []Redelegation       `json:"redelegations"`
	Rewards      []Reward              `json:"rewards"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// CrossChainAccountState represents account state across multiple chains
type CrossChainAccountState struct {
	Address   string                   `json:"address"`
	Chains    map[string]AccountState  `json:"chains"`
	Totals    CrossChainTotals         `json:"totals"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// CrossChainTotals represents aggregated totals across chains
type CrossChainTotals struct {
	TotalBalance    map[string]string `json:"total_balance"`    // denom -> total amount
	TotalDelegated  map[string]string `json:"total_delegated"`  // denom -> total delegated
	TotalUnbonding  map[string]string `json:"total_unbonding"`  // denom -> total unbonding
	TotalRewards    map[string]string `json:"total_rewards"`    // denom -> total rewards
}

// ChainInfo represents chain information
type ChainInfo struct {
	Name         string    `json:"name"`
	ChainID      string    `json:"chain_id"`
	Status       string    `json:"status"`
	LatestHeight int64     `json:"latest_height"`
	LatestTime   time.Time `json:"latest_time"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// API response types
type AccountState struct {
	ChainName   string       `json:"chain_name"`
	Address     string       `json:"address"`
	Balances    []Balance    `json:"balances"`
	Delegations []Delegation `json:"delegations"`
	// TODO: Add unbonding, redelegations, rewards
}


type CrossChainAccountState struct {
	Address string                    `json:"address"`
	Chains  map[string]AccountState   `json:"chains"`
	Totals  CrossChainTotals         `json:"totals"`
}

type CrossChainTotals struct {
	TotalBalance   map[string]string `json:"total_balance"`   // denom -> amount
	TotalDelegated map[string]string `json:"total_delegated"` // denom -> amount
	TotalUnbonding map[string]string `json:"total_unbonding"` // denom -> amount
	TotalRewards   map[string]string `json:"total_rewards"`   // denom -> amount
}
