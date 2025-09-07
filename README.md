# Cosmos State Mesh

A unified state query layer for Cosmos SDK blockchains that provides cross-module and cross-chain aggregation capabilities.

## Overview

Cosmos State Mesh addresses the developer friction of querying multiple endpoints across Cosmos SDK modules and chains by providing:

- **Chain-agnostic unified query layer** for Cosmos state
- **Cross-module aggregation** (balances + delegations + rewards in one response)
- **Cross-chain federation** (total ATOM staked across Hub + consumer chains)
- **Developer-friendly APIs** (GraphQL/REST/SDKs)
- **Real-time state streaming** using ADR-038 State Listening

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Cosmos Hub    │    │    Osmosis      │    │      Juno       │
│   gRPC:9090     │    │   gRPC:9090     │    │   gRPC:9090     │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼─────────────┐
                    │     State Mesh Ingester   │
                    │   (ADR-038 Streaming)     │
                    └─────────────┬─────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │      Kafka Streaming      │
                    └─────────────┬─────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
┌─────────▼─────────┐   ┌─────────▼─────────┐   ┌─────────▼─────────┐
│    PostgreSQL     │   │    ClickHouse     │   │   GraphQL API     │
│ (Normalized State)│   │   (Analytics)     │   │ (Query Interface) │
└───────────────────┘   └───────────────────┘   └───────────────────┘
```

## Features

### Core Features
- **State Ingestion**: Connect to chain gRPC endpoints and subscribe to module state
- **Cross-Module Aggregation**: Unified user/account-centric views
- **Cross-Chain Federation**: Multi-chain queries and aggregation
- **Developer API**: GraphQL and REST endpoints with TypeScript/Rust SDKs
- **Observability**: Metrics, monitoring, and error handling

### Supported Modules
- Bank (balances, supply, metadata)
- Staking (delegations, validators, rewards)
- Distribution (rewards, commission)
- Governance (proposals, votes)
- Mint (inflation, supply)
- Slashing (validator penalties)
- Authz (authorization grants)
- Feegrant (fee allowances)

## Quick Start

### Prerequisites
- Go 1.22+
- PostgreSQL 14+
- ClickHouse 23+
- Kafka (optional, for production streaming)

### Installation

```bash
# Clone the repository
git clone https://github.com/cosmos/state-mesh
cd state-mesh

# Install dependencies
go mod download

# Build the application
make build

# Run database migrations
make migrate

# Start the ingester
./bin/state-mesh ingest --config config.yaml

# Start the API server
./bin/state-mesh serve --config config.yaml
```

### Configuration

```yaml
# config.yaml
chains:
  - name: "cosmoshub"
    grpc_endpoint: "localhost:9090"
    modules: ["bank", "staking", "distribution", "gov"]
  - name: "osmosis"
    grpc_endpoint: "osmosis.grpc.endpoint:9090"
    modules: ["bank", "staking"]

database:
  postgres:
    host: "localhost"
    port: 5432
    database: "statemesh"
    user: "postgres"
    password: "password"
  
  clickhouse:
    host: "localhost"
    port: 9000
    database: "statemesh_analytics"

streaming:
  kafka:
    brokers: ["localhost:9092"]
    topic: "cosmos-state-changes"

api:
  graphql:
    port: 8080
    playground: true
  rest:
    port: 8081

observability:
  metrics:
    port: 9090
  logging:
    level: "info"
```

## API Examples

### GraphQL Queries

```graphql
# Get unified account state across modules
query GetAccountState($address: String!) {
  account(address: $address) {
    balances {
      denom
      amount
    }
    delegations {
      validator
      amount
      rewards
    }
    unbonding {
      validator
      amount
      completionTime
    }
  }
}

# Cross-chain aggregation
query GetCrossChainBalances($address: String!) {
  chains {
    name
    account(address: $address) {
      balances {
        denom
        amount
      }
    }
  }
}
```

### REST API

```bash
# Get account balances across all chains
GET /api/v1/accounts/{address}/balances

# Get staking information
GET /api/v1/accounts/{address}/staking

# Get governance proposals
GET /api/v1/governance/proposals?status=voting

# Cross-chain validator information
GET /api/v1/validators?chains=cosmoshub,osmosis
```

## Development

### Project Structure

```
├── cmd/                    # CLI commands
├── internal/
│   ├── config/            # Configuration management
│   ├── ingester/          # State ingestion logic
│   ├── storage/           # Database interfaces
│   ├── api/               # GraphQL/REST handlers
│   └── streaming/         # Kafka integration
├── pkg/
│   ├── cosmos/            # Cosmos SDK client
│   ├── types/             # Shared types
│   └── utils/             # Utilities
├── schema/                # GraphQL schema definitions
├── migrations/            # Database migrations
├── deployments/           # Kubernetes/Helm charts
└── docs/                  # Documentation
```

### Running Tests

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run with coverage
make test-coverage
```

### Building

```bash
# Build binary
make build

# Build Docker image
make docker-build

# Build and push
make docker-push
```

## Deployment

### Docker Compose (Development)

```bash
docker-compose up -d
```

### Kubernetes (Production)

```bash
# Install with Helm
helm repo add state-mesh https://cosmos.github.io/state-mesh
helm install state-mesh state-mesh/state-mesh
```

## Monitoring

State Mesh exposes Prometheus metrics on `/metrics` endpoint:

- `statemesh_ingestion_rate` - State changes ingested per second
- `statemesh_query_duration` - API query response times
- `statemesh_chain_availability` - Chain endpoint availability
- `statemesh_storage_operations` - Database operation metrics

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

Apache 2.0

## Roadmap

### Phase 1 (MVP - 4-6 weeks)
- [x] Bank + Staking modules for Cosmos Hub
- [x] GraphQL API + PostgreSQL backend
- [x] CLI demo + documentation

### Phase 2 (6-12 weeks)
- [ ] Distribution, Governance, Mint modules
- [ ] Support for 3 chains (Hub, Osmosis, Juno)
- [ ] TypeScript and Rust SDKs

### Phase 3 (12-24 weeks)
- [ ] Cross-chain federation queries
- [ ] Kafka streaming → ClickHouse analytics
- [ ] Grafana dashboards + monitoring
- [ ] IBC integration for interchain metrics

## Support

- Documentation: [docs.statemesh.cosmos.network](https://docs.statemesh.cosmos.network)
- Discord: [#state-mesh](https://discord.gg/cosmosnetwork)
- Issues: [GitHub Issues](https://github.com/cosmos/state-mesh/issues)
