package entities

import (
	"time"
)

// IndexerState tracks the indexing progress for a token
type IndexerState struct {
	TokenAddress      string    `db:"token_address"`
	LastIndexedBlock  int64     `db:"last_indexed_block"`
	IsBackfilling     bool      `db:"is_backfilling"`
	BackfillFromBlock *int64    `db:"backfill_from_block"`
	BackfillToBlock   *int64    `db:"backfill_to_block"`
	UpdatedAt         time.Time `db:"updated_at"`
}
