package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
)

// Ensure IndexerStateRepo implements IndexerStateRepository
var _ repositories.IndexerStateRepository = (*IndexerStateRepo)(nil)

// IndexerStateRepo implements IndexerStateRepository using PostgreSQL
type IndexerStateRepo struct {
	db *sqlx.DB
}

// NewIndexerStateRepo creates a new indexer state repository
func NewIndexerStateRepo(db *sqlx.DB) *IndexerStateRepo {
	return &IndexerStateRepo{db: db}
}

// Get retrieves the indexer state for a token
func (r *IndexerStateRepo) Get(ctx context.Context, tokenAddress string) (*entities.IndexerState, error) {
	var state entities.IndexerState
	query := `SELECT * FROM indexer_state WHERE token_address = $1`

	if err := r.db.GetContext(ctx, &state, query, tokenAddress); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get indexer state: %w", err)
	}

	return &state, nil
}

// Upsert creates or updates the indexer state
func (r *IndexerStateRepo) Upsert(ctx context.Context, state *entities.IndexerState) error {
	query := `
		INSERT INTO indexer_state (token_address, last_indexed_block, is_backfilling, backfill_from_block, backfill_to_block)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (token_address) DO UPDATE SET
			last_indexed_block = EXCLUDED.last_indexed_block,
			is_backfilling = EXCLUDED.is_backfilling,
			backfill_from_block = EXCLUDED.backfill_from_block,
			backfill_to_block = EXCLUDED.backfill_to_block,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query,
		state.TokenAddress,
		state.LastIndexedBlock,
		state.IsBackfilling,
		state.BackfillFromBlock,
		state.BackfillToBlock,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert indexer state: %w", err)
	}

	return nil
}

// UpdateLastBlock updates the last indexed block for a token
func (r *IndexerStateRepo) UpdateLastBlock(ctx context.Context, tokenAddress string, blockNumber int64) error {
	query := `
		UPDATE indexer_state SET
			last_indexed_block = $2,
			updated_at = NOW()
		WHERE token_address = $1
	`

	result, err := r.db.ExecContext(ctx, query, tokenAddress, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to update last block: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		// Insert if not exists
		return r.Upsert(ctx, &entities.IndexerState{
			TokenAddress:     tokenAddress,
			LastIndexedBlock: blockNumber,
		})
	}

	return nil
}

// SetBackfilling sets the backfilling state for a token
func (r *IndexerStateRepo) SetBackfilling(ctx context.Context, tokenAddress string, isBackfilling bool, fromBlock, toBlock *int64) error {
	query := `
		UPDATE indexer_state SET
			is_backfilling = $2,
			backfill_from_block = $3,
			backfill_to_block = $4,
			updated_at = NOW()
		WHERE token_address = $1
	`

	_, err := r.db.ExecContext(ctx, query, tokenAddress, isBackfilling, fromBlock, toBlock)
	if err != nil {
		return fmt.Errorf("failed to set backfilling: %w", err)
	}

	return nil
}
