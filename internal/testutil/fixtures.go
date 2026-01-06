package testutil

import (
	"math/big"
	"time"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// Common test addresses
const (
	USDTAddress  = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	USDCAddress  = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	AliceAddress = "0x1111111111111111111111111111111111111111"
	BobAddress   = "0x2222222222222222222222222222222222222222"
	CharlieAddr  = "0x3333333333333333333333333333333333333333"
)

// CreateTestTransfer creates a test transfer with default values
func CreateTestTransfer(opts ...TransferOption) entities.Transfer {
	t := entities.Transfer{
		ID:             1,
		TxHash:         "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		LogIndex:       0,
		BlockNumber:    12345678,
		BlockTimestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		TokenAddress:   USDTAddress,
		FromAddress:    AliceAddress,
		ToAddress:      BobAddress,
		Value:          big.NewInt(1000000), // 1 USDT
		ValueString:    "1000000",
		CreatedAt:      time.Now(),
	}

	for _, opt := range opts {
		opt(&t)
	}

	return t
}

type TransferOption func(*entities.Transfer)

func WithID(id int64) TransferOption {
	return func(t *entities.Transfer) {
		t.ID = id
	}
}

func WithTxHash(hash string) TransferOption {
	return func(t *entities.Transfer) {
		t.TxHash = hash
	}
}

func WithLogIndex(idx int) TransferOption {
	return func(t *entities.Transfer) {
		t.LogIndex = idx
	}
}

func WithBlockNumber(num int64) TransferOption {
	return func(t *entities.Transfer) {
		t.BlockNumber = num
	}
}

func WithBlockTimestamp(ts time.Time) TransferOption {
	return func(t *entities.Transfer) {
		t.BlockTimestamp = ts
	}
}

func WithTokenAddress(addr string) TransferOption {
	return func(t *entities.Transfer) {
		t.TokenAddress = addr
	}
}

func WithFromAddress(addr string) TransferOption {
	return func(t *entities.Transfer) {
		t.FromAddress = addr
	}
}

func WithToAddress(addr string) TransferOption {
	return func(t *entities.Transfer) {
		t.ToAddress = addr
	}
}

func WithValue(val *big.Int) TransferOption {
	return func(t *entities.Transfer) {
		t.Value = val
		t.ValueString = val.String()
	}
}

// CreateTestToken creates a test token with default values
func CreateTestToken(opts ...TokenOption) *entities.Token {
	firstSeenBlock := int64(12000000)
	lastSeenBlock := int64(12345678)
	t := &entities.Token{
		Address:               USDTAddress,
		Name:                  "Tether USD",
		Symbol:                "USDT",
		Decimals:              6,
		TotalIndexedTransfers: 0,
		FirstSeenBlock:        &firstSeenBlock,
		LastSeenBlock:         &lastSeenBlock,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

type TokenOption func(*entities.Token)

func TokenWithAddress(addr string) TokenOption {
	return func(t *entities.Token) {
		t.Address = addr
	}
}

func TokenWithName(name string) TokenOption {
	return func(t *entities.Token) {
		t.Name = name
	}
}

func TokenWithSymbol(symbol string) TokenOption {
	return func(t *entities.Token) {
		t.Symbol = symbol
	}
}

func TokenWithDecimals(dec int) TokenOption {
	return func(t *entities.Token) {
		t.Decimals = dec
	}
}

func TokenWithTotalTransfers(count int64) TokenOption {
	return func(t *entities.Token) {
		t.TotalIndexedTransfers = count
	}
}

func TokenWithFirstSeenBlock(block int64) TokenOption {
	return func(t *entities.Token) {
		t.FirstSeenBlock = &block
	}
}

func TokenWithLastSeenBlock(block int64) TokenOption {
	return func(t *entities.Token) {
		t.LastSeenBlock = &block
	}
}

// CreateTestIndexerState creates a test indexer state
func CreateTestIndexerState(opts ...IndexerStateOption) *entities.IndexerState {
	s := &entities.IndexerState{
		TokenAddress:     USDTAddress,
		LastIndexedBlock: 12345678,
		IsBackfilling:    false,
		UpdatedAt:        time.Now(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type IndexerStateOption func(*entities.IndexerState)

func StateWithTokenAddress(addr string) IndexerStateOption {
	return func(s *entities.IndexerState) {
		s.TokenAddress = addr
	}
}

func StateWithLastIndexedBlock(block int64) IndexerStateOption {
	return func(s *entities.IndexerState) {
		s.LastIndexedBlock = block
	}
}

func StateWithBackfilling(isBackfilling bool, fromBlock, toBlock *int64) IndexerStateOption {
	return func(s *entities.IndexerState) {
		s.IsBackfilling = isBackfilling
		s.BackfillFromBlock = fromBlock
		s.BackfillToBlock = toBlock
	}
}

// CreateMultipleTransfers creates multiple test transfers for testing pagination
func CreateMultipleTransfers(count int, opts ...TransferOption) []entities.Transfer {
	transfers := make([]entities.Transfer, count)
	for i := 0; i < count; i++ {
		t := CreateTestTransfer(opts...)
		t.ID = int64(i + 1)
		t.LogIndex = i
		t.BlockNumber = int64(12345678 + i)
		t.BlockTimestamp = t.BlockTimestamp.Add(time.Duration(i) * time.Minute)
		t.TxHash = generateTxHash(i)
		transfers[i] = t
	}
	return transfers
}

func generateTxHash(index int) string {
	// Generate a unique tx hash based on index
	hash := "0x"
	for i := 0; i < 64; i++ {
		hash += string(rune('a' + (index+i)%6))
	}
	return hash
}

// PointerTo returns a pointer to the given value
func PointerTo[T any](v T) *T {
	return &v
}
