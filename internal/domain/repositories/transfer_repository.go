package repositories

import (
	"context"
	"time"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// TokenStatsResult holds aggregated statistics for a token
type TokenStatsResult struct {
	TotalTransfers  int64
	UniqueFromAddrs int64
	UniqueToAddrs   int64
	TotalVolume     string
	Transfers24h    int64
	Volume24h       string
	Transfers7d     int64
	Volume7d        string
	FirstTransferAt *time.Time
	LastTransferAt  *time.Time
}

// HolderBalance represents an address and its token balance
type HolderBalance struct {
	Address string
	Balance string // big number as string to preserve precision
	Rank    int
}

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

	// GetTokenStats returns aggregated transfer statistics for a token
	GetTokenStats(ctx context.Context, tokenAddress string) (*TokenStatsResult, error)

	// GetTopHolders returns top token holders sorted by balance
	GetTopHolders(ctx context.Context, tokenAddress string, limit int) ([]HolderBalance, error)

	// GetHolderBalance returns balance for a specific holder
	GetHolderBalance(ctx context.Context, tokenAddress, holderAddress string) (*HolderBalance, error)
}
