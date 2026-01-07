package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func TestPortfolioService_GetPortfolio(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("returns portfolio successfully", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletHoldingsFunc = func(ctx context.Context, walletAddress string) ([]entities.TokenHolding, error) {
			return []entities.TokenHolding{
				{
					TokenAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					TokenName:    "Tether USD",
					TokenSymbol:  "USDT",
					Decimals:     6,
					BalanceStr:   "1000000000",
					BalanceHuman: "1000.000000",
				},
				{
					TokenAddress: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					TokenName:    "USD Coin",
					TokenSymbol:  "USDC",
					Decimals:     6,
					BalanceStr:   "500000000",
					BalanceHuman: "500.000000",
				},
			}, nil
		}
		mockRepo.GetWalletTransferSummaryFunc = func(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
			return &repositories.WalletTransferSummary{
				TotalTransfersIn:  150,
				TotalTransfersOut: 75,
				TotalVolumeIn:     "5000000000000",
				TotalVolumeOut:    "2500000000000",
				UniqueTokens:      2,
			}, nil
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		result, err := service.GetPortfolio(ctx, "0x1234567890123456789012345678901234567890")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("expected result, got nil")
		}

		if len(result.Data.Holdings) != 2 {
			t.Errorf("expected 2 holdings, got %d", len(result.Data.Holdings))
		}

		if result.Data.Summary.TotalTokens != 2 {
			t.Errorf("expected TotalTokens 2, got %d", result.Data.Summary.TotalTokens)
		}

		if result.Data.Summary.TotalTransfersIn != 150 {
			t.Errorf("expected TotalTransfersIn 150, got %d", result.Data.Summary.TotalTransfersIn)
		}

		if result.Data.Summary.TotalTransfersOut != 75 {
			t.Errorf("expected TotalTransfersOut 75, got %d", result.Data.Summary.TotalTransfersOut)
		}
	})

	t.Run("returns error when holdings fail", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletHoldingsFunc = func(ctx context.Context, walletAddress string) ([]entities.TokenHolding, error) {
			return nil, errors.New("database error")
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		_, err := service.GetPortfolio(ctx, "0x1234567890123456789012345678901234567890")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when summary fails", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletHoldingsFunc = func(ctx context.Context, walletAddress string) ([]entities.TokenHolding, error) {
			return []entities.TokenHolding{}, nil
		}
		mockRepo.GetWalletTransferSummaryFunc = func(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
			return nil, errors.New("database error")
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		_, err := service.GetPortfolio(ctx, "0x1234567890123456789012345678901234567890")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("lowercases wallet address", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		var capturedAddress string
		mockRepo.GetWalletHoldingsFunc = func(ctx context.Context, walletAddress string) ([]entities.TokenHolding, error) {
			capturedAddress = walletAddress
			return []entities.TokenHolding{}, nil
		}
		mockRepo.GetWalletTransferSummaryFunc = func(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
			return &repositories.WalletTransferSummary{}, nil
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		_, err := service.GetPortfolio(ctx, "0xABCDEF1234567890123456789012345678901234")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedAddress != "0xabcdef1234567890123456789012345678901234" {
			t.Errorf("expected lowercased address, got %s", capturedAddress)
		}
	})
}

func TestPortfolioService_GetPortfolioByToken(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("returns token holding successfully", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletHoldingByTokenFunc = func(ctx context.Context, walletAddress, tokenAddress string) (*entities.TokenHolding, error) {
			return &entities.TokenHolding{
				TokenAddress: tokenAddress,
				TokenName:    "Tether USD",
				TokenSymbol:  "USDT",
				Decimals:     6,
				BalanceStr:   "1000000000",
				BalanceHuman: "1000.000000",
			}, nil
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		result, err := service.GetPortfolioByToken(
			ctx,
			"0x1234567890123456789012345678901234567890",
			"0xdac17f958d2ee523a2206206994597c13d831ec7",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("expected result, got nil")
		}

		if result.Data.TokenSymbol != "USDT" {
			t.Errorf("expected TokenSymbol USDT, got %s", result.Data.TokenSymbol)
		}
	})

	t.Run("returns nil when token not found", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletHoldingByTokenFunc = func(ctx context.Context, walletAddress, tokenAddress string) (*entities.TokenHolding, error) {
			return nil, nil
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		result, err := service.GetPortfolioByToken(
			ctx,
			"0x1234567890123456789012345678901234567890",
			"0xdac17f958d2ee523a2206206994597c13d831ec7",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != nil {
			t.Error("expected nil result for not found token")
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletHoldingByTokenFunc = func(ctx context.Context, walletAddress, tokenAddress string) (*entities.TokenHolding, error) {
			return nil, errors.New("database error")
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		_, err := service.GetPortfolioByToken(
			ctx,
			"0x1234567890123456789012345678901234567890",
			"0xdac17f958d2ee523a2206206994597c13d831ec7",
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPortfolioService_GetWalletSummary(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("returns summary successfully", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		firstTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		lastTime := time.Date(2026, 1, 7, 8, 45, 0, 0, time.UTC)
		mockRepo.GetWalletTransferSummaryFunc = func(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
			return &repositories.WalletTransferSummary{
				TotalTransfersIn:  150,
				TotalTransfersOut: 75,
				TotalVolumeIn:     "5000000000000",
				TotalVolumeOut:    "2500000000000",
				UniqueTokens:      5,
				FirstTransferAt:   &firstTime,
				LastTransferAt:    &lastTime,
			}, nil
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		result, err := service.GetWalletSummary(ctx, "0x1234567890123456789012345678901234567890")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("expected result, got nil")
		}

		if result.Data.TotalTransfersIn != 150 {
			t.Errorf("expected TotalTransfersIn 150, got %d", result.Data.TotalTransfersIn)
		}

		if result.Data.TotalTransfersOut != 75 {
			t.Errorf("expected TotalTransfersOut 75, got %d", result.Data.TotalTransfersOut)
		}

		if result.Data.UniqueTokens != 5 {
			t.Errorf("expected UniqueTokens 5, got %d", result.Data.UniqueTokens)
		}

		if result.Data.FirstTransferAt == nil {
			t.Error("expected FirstTransferAt to be set")
		}

		if result.Data.LastTransferAt == nil {
			t.Error("expected LastTransferAt to be set")
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletTransferSummaryFunc = func(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
			return nil, errors.New("database error")
		}

		service := NewPortfolioService(mockRepo, nil, logger)

		_, err := service.GetWalletSummary(ctx, "0x1234567890123456789012345678901234567890")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
