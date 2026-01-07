package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
)

// Ensure PortfolioRepo implements PortfolioRepository
var _ repositories.PortfolioRepository = (*PortfolioRepo)(nil)

// PortfolioRepo implements PortfolioRepository using PostgreSQL
type PortfolioRepo struct {
	db *sqlx.DB
}

// NewPortfolioRepo creates a new portfolio repository
func NewPortfolioRepo(db *sqlx.DB) *PortfolioRepo {
	return &PortfolioRepo{db: db}
}

// holdingRow holds the result of the holdings query
type holdingRow struct {
	TokenAddress string `db:"token_address"`
	TokenName    string `db:"name"`
	TokenSymbol  string `db:"symbol"`
	Decimals     int    `db:"decimals"`
	Balance      string `db:"balance"`
}

// GetWalletHoldings retrieves all token holdings for a wallet
func (r *PortfolioRepo) GetWalletHoldings(ctx context.Context, walletAddress string) ([]entities.TokenHolding, error) {
	query := `
		WITH balances AS (
			SELECT
				token_address,
				SUM(CASE WHEN to_address = $1 THEN value ELSE 0 END) -
				SUM(CASE WHEN from_address = $1 THEN value ELSE 0 END) as balance
			FROM transfers
			WHERE from_address = $1 OR to_address = $1
			GROUP BY token_address
			HAVING SUM(CASE WHEN to_address = $1 THEN value ELSE 0 END) -
				   SUM(CASE WHEN from_address = $1 THEN value ELSE 0 END) > 0
		)
		SELECT
			b.token_address,
			t.name,
			t.symbol,
			t.decimals,
			b.balance::text as balance
		FROM balances b
		JOIN tokens t ON t.address = b.token_address
		ORDER BY b.balance DESC
	`

	var rows []holdingRow
	if err := r.db.SelectContext(ctx, &rows, query, walletAddress); err != nil {
		return nil, fmt.Errorf("failed to get wallet holdings: %w", err)
	}

	holdings := make([]entities.TokenHolding, len(rows))
	for i, row := range rows {
		holdings[i] = entities.TokenHolding{
			TokenAddress: row.TokenAddress,
			TokenName:    row.TokenName,
			TokenSymbol:  row.TokenSymbol,
			Decimals:     row.Decimals,
			BalanceStr:   row.Balance,
			BalanceHuman: formatBalance(row.Balance, row.Decimals),
		}
	}

	return holdings, nil
}

// GetWalletHoldingByToken retrieves holding for specific token
func (r *PortfolioRepo) GetWalletHoldingByToken(ctx context.Context, walletAddress, tokenAddress string) (*entities.TokenHolding, error) {
	query := `
		SELECT
			$2 as token_address,
			t.name,
			t.symbol,
			t.decimals,
			COALESCE(
				SUM(CASE WHEN tr.to_address = $1 THEN tr.value ELSE 0 END) -
				SUM(CASE WHEN tr.from_address = $1 THEN tr.value ELSE 0 END),
				0
			)::text as balance
		FROM tokens t
		LEFT JOIN transfers tr ON tr.token_address = t.address
			AND (tr.from_address = $1 OR tr.to_address = $1)
		WHERE t.address = $2
		GROUP BY t.name, t.symbol, t.decimals
	`

	var row holdingRow
	if err := r.db.GetContext(ctx, &row, query, walletAddress, tokenAddress); err != nil {
		return nil, fmt.Errorf("failed to get wallet holding by token: %w", err)
	}

	return &entities.TokenHolding{
		TokenAddress: row.TokenAddress,
		TokenName:    row.TokenName,
		TokenSymbol:  row.TokenSymbol,
		Decimals:     row.Decimals,
		BalanceStr:   row.Balance,
		BalanceHuman: formatBalance(row.Balance, row.Decimals),
	}, nil
}

// GetWalletTokenCount returns count of tokens held by wallet
func (r *PortfolioRepo) GetWalletTokenCount(ctx context.Context, walletAddress string) (int64, error) {
	query := `
		WITH balances AS (
			SELECT
				token_address,
				SUM(CASE WHEN to_address = $1 THEN value ELSE 0 END) -
				SUM(CASE WHEN from_address = $1 THEN value ELSE 0 END) as balance
			FROM transfers
			WHERE from_address = $1 OR to_address = $1
			GROUP BY token_address
			HAVING SUM(CASE WHEN to_address = $1 THEN value ELSE 0 END) -
				   SUM(CASE WHEN from_address = $1 THEN value ELSE 0 END) > 0
		)
		SELECT COUNT(*) FROM balances
	`

	var count int64
	if err := r.db.GetContext(ctx, &count, query, walletAddress); err != nil {
		return 0, fmt.Errorf("failed to get wallet token count: %w", err)
	}

	return count, nil
}

// summaryRow holds the result of the summary query
type summaryRow struct {
	TotalIn       int64   `db:"total_in"`
	TotalOut      int64   `db:"total_out"`
	VolumeIn      string  `db:"volume_in"`
	VolumeOut     string  `db:"volume_out"`
	UniqueTokens  int64   `db:"unique_tokens"`
	FirstTransfer *string `db:"first_transfer"`
	LastTransfer  *string `db:"last_transfer"`
}

// GetWalletTransferSummary returns transfer stats for a wallet
func (r *PortfolioRepo) GetWalletTransferSummary(ctx context.Context, walletAddress string) (*repositories.WalletTransferSummary, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE to_address = $1) as total_in,
			COUNT(*) FILTER (WHERE from_address = $1) as total_out,
			COALESCE(SUM(value) FILTER (WHERE to_address = $1), 0)::text as volume_in,
			COALESCE(SUM(value) FILTER (WHERE from_address = $1), 0)::text as volume_out,
			COUNT(DISTINCT token_address) as unique_tokens,
			MIN(block_timestamp)::text as first_transfer,
			MAX(block_timestamp)::text as last_transfer
		FROM transfers
		WHERE from_address = $1 OR to_address = $1
	`

	var row summaryRow
	if err := r.db.GetContext(ctx, &row, query, walletAddress); err != nil {
		return nil, fmt.Errorf("failed to get wallet transfer summary: %w", err)
	}

	result := &repositories.WalletTransferSummary{
		TotalTransfersIn:  row.TotalIn,
		TotalTransfersOut: row.TotalOut,
		TotalVolumeIn:     row.VolumeIn,
		TotalVolumeOut:    row.VolumeOut,
		UniqueTokens:      row.UniqueTokens,
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

// formatBalance converts raw balance to human readable format with decimals
func formatBalance(balance string, decimals int) string {
	if balance == "" || balance == "0" {
		return "0"
	}

	// Pad with leading zeros if necessary
	for len(balance) <= decimals {
		balance = "0" + balance
	}

	// Insert decimal point
	if decimals > 0 {
		insertPos := len(balance) - decimals
		intPart := balance[:insertPos]
		decPart := balance[insertPos:]

		// Trim trailing zeros from decimal part
		decPart = trimTrailingZeros(decPart)

		if decPart == "" {
			return intPart
		}
		return intPart + "." + decPart
	}

	return balance
}

// trimTrailingZeros removes trailing zeros from a string
func trimTrailingZeros(s string) string {
	i := len(s) - 1
	for i >= 0 && s[i] == '0' {
		i--
	}
	if i < 0 {
		return ""
	}
	return s[:i+1]
}
