package testutil

import (
	"context"
	"testing"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

func TestMockTransferRepository_GetByFilter(t *testing.T) {
	repo := NewMockTransferRepository()

	// Add test data
	repo.AddTransfers(
		CreateTestTransfer(WithID(1), WithTokenAddress(USDTAddress), WithFromAddress(AliceAddress)),
		CreateTestTransfer(WithID(2), WithTokenAddress(USDTAddress), WithFromAddress(BobAddress)),
		CreateTestTransfer(WithID(3), WithTokenAddress(USDCAddress), WithFromAddress(AliceAddress)),
	)

	ctx := context.Background()

	// Test filter by token address
	tokenAddr := USDTAddress
	filter := entities.TransferFilter{
		TokenAddress: &tokenAddr,
		Limit:        100,
	}

	transfers, err := repo.GetByFilter(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transfers) != 2 {
		t.Errorf("expected 2 transfers, got %d", len(transfers))
	}

	// Test filter by from address
	fromAddr := AliceAddress
	filter = entities.TransferFilter{
		FromAddress: &fromAddr,
		Limit:       100,
	}

	transfers, err = repo.GetByFilter(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transfers) != 2 {
		t.Errorf("expected 2 transfers from Alice, got %d", len(transfers))
	}

	// Test call tracking
	if len(repo.Calls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(repo.Calls))
	}
}

func TestMockTransferRepository_BatchInsert(t *testing.T) {
	repo := NewMockTransferRepository()
	ctx := context.Background()

	transfers := CreateMultipleTransfers(5)
	err := repo.BatchInsert(ctx, transfers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify transfers were added
	all, err := repo.GetByFilter(ctx, entities.TransferFilter{Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5 transfers, got %d", len(all))
	}
}

func TestMockTransferRepository_GetLatestBlock(t *testing.T) {
	repo := NewMockTransferRepository()

	repo.AddTransfers(
		CreateTestTransfer(WithBlockNumber(100), WithTokenAddress(USDTAddress)),
		CreateTestTransfer(WithBlockNumber(200), WithTokenAddress(USDTAddress)),
		CreateTestTransfer(WithBlockNumber(150), WithTokenAddress(USDCAddress)),
	)

	ctx := context.Background()

	latest, err := repo.GetLatestBlock(ctx, USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != 200 {
		t.Errorf("expected 200, got %d", latest)
	}

	latest, err = repo.GetLatestBlock(ctx, USDCAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != 150 {
		t.Errorf("expected 150, got %d", latest)
	}
}

func TestMockTokenRepository(t *testing.T) {
	repo := NewMockTokenRepository()
	ctx := context.Background()

	// Test Upsert
	token := CreateTestToken()
	err := repo.Upsert(ctx, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test GetByAddress
	retrieved, err := repo.GetByAddress(ctx, USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected token, got nil")
	}
	if retrieved.Symbol != "USDT" {
		t.Errorf("expected USDT, got %s", retrieved.Symbol)
	}

	// Test GetAll
	repo.AddToken(CreateTestToken(TokenWithAddress(USDCAddress), TokenWithSymbol("USDC")))
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(all))
	}

	// Test UpdateStats
	err = repo.UpdateStats(ctx, USDTAddress, 100, 12500000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ = repo.GetByAddress(ctx, USDTAddress)
	if retrieved.TotalIndexedTransfers != 100 {
		t.Errorf("expected 100 transfers, got %d", retrieved.TotalIndexedTransfers)
	}
}

func TestMockIndexerStateRepository(t *testing.T) {
	repo := NewMockIndexerStateRepository()
	ctx := context.Background()

	// Test Upsert
	state := CreateTestIndexerState()
	err := repo.Upsert(ctx, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Get
	retrieved, err := repo.Get(ctx, USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected state, got nil")
	}
	if retrieved.LastIndexedBlock != 12345678 {
		t.Errorf("expected block 12345678, got %d", retrieved.LastIndexedBlock)
	}

	// Test UpdateLastBlock
	err = repo.UpdateLastBlock(ctx, USDTAddress, 12500000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ = repo.Get(ctx, USDTAddress)
	if retrieved.LastIndexedBlock != 12500000 {
		t.Errorf("expected block 12500000, got %d", retrieved.LastIndexedBlock)
	}

	// Test SetBackfilling
	fromBlock := int64(10000000)
	toBlock := int64(11000000)
	err = repo.SetBackfilling(ctx, USDTAddress, true, &fromBlock, &toBlock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ = repo.Get(ctx, USDTAddress)
	if !retrieved.IsBackfilling {
		t.Error("expected backfilling to be true")
	}
	if *retrieved.BackfillFromBlock != 10000000 {
		t.Errorf("expected from block 10000000, got %d", *retrieved.BackfillFromBlock)
	}
}

func TestCreateTestTransfer(t *testing.T) {
	// Test default values
	transfer := CreateTestTransfer()
	if transfer.TokenAddress != USDTAddress {
		t.Errorf("expected USDT address, got %s", transfer.TokenAddress)
	}
	if transfer.FromAddress != AliceAddress {
		t.Errorf("expected Alice address, got %s", transfer.FromAddress)
	}

	// Test with options
	transfer = CreateTestTransfer(
		WithBlockNumber(999),
		WithTokenAddress(USDCAddress),
	)
	if transfer.BlockNumber != 999 {
		t.Errorf("expected block 999, got %d", transfer.BlockNumber)
	}
	if transfer.TokenAddress != USDCAddress {
		t.Errorf("expected USDC address, got %s", transfer.TokenAddress)
	}
}

func TestCreateMultipleTransfers(t *testing.T) {
	transfers := CreateMultipleTransfers(10)
	if len(transfers) != 10 {
		t.Errorf("expected 10 transfers, got %d", len(transfers))
	}

	// Verify each has unique ID and LogIndex
	ids := make(map[int64]bool)
	for _, tr := range transfers {
		if ids[tr.ID] {
			t.Errorf("duplicate ID: %d", tr.ID)
		}
		ids[tr.ID] = true
	}
}

func TestPointerTo(t *testing.T) {
	intVal := 42
	ptr := PointerTo(intVal)
	if *ptr != 42 {
		t.Errorf("expected 42, got %d", *ptr)
	}

	strVal := "hello"
	strPtr := PointerTo(strVal)
	if *strPtr != "hello" {
		t.Errorf("expected hello, got %s", *strPtr)
	}
}
