package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/infrastructure/cache"
)

type PortfolioService struct {
	portfolioRepo repositories.PortfolioRepository
	cache         *cache.RedisCache
	logger        *zap.Logger
}

func NewPortfolioService(
	portfolioRepo repositories.PortfolioRepository,
	cache *cache.RedisCache,
	logger *zap.Logger,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo: portfolioRepo,
		cache:         cache,
		logger:        logger,
	}
}

type TokenHoldingDTO struct {
	TokenAddress     string `json:"token_address"`
	TokenName        string `json:"token_name"`
	TokenSymbol      string `json:"token_symbol"`
	Decimals         int    `json:"decimals"`
	Balance          string `json:"balance"`           // Raw wei
	BalanceFormatted string `json:"balance_formatted"` // Human readable
}

type PortfolioSummary struct {
	TotalTokens       int   `json:"total_tokens"`
	TotalTransfersIn  int64 `json:"total_transfers_in"`
	TotalTransfersOut int64 `json:"total_transfers_out"`
}

type PortfolioDTO struct {
	WalletAddress string            `json:"wallet_address"`
	Holdings      []TokenHoldingDTO `json:"holdings"`
	Summary       PortfolioSummary  `json:"summary"`
	UpdatedAt     string            `json:"updated_at"`
}

type PortfolioResponse struct {
	Data PortfolioDTO `json:"data"`
}

type TokenHoldingResponse struct {
	Data TokenHoldingDTO `json:"data"`
}

type WalletSummaryDTO struct {
	WalletAddress     string  `json:"wallet_address"`
	TotalTransfersIn  int64   `json:"total_transfers_in"`
	TotalTransfersOut int64   `json:"total_transfers_out"`
	TotalVolumeIn     string  `json:"total_volume_in"`
	TotalVolumeOut    string  `json:"total_volume_out"`
	UniqueTokens      int64   `json:"unique_tokens"`
	FirstTransferAt   *string `json:"first_transfer_at,omitempty"`
	LastTransferAt    *string `json:"last_transfer_at,omitempty"`
}

type WalletSummaryResponse struct {
	Data WalletSummaryDTO `json:"data"`
}

func (s *PortfolioService) GetPortfolio(ctx context.Context, walletAddress string) (*PortfolioResponse, error) {
	walletAddress = strings.ToLower(walletAddress)

	cacheKey := fmt.Sprintf("portfolio:%s", walletAddress)

	var cached PortfolioResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	holdings, err := s.portfolioRepo.GetWalletHoldings(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet holdings: %w", err)
	}

	summary, err := s.portfolioRepo.GetWalletTransferSummary(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet summary: %w", err)
	}

	holdingsDTO := make([]TokenHoldingDTO, len(holdings))
	for i, h := range holdings {
		holdingsDTO[i] = TokenHoldingDTO{
			TokenAddress:     h.TokenAddress,
			TokenName:        h.TokenName,
			TokenSymbol:      h.TokenSymbol,
			Decimals:         h.Decimals,
			Balance:          h.BalanceStr,
			BalanceFormatted: h.BalanceHuman,
		}
	}

	response := &PortfolioResponse{
		Data: PortfolioDTO{
			WalletAddress: walletAddress,
			Holdings:      holdingsDTO,
			Summary: PortfolioSummary{
				TotalTokens:       len(holdings),
				TotalTransfersIn:  summary.TotalTransfersIn,
				TotalTransfersOut: summary.TotalTransfersOut,
			},
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	// Cache the response (2 minutes TTL for portfolio)
	if s.cache != nil {
		if err := s.cache.SetWithTTL(ctx, cacheKey, response, 2*time.Minute); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}

func (s *PortfolioService) GetPortfolioByToken(ctx context.Context, walletAddress, tokenAddress string) (*TokenHoldingResponse, error) {
	walletAddress = strings.ToLower(walletAddress)
	tokenAddress = strings.ToLower(tokenAddress)

	cacheKey := fmt.Sprintf("portfolio:%s:%s", walletAddress, tokenAddress)

	var cached TokenHoldingResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	holding, err := s.portfolioRepo.GetWalletHoldingByToken(ctx, walletAddress, tokenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet holding by token: %w", err)
	}

	if holding == nil {
		return nil, nil
	}

	response := &TokenHoldingResponse{
		Data: TokenHoldingDTO{
			TokenAddress:     holding.TokenAddress,
			TokenName:        holding.TokenName,
			TokenSymbol:      holding.TokenSymbol,
			Decimals:         holding.Decimals,
			Balance:          holding.BalanceStr,
			BalanceFormatted: holding.BalanceHuman,
		},
	}

	// Cache the response (2 minutes TTL)
	if s.cache != nil {
		if err := s.cache.SetWithTTL(ctx, cacheKey, response, 2*time.Minute); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}

func (s *PortfolioService) GetWalletSummary(ctx context.Context, walletAddress string) (*WalletSummaryResponse, error) {
	walletAddress = strings.ToLower(walletAddress)

	cacheKey := fmt.Sprintf("wallet_summary:%s", walletAddress)

	var cached WalletSummaryResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	summary, err := s.portfolioRepo.GetWalletTransferSummary(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet summary: %w", err)
	}

	var firstTransferAt, lastTransferAt *string
	if summary.FirstTransferAt != nil {
		t := summary.FirstTransferAt.Format(time.RFC3339)
		firstTransferAt = &t
	}
	if summary.LastTransferAt != nil {
		t := summary.LastTransferAt.Format(time.RFC3339)
		lastTransferAt = &t
	}

	response := &WalletSummaryResponse{
		Data: WalletSummaryDTO{
			WalletAddress:     walletAddress,
			TotalTransfersIn:  summary.TotalTransfersIn,
			TotalTransfersOut: summary.TotalTransfersOut,
			TotalVolumeIn:     summary.TotalVolumeIn,
			TotalVolumeOut:    summary.TotalVolumeOut,
			UniqueTokens:      summary.UniqueTokens,
			FirstTransferAt:   firstTransferAt,
			LastTransferAt:    lastTransferAt,
		},
	}

	// Cache the response (5 minutes TTL for summary)
	if s.cache != nil {
		if err := s.cache.SetWithTTL(ctx, cacheKey, response, 5*time.Minute); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}
