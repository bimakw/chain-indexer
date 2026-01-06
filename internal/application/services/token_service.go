package services

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/infrastructure/cache"
)

// TokenService provides business logic for token queries
type TokenService struct {
	tokenRepo repositories.TokenRepository
	cache     *cache.RedisCache
	logger    *zap.Logger
}

// NewTokenService creates a new token service
func NewTokenService(
	tokenRepo repositories.TokenRepository,
	cache *cache.RedisCache,
	logger *zap.Logger,
) *TokenService {
	return &TokenService{
		tokenRepo: tokenRepo,
		cache:     cache,
		logger:    logger,
	}
}

// TokenDTO is the API representation of a token
type TokenDTO struct {
	Address               string `json:"address"`
	Name                  string `json:"name"`
	Symbol                string `json:"symbol"`
	Decimals              int    `json:"decimals"`
	TotalIndexedTransfers int64  `json:"total_indexed_transfers"`
	FirstSeenBlock        *int64 `json:"first_seen_block"`
	LastSeenBlock         *int64 `json:"last_seen_block"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

// TokenListResponse is the API response for token list queries
type TokenListResponse struct {
	Data       []TokenDTO         `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// TokenResponse is the API response for single token queries
type TokenResponse struct {
	Data TokenDTO `json:"data"`
}

// PaginationResponse contains pagination metadata
type PaginationResponse struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// GetAllTokens retrieves all tokens with pagination and sorting
func (s *TokenService) GetAllTokens(ctx context.Context, limit, offset int, sortBy, sortOrder string) (*TokenListResponse, error) {
	// Generate cache key
	cacheKey := fmt.Sprintf("tokens:list:%d:%d:%s:%s", limit, offset, sortBy, sortOrder)

	// Try cache first
	var cached TokenListResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	// Query database
	tokens, total, err := s.tokenRepo.GetAllPaginated(ctx, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	// Convert to DTOs
	dtos := make([]TokenDTO, len(tokens))
	for i, t := range tokens {
		dtos[i] = tokenToDTO(t)
	}

	response := &TokenListResponse{
		Data: dtos,
		Pagination: PaginationResponse{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	}

	// Cache the response
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, response); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}

// GetByAddress retrieves a single token by address
func (s *TokenService) GetByAddress(ctx context.Context, address string) (*TokenResponse, error) {
	address = strings.ToLower(address)

	// Generate cache key
	cacheKey := fmt.Sprintf("tokens:%s", address)

	// Try cache first
	var cached TokenResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	// Query database
	token, err := s.tokenRepo.GetByAddress(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return nil, nil
	}

	response := &TokenResponse{
		Data: tokenToDTO(token),
	}

	// Cache the response
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, response); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}

// tokenToDTO converts a token entity to a DTO
func tokenToDTO(t *entities.Token) TokenDTO {
	return TokenDTO{
		Address:               t.Address,
		Name:                  t.Name,
		Symbol:                t.Symbol,
		Decimals:              t.Decimals,
		TotalIndexedTransfers: t.TotalIndexedTransfers,
		FirstSeenBlock:        t.FirstSeenBlock,
		LastSeenBlock:         t.LastSeenBlock,
		CreatedAt:             t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:             t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
