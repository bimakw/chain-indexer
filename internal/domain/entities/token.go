package entities

import (
	"time"
)

// Token represents an ERC-20 token being indexed
type Token struct {
	Address               string    `db:"address"`
	Name                  string    `db:"name"`
	Symbol                string    `db:"symbol"`
	Decimals              int       `db:"decimals"`
	TotalIndexedTransfers int64     `db:"total_indexed_transfers"`
	FirstSeenBlock        *int64    `db:"first_seen_block"`
	LastSeenBlock         *int64    `db:"last_seen_block"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}
