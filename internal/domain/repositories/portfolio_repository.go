package repositories

import (
	"context"
	"time"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

type WalletTransferSummary struct {
	TotalTransfersIn  int64
	TotalTransfersOut int64
	TotalVolumeIn     string
	TotalVolumeOut    string
	UniqueTokens      int64
	FirstTransferAt   *time.Time
	LastTransferAt    *time.Time
}

type PortfolioRepository interface {
	// Calculates balance from transfers: SUM(received) - SUM(sent)
	GetWalletHoldings(ctx context.Context, walletAddress string) ([]entities.TokenHolding, error)

	GetWalletHoldingByToken(ctx context.Context, walletAddress, tokenAddress string) (*entities.TokenHolding, error)

	GetWalletTokenCount(ctx context.Context, walletAddress string) (int64, error)

	GetWalletTransferSummary(ctx context.Context, walletAddress string) (*WalletTransferSummary, error)
}
