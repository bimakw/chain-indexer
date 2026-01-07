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

// PortfolioService provides business logic for wallet portfolios
type PortfolioService struct {
	portfolioRepo repositories.PortfolioRepository
	cache         *cache.RedisCache
	logger        *zap.Logger
}

// NewPortfolioService creates a new portfolio service
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

// TokenHoldingDTO is the API representation of a token holding
type TokenHoldingDTO struct {
	TokenAddress     string `json:"token_address"`
	TokenName        string `json:"token_name"`
	TokenSymbol      string `json:"token_symbol"`
	Decimals         int    `json:"decimals"`
	Balance          string `json:"balance"`           // Raw wei
	BalanceFormatted string `json:"balance_formatted"` // Human readable
}

// PortfolioSummary contains summary information for a portfolio
type PortfolioSummary struct {
	TotalTokens       int   `json:"total_tokens"`
	TotalTransfersIn  int64 `json:"total_transfers_in"`
	TotalTransfersOut int64 `json:"total_transfers_out"`
}

// PortfolioDTO is the API representation of a wallet portfolio
type PortfolioDTO struct {
	WalletAddress string            `json:"wallet_address"`
	Holdings      []TokenHoldingDTO `json:"holdings"`
	Summary       PortfolioSummary  `json:"summary"`
	UpdatedAt     string            `json:"updated_at"`
}

// PortfolioResponse wraps portfolio data for API response
type PortfolioResponse struct {
	Data PortfolioDTO `json:"data"`
}

// TokenHoldingResponse wraps single token holding for API response
type TokenHoldingResponse struct {
	Data TokenHoldingDTO `json:"data"`
}

// WalletSummaryDTO is the API representation of wallet summary
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

// WalletSummaryResponse wraps wallet summary for API response
type WalletSummaryResponse struct {
	Data WalletSummaryDTO `json:"data"`
}

// GetPortfolio retrieves complete portfolio for a wallet address
func (s *PortfolioService) GetPortfolio(ctx context.Context, walletAddress string) (*PortfolioResponse, error) {
	walletAddress = strings.ToLower(walletAddress)

	// Generate cache key
	cacheKey := fmt.Sprintf("portfolio:%s", walletAddress)

	// Try cache first
	var cached PortfolioResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	// Get holdings from database
	holdings, err := s.portfolioRepo.GetWalletHoldings(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet holdings: %w", err)
	}

	// Get transfer summary for the wallet
	summary, err := s.portfolioRepo.GetWalletTransferSummary(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet summary: %w", err)
	}

	// Build response
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

// GetPortfolioByToken retrieves holding for specific token in a wallet
func (s *PortfolioService) GetPortfolioByToken(ctx context.Context, walletAddress, tokenAddress string) (*TokenHoldingResponse, error) {
	walletAddress = strings.ToLower(walletAddress)
	tokenAddress = strings.ToLower(tokenAddress)

	// Generate cache key
	cacheKey := fmt.Sprintf("portfolio:%s:%s", walletAddress, tokenAddress)

	// Try cache first
	var cached TokenHoldingResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	// Get holding from database
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

// GetWalletSummary retrieves transfer summary for a wallet
func (s *PortfolioService) GetWalletSummary(ctx context.Context, walletAddress string) (*WalletSummaryResponse, error) {
	walletAddress = strings.ToLower(walletAddress)

	// Generate cache key
	cacheKey := fmt.Sprintf("wallet_summary:%s", walletAddress)

	// Try cache first
	var cached WalletSummaryResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	// Get summary from database
	summary, err := s.portfolioRepo.GetWalletTransferSummary(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet summary: %w", err)
	}

	// Format timestamps
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
