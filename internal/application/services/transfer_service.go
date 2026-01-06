package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/infrastructure/cache"
)

// TransferService provides business logic for transfer queries
type TransferService struct {
	transferRepo repositories.TransferRepository
	tokenRepo    repositories.TokenRepository
	cache        *cache.RedisCache
	logger       *zap.Logger
}

// NewTransferService creates a new transfer service
func NewTransferService(
	transferRepo repositories.TransferRepository,
	tokenRepo repositories.TokenRepository,
	cache *cache.RedisCache,
	logger *zap.Logger,
) *TransferService {
	return &TransferService{
		transferRepo: transferRepo,
		tokenRepo:    tokenRepo,
		cache:        cache,
		logger:       logger,
	}
}

// TransferResponse is the API response for transfer queries
type TransferResponse struct {
	Transfers  []TransferDTO `json:"transfers"`
	Total      int64         `json:"total"`
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
	HasMore    bool          `json:"has_more"`
}

// TransferDTO is the API representation of a transfer
type TransferDTO struct {
	TxHash         string `json:"tx_hash"`
	LogIndex       int    `json:"log_index"`
	BlockNumber    int64  `json:"block_number"`
	BlockTimestamp string `json:"block_timestamp"`
	TokenAddress   string `json:"token_address"`
	FromAddress    string `json:"from_address"`
	ToAddress      string `json:"to_address"`
	Value          string `json:"value"`
}

// GetTransfers retrieves transfers based on filter
func (s *TransferService) GetTransfers(ctx context.Context, filter entities.TransferFilter) (*TransferResponse, error) {
	// Generate cache key
	cacheKey := s.generateCacheKey(filter)

	// Try cache first
	var cached TransferResponse
	if s.cache != nil {
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			s.logger.Debug("Cache hit", zap.String("key", cacheKey))
			return &cached, nil
		}
	}

	// Query database
	transfers, err := s.transferRepo.GetByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfers: %w", err)
	}

	total, err := s.transferRepo.GetCount(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer count: %w", err)
	}

	// Convert to DTOs
	dtos := make([]TransferDTO, len(transfers))
	for i, t := range transfers {
		dtos[i] = TransferDTO{
			TxHash:         t.TxHash,
			LogIndex:       t.LogIndex,
			BlockNumber:    t.BlockNumber,
			BlockTimestamp: t.BlockTimestamp.Format("2006-01-02T15:04:05Z"),
			TokenAddress:   t.TokenAddress,
			FromAddress:    t.FromAddress,
			ToAddress:      t.ToAddress,
			Value:          t.ValueString,
		}
	}

	response := &TransferResponse{
		Transfers: dtos,
		Total:     total,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
		HasMore:   int64(filter.Offset+len(transfers)) < total,
	}

	// Cache the response
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, response); err != nil {
			s.logger.Warn("Failed to cache response", zap.Error(err))
		}
	}

	return response, nil
}

// GetTransfersByAddress retrieves transfers involving a specific address
func (s *TransferService) GetTransfersByAddress(ctx context.Context, address string, limit, offset int) (*TransferResponse, error) {
	address = strings.ToLower(address)
	filter := entities.TransferFilter{
		Address: &address,
		Limit:   limit,
		Offset:  offset,
	}
	return s.GetTransfers(ctx, filter)
}

// GetTransfersByToken retrieves transfers for a specific token
func (s *TransferService) GetTransfersByToken(ctx context.Context, tokenAddress string, limit, offset int) (*TransferResponse, error) {
	tokenAddress = strings.ToLower(tokenAddress)
	filter := entities.TransferFilter{
		TokenAddress: &tokenAddress,
		Limit:        limit,
		Offset:       offset,
	}
	return s.GetTransfers(ctx, filter)
}

// generateCacheKey generates a unique cache key for the filter
func (s *TransferService) generateCacheKey(filter entities.TransferFilter) string {
	var parts []string

	if filter.TokenAddress != nil {
		parts = append(parts, "token:"+*filter.TokenAddress)
	}
	if filter.FromAddress != nil {
		parts = append(parts, "from:"+*filter.FromAddress)
	}
	if filter.ToAddress != nil {
		parts = append(parts, "to:"+*filter.ToAddress)
	}
	if filter.Address != nil {
		parts = append(parts, "addr:"+*filter.Address)
	}
	if filter.FromBlock != nil {
		parts = append(parts, fmt.Sprintf("fb:%d", *filter.FromBlock))
	}
	if filter.ToBlock != nil {
		parts = append(parts, fmt.Sprintf("tb:%d", *filter.ToBlock))
	}

	parts = append(parts, fmt.Sprintf("l:%d:o:%d", filter.Limit, filter.Offset))

	key := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(key))
	return "transfers:" + hex.EncodeToString(hash[:8])
}
