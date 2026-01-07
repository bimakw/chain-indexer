package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/application/services"
	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func setupPortfolioHandler(mockRepo *testutil.MockPortfolioRepository) *PortfolioHandler {
	logger := zap.NewNop()
	portfolioService := services.NewPortfolioService(mockRepo, nil, logger)
	return NewPortfolioHandler(portfolioService, logger)
}

func TestPortfolioHandler_GetPortfolio(t *testing.T) {
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
			}, nil
		}
		mockRepo.GetWalletTransferSummaryFunc = func(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
			return &repositories.WalletTransferSummary{
				TotalTransfersIn:  100,
				TotalTransfersOut: 50,
				TotalVolumeIn:     "5000000000",
				TotalVolumeOut:    "2500000000",
				UniqueTokens:      1,
			}, nil
		}

		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/portfolio", handler.GetPortfolio)

		req := httptest.NewRequest("GET", "/wallets/0x1234567890123456789012345678901234567890/portfolio", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response services.PortfolioResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(response.Data.Holdings) != 1 {
			t.Errorf("expected 1 holding, got %d", len(response.Data.Holdings))
		}
	})

	t.Run("returns error for invalid address", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/portfolio", handler.GetPortfolio)

		req := httptest.NewRequest("GET", "/wallets/invalid-address/portfolio", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns error when service fails", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletHoldingsFunc = func(ctx context.Context, walletAddress string) ([]entities.TokenHolding, error) {
			return nil, errors.New("database error")
		}

		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/portfolio", handler.GetPortfolio)

		req := httptest.NewRequest("GET", "/wallets/0x1234567890123456789012345678901234567890/portfolio", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}

func TestPortfolioHandler_GetTokenHolding(t *testing.T) {
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

		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/portfolio/tokens/{tokenAddress}", handler.GetTokenHolding)

		req := httptest.NewRequest("GET", "/wallets/0x1234567890123456789012345678901234567890/portfolio/tokens/0xdac17f958d2ee523a2206206994597c13d831ec7", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response services.TokenHoldingResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Data.TokenSymbol != "USDT" {
			t.Errorf("expected TokenSymbol USDT, got %s", response.Data.TokenSymbol)
		}
	})

	t.Run("returns error for invalid wallet address", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/portfolio/tokens/{tokenAddress}", handler.GetTokenHolding)

		req := httptest.NewRequest("GET", "/wallets/invalid/portfolio/tokens/0xdac17f958d2ee523a2206206994597c13d831ec7", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns error for invalid token address", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/portfolio/tokens/{tokenAddress}", handler.GetTokenHolding)

		req := httptest.NewRequest("GET", "/wallets/0x1234567890123456789012345678901234567890/portfolio/tokens/invalid", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns not found when token not found", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletHoldingByTokenFunc = func(ctx context.Context, walletAddress, tokenAddress string) (*entities.TokenHolding, error) {
			return nil, nil
		}

		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/portfolio/tokens/{tokenAddress}", handler.GetTokenHolding)

		req := httptest.NewRequest("GET", "/wallets/0x1234567890123456789012345678901234567890/portfolio/tokens/0xdac17f958d2ee523a2206206994597c13d831ec7", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestPortfolioHandler_GetWalletSummary(t *testing.T) {
	t.Run("returns wallet summary successfully", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletTransferSummaryFunc = func(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
			return &repositories.WalletTransferSummary{
				TotalTransfersIn:  100,
				TotalTransfersOut: 50,
				TotalVolumeIn:     "5000000000",
				TotalVolumeOut:    "2500000000",
				UniqueTokens:      5,
			}, nil
		}

		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/summary", handler.GetWalletSummary)

		req := httptest.NewRequest("GET", "/wallets/0x1234567890123456789012345678901234567890/summary", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response services.WalletSummaryResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Data.TotalTransfersIn != 100 {
			t.Errorf("expected TotalTransfersIn 100, got %d", response.Data.TotalTransfersIn)
		}

		if response.Data.UniqueTokens != 5 {
			t.Errorf("expected UniqueTokens 5, got %d", response.Data.UniqueTokens)
		}
	})

	t.Run("returns error for invalid address", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/summary", handler.GetWalletSummary)

		req := httptest.NewRequest("GET", "/wallets/invalid/summary", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns error when service fails", func(t *testing.T) {
		mockRepo := testutil.NewMockPortfolioRepository()
		mockRepo.GetWalletTransferSummaryFunc = func(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
			return nil, errors.New("database error")
		}

		handler := setupPortfolioHandler(mockRepo)

		r := chi.NewRouter()
		r.Get("/wallets/{address}/summary", handler.GetWalletSummary)

		req := httptest.NewRequest("GET", "/wallets/0x1234567890123456789012345678901234567890/summary", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}
