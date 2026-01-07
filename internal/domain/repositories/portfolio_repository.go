package repositories

import (
	"context"
	"time"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// WalletTransferSummary holds transfer statistics for a wallet
type WalletTransferSummary struct {
	TotalTransfersIn  int64
	TotalTransfersOut int64
	TotalVolumeIn     string
	TotalVolumeOut    string
	UniqueTokens      int64
	FirstTransferAt   *time.Time
	LastTransferAt    *time.Time
}

// PortfolioRepository defines interface for portfolio data operations
type PortfolioRepository interface {
	// GetWalletHoldings retrieves all token holdings for a wallet
	// Calculates balance from transfers: SUM(received) - SUM(sent)
	GetWalletHoldings(ctx context.Context, walletAddress string) ([]entities.TokenHolding, error)

	// GetWalletHoldingByToken retrieves holding for specific token
	GetWalletHoldingByToken(ctx context.Context, walletAddress, tokenAddress string) (*entities.TokenHolding, error)

	// GetWalletTokenCount returns count of tokens held by wallet
	GetWalletTokenCount(ctx context.Context, walletAddress string) (int64, error)

	// GetWalletTransferSummary returns transfer stats for a wallet
	GetWalletTransferSummary(ctx context.Context, walletAddress string) (*WalletTransferSummary, error)
}
