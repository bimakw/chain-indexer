package entities

import (
	"math/big"
	"time"
)

// Transfer represents an ERC-20 Transfer event
type Transfer struct {
	ID             int64     `db:"id"`
	TxHash         string    `db:"tx_hash"`
	LogIndex       int       `db:"log_index"`
	BlockNumber    int64     `db:"block_number"`
	BlockTimestamp time.Time `db:"block_timestamp"`
	TokenAddress   string    `db:"token_address"`
	FromAddress    string    `db:"from_address"`
	ToAddress      string    `db:"to_address"`
	Value          *big.Int  `db:"-"` // Handled separately due to NUMERIC type
	ValueString    string    `db:"value"`
	CreatedAt      time.Time `db:"created_at"`
}

// TransferFilter contains filters for querying transfers
type TransferFilter struct {
	TokenAddress *string
	FromAddress  *string
	ToAddress    *string
	Address      *string // matches either from or to
	FromBlock    *int64
	ToBlock      *int64
	FromTime     *time.Time
	ToTime       *time.Time
	Limit        int
	Offset       int
}

// DefaultTransferFilter returns a filter with sensible defaults
func DefaultTransferFilter() TransferFilter {
	return TransferFilter{
		Limit:  100,
		Offset: 0,
	}
}
