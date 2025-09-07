# PRD: Cosmos State Mesh (Unified State Query Layer)

---

## 1. Problem Statement
Current Cosmos SDK chains expose state (balances, staking, params, governance, etc.) through per-module gRPC/REST APIs. This works per-chain, but:

- Developers must query multiple endpoints across modules (bank, staking, distribution).  
- Aggregating across multiple chains (Cosmos Hub, Osmosis, Juno, etc.) requires custom scripts per chain/version.  
- Explorers/indexers focus on blocks/txs, not stateful views of users, staking, params, or assets.  
- No standardized query mesh exists to unify state across the AEZ.

**Result:** Developers, wallets, and protocols duplicate work, slowing ecosystem growth.

---

## 2. Goals
- Provide a chain-agnostic, unified query layer for Cosmos state.  
- Enable cross-module aggregation (e.g., balances + delegations in one response).  
- Enable cross-chain aggregation (e.g., “show total ATOM staked across Hub + consumer chains”).  
- Expose a developer-friendly API (GraphQL/REST/SDKs).  
- Be open-source + modular, deployable as a sidecar per chain or as a multi-chain SaaS.  

---

## 3. Non-Goals
- Not replacing existing explorers ([Mintscan](https://www.mintscan.io/), [Ping.pub](https://ping.pub/)).  
- Not a tx/block indexer (skip Tendermint history indexing).  
- Not a DeFi analytics tool (but can be consumed by them).  

---

## 4. Users & Personas
- **Wallet developers**: need unified balances/staking for multi-chain wallets.  
- **DeFi protocols**: need cross-chain liquidity/state data for risk mgmt.  
- **Indexers/explorers**: want a standard backend instead of custom per-chain logic.  
- **Researchers/analysts**: query staking params, inflation, supply across chains.  

---

## 5. Product Scope

### Core Features
1. **State Ingestion**
   - Connect to chain gRPC endpoints.  
   - Subscribe to module state (bank, staking, distribution, gov, mint, slashing).  
   - Normalize into standard schemas.  

2. **Cross-Module Aggregation**
   - E.g., “get balances + staking + distribution rewards for address X.”  
   - Schema unifies user/account-centric views.  

3. **Cross-Chain Federation**
   - Support multiple chains simultaneously.  
   - Queries resolve across chains (per denom, per address, etc.).  

4. **Developer API**
   - **GraphQL** for flexible queries.  
   - **REST** for simple integrations.  
   - SDKs in **TypeScript & Rust**.  

5. **Observability**
   - Metrics: query volume, latency, chain availability.  
   - Error handling for chain downtime.  

---

### Future Extensions
- State snapshots / historical state queries.  
- AI query assistant (“explain staking health of AEZ”).  
- Integration with IBC/ICS for live interchain metrics.  

---

## 6. Tech Stack

### Backend
- **Cosmos SDK gRPC + Tendermint RPC** as data source.  
- **Go services** for ingestion (native to SDK ecosystem).  
- **Kafka / NATS streaming** (for real-time state change streaming).  
- **Postgres** (normalized storage).  
- **ClickHouse** (for analytics queries at scale).  

### API Layer
- **Hasura GraphQL Engine** or custom **Apollo Server** on top of Postgres.  
- REST API via **Go Gin/Fiber** service.  

### Deployment
- **Docker / Kubernetes** (sidecar or multi-chain deployment).  
- **Helm charts** for validators/chains to self-host.  

### Observability
- **Prometheus + Grafana dashboards** for chain status + query metrics.  

---

## 7. References (Docs to Consult)

1. **Cosmos SDK Modules**
   - Bank: [Bank Module Client](https://docs.cosmos.network/v0.46/modules/bank/06_client.html)  
   - Staking: [Staking Module Client](https://docs.cosmos.network/v0.46/modules/staking/09_client.html)  
   - Distribution: [Distribution Module](https://docs.cosmos.network/v0.46/modules/distribution/)  
   - Governance: [Governance Module](https://docs.cosmos.network/v0.46/modules/gov/)  
   - Mint: [Mint Module](https://docs.cosmos.network/v0.46/modules/mint/)  
   - Slashing: [Slashing Module](https://docs.cosmos.network/v0.46/modules/slashing/)  
   - Authz: [Authz Module](https://docs.cosmos.network/v0.46/modules/authz/)  
   - Feegrant: [Feegrant Module](https://docs.cosmos.network/v0.46/modules/feegrant/)  

2. **Cosmos gRPC & REST APIs**
   - [Cosmos SDK API Reference](https://docs.cosmos.network/main/build/modules)  

3. **ADR-038: State Listening / Streaming**
   - [ADR-038 on GitHub](https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-038-state-listening.md)  

4. **IBC & ICS**
   - [IBC Overview](https://ibc.cosmos.network/main/)  
   - [ICS Overview](https://docs.cosmos.network/main/ibc/ics-overview.html)  

5. **Existing Indexers**
   - [Ping.pub Explorer](https://ping.pub/)  
   - [Mintscan Explorer](https://www.mintscan.io/)  
   - [Big Dipper Explorer](https://bigdipper.live/)  

📚 Estimated docs to reference: **~15–20 official pages**

---

## 8. Success Metrics
- **MVP adoption**: 2 wallets + 1 explorer integrate within 2 months.  
- **Performance**: <200ms response time for single-chain queries.  
- **Coverage**: Support at least 3 major chains (Hub, Osmosis, Juno).  
- **Community adoption**: 50+ devs using GraphQL API within 6 months.  

---

## 9. Rollout Plan

### Phase 1 (4–6 weeks)
- Ingest **Bank + Staking modules** for 1 chain (Cosmos Hub).  
- GraphQL API + Postgres backend.  
- CLI demo + docs.

### Phase 2 (6–12 weeks)
- Add **Distribution, Governance, Mint** modules.  
- Support **3 chains**.  
- SDKs (TypeScript, Rust).

### Phase 3 (12–24 weeks)
- Cross-chain federation queries.  
- Kafka streaming → ClickHouse.  
- Grafana dashboards + monitoring.  

---

## 10. Risks & Mitigations
- **Chain downtime** → fallback retries + error surfaces.  
- **Module version drift** → version-aware schema registry.  
- **Scaling queries** → ClickHouse backend for analytics workloads.  

---
