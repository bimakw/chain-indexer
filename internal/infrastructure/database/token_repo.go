package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
)

// Ensure TokenRepo implements TokenRepository
var _ repositories.TokenRepository = (*TokenRepo)(nil)

// TokenRepo implements TokenRepository using PostgreSQL
type TokenRepo struct {
	db *sqlx.DB
}

// NewTokenRepo creates a new token repository
func NewTokenRepo(db *sqlx.DB) *TokenRepo {
	return &TokenRepo{db: db}
}

// GetByAddress retrieves a token by its address
func (r *TokenRepo) GetByAddress(ctx context.Context, address string) (*entities.Token, error) {
	var token entities.Token
	query := `SELECT * FROM tokens WHERE address = $1`

	if err := r.db.GetContext(ctx, &token, query, address); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return &token, nil
}

// GetAll retrieves all tokens
func (r *TokenRepo) GetAll(ctx context.Context) ([]entities.Token, error) {
	var tokens []entities.Token
	query := `SELECT * FROM tokens ORDER BY symbol`

	if err := r.db.SelectContext(ctx, &tokens, query); err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	return tokens, nil
}

// Upsert creates or updates a token
func (r *TokenRepo) Upsert(ctx context.Context, token *entities.Token) error {
	query := `
		INSERT INTO tokens (address, name, symbol, decimals, first_seen_block)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (address) DO UPDATE SET
			name = EXCLUDED.name,
			symbol = EXCLUDED.symbol,
			decimals = EXCLUDED.decimals,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query,
		token.Address,
		token.Name,
		token.Symbol,
		token.Decimals,
		token.FirstSeenBlock,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert token: %w", err)
	}

	return nil
}

// UpdateStats updates token statistics
func (r *TokenRepo) UpdateStats(ctx context.Context, address string, transferCount int64, lastBlock int64) error {
	query := `
		UPDATE tokens SET
			total_indexed_transfers = total_indexed_transfers + $2,
			last_seen_block = GREATEST(COALESCE(last_seen_block, 0), $3),
			updated_at = NOW()
		WHERE address = $1
	`

	_, err := r.db.ExecContext(ctx, query, address, transferCount, lastBlock)
	if err != nil {
		return fmt.Errorf("failed to update token stats: %w", err)
	}

	return nil
}
