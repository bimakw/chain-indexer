package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
)

// Ensure TransferRepo implements TransferRepository
var _ repositories.TransferRepository = (*TransferRepo)(nil)

// TransferRepo implements TransferRepository using PostgreSQL
type TransferRepo struct {
	db *sqlx.DB
}

// NewTransferRepo creates a new transfer repository
func NewTransferRepo(db *sqlx.DB) *TransferRepo {
	return &TransferRepo{db: db}
}

// GetByFilter retrieves transfers matching the given filter
func (r *TransferRepo) GetByFilter(ctx context.Context, filter entities.TransferFilter) ([]entities.Transfer, error) {
	query, args := r.buildFilterQuery(filter, false)

	var transfers []entities.Transfer
	if err := r.db.SelectContext(ctx, &transfers, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get transfers: %w", err)
	}

	return transfers, nil
}

// GetCount returns the count of transfers matching the filter
func (r *TransferRepo) GetCount(ctx context.Context, filter entities.TransferFilter) (int64, error) {
	query, args := r.buildFilterQuery(filter, true)

	var count int64
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to get transfer count: %w", err)
	}

	return count, nil
}

// buildFilterQuery builds the SQL query for filtering transfers
func (r *TransferRepo) buildFilterQuery(filter entities.TransferFilter, countOnly bool) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.TokenAddress != nil {
		conditions = append(conditions, fmt.Sprintf("token_address = $%d", argIdx))
		args = append(args, *filter.TokenAddress)
		argIdx++
	}

	if filter.FromAddress != nil {
		conditions = append(conditions, fmt.Sprintf("from_address = $%d", argIdx))
		args = append(args, *filter.FromAddress)
		argIdx++
	}

	if filter.ToAddress != nil {
		conditions = append(conditions, fmt.Sprintf("to_address = $%d", argIdx))
		args = append(args, *filter.ToAddress)
		argIdx++
	}

	if filter.Address != nil {
		conditions = append(conditions, fmt.Sprintf("(from_address = $%d OR to_address = $%d)", argIdx, argIdx))
		args = append(args, *filter.Address)
		argIdx++
	}

	if filter.FromBlock != nil {
		conditions = append(conditions, fmt.Sprintf("block_number >= $%d", argIdx))
		args = append(args, *filter.FromBlock)
		argIdx++
	}

	if filter.ToBlock != nil {
		conditions = append(conditions, fmt.Sprintf("block_number <= $%d", argIdx))
		args = append(args, *filter.ToBlock)
		argIdx++
	}

	if filter.FromTime != nil {
		conditions = append(conditions, fmt.Sprintf("block_timestamp >= $%d", argIdx))
		args = append(args, *filter.FromTime)
		argIdx++
	}

	if filter.ToTime != nil {
		conditions = append(conditions, fmt.Sprintf("block_timestamp <= $%d", argIdx))
		args = append(args, *filter.ToTime)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	if countOnly {
		return fmt.Sprintf("SELECT COUNT(*) FROM transfers %s", whereClause), args
	}

	query := fmt.Sprintf(`
		SELECT id, tx_hash, log_index, block_number, block_timestamp,
			   token_address, from_address, to_address, value, created_at
		FROM transfers
		%s
		ORDER BY block_timestamp DESC, log_index DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	return query, args
}

// BatchInsert inserts multiple transfers in a single transaction
func (r *TransferRepo) BatchInsert(ctx context.Context, transfers []entities.Transfer) error {
	if len(transfers) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `
		INSERT INTO transfers (tx_hash, log_index, block_number, block_timestamp,
							   token_address, from_address, to_address, value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tx_hash, log_index, block_timestamp) DO NOTHING
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, t := range transfers {
		_, err := stmt.ExecContext(ctx,
			t.TxHash,
			t.LogIndex,
			t.BlockNumber,
			t.BlockTimestamp,
			t.TokenAddress,
			t.FromAddress,
			t.ToAddress,
			t.ValueString,
		)
		if err != nil {
			return fmt.Errorf("failed to insert transfer: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetLatestBlock returns the latest indexed block for a token
func (r *TransferRepo) GetLatestBlock(ctx context.Context, tokenAddress string) (int64, error) {
	query := `SELECT COALESCE(MAX(block_number), 0) FROM transfers WHERE token_address = $1`

	var blockNumber int64
	if err := r.db.GetContext(ctx, &blockNumber, query, tokenAddress); err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}

	return blockNumber, nil
}
