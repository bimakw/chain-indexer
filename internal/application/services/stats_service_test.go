package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func setupStatsServiceTest() (*StatsService, *testutil.MockTransferRepository, *testutil.MockTokenRepository) {
	transferRepo := testutil.NewMockTransferRepository()
	tokenRepo := testutil.NewMockTokenRepository()
	logger := zap.NewNop()

	service := NewStatsService(transferRepo, tokenRepo, nil, logger)
	return service, transferRepo, tokenRepo
}

func TestNewStatsService(t *testing.T) {
	service, _, _ := setupStatsServiceTest()
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestStatsService_GetTokenStats_Success(t *testing.T) {
	service, transferRepo, tokenRepo := setupStatsServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithSymbol("USDT"),
	))

	// Setup mock stats response
	firstTransfer := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	lastTransfer := time.Date(2024, 6, 20, 15, 45, 0, 0, time.UTC)
	transferRepo.GetTokenStatsFunc = func(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error) {
		return &repositories.TokenStatsResult{
			TotalTransfers:  1234567,
			UniqueFromAddrs: 50000,
			UniqueToAddrs:   75000,
			TotalVolume:     "999999999999999999999",
			Transfers24h:    5000,
			Volume24h:       "1000000000000",
			Transfers7d:     35000,
			Volume7d:        "7000000000000",
			FirstTransferAt: &firstTransfer,
			LastTransferAt:  &lastTransfer,
		}, nil
	}

	response, err := service.GetTokenStats(ctx, testutil.USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}

	stats := response.Data
	if stats.TokenAddress != testutil.USDTAddress {
		t.Errorf("expected token address %s, got %s", testutil.USDTAddress, stats.TokenAddress)
	}
	if stats.TotalTransfers != 1234567 {
		t.Errorf("expected total transfers 1234567, got %d", stats.TotalTransfers)
	}
	if stats.UniqueFromAddresses != 50000 {
		t.Errorf("expected unique from addresses 50000, got %d", stats.UniqueFromAddresses)
	}
	if stats.UniqueToAddresses != 75000 {
		t.Errorf("expected unique to addresses 75000, got %d", stats.UniqueToAddresses)
	}
	if stats.TotalVolume != "999999999999999999999" {
		t.Errorf("expected total volume '999999999999999999999', got %s", stats.TotalVolume)
	}
	if stats.Transfers24h != 5000 {
		t.Errorf("expected transfers 24h 5000, got %d", stats.Transfers24h)
	}
	if stats.Volume24h != "1000000000000" {
		t.Errorf("expected volume 24h '1000000000000', got %s", stats.Volume24h)
	}
	if stats.Transfers7d != 35000 {
		t.Errorf("expected transfers 7d 35000, got %d", stats.Transfers7d)
	}
	if stats.Volume7d != "7000000000000" {
		t.Errorf("expected volume 7d '7000000000000', got %s", stats.Volume7d)
	}
	if stats.FirstTransferAt != "2024-01-15T10:30:00Z" {
		t.Errorf("expected first transfer at '2024-01-15T10:30:00Z', got %s", stats.FirstTransferAt)
	}
	if stats.LastTransferAt != "2024-06-20T15:45:00Z" {
		t.Errorf("expected last transfer at '2024-06-20T15:45:00Z', got %s", stats.LastTransferAt)
	}
}

func TestStatsService_GetTokenStats_TokenNotFound(t *testing.T) {
	service, _, _ := setupStatsServiceTest()
	ctx := context.Background()

	response, err := service.GetTokenStats(ctx, testutil.USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response != nil {
		t.Error("expected nil response for non-existent token")
	}
}

func TestStatsService_GetTokenStats_NoTransfers(t *testing.T) {
	service, transferRepo, tokenRepo := setupStatsServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock stats response for token with no transfers
	transferRepo.GetTokenStatsFunc = func(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error) {
		return &repositories.TokenStatsResult{
			TotalTransfers:  0,
			UniqueFromAddrs: 0,
			UniqueToAddrs:   0,
			TotalVolume:     "0",
			Transfers24h:    0,
			Volume24h:       "0",
			Transfers7d:     0,
			Volume7d:        "0",
			FirstTransferAt: nil,
			LastTransferAt:  nil,
		}, nil
	}

	response, err := service.GetTokenStats(ctx, testutil.USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}

	stats := response.Data
	if stats.TotalTransfers != 0 {
		t.Errorf("expected 0 transfers, got %d", stats.TotalTransfers)
	}
	if stats.FirstTransferAt != "" {
		t.Errorf("expected empty first transfer at, got %s", stats.FirstTransferAt)
	}
	if stats.LastTransferAt != "" {
		t.Errorf("expected empty last transfer at, got %s", stats.LastTransferAt)
	}
}

func TestStatsService_GetTokenStats_Lowercase(t *testing.T) {
	service, transferRepo, tokenRepo := setupStatsServiceTest()
	ctx := context.Background()

	// Add token with lowercase address
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Track which address was queried
	var queriedAddress string
	transferRepo.GetTokenStatsFunc = func(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error) {
		queriedAddress = tokenAddress
		return &repositories.TokenStatsResult{
			TotalTransfers: 100,
			TotalVolume:    "1000",
			Volume24h:      "0",
			Volume7d:       "0",
		}, nil
	}

	// Use uppercase address
	upperAddr := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	_, err := service.GetTokenStats(ctx, upperAddr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if queriedAddress != testutil.USDTAddress {
		t.Errorf("expected lowercase address %s, got %s", testutil.USDTAddress, queriedAddress)
	}
}

func TestStatsService_GetTokenStats_TokenRepoError(t *testing.T) {
	service, _, tokenRepo := setupStatsServiceTest()
	ctx := context.Background()

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database connection failed")
	}

	_, err := service.GetTokenStats(ctx, testutil.USDTAddress)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to check token: database connection failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestStatsService_GetTokenStats_TransferRepoError(t *testing.T) {
	service, transferRepo, tokenRepo := setupStatsServiceTest()
	ctx := context.Background()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	transferRepo.GetTokenStatsFunc = func(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error) {
		return nil, errors.New("query timeout")
	}

	_, err := service.GetTokenStats(ctx, testutil.USDTAddress)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to get token stats: query timeout" {
		t.Errorf("unexpected error message: %v", err)
	}
}
