-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Tokens table: stores token metadata
CREATE TABLE IF NOT EXISTS tokens (
    address VARCHAR(42) PRIMARY KEY,
    name VARCHAR(255),
    symbol VARCHAR(32),
    decimals INTEGER DEFAULT 18,
    total_indexed_transfers BIGINT DEFAULT 0,
    first_seen_block BIGINT,
    last_seen_block BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Transfers table: stores ERC-20 Transfer events
CREATE TABLE IF NOT EXISTS transfers (
    id BIGSERIAL,
    tx_hash VARCHAR(66) NOT NULL,
    log_index INTEGER NOT NULL,
    block_number BIGINT NOT NULL,
    block_timestamp TIMESTAMPTZ NOT NULL,
    token_address VARCHAR(42) NOT NULL REFERENCES tokens(address),
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42) NOT NULL,
    value NUMERIC(78, 0) NOT NULL, -- uint256 max is 78 digits
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, block_timestamp)
);

-- Convert transfers to hypertable for time-series optimization
SELECT create_hypertable('transfers', 'block_timestamp',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Indexer state: tracks indexing progress per token
CREATE TABLE IF NOT EXISTS indexer_state (
    token_address VARCHAR(42) PRIMARY KEY REFERENCES tokens(address),
    last_indexed_block BIGINT NOT NULL DEFAULT 0,
    is_backfilling BOOLEAN DEFAULT FALSE,
    backfill_from_block BIGINT,
    backfill_to_block BIGINT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_transfers_token ON transfers (token_address, block_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_from ON transfers (from_address, block_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_to ON transfers (to_address, block_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_block ON transfers (block_number);
CREATE INDEX IF NOT EXISTS idx_transfers_tx_hash ON transfers (tx_hash);

-- Unique constraint to prevent duplicate events
CREATE UNIQUE INDEX IF NOT EXISTS idx_transfers_unique
    ON transfers (tx_hash, log_index, block_timestamp);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_tokens_updated_at
    BEFORE UPDATE ON tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_indexer_state_updated_at
    BEFORE UPDATE ON indexer_state
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Compression policy for old data (after 7 days)
SELECT add_compression_policy('transfers', INTERVAL '7 days', if_not_exists => TRUE);

-- Retention policy (optional, keep 1 year of data)
-- SELECT add_retention_policy('transfers', INTERVAL '1 year', if_not_exists => TRUE);
