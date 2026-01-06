package repositories

import (
	"context"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// TransferRepository defines the interface for transfer data operations
type TransferRepository interface {
	// GetByFilter retrieves transfers matching the given filter
	GetByFilter(ctx context.Context, filter entities.TransferFilter) ([]entities.Transfer, error)

	// GetCount returns the count of transfers matching the filter
	GetCount(ctx context.Context, filter entities.TransferFilter) (int64, error)

	// BatchInsert inserts multiple transfers in a single transaction
	BatchInsert(ctx context.Context, transfers []entities.Transfer) error

	// GetLatestBlock returns the latest indexed block for a token
	GetLatestBlock(ctx context.Context, tokenAddress string) (int64, error)
}
