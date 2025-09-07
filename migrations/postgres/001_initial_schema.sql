-- Initial PostgreSQL schema for Cosmos State Mesh
-- This migration creates the core tables for storing normalized blockchain state

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Chains table
CREATE TABLE chains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(64) NOT NULL UNIQUE,
    chain_id VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    latest_height BIGINT NOT NULL DEFAULT 0,
    latest_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on chain name for fast lookups
CREATE INDEX idx_chains_name ON chains(name);
CREATE INDEX idx_chains_status ON chains(status);

-- Accounts table
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    address VARCHAR(128) NOT NULL,
    account_number BIGINT,
    sequence BIGINT,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name, address)
);

-- Create indexes for accounts
CREATE INDEX idx_accounts_chain_name ON accounts(chain_name);
CREATE INDEX idx_accounts_address ON accounts(address);
CREATE INDEX idx_accounts_chain_address ON accounts(chain_name, address);

-- Balances table
CREATE TABLE balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    address VARCHAR(128) NOT NULL,
    denom VARCHAR(128) NOT NULL,
    amount DECIMAL(78, 0) NOT NULL DEFAULT 0, -- Support very large numbers
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name, address, denom)
);

-- Create indexes for balances
CREATE INDEX idx_balances_chain_name ON balances(chain_name);
CREATE INDEX idx_balances_address ON balances(address);
CREATE INDEX idx_balances_denom ON balances(denom);
CREATE INDEX idx_balances_chain_address ON balances(chain_name, address);
CREATE INDEX idx_balances_chain_denom ON balances(chain_name, denom);

-- Validators table
CREATE TABLE validators (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    operator_address VARCHAR(128) NOT NULL,
    consensus_address VARCHAR(128) NOT NULL,
    consensus_pubkey TEXT,
    moniker VARCHAR(256),
    identity VARCHAR(64),
    website VARCHAR(256),
    security_contact VARCHAR(256),
    details TEXT,
    commission_rate DECIMAL(20, 18) NOT NULL DEFAULT 0,
    commission_max_rate DECIMAL(20, 18) NOT NULL DEFAULT 0,
    commission_max_change_rate DECIMAL(20, 18) NOT NULL DEFAULT 0,
    min_self_delegation DECIMAL(78, 0) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL,
    jailed BOOLEAN NOT NULL DEFAULT FALSE,
    tokens DECIMAL(78, 0) NOT NULL DEFAULT 0,
    delegator_shares DECIMAL(78, 18) NOT NULL DEFAULT 0,
    unbonding_height BIGINT NOT NULL DEFAULT 0,
    unbonding_time TIMESTAMP WITH TIME ZONE,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name, operator_address)
);

-- Create indexes for validators
CREATE INDEX idx_validators_chain_name ON validators(chain_name);
CREATE INDEX idx_validators_operator_address ON validators(operator_address);
CREATE INDEX idx_validators_consensus_address ON validators(consensus_address);
CREATE INDEX idx_validators_status ON validators(status);
CREATE INDEX idx_validators_jailed ON validators(jailed);
CREATE INDEX idx_validators_tokens ON validators(tokens DESC);

-- Delegations table
CREATE TABLE delegations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    delegator_address VARCHAR(128) NOT NULL,
    validator_address VARCHAR(128) NOT NULL,
    shares DECIMAL(78, 18) NOT NULL DEFAULT 0,
    amount DECIMAL(78, 0) NOT NULL DEFAULT 0,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name, delegator_address, validator_address)
);

-- Create indexes for delegations
CREATE INDEX idx_delegations_chain_name ON delegations(chain_name);
CREATE INDEX idx_delegations_delegator_address ON delegations(delegator_address);
CREATE INDEX idx_delegations_validator_address ON delegations(validator_address);
CREATE INDEX idx_delegations_chain_delegator ON delegations(chain_name, delegator_address);
CREATE INDEX idx_delegations_chain_validator ON delegations(chain_name, validator_address);

-- Unbonding delegations table
CREATE TABLE unbonding_delegations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    delegator_address VARCHAR(128) NOT NULL,
    validator_address VARCHAR(128) NOT NULL,
    creation_height BIGINT NOT NULL,
    completion_time TIMESTAMP WITH TIME ZONE NOT NULL,
    initial_balance DECIMAL(78, 0) NOT NULL,
    balance DECIMAL(78, 0) NOT NULL,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for unbonding delegations
CREATE INDEX idx_unbonding_delegations_chain_name ON unbonding_delegations(chain_name);
CREATE INDEX idx_unbonding_delegations_delegator_address ON unbonding_delegations(delegator_address);
CREATE INDEX idx_unbonding_delegations_validator_address ON unbonding_delegations(validator_address);
CREATE INDEX idx_unbonding_delegations_completion_time ON unbonding_delegations(completion_time);

-- Redelegations table
CREATE TABLE redelegations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    delegator_address VARCHAR(128) NOT NULL,
    validator_src_address VARCHAR(128) NOT NULL,
    validator_dst_address VARCHAR(128) NOT NULL,
    creation_height BIGINT NOT NULL,
    completion_time TIMESTAMP WITH TIME ZONE NOT NULL,
    initial_balance DECIMAL(78, 0) NOT NULL,
    shares_dst DECIMAL(78, 18) NOT NULL,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for redelegations
CREATE INDEX idx_redelegations_chain_name ON redelegations(chain_name);
CREATE INDEX idx_redelegations_delegator_address ON redelegations(delegator_address);
CREATE INDEX idx_redelegations_validator_src_address ON redelegations(validator_src_address);
CREATE INDEX idx_redelegations_validator_dst_address ON redelegations(validator_dst_address);
CREATE INDEX idx_redelegations_completion_time ON redelegations(completion_time);

-- Proposals table
CREATE TABLE proposals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    proposal_id BIGINT NOT NULL,
    content JSONB NOT NULL,
    status VARCHAR(64) NOT NULL,
    final_tally_result JSONB,
    submit_time TIMESTAMP WITH TIME ZONE NOT NULL,
    deposit_end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    total_deposit JSONB,
    voting_start_time TIMESTAMP WITH TIME ZONE,
    voting_end_time TIMESTAMP WITH TIME ZONE,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name, proposal_id)
);

-- Create indexes for proposals
CREATE INDEX idx_proposals_chain_name ON proposals(chain_name);
CREATE INDEX idx_proposals_proposal_id ON proposals(proposal_id);
CREATE INDEX idx_proposals_status ON proposals(status);
CREATE INDEX idx_proposals_submit_time ON proposals(submit_time DESC);
CREATE INDEX idx_proposals_voting_start_time ON proposals(voting_start_time);
CREATE INDEX idx_proposals_voting_end_time ON proposals(voting_end_time);

-- Votes table
CREATE TABLE votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    proposal_id BIGINT NOT NULL,
    voter VARCHAR(128) NOT NULL,
    option VARCHAR(32) NOT NULL,
    weight DECIMAL(20, 18) NOT NULL DEFAULT 1.0,
    height BIGINT NOT NULL,
    tx_hash VARCHAR(64),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name, proposal_id, voter)
);

-- Create indexes for votes
CREATE INDEX idx_votes_chain_name ON votes(chain_name);
CREATE INDEX idx_votes_proposal_id ON votes(proposal_id);
CREATE INDEX idx_votes_voter ON votes(voter);
CREATE INDEX idx_votes_option ON votes(option);
CREATE INDEX idx_votes_chain_proposal ON votes(chain_name, proposal_id);

-- Supply table
CREATE TABLE supply (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    denom VARCHAR(128) NOT NULL,
    amount DECIMAL(78, 0) NOT NULL DEFAULT 0,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name, denom)
);

-- Create indexes for supply
CREATE INDEX idx_supply_chain_name ON supply(chain_name);
CREATE INDEX idx_supply_denom ON supply(denom);
CREATE INDEX idx_supply_amount ON supply(amount DESC);

-- Mint parameters table
CREATE TABLE mint_params (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    mint_denom VARCHAR(128) NOT NULL,
    inflation_rate_change DECIMAL(20, 18) NOT NULL,
    inflation_max DECIMAL(20, 18) NOT NULL,
    inflation_min DECIMAL(20, 18) NOT NULL,
    goal_bonded DECIMAL(20, 18) NOT NULL,
    blocks_per_year BIGINT NOT NULL,
    current_inflation DECIMAL(20, 18) NOT NULL,
    annual_provisions DECIMAL(78, 0) NOT NULL,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name)
);

-- Create indexes for mint parameters
CREATE INDEX idx_mint_params_chain_name ON mint_params(chain_name);

-- Slashing info table
CREATE TABLE slashing_info (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_name VARCHAR(64) NOT NULL REFERENCES chains(name) ON DELETE CASCADE,
    consensus_address VARCHAR(128) NOT NULL,
    start_height BIGINT NOT NULL,
    index_offset BIGINT NOT NULL,
    jailed_until TIMESTAMP WITH TIME ZONE NOT NULL,
    tombstoned BOOLEAN NOT NULL DEFAULT FALSE,
    missed_blocks_counter BIGINT NOT NULL DEFAULT 0,
    height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(chain_name, consensus_address)
);

-- Create indexes for slashing info
CREATE INDEX idx_slashing_info_chain_name ON slashing_info(chain_name);
CREATE INDEX idx_slashing_info_consensus_address ON slashing_info(consensus_address);
CREATE INDEX idx_slashing_info_jailed_until ON slashing_info(jailed_until);
CREATE INDEX idx_slashing_info_tombstoned ON slashing_info(tombstoned);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add updated_at triggers to all tables
CREATE TRIGGER update_chains_updated_at BEFORE UPDATE ON chains FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_accounts_updated_at BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_balances_updated_at BEFORE UPDATE ON balances FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_validators_updated_at BEFORE UPDATE ON validators FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_delegations_updated_at BEFORE UPDATE ON delegations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_unbonding_delegations_updated_at BEFORE UPDATE ON unbonding_delegations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_redelegations_updated_at BEFORE UPDATE ON redelegations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_proposals_updated_at BEFORE UPDATE ON proposals FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_supply_updated_at BEFORE UPDATE ON supply FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mint_params_updated_at BEFORE UPDATE ON mint_params FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_slashing_info_updated_at BEFORE UPDATE ON slashing_info FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
