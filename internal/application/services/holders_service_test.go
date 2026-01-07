package services

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func setupHoldersServiceTest() (*HoldersService, *testutil.MockTransferRepository, *testutil.MockTokenRepository) {
	transferRepo := testutil.NewMockTransferRepository()
	tokenRepo := testutil.NewMockTokenRepository()
	logger := zap.NewNop()

	service := NewHoldersService(transferRepo, tokenRepo, nil, logger)
	return service, transferRepo, tokenRepo
}

func TestNewHoldersService(t *testing.T) {
	service, _, _ := setupHoldersServiceTest()
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestHoldersService_GetTopHolders_Success(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithSymbol("USDT"),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 100, nil
	}

	// Setup mock holders response
	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		return []repositories.HolderBalance{
			{Address: "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503", Balance: "999999999999999999999", Rank: 1},
			{Address: "0x1111111111111111111111111111111111111111", Balance: "500000000000000000000", Rank: 2},
			{Address: "0x2222222222222222222222222222222222222222", Balance: "250000000000000000000", Rank: 3},
		}, nil
	}

	response, err := service.GetTopHolders(ctx, testutil.USDTAddress, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}

	if len(response.Data) != 3 {
		t.Errorf("expected 3 holders, got %d", len(response.Data))
	}

	// Check first holder
	holder := response.Data[0]
	if holder.Address != "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503" {
		t.Errorf("expected address 0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503, got %s", holder.Address)
	}
	if holder.Balance != "999999999999999999999" {
		t.Errorf("expected balance '999999999999999999999', got %s", holder.Balance)
	}
	if holder.Rank != 1 {
		t.Errorf("expected rank 1, got %d", holder.Rank)
	}

	// Check pagination metadata
	if response.Pagination.Total != 100 {
		t.Errorf("expected total 100, got %d", response.Pagination.Total)
	}
	if response.Pagination.Limit != 100 {
		t.Errorf("expected limit 100, got %d", response.Pagination.Limit)
	}
	if response.Pagination.Offset != 0 {
		t.Errorf("expected offset 0, got %d", response.Pagination.Offset)
	}
	// With total=100, limit=100, offset=0: 0+100 >= 100, so hasMore should be false
	if response.Pagination.HasMore {
		t.Error("expected has_more to be false (0 + 100 >= 100)")
	}
}

func TestHoldersService_GetTopHolders_TokenNotFound(t *testing.T) {
	service, _, _ := setupHoldersServiceTest()
	ctx := context.Background()

	response, err := service.GetTopHolders(ctx, testutil.USDTAddress, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response != nil {
		t.Error("expected nil response for non-existent token")
	}
}

func TestHoldersService_GetTopHolders_EmptyResult(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 0, nil
	}

	// Setup mock empty holders response
	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		return []repositories.HolderBalance{}, nil
	}

	response, err := service.GetTopHolders(ctx, testutil.USDTAddress, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}

	if len(response.Data) != 0 {
		t.Errorf("expected 0 holders, got %d", len(response.Data))
	}

	if response.Pagination.HasMore {
		t.Error("expected has_more to be false for empty result")
	}
}

func TestHoldersService_GetTopHolders_LimitValidation(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 0, nil
	}

	// Track the limit passed to repo
	var capturedLimit int
	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		capturedLimit = limit
		return []repositories.HolderBalance{}, nil
	}

	// Test default limit (when 0 is passed)
	_, _ = service.GetTopHolders(ctx, testutil.USDTAddress, 0, 0)
	if capturedLimit != 100 {
		t.Errorf("expected default limit 100, got %d", capturedLimit)
	}

	// Test max limit (when > 1000 is passed)
	_, _ = service.GetTopHolders(ctx, testutil.USDTAddress, 5000, 0)
	if capturedLimit != 1000 {
		t.Errorf("expected max limit 1000, got %d", capturedLimit)
	}
}

func TestHoldersService_GetTopHolders_Lowercase(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token with lowercase address
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 0, nil
	}

	// Track which address was queried
	var queriedAddress string
	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		queriedAddress = tokenAddress
		return []repositories.HolderBalance{}, nil
	}

	// Use uppercase address
	upperAddr := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	_, err := service.GetTopHolders(ctx, upperAddr, 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if queriedAddress != testutil.USDTAddress {
		t.Errorf("expected lowercase address %s, got %s", testutil.USDTAddress, queriedAddress)
	}
}

func TestHoldersService_GetTopHolders_TokenRepoError(t *testing.T) {
	service, _, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database connection failed")
	}

	_, err := service.GetTopHolders(ctx, testutil.USDTAddress, 100, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to check token: database connection failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHoldersService_GetTopHolders_TransferRepoError(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 100, nil
	}

	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		return nil, errors.New("query timeout")
	}

	_, err := service.GetTopHolders(ctx, testutil.USDTAddress, 100, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to get top holders: query timeout" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHoldersService_GetTopHolders_Pagination(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 150, nil
	}

	// Track offset passed to repo
	var capturedOffset int
	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		capturedOffset = offset
		return []repositories.HolderBalance{
			{Address: "0x1111111111111111111111111111111111111111", Balance: "1000", Rank: offset + 1},
		}, nil
	}

	// Test with offset 100
	response, err := service.GetTopHolders(ctx, testutil.USDTAddress, 50, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedOffset != 100 {
		t.Errorf("expected offset 100, got %d", capturedOffset)
	}

	if response.Pagination.Offset != 100 {
		t.Errorf("expected pagination offset 100, got %d", response.Pagination.Offset)
	}

	if response.Pagination.HasMore {
		t.Error("expected has_more to be false (100 + 50 >= 150)")
	}

	// Test with offset 0
	response, err = service.GetTopHolders(ctx, testutil.USDTAddress, 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !response.Pagination.HasMore {
		t.Error("expected has_more to be true (0 + 50 < 150)")
	}
}

func TestHoldersService_GetTopHolders_HolderCountError(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count error
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 0, errors.New("count query failed")
	}

	_, err := service.GetTopHolders(ctx, testutil.USDTAddress, 100, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to get holder count: count query failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHoldersService_GetHolderBalance_Success(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithSymbol("USDT"),
	))

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	// Setup mock holder balance response
	transferRepo.GetHolderBalanceFunc = func(ctx context.Context, tokenAddr, holderAddr string) (*repositories.HolderBalance, error) {
		return &repositories.HolderBalance{
			Address: holderAddr,
			Balance: "999999999999999999999",
			Rank:    1,
		}, nil
	}

	response, err := service.GetHolderBalance(ctx, testutil.USDTAddress, holderAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}

	holder := response.Data
	if holder.Address != holderAddress {
		t.Errorf("expected address %s, got %s", holderAddress, holder.Address)
	}
	if holder.Balance != "999999999999999999999" {
		t.Errorf("expected balance '999999999999999999999', got %s", holder.Balance)
	}
	if holder.Rank != 1 {
		t.Errorf("expected rank 1, got %d", holder.Rank)
	}
}

func TestHoldersService_GetHolderBalance_TokenNotFound(t *testing.T) {
	service, _, _ := setupHoldersServiceTest()
	ctx := context.Background()

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	response, err := service.GetHolderBalance(ctx, testutil.USDTAddress, holderAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response != nil {
		t.Error("expected nil response for non-existent token")
	}
}

func TestHoldersService_GetHolderBalance_ZeroBalance(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	// Setup mock holder balance with zero balance
	transferRepo.GetHolderBalanceFunc = func(ctx context.Context, tokenAddr, holderAddr string) (*repositories.HolderBalance, error) {
		return &repositories.HolderBalance{
			Address: holderAddr,
			Balance: "0",
			Rank:    0,
		}, nil
	}

	response, err := service.GetHolderBalance(ctx, testutil.USDTAddress, holderAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}

	if response.Data.Balance != "0" {
		t.Errorf("expected balance '0', got %s", response.Data.Balance)
	}
}

func TestHoldersService_GetHolderBalance_Lowercase(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token with lowercase address
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Track which addresses were queried
	var queriedTokenAddr, queriedHolderAddr string
	transferRepo.GetHolderBalanceFunc = func(ctx context.Context, tokenAddr, holderAddr string) (*repositories.HolderBalance, error) {
		queriedTokenAddr = tokenAddr
		queriedHolderAddr = holderAddr
		return &repositories.HolderBalance{
			Address: holderAddr,
			Balance: "1000",
			Rank:    1,
		}, nil
	}

	// Use uppercase addresses
	upperToken := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	upperHolder := "0x47AC0FB4F2D84898E4D9E7B4DAB3C24507A6D503"
	_, err := service.GetHolderBalance(ctx, upperToken, upperHolder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if queriedTokenAddr != testutil.USDTAddress {
		t.Errorf("expected lowercase token address %s, got %s", testutil.USDTAddress, queriedTokenAddr)
	}
	if queriedHolderAddr != "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503" {
		t.Errorf("expected lowercase holder address, got %s", queriedHolderAddr)
	}
}

func TestHoldersService_GetHolderBalance_TokenRepoError(t *testing.T) {
	service, _, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database connection failed")
	}

	_, err := service.GetHolderBalance(ctx, testutil.USDTAddress, holderAddress)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to check token: database connection failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHoldersService_GetHolderBalance_TransferRepoError(t *testing.T) {
	service, transferRepo, tokenRepo := setupHoldersServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	transferRepo.GetHolderBalanceFunc = func(ctx context.Context, tokenAddr, holderAddr string) (*repositories.HolderBalance, error) {
		return nil, errors.New("query timeout")
	}

	_, err := service.GetHolderBalance(ctx, testutil.USDTAddress, holderAddress)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to get holder balance: query timeout" {
		t.Errorf("unexpected error message: %v", err)
	}
}
