# Chain Indexer

High-performance blockchain event indexer for ERC-20 Transfer events with REST API.

## Features

- **Real-time Indexing**: Continuously indexes new blocks with configurable confirmation depth
- **Historical Backfill**: Efficiently backfill historical data with batched processing
- **REST API**: Query transfers by address, token, block range, or time range
- **Caching**: Redis-based caching for frequently accessed data
- **Metrics**: Prometheus metrics for monitoring indexer performance
- **Production Ready**: Docker support, graceful shutdown, health checks

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Ethereum   │────▶│   Indexer   │────▶│ PostgreSQL  │
│    Node     │     │   (Go)      │     │ TimescaleDB │
└─────────────┘     └─────────────┘     └─────────────┘
                          │                    │
                          ▼                    │
                    ┌───────────┐              │
                    │   Redis   │              │
                    │  (cache)  │              │
                    └───────────┘              │
                          │                    │
                          ▼                    ▼
                    ┌─────────────────────────────┐
                    │         REST API            │
                    │  /transfers, /tokens, etc   │
                    └─────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- (Optional) Foundry for local Anvil node

### Development Setup

1. Clone the repository:
```bash
git clone https://github.com/bimakw/chain-indexer.git
cd chain-indexer
```

2. Start infrastructure:
```bash
docker-compose up -d
```

3. Run the indexer:
```bash
make run-indexer
```

4. Run the API (in another terminal):
```bash
make run-api
```

### Using Anvil (Local Fork)

Start Anvil with Ethereum mainnet fork:
```bash
make anvil
```

Or use Docker Compose which includes Anvil:
```bash
docker-compose up -d anvil
```

## API Reference

### Get Transfers

```bash
# Get all transfers
GET /api/v1/transfers

# Filter by token
GET /api/v1/transfers?token=0xdAC17F958D2ee523a2206206994597C13D831ec7

# Filter by address (sender or receiver)
GET /api/v1/transfers?address=0x...

# Filter by block range
GET /api/v1/transfers?from_block=19000000&to_block=19001000

# Filter by time range
GET /api/v1/transfers?from_time=2024-01-01T00:00:00Z&to_time=2024-01-02T00:00:00Z

# Pagination
GET /api/v1/transfers?limit=50&offset=100
```

### Get Transfers by Address

```bash
GET /api/v1/transfers/address/0x...
```

### Get Transfers by Token

```bash
GET /api/v1/tokens/0xdAC17F958D2ee523a2206206994597C13D831ec7/transfers
```

### Health Check

```bash
GET /health    # Detailed health status
GET /ready     # Kubernetes readiness probe
GET /live      # Kubernetes liveness probe
```

### Metrics

```bash
GET /metrics   # Prometheus metrics
```

## Configuration

Configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ETH_RPC_URL` | `http://localhost:8545` | Ethereum RPC endpoint |
| `ETH_CHAIN_ID` | `1` | Expected chain ID |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `indexer` | PostgreSQL user |
| `DB_PASSWORD` | `indexer` | PostgreSQL password |
| `DB_NAME` | `chain_indexer` | PostgreSQL database |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `API_PORT` | `8081` | API server port |
| `INDEXER_METRICS_PORT` | `8080` | Indexer metrics port |
| `INDEXER_BATCH_SIZE` | `100` | Blocks per batch |
| `INDEXER_BLOCK_CONFIRMATIONS` | `12` | Block confirmations |
| `INDEXER_TOKEN_ADDRESSES` | USDT,USDC | Comma-separated token addresses |

See `.env.example` for all options.

## Project Structure

```
chain-indexer/
├── cmd/
│   ├── indexer/          # Indexer entrypoint
│   └── api/              # API server entrypoint
├── internal/
│   ├── config/           # Configuration management
│   ├── domain/
│   │   ├── entities/     # Domain models
│   │   └── repositories/ # Repository interfaces
│   ├── infrastructure/
│   │   ├── ethereum/     # Ethereum client & parser
│   │   ├── database/     # PostgreSQL repositories
│   │   └── cache/        # Redis cache
│   ├── application/
│   │   └── services/     # Business logic
│   └── presentation/
│       ├── handlers/     # HTTP handlers
│       └── middleware/   # HTTP middleware
├── migrations/           # Database migrations
├── deployments/          # Docker & K8s configs
└── scripts/              # Utility scripts
```

## Default Tokens

By default, the indexer tracks:
- **USDT**: `0xdAC17F958D2ee523a2206206994597C13D831ec7`
- **USDC**: `0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48`

## Production Deployment

Build Docker images:
```bash
docker build --target indexer -t chain-indexer-indexer .
docker build --target api -t chain-indexer-api .
```

Run with production compose:
```bash
cd deployments
docker-compose -f docker-compose.prod.yml up -d
```

## Monitoring

Access Prometheus metrics at `/metrics`:
- `indexer_blocks_indexed_total` - Total blocks indexed
- `indexer_transfers_indexed_total` - Total transfers indexed
- `indexer_last_indexed_block` - Current block height
- `http_requests_total` - API request count
- `http_request_duration_seconds` - API latency

Enable Grafana dashboard:
```bash
docker-compose --profile monitoring up -d
```

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Format
go fmt ./...
```

## License

MIT License - see LICENSE file for details.
