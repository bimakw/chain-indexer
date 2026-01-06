package repositories

import (
	"context"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// IndexerStateRepository defines the interface for indexer state operations
type IndexerStateRepository interface {
	// Get retrieves the indexer state for a token
	Get(ctx context.Context, tokenAddress string) (*entities.IndexerState, error)

	// Upsert creates or updates the indexer state
	Upsert(ctx context.Context, state *entities.IndexerState) error

	// UpdateLastBlock updates the last indexed block for a token
	UpdateLastBlock(ctx context.Context, tokenAddress string, blockNumber int64) error

	// SetBackfilling sets the backfilling state for a token
	SetBackfilling(ctx context.Context, tokenAddress string, isBackfilling bool, fromBlock, toBlock *int64) error
}
