package services

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func setupTransferServiceTest() (*TransferService, *testutil.MockTransferRepository, *testutil.MockTokenRepository) {
	transferRepo := testutil.NewMockTransferRepository()
	tokenRepo := testutil.NewMockTokenRepository()
	logger := zap.NewNop()

	service := NewTransferService(transferRepo, tokenRepo, nil, logger)
	return service, transferRepo, tokenRepo
}

func TestNewTransferService(t *testing.T) {
	service, _, _ := setupTransferServiceTest()
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestTransferService_GetTransfers_Success(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	// Add test data
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(
			testutil.WithID(1),
			testutil.WithBlockNumber(100),
			testutil.WithBlockTimestamp(timestamp),
			testutil.WithValue(big.NewInt(1000000)),
		),
		testutil.CreateTestTransfer(
			testutil.WithID(2),
			testutil.WithBlockNumber(101),
			testutil.WithBlockTimestamp(timestamp.Add(time.Minute)),
			testutil.WithValue(big.NewInt(2000000)),
		),
	)

	filter := entities.TransferFilter{
		Limit:  100,
		Offset: 0,
	}

	response, err := service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
	if len(response.Transfers) != 2 {
		t.Errorf("expected 2 transfers, got %d", len(response.Transfers))
	}
	if response.Limit != 100 {
		t.Errorf("expected limit 100, got %d", response.Limit)
	}
	if response.Offset != 0 {
		t.Errorf("expected offset 0, got %d", response.Offset)
	}
	if response.HasMore {
		t.Error("expected HasMore to be false")
	}
}

func TestTransferService_GetTransfers_Pagination(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	// Add 10 transfers
	transfers := testutil.CreateMultipleTransfers(10)
	transferRepo.AddTransfers(transfers...)

	// Test first page
	filter := entities.TransferFilter{
		Limit:  3,
		Offset: 0,
	}

	response, err := service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 10 {
		t.Errorf("expected total 10, got %d", response.Total)
	}
	if len(response.Transfers) != 3 {
		t.Errorf("expected 3 transfers, got %d", len(response.Transfers))
	}
	if !response.HasMore {
		t.Error("expected HasMore to be true")
	}

	// Test second page
	filter.Offset = 3
	response, err = service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Transfers) != 3 {
		t.Errorf("expected 3 transfers, got %d", len(response.Transfers))
	}
	if !response.HasMore {
		t.Error("expected HasMore to be true")
	}

	// Test last page
	filter.Offset = 9
	response, err = service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Transfers) != 1 {
		t.Errorf("expected 1 transfer, got %d", len(response.Transfers))
	}
	if response.HasMore {
		t.Error("expected HasMore to be false")
	}
}

func TestTransferService_GetTransfers_FilterByToken(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithTokenAddress(testutil.USDTAddress)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithTokenAddress(testutil.USDTAddress)),
		testutil.CreateTestTransfer(testutil.WithID(3), testutil.WithTokenAddress(testutil.USDCAddress)),
	)

	tokenAddr := testutil.USDTAddress
	filter := entities.TransferFilter{
		TokenAddress: &tokenAddr,
		Limit:        100,
	}

	response, err := service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
	if len(response.Transfers) != 2 {
		t.Errorf("expected 2 transfers, got %d", len(response.Transfers))
	}
}

func TestTransferService_GetTransfers_FilterByAddress(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithFromAddress(testutil.AliceAddress), testutil.WithToAddress(testutil.BobAddress)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithFromAddress(testutil.BobAddress), testutil.WithToAddress(testutil.CharlieAddr)),
		testutil.CreateTestTransfer(testutil.WithID(3), testutil.WithFromAddress(testutil.CharlieAddr), testutil.WithToAddress(testutil.AliceAddress)),
	)

	// Alice is involved in 2 transfers (as sender and receiver)
	aliceAddr := testutil.AliceAddress
	filter := entities.TransferFilter{
		Address: &aliceAddr,
		Limit:   100,
	}

	response, err := service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
}

func TestTransferService_GetTransfers_FilterByBlockRange(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithBlockNumber(100)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithBlockNumber(150)),
		testutil.CreateTestTransfer(testutil.WithID(3), testutil.WithBlockNumber(200)),
		testutil.CreateTestTransfer(testutil.WithID(4), testutil.WithBlockNumber(250)),
	)

	fromBlock := int64(120)
	toBlock := int64(200)
	filter := entities.TransferFilter{
		FromBlock: &fromBlock,
		ToBlock:   &toBlock,
		Limit:     100,
	}

	response, err := service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include block 150 and 200 (between 120 and 200 inclusive)
	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
}

func TestTransferService_GetTransfers_EmptyResult(t *testing.T) {
	service, _, _ := setupTransferServiceTest()
	ctx := context.Background()

	filter := entities.TransferFilter{
		Limit: 100,
	}

	response, err := service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 0 {
		t.Errorf("expected total 0, got %d", response.Total)
	}
	if len(response.Transfers) != 0 {
		t.Errorf("expected 0 transfers, got %d", len(response.Transfers))
	}
	if response.HasMore {
		t.Error("expected HasMore to be false")
	}
}

func TestTransferService_GetTransfers_RepositoryError(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	// Simulate repository error
	transferRepo.GetByFilterFunc = func(ctx context.Context, filter entities.TransferFilter) ([]entities.Transfer, error) {
		return nil, errors.New("database connection failed")
	}

	filter := entities.TransferFilter{Limit: 100}
	_, err := service.GetTransfers(ctx, filter)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to get transfers: database connection failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTransferService_GetTransfers_CountError(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	// Simulate count error
	transferRepo.GetCountFunc = func(ctx context.Context, filter entities.TransferFilter) (int64, error) {
		return 0, errors.New("count query failed")
	}

	filter := entities.TransferFilter{Limit: 100}
	_, err := service.GetTransfers(ctx, filter)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to get transfer count: count query failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTransferService_GetTransfersByAddress(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithFromAddress(testutil.AliceAddress)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithToAddress(testutil.AliceAddress)),
		testutil.CreateTestTransfer(testutil.WithID(3), testutil.WithFromAddress(testutil.BobAddress)),
	)

	response, err := service.GetTransfersByAddress(ctx, testutil.AliceAddress, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
}

func TestTransferService_GetTransfersByAddress_Lowercase(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithFromAddress(testutil.AliceAddress)),
	)

	// Use uppercase address
	upperAddr := "0x1111111111111111111111111111111111111111"
	response, err := service.GetTransfersByAddress(ctx, upperAddr, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 1 {
		t.Errorf("expected total 1, got %d", response.Total)
	}
}

func TestTransferService_GetTransfersByToken(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithTokenAddress(testutil.USDTAddress)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithTokenAddress(testutil.USDTAddress)),
		testutil.CreateTestTransfer(testutil.WithID(3), testutil.WithTokenAddress(testutil.USDCAddress)),
	)

	response, err := service.GetTransfersByToken(ctx, testutil.USDTAddress, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
}

func TestTransferService_GetTransfersByToken_Lowercase(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithTokenAddress(testutil.USDTAddress)),
	)

	// Use uppercase token address
	upperAddr := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	response, err := service.GetTransfersByToken(ctx, upperAddr, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Total != 1 {
		t.Errorf("expected total 1, got %d", response.Total)
	}
}

func TestTransferDTO_Formatting(t *testing.T) {
	service, transferRepo, _ := setupTransferServiceTest()
	ctx := context.Background()

	timestamp := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(
			testutil.WithTxHash("0xabcd1234"),
			testutil.WithLogIndex(5),
			testutil.WithBlockNumber(12345),
			testutil.WithBlockTimestamp(timestamp),
			testutil.WithTokenAddress(testutil.USDTAddress),
			testutil.WithFromAddress(testutil.AliceAddress),
			testutil.WithToAddress(testutil.BobAddress),
			testutil.WithValue(big.NewInt(1000000)),
		),
	)

	filter := entities.TransferFilter{Limit: 100}
	response, err := service.GetTransfers(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Transfers) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(response.Transfers))
	}

	dto := response.Transfers[0]
	if dto.TxHash != "0xabcd1234" {
		t.Errorf("TxHash mismatch: %s", dto.TxHash)
	}
	if dto.LogIndex != 5 {
		t.Errorf("LogIndex mismatch: %d", dto.LogIndex)
	}
	if dto.BlockNumber != 12345 {
		t.Errorf("BlockNumber mismatch: %d", dto.BlockNumber)
	}
	if dto.BlockTimestamp != "2024-01-15T10:30:45Z" {
		t.Errorf("BlockTimestamp mismatch: %s", dto.BlockTimestamp)
	}
	if dto.TokenAddress != testutil.USDTAddress {
		t.Errorf("TokenAddress mismatch: %s", dto.TokenAddress)
	}
	if dto.FromAddress != testutil.AliceAddress {
		t.Errorf("FromAddress mismatch: %s", dto.FromAddress)
	}
	if dto.ToAddress != testutil.BobAddress {
		t.Errorf("ToAddress mismatch: %s", dto.ToAddress)
	}
	if dto.Value != "1000000" {
		t.Errorf("Value mismatch: %s", dto.Value)
	}
}

func TestGenerateCacheKey_DifferentFilters(t *testing.T) {
	service, _, _ := setupTransferServiceTest()

	tokenAddr := testutil.USDTAddress
	fromAddr := testutil.AliceAddress
	toAddr := testutil.BobAddress
	addr := testutil.CharlieAddr
	fromBlock := int64(100)
	toBlock := int64(200)

	tests := []struct {
		name    string
		filter1 entities.TransferFilter
		filter2 entities.TransferFilter
		same    bool
	}{
		{
			name:    "same filters produce same key",
			filter1: entities.TransferFilter{TokenAddress: &tokenAddr, Limit: 100, Offset: 0},
			filter2: entities.TransferFilter{TokenAddress: &tokenAddr, Limit: 100, Offset: 0},
			same:    true,
		},
		{
			name:    "different token produces different key",
			filter1: entities.TransferFilter{TokenAddress: &tokenAddr, Limit: 100},
			filter2: entities.TransferFilter{FromAddress: &fromAddr, Limit: 100},
			same:    false,
		},
		{
			name:    "different limit produces different key",
			filter1: entities.TransferFilter{TokenAddress: &tokenAddr, Limit: 100},
			filter2: entities.TransferFilter{TokenAddress: &tokenAddr, Limit: 50},
			same:    false,
		},
		{
			name:    "different offset produces different key",
			filter1: entities.TransferFilter{TokenAddress: &tokenAddr, Limit: 100, Offset: 0},
			filter2: entities.TransferFilter{TokenAddress: &tokenAddr, Limit: 100, Offset: 10},
			same:    false,
		},
		{
			name:    "all filters combined",
			filter1: entities.TransferFilter{TokenAddress: &tokenAddr, FromAddress: &fromAddr, ToAddress: &toAddr, Address: &addr, FromBlock: &fromBlock, ToBlock: &toBlock, Limit: 100},
			filter2: entities.TransferFilter{TokenAddress: &tokenAddr, FromAddress: &fromAddr, ToAddress: &toAddr, Address: &addr, FromBlock: &fromBlock, ToBlock: &toBlock, Limit: 100},
			same:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := service.generateCacheKey(tt.filter1)
			key2 := service.generateCacheKey(tt.filter2)

			if tt.same && key1 != key2 {
				t.Errorf("expected same cache keys, got %s and %s", key1, key2)
			}
			if !tt.same && key1 == key2 {
				t.Errorf("expected different cache keys, but both are %s", key1)
			}
		})
	}
}

func TestGenerateCacheKey_Format(t *testing.T) {
	service, _, _ := setupTransferServiceTest()

	filter := entities.TransferFilter{Limit: 100}
	key := service.generateCacheKey(filter)

	// Key should start with "transfers:"
	if len(key) < 10 || key[:10] != "transfers:" {
		t.Errorf("cache key should start with 'transfers:', got %s", key)
	}

	// Key should be consistent length (prefix + 16 hex chars)
	expectedLen := 10 + 16
	if len(key) != expectedLen {
		t.Errorf("expected key length %d, got %d", expectedLen, len(key))
	}
}
