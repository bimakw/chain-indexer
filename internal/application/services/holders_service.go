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

type HoldersService struct {
	transferRepo repositories.TransferRepository
	tokenRepo    repositories.TokenRepository
	cache        *cache.RedisCache
	logger       *zap.Logger
}

func NewHoldersService(
	transferRepo repositories.TransferRepository,
	tokenRepo repositories.TokenRepository,
	cache *cache.RedisCache,
	logger *zap.Logger,
) *HoldersService {
	return &HoldersService{
		transferRepo: transferRepo,
		tokenRepo:    tokenRepo,
		cache:        cache,
		logger:       logger,
	}
}

type HolderDTO struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
	Rank    int    `json:"rank"`
}

type PaginationMetadata struct {
	Total   int64 `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	HasMore bool  `json:"has_more"`
}

type TopHoldersResponse struct {
	Data       []HolderDTO        `json:"data"`
	Pagination PaginationMetadata `json:"pagination"`
}

type HolderBalanceResponse struct {
	Data HolderDTO `json:"data"`
}

func (s *HoldersService) GetTopHolders(ctx context.Context, tokenAddress string, limit, offset int) (*TopHoldersResponse, error) {
	tokenAddress = strings.ToLower(tokenAddress)

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	if offset < 0 {
		offset = 0
	}

	cacheKey := fmt.Sprintf("holders:%s:%d:%d", tokenAddress, limit, offset)

	var cached TopHoldersResponse
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

	// Get total holder count (with separate cache key)
	var total int64
	countCacheKey := fmt.Sprintf("holders_count:%s", tokenAddress)
	if s.cache != nil {
		if cacheErr := s.cache.Get(ctx, countCacheKey, &total); cacheErr != nil {
			var countErr error
			total, countErr = s.transferRepo.GetHolderCount(ctx, tokenAddress)
			if countErr != nil {
				return nil, fmt.Errorf("failed to get holder count: %w", countErr)
			}
			// Cache the count with 5 min TTL
			if setErr := s.cache.SetWithTTL(ctx, countCacheKey, total, 5*time.Minute); setErr != nil {
				s.logger.Warn("Failed to cache holder count", zap.Error(setErr))
			}
		}
	} else {
		total, err = s.transferRepo.GetHolderCount(ctx, tokenAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to get holder count: %w", err)
		}
	}

	holders, err := s.transferRepo.GetTopHoldersWithOffset(ctx, tokenAddress, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get top holders: %w", err)
	}

	data := make([]HolderDTO, len(holders))
	for i, h := range holders {
		data[i] = HolderDTO{
			Address: h.Address,
			Balance: h.Balance,
			Rank:    h.Rank,
		}
	}

	hasMore := int64(offset+limit) < total

	response := &TopHoldersResponse{
		Data: data,
		Pagination: PaginationMetadata{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasMore: hasMore,
		},
	}

	// Cache the response (5 minutes TTL for holders)
	if s.cache != nil {
		if err := s.cache.SetWithTTL(ctx, cacheKey, response, 5*time.Minute); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}

func (s *HoldersService) GetHolderBalance(ctx context.Context, tokenAddress, holderAddress string) (*HolderBalanceResponse, error) {
	tokenAddress = strings.ToLower(tokenAddress)
	holderAddress = strings.ToLower(holderAddress)

	cacheKey := fmt.Sprintf("holder:%s:%s", tokenAddress, holderAddress)

	var cached HolderBalanceResponse
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

	holder, err := s.transferRepo.GetHolderBalance(ctx, tokenAddress, holderAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get holder balance: %w", err)
	}

	response := &HolderBalanceResponse{
		Data: HolderDTO{
			Address: holder.Address,
			Balance: holder.Balance,
			Rank:    holder.Rank,
		},
	}

	// Cache the response (1 minute TTL for individual holder)
	if s.cache != nil {
		if err := s.cache.SetWithTTL(ctx, cacheKey, response, time.Minute); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}
