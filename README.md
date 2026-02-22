# Chain Indexer

[![CI](https://img.shields.io/github/actions/workflow/status/bimakw/chain-indexer/ci.yml?branch=main)](https://github.com/bimakw/chain-indexer/actions)

Real-time blockchain event indexer for ERC-20 transfers. Indexes Ethereum blocks, stores events in PostgreSQL (TimescaleDB), caches hot queries with Redis, and streams updates via WebSocket.

## Features

- **Block Indexer** — processes blocks concurrently with configurable batch size
- **TimescaleDB** — hypertables for time-series transfer data with automatic partitioning
- **Redis Cache** — caches frequent queries (token balances, recent transfers)
- **WebSocket** — real-time transfer notifications per address/token
- **REST API** — filter transfers by token, address, block range, time range
- **Prometheus** — indexer lag, blocks processed, API latency metrics
- **Graceful Shutdown** — drains in-flight blocks before stopping

## Stack

| Component | Tech |
|-----------|------|
| Language | Go 1.22+ |
| Blockchain | go-ethereum (ethclient) |
| Database | PostgreSQL 15 + TimescaleDB |
| Cache | Redis 7 |
| API | net/http + chi router |
| Monitoring | Prometheus + pprof |
| Testing | go test + testcontainers |

## Running

```bash
docker-compose up -d   # postgres + redis + anvil
make run-indexer       # start indexing
make run-api           # separate terminal — REST + WebSocket
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/transfers` | Filter by token, address, block range, time |
| GET | `/api/v1/transfers/address/{addr}` | Transfers by address |
| GET | `/api/v1/tokens/{addr}/transfers` | Transfers by token contract |
| WS | `/ws/transfers` | Real-time transfer stream |
| GET | `/health` `/ready` `/live` | Health checks |
| GET | `/metrics` | Prometheus metrics |

See `.env.example` for full configuration.

## Project Structure

```
cmd/
  indexer/       block indexer entrypoint
  api/           REST + WebSocket server
internal/
  indexer/       block processing, event parsing
  store/         PostgreSQL repository layer
  cache/         Redis caching
  ws/            WebSocket hub + connections
  models/        domain types
migrations/      SQL + TimescaleDB setup
```

## Testing

```bash
make test
```

## License

MIT
