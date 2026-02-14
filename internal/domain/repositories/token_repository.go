package repositories

import (
	"context"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

type TokenRepository interface {
	GetByAddress(ctx context.Context, address string) (*entities.Token, error)

	GetAll(ctx context.Context) ([]entities.Token, error)

	GetAllPaginated(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]*entities.Token, int64, error)

	Count(ctx context.Context) (int64, error)

	Upsert(ctx context.Context, token *entities.Token) error

	UpdateStats(ctx context.Context, address string, transferCount int64, lastBlock int64) error
}
