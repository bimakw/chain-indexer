package repositories

import (
	"context"
	"time"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

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

type HolderBalance struct {
	Address string
	Balance string // big number as string to preserve precision
	Rank    int
}

type TransferRepository interface {
	GetByFilter(ctx context.Context, filter entities.TransferFilter) ([]entities.Transfer, error)

	GetCount(ctx context.Context, filter entities.TransferFilter) (int64, error)

	BatchInsert(ctx context.Context, transfers []entities.Transfer) error

	GetLatestBlock(ctx context.Context, tokenAddress string) (int64, error)

	GetTokenStats(ctx context.Context, tokenAddress string) (*TokenStatsResult, error)

	GetTopHolders(ctx context.Context, tokenAddress string, limit int) ([]HolderBalance, error)

	GetHolderBalance(ctx context.Context, tokenAddress, holderAddress string) (*HolderBalance, error)

	GetHolderCount(ctx context.Context, tokenAddress string) (int64, error)

	GetTopHoldersWithOffset(ctx context.Context, tokenAddress string, limit, offset int) ([]HolderBalance, error)
}
