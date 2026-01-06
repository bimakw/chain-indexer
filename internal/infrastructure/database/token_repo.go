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

// validSortColumns defines allowed sort columns to prevent SQL injection
var validSortColumns = map[string]bool{
	"address":                 true,
	"name":                    true,
	"symbol":                  true,
	"decimals":                true,
	"total_indexed_transfers": true,
	"first_seen_block":        true,
	"last_seen_block":         true,
	"created_at":              true,
	"updated_at":              true,
}

// GetAllPaginated retrieves tokens with pagination and sorting
func (r *TokenRepo) GetAllPaginated(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]*entities.Token, int64, error) {
	// Validate sort column
	if !validSortColumns[sortBy] {
		sortBy = "total_indexed_transfers"
	}

	// Validate sort order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM tokens`
	if err := r.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, 0, fmt.Errorf("failed to count tokens: %w", err)
	}

	// Get paginated tokens
	query := fmt.Sprintf(`SELECT * FROM tokens ORDER BY %s %s LIMIT $1 OFFSET $2`, sortBy, sortOrder)
	var tokens []*entities.Token
	if err := r.db.SelectContext(ctx, &tokens, query, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("failed to get tokens: %w", err)
	}

	return tokens, total, nil
}

// Count returns the total number of tokens
func (r *TokenRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM tokens`

	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("failed to count tokens: %w", err)
	}

	return count, nil
}
