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

type StatsService struct {
	transferRepo repositories.TransferRepository
	tokenRepo    repositories.TokenRepository
	cache        *cache.RedisCache
	logger       *zap.Logger
}

func NewStatsService(
	transferRepo repositories.TransferRepository,
	tokenRepo repositories.TokenRepository,
	cache *cache.RedisCache,
	logger *zap.Logger,
) *StatsService {
	return &StatsService{
		transferRepo: transferRepo,
		tokenRepo:    tokenRepo,
		cache:        cache,
		logger:       logger,
	}
}

type TokenStats struct {
	TokenAddress        string `json:"token_address"`
	TotalTransfers      int64  `json:"total_transfers"`
	UniqueFromAddresses int64  `json:"unique_from_addresses"`
	UniqueToAddresses   int64  `json:"unique_to_addresses"`
	TotalVolume         string `json:"total_volume"`
	Transfers24h        int64  `json:"transfers_24h"`
	Volume24h           string `json:"volume_24h"`
	Transfers7d         int64  `json:"transfers_7d"`
	Volume7d            string `json:"volume_7d"`
	FirstTransferAt     string `json:"first_transfer_at"`
	LastTransferAt      string `json:"last_transfer_at"`
}

type HolderCountResponse struct {
	Data HolderCountDTO `json:"data"`
}

type HolderCountDTO struct {
	TokenAddress string `json:"token_address"`
	HolderCount  int64  `json:"holder_count"`
}

type TokenStatsResponse struct {
	Data TokenStats `json:"data"`
}

func (s *StatsService) GetTokenStats(ctx context.Context, tokenAddress string) (*TokenStatsResponse, error) {
	tokenAddress = strings.ToLower(tokenAddress)

	cacheKey := fmt.Sprintf("stats:%s", tokenAddress)

	var cached TokenStatsResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	token, err := s.tokenRepo.GetByAddress(ctx, tokenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to check token: %w", err)
	}
	if token == nil {
		return nil, nil // Token not found
	}

	stats, err := s.transferRepo.GetTokenStats(ctx, tokenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get token stats: %w", err)
	}

	response := &TokenStatsResponse{
		Data: TokenStats{
			TokenAddress:        tokenAddress,
			TotalTransfers:      stats.TotalTransfers,
			UniqueFromAddresses: stats.UniqueFromAddrs,
			UniqueToAddresses:   stats.UniqueToAddrs,
			TotalVolume:         stats.TotalVolume,
			Transfers24h:        stats.Transfers24h,
			Volume24h:           stats.Volume24h,
			Transfers7d:         stats.Transfers7d,
			Volume7d:            stats.Volume7d,
			FirstTransferAt:     "",
			LastTransferAt:      "",
		},
	}

	if stats.FirstTransferAt != nil {
		response.Data.FirstTransferAt = stats.FirstTransferAt.Format("2006-01-02T15:04:05Z")
	}
	if stats.LastTransferAt != nil {
		response.Data.LastTransferAt = stats.LastTransferAt.Format("2006-01-02T15:04:05Z")
	}

	// Cache the response with shorter TTL (60 seconds for stats)
	if s.cache != nil {
		if err := s.cache.SetWithTTL(ctx, cacheKey, response, 60*time.Second); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}

func (s *StatsService) GetHolderCount(ctx context.Context, tokenAddress string) (*HolderCountResponse, error) {
	tokenAddress = strings.ToLower(tokenAddress)

	cacheKey := fmt.Sprintf("holder_count:%s", tokenAddress)

	var cached HolderCountResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	token, err := s.tokenRepo.GetByAddress(ctx, tokenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to check token: %w", err)
	}
	if token == nil {
		return nil, nil // Token not found
	}

	count, err := s.transferRepo.GetHolderCount(ctx, tokenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get holder count: %w", err)
	}

	response := &HolderCountResponse{
		Data: HolderCountDTO{
			TokenAddress: tokenAddress,
			HolderCount:  count,
		},
	}

	// Cache the response with 5 minutes TTL (holder count changes slowly)
	if s.cache != nil {
		if err := s.cache.SetWithTTL(ctx, cacheKey, response, 300*time.Second); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}
