package repositories

import (
	"context"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

type IndexerStateRepository interface {
	Get(ctx context.Context, tokenAddress string) (*entities.IndexerState, error)

	Upsert(ctx context.Context, state *entities.IndexerState) error

	UpdateLastBlock(ctx context.Context, tokenAddress string, blockNumber int64) error

	SetBackfilling(ctx context.Context, tokenAddress string, isBackfilling bool, fromBlock, toBlock *int64) error
}
