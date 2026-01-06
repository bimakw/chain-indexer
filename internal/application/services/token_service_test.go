package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func setupTokenServiceTest() (*TokenService, *testutil.MockTokenRepository) {
	tokenRepo := testutil.NewMockTokenRepository()
	logger := zap.NewNop()

	service := NewTokenService(tokenRepo, nil, logger)
	return service, tokenRepo
}

func TestNewTokenService(t *testing.T) {
	service, _ := setupTokenServiceTest()
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestTokenService_GetAllTokens_Success(t *testing.T) {
	service, tokenRepo := setupTokenServiceTest()
	ctx := context.Background()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithSymbol("USDT"),
		testutil.TokenWithTotalTransfers(1000),
	))
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDCAddress),
		testutil.TokenWithSymbol("USDC"),
		testutil.TokenWithTotalTransfers(500),
	))

	response, err := service.GetAllTokens(ctx, 100, 0, "total_indexed_transfers", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Pagination.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Pagination.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(response.Data))
	}
	if response.Pagination.Limit != 100 {
		t.Errorf("expected limit 100, got %d", response.Pagination.Limit)
	}
	if response.Pagination.Offset != 0 {
		t.Errorf("expected offset 0, got %d", response.Pagination.Offset)
	}
}

func TestTokenService_GetAllTokens_Pagination(t *testing.T) {
	service, tokenRepo := setupTokenServiceTest()
	ctx := context.Background()

	// Add 5 tokens
	for i := 0; i < 5; i++ {
		addr := "0x" + string(rune('a'+i)) + "000000000000000000000000000000000000000"
		tokenRepo.AddToken(testutil.CreateTestToken(
			testutil.TokenWithAddress(addr),
			testutil.TokenWithSymbol("TKN"+string(rune('A'+i))),
		))
	}

	// First page
	response, err := service.GetAllTokens(ctx, 2, 0, "symbol", "asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Pagination.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Pagination.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(response.Data))
	}

	// Second page
	response, err = service.GetAllTokens(ctx, 2, 2, "symbol", "asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Data) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(response.Data))
	}

	// Last page
	response, err = service.GetAllTokens(ctx, 2, 4, "symbol", "asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Data) != 1 {
		t.Errorf("expected 1 token, got %d", len(response.Data))
	}
}

func TestTokenService_GetAllTokens_EmptyResult(t *testing.T) {
	service, _ := setupTokenServiceTest()
	ctx := context.Background()

	response, err := service.GetAllTokens(ctx, 100, 0, "symbol", "asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Pagination.Total != 0 {
		t.Errorf("expected total 0, got %d", response.Pagination.Total)
	}
	if len(response.Data) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(response.Data))
	}
}

func TestTokenService_GetAllTokens_RepositoryError(t *testing.T) {
	service, tokenRepo := setupTokenServiceTest()
	ctx := context.Background()

	tokenRepo.GetAllPaginatedFunc = func(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]*entities.Token, int64, error) {
		return nil, 0, errors.New("database connection failed")
	}

	_, err := service.GetAllTokens(ctx, 100, 0, "symbol", "asc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to get tokens: database connection failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTokenService_GetByAddress_Success(t *testing.T) {
	service, tokenRepo := setupTokenServiceTest()
	ctx := context.Background()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithName("Tether USD"),
		testutil.TokenWithSymbol("USDT"),
		testutil.TokenWithDecimals(6),
		testutil.TokenWithTotalTransfers(12345),
	))

	response, err := service.GetByAddress(ctx, testutil.USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if response.Data.Address != testutil.USDTAddress {
		t.Errorf("expected address %s, got %s", testutil.USDTAddress, response.Data.Address)
	}
	if response.Data.Name != "Tether USD" {
		t.Errorf("expected name 'Tether USD', got %s", response.Data.Name)
	}
	if response.Data.Symbol != "USDT" {
		t.Errorf("expected symbol 'USDT', got %s", response.Data.Symbol)
	}
	if response.Data.Decimals != 6 {
		t.Errorf("expected decimals 6, got %d", response.Data.Decimals)
	}
	if response.Data.TotalIndexedTransfers != 12345 {
		t.Errorf("expected total transfers 12345, got %d", response.Data.TotalIndexedTransfers)
	}
}

func TestTokenService_GetByAddress_NotFound(t *testing.T) {
	service, _ := setupTokenServiceTest()
	ctx := context.Background()

	response, err := service.GetByAddress(ctx, testutil.USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response != nil {
		t.Error("expected nil response for non-existent token")
	}
}

func TestTokenService_GetByAddress_Lowercase(t *testing.T) {
	service, tokenRepo := setupTokenServiceTest()
	ctx := context.Background()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Use uppercase address
	upperAddr := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	response, err := service.GetByAddress(ctx, upperAddr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if response.Data.Address != testutil.USDTAddress {
		t.Errorf("expected lowercase address %s, got %s", testutil.USDTAddress, response.Data.Address)
	}
}

func TestTokenService_GetByAddress_RepositoryError(t *testing.T) {
	service, tokenRepo := setupTokenServiceTest()
	ctx := context.Background()

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database error")
	}

	_, err := service.GetByAddress(ctx, testutil.USDTAddress)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "failed to get token: database error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTokenDTO_Formatting(t *testing.T) {
	service, tokenRepo := setupTokenServiceTest()
	ctx := context.Background()

	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 20, 15, 45, 0, 0, time.UTC)
	firstSeenBlock := int64(4634748)
	lastSeenBlock := int64(19500000)

	token := &entities.Token{
		Address:               testutil.USDTAddress,
		Name:                  "Tether USD",
		Symbol:                "USDT",
		Decimals:              6,
		TotalIndexedTransfers: 1234567,
		FirstSeenBlock:        &firstSeenBlock,
		LastSeenBlock:         &lastSeenBlock,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	}
	tokenRepo.AddToken(token)

	response, err := service.GetByAddress(ctx, testutil.USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dto := response.Data
	if dto.Address != testutil.USDTAddress {
		t.Errorf("Address mismatch: %s", dto.Address)
	}
	if dto.Name != "Tether USD" {
		t.Errorf("Name mismatch: %s", dto.Name)
	}
	if dto.Symbol != "USDT" {
		t.Errorf("Symbol mismatch: %s", dto.Symbol)
	}
	if dto.Decimals != 6 {
		t.Errorf("Decimals mismatch: %d", dto.Decimals)
	}
	if dto.TotalIndexedTransfers != 1234567 {
		t.Errorf("TotalIndexedTransfers mismatch: %d", dto.TotalIndexedTransfers)
	}
	if *dto.FirstSeenBlock != 4634748 {
		t.Errorf("FirstSeenBlock mismatch: %d", *dto.FirstSeenBlock)
	}
	if *dto.LastSeenBlock != 19500000 {
		t.Errorf("LastSeenBlock mismatch: %d", *dto.LastSeenBlock)
	}
	if dto.CreatedAt != "2024-01-15T10:30:00Z" {
		t.Errorf("CreatedAt mismatch: %s", dto.CreatedAt)
	}
	if dto.UpdatedAt != "2024-01-20T15:45:00Z" {
		t.Errorf("UpdatedAt mismatch: %s", dto.UpdatedAt)
	}
}

func TestTokenDTO_NilBlocks(t *testing.T) {
	service, tokenRepo := setupTokenServiceTest()
	ctx := context.Background()

	token := &entities.Token{
		Address:               testutil.USDTAddress,
		Name:                  "Test Token",
		Symbol:                "TEST",
		Decimals:              18,
		TotalIndexedTransfers: 0,
		FirstSeenBlock:        nil,
		LastSeenBlock:         nil,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	tokenRepo.AddToken(token)

	response, err := service.GetByAddress(ctx, testutil.USDTAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dto := response.Data
	if dto.FirstSeenBlock != nil {
		t.Errorf("expected nil FirstSeenBlock, got %d", *dto.FirstSeenBlock)
	}
	if dto.LastSeenBlock != nil {
		t.Errorf("expected nil LastSeenBlock, got %d", *dto.LastSeenBlock)
	}
}
