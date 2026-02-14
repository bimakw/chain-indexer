# Chain Indexer

Blockchain event indexer for ERC-20 Transfer events. Indexes blocks from Ethereum (or local Anvil fork), stores in PostgreSQL (TimescaleDB), caches with Redis, and serves via REST API.

## Running

```bash
docker-compose up -d
make run-indexer
make run-api   # separate terminal
```

## Endpoints

- `GET /api/v1/transfers` — filter by token, address, block range, time range
- `GET /api/v1/transfers/address/{addr}` — by address
- `GET /api/v1/tokens/{addr}/transfers` — by token
- `GET /health` / `/ready` / `/live`
- `GET /metrics` — Prometheus

See `.env.example` for config.

## Testing

```bash
make test
```

## License

MIT
