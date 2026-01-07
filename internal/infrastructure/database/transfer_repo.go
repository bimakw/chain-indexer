package database

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// statsRow holds the result of the stats query
type statsRow struct {
	TotalTransfers  int64   `db:"total_transfers"`
	UniqueFrom      int64   `db:"unique_from"`
	UniqueTo        int64   `db:"unique_to"`
	TotalVolume     string  `db:"total_volume"`
	FirstTransfer   *string `db:"first_transfer"`
	LastTransfer    *string `db:"last_transfer"`
	Transfers24h    int64   `db:"transfers_24h"`
	Volume24h       string  `db:"volume_24h"`
	Transfers7d     int64   `db:"transfers_7d"`
	Volume7d        string  `db:"volume_7d"`
}

// GetTokenStats returns aggregated transfer statistics for a token
func (r *TransferRepo) GetTokenStats(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error) {
	query := `
		WITH stats AS (
			SELECT
				COUNT(*) as total_transfers,
				COUNT(DISTINCT from_address) as unique_from,
				COUNT(DISTINCT to_address) as unique_to,
				COALESCE(SUM(value), 0)::TEXT as total_volume,
				MIN(block_timestamp)::TEXT as first_transfer,
				MAX(block_timestamp)::TEXT as last_transfer
			FROM transfers
			WHERE token_address = $1
		),
		stats_24h AS (
			SELECT
				COUNT(*) as transfers,
				COALESCE(SUM(value), 0)::TEXT as volume
			FROM transfers
			WHERE token_address = $1
			AND block_timestamp >= NOW() - INTERVAL '24 hours'
		),
		stats_7d AS (
			SELECT
				COUNT(*) as transfers,
				COALESCE(SUM(value), 0)::TEXT as volume
			FROM transfers
			WHERE token_address = $1
			AND block_timestamp >= NOW() - INTERVAL '7 days'
		)
		SELECT
			s.total_transfers, s.unique_from, s.unique_to, s.total_volume,
			s.first_transfer, s.last_transfer,
			s24.transfers as transfers_24h, s24.volume as volume_24h,
			s7.transfers as transfers_7d, s7.volume as volume_7d
		FROM stats s, stats_24h s24, stats_7d s7
	`

	var row statsRow
	if err := r.db.GetContext(ctx, &row, query, tokenAddress); err != nil {
		return nil, fmt.Errorf("failed to get token stats: %w", err)
	}

	result := &repositories.TokenStatsResult{
		TotalTransfers:  row.TotalTransfers,
		UniqueFromAddrs: row.UniqueFrom,
		UniqueToAddrs:   row.UniqueTo,
		TotalVolume:     row.TotalVolume,
		Transfers24h:    row.Transfers24h,
		Volume24h:       row.Volume24h,
		Transfers7d:     row.Transfers7d,
		Volume7d:        row.Volume7d,
	}

	// Parse timestamps if they exist
	if row.FirstTransfer != nil && *row.FirstTransfer != "" {
		t, err := parseTimestamp(*row.FirstTransfer)
		if err == nil {
			result.FirstTransferAt = &t
		}
	}
	if row.LastTransfer != nil && *row.LastTransfer != "" {
		t, err := parseTimestamp(*row.LastTransfer)
		if err == nil {
			result.LastTransferAt = &t
		}
	}

	return result, nil
}

// parseTimestamp parses a timestamp string from the database
func parseTimestamp(s string) (time.Time, error) {
	// Try parsing various formats
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse timestamp: %s", s)
}

// holderBalanceRow holds the result of the holder balance query
type holderBalanceRow struct {
	Address string `db:"address"`
	Balance string `db:"balance"`
	Rank    int    `db:"rank"`
}

// GetTopHolders returns top token holders sorted by balance
func (r *TransferRepo) GetTopHolders(ctx context.Context, tokenAddress string, limit int) ([]repositories.HolderBalance, error) {
	query := `
		WITH balances AS (
			SELECT
				address,
				SUM(amount) as balance
			FROM (
				-- Incoming transfers (positive)
				SELECT to_address as address, value as amount
				FROM transfers
				WHERE token_address = $1

				UNION ALL

				-- Outgoing transfers (negative)
				SELECT from_address as address, -value as amount
				FROM transfers
				WHERE token_address = $1
			) t
			GROUP BY address
			HAVING SUM(amount) > 0
		)
		SELECT
			address,
			balance::TEXT as balance,
			ROW_NUMBER() OVER (ORDER BY balance DESC)::INTEGER as rank
		FROM balances
		ORDER BY balance DESC
		LIMIT $2
	`

	var rows []holderBalanceRow
	if err := r.db.SelectContext(ctx, &rows, query, tokenAddress, limit); err != nil {
		return nil, fmt.Errorf("failed to get top holders: %w", err)
	}

	result := make([]repositories.HolderBalance, len(rows))
	for i, row := range rows {
		result[i] = repositories.HolderBalance{
			Address: row.Address,
			Balance: row.Balance,
			Rank:    row.Rank,
		}
	}

	return result, nil
}

// GetHolderBalance returns balance for a specific holder
func (r *TransferRepo) GetHolderBalance(ctx context.Context, tokenAddress, holderAddress string) (*repositories.HolderBalance, error) {
	// First get the balance
	balanceQuery := `
		SELECT
			COALESCE(SUM(
				CASE
					WHEN to_address = $2 THEN value
					WHEN from_address = $2 THEN -value
					ELSE 0
				END
			), 0)::TEXT as balance
		FROM transfers
		WHERE token_address = $1
		AND (to_address = $2 OR from_address = $2)
	`

	var balance string
	if err := r.db.GetContext(ctx, &balance, balanceQuery, tokenAddress, holderAddress); err != nil {
		return nil, fmt.Errorf("failed to get holder balance: %w", err)
	}

	// Get the rank by counting addresses with higher balance
	rankQuery := `
		WITH balances AS (
			SELECT
				address,
				SUM(amount) as balance
			FROM (
				SELECT to_address as address, value as amount
				FROM transfers
				WHERE token_address = $1

				UNION ALL

				SELECT from_address as address, -value as amount
				FROM transfers
				WHERE token_address = $1
			) t
			GROUP BY address
			HAVING SUM(amount) > 0
		)
		SELECT COUNT(*) + 1 as rank
		FROM balances
		WHERE balance > (
			SELECT COALESCE(SUM(
				CASE
					WHEN to_address = $2 THEN value
					WHEN from_address = $2 THEN -value
					ELSE 0
				END
			), 0)
			FROM transfers
			WHERE token_address = $1
			AND (to_address = $2 OR from_address = $2)
		)
	`

	var rank int
	if err := r.db.GetContext(ctx, &rank, rankQuery, tokenAddress, holderAddress); err != nil {
		return nil, fmt.Errorf("failed to get holder rank: %w", err)
	}

	return &repositories.HolderBalance{
		Address: holderAddress,
		Balance: balance,
		Rank:    rank,
	}, nil
}
