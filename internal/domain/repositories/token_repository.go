package repositories

import (
	"context"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// TokenRepository defines the interface for token data operations
type TokenRepository interface {
	// GetByAddress retrieves a token by its address
	GetByAddress(ctx context.Context, address string) (*entities.Token, error)

	// GetAll retrieves all tokens
	GetAll(ctx context.Context) ([]entities.Token, error)

	// GetAllPaginated retrieves tokens with pagination and sorting
	GetAllPaginated(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]*entities.Token, int64, error)

	// Count returns the total number of tokens
	Count(ctx context.Context) (int64, error)

	// Upsert creates or updates a token
	Upsert(ctx context.Context, token *entities.Token) error

	// UpdateStats updates token statistics
	UpdateStats(ctx context.Context, address string, transferCount int64, lastBlock int64) error
}
