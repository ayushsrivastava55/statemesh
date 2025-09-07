-- Initial ClickHouse schema for Cosmos State Mesh Analytics
-- This migration creates the analytics tables optimized for time-series queries

-- Balance events table for analytics
CREATE TABLE balance_events (
    chain_name LowCardinality(String),
    address String,
    denom LowCardinality(String),
    amount String,
    event_type LowCardinality(String),
    height UInt64,
    timestamp DateTime64(3),
    date Date MATERIALIZED toDate(timestamp)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (chain_name, address, denom, timestamp)
SETTINGS index_granularity = 8192;

-- Delegation events table for analytics
CREATE TABLE delegation_events (
    chain_name LowCardinality(String),
    delegator_address String,
    validator_address String,
    amount String,
    event_type LowCardinality(String),
    height UInt64,
    timestamp DateTime64(3),
    date Date MATERIALIZED toDate(timestamp)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (chain_name, delegator_address, validator_address, timestamp)
SETTINGS index_granularity = 8192;

-- Validator performance metrics
CREATE TABLE validator_metrics (
    chain_name LowCardinality(String),
    validator_address String,
    operator_address String,
    moniker String,
    voting_power UInt64,
    commission_rate Float64,
    self_delegation String,
    total_delegation String,
    uptime_percentage Float64,
    blocks_signed UInt64,
    blocks_missed UInt64,
    jailed UInt8,
    height UInt64,
    timestamp DateTime64(3),
    date Date MATERIALIZED toDate(timestamp)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (chain_name, validator_address, timestamp)
SETTINGS index_granularity = 8192;

-- Chain statistics aggregated by hour
CREATE TABLE chain_stats_hourly (
    chain_name LowCardinality(String),
    hour DateTime,
    total_validators UInt32,
    active_validators UInt32,
    total_delegated String,
    total_supply String,
    inflation_rate Float64,
    avg_block_time Float64,
    transactions_count UInt64,
    unique_addresses UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (chain_name, hour)
SETTINGS index_granularity = 8192;

-- Token holder distribution
CREATE TABLE token_holders (
    chain_name LowCardinality(String),
    address String,
    denom LowCardinality(String),
    balance String,
    balance_usd Float64,
    rank UInt32,
    percentage Float64,
    height UInt64,
    timestamp DateTime64(3),
    date Date MATERIALIZED toDate(timestamp)
) ENGINE = ReplacingMergeTree(timestamp)
PARTITION BY (chain_name, denom, toYYYYMM(date))
ORDER BY (chain_name, denom, rank, address)
SETTINGS index_granularity = 8192;

-- Governance proposal analytics
CREATE TABLE proposal_analytics (
    chain_name LowCardinality(String),
    proposal_id UInt64,
    proposal_type LowCardinality(String),
    status LowCardinality(String),
    yes_votes String,
    no_votes String,
    abstain_votes String,
    no_with_veto_votes String,
    total_votes String,
    turnout_percentage Float64,
    submit_time DateTime64(3),
    voting_start_time DateTime64(3),
    voting_end_time DateTime64(3),
    height UInt64,
    timestamp DateTime64(3),
    date Date MATERIALIZED toDate(timestamp)
) ENGINE = ReplacingMergeTree(timestamp)
PARTITION BY toYYYYMM(date)
ORDER BY (chain_name, proposal_id, timestamp)
SETTINGS index_granularity = 8192;

-- Cross-chain transfer events
CREATE TABLE ibc_transfer_events (
    source_chain LowCardinality(String),
    destination_chain LowCardinality(String),
    sender String,
    receiver String,
    denom String,
    amount String,
    channel_id String,
    sequence UInt64,
    timeout_height UInt64,
    timeout_timestamp UInt64,
    success UInt8,
    height UInt64,
    timestamp DateTime64(3),
    date Date MATERIALIZED toDate(timestamp)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (source_chain, destination_chain, timestamp)
SETTINGS index_granularity = 8192;

-- Network activity metrics
CREATE TABLE network_activity (
    chain_name LowCardinality(String),
    block_height UInt64,
    block_time DateTime64(3),
    num_txs UInt32,
    gas_used UInt64,
    gas_wanted UInt64,
    proposer_address String,
    validator_hash String,
    date Date MATERIALIZED toDate(block_time)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (chain_name, block_height)
SETTINGS index_granularity = 8192;

-- Materialized views for common aggregations

-- Daily balance changes
CREATE MATERIALIZED VIEW balance_changes_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (chain_name, address, denom, date)
AS SELECT
    chain_name,
    address,
    denom,
    toDate(timestamp) as date,
    count() as events_count,
    sum(toFloat64OrZero(amount)) as total_amount_change
FROM balance_events
GROUP BY chain_name, address, denom, date;

-- Hourly delegation changes
CREATE MATERIALIZED VIEW delegation_changes_hourly
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (chain_name, validator_address, hour)
AS SELECT
    chain_name,
    validator_address,
    toStartOfHour(timestamp) as hour,
    count() as events_count,
    sum(toFloat64OrZero(amount)) as total_delegation_change
FROM delegation_events
GROUP BY chain_name, validator_address, hour;

-- Daily chain metrics
CREATE MATERIALIZED VIEW chain_metrics_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (chain_name, date)
AS SELECT
    chain_name,
    toDate(block_time) as date,
    count() as blocks_count,
    sum(num_txs) as total_transactions,
    avg(gas_used) as avg_gas_used,
    uniq(proposer_address) as unique_proposers
FROM network_activity
GROUP BY chain_name, date;

-- Top token holders view
CREATE MATERIALIZED VIEW top_token_holders
ENGINE = ReplacingMergeTree(timestamp)
PARTITION BY (chain_name, denom)
ORDER BY (chain_name, denom, balance_rank)
AS SELECT
    chain_name,
    denom,
    address,
    balance,
    balance_usd,
    row_number() OVER (PARTITION BY chain_name, denom ORDER BY toFloat64OrZero(balance) DESC) as balance_rank,
    timestamp
FROM token_holders
WHERE balance != '0';

-- Validator performance summary
CREATE MATERIALIZED VIEW validator_performance_summary
ENGINE = ReplacingMergeTree(timestamp)
PARTITION BY chain_name
ORDER BY (chain_name, validator_address)
AS SELECT
    chain_name,
    validator_address,
    operator_address,
    moniker,
    avg(voting_power) as avg_voting_power,
    avg(commission_rate) as avg_commission_rate,
    avg(uptime_percentage) as avg_uptime,
    sum(blocks_signed) as total_blocks_signed,
    sum(blocks_missed) as total_blocks_missed,
    max(timestamp) as timestamp
FROM validator_metrics
GROUP BY chain_name, validator_address, operator_address, moniker;

-- Create indexes for better query performance
-- Note: ClickHouse uses different indexing approach, these are skip indexes

-- Skip index for addresses in balance events
ALTER TABLE balance_events ADD INDEX idx_address address TYPE bloom_filter GRANULARITY 1;

-- Skip index for validator addresses in delegation events  
ALTER TABLE delegation_events ADD INDEX idx_validator validator_address TYPE bloom_filter GRANULARITY 1;

-- Skip index for proposal status
ALTER TABLE proposal_analytics ADD INDEX idx_status status TYPE set(10) GRANULARITY 1;

-- Skip index for IBC channels
ALTER TABLE ibc_transfer_events ADD INDEX idx_channel channel_id TYPE bloom_filter GRANULARITY 1;
