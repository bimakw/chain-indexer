-- Drop triggers
DROP TRIGGER IF EXISTS update_indexer_state_updated_at ON indexer_state;
DROP TRIGGER IF EXISTS update_tokens_updated_at ON tokens;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (order matters due to foreign keys)
DROP TABLE IF EXISTS indexer_state;
DROP TABLE IF EXISTS transfers;
DROP TABLE IF EXISTS tokens;
