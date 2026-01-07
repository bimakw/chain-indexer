package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/application/services"
	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func setupHoldersHandlerTest() (*HoldersHandler, *testutil.MockTransferRepository, *testutil.MockTokenRepository) {
	transferRepo := testutil.NewMockTransferRepository()
	tokenRepo := testutil.NewMockTokenRepository()
	logger := zap.NewNop()

	service := services.NewHoldersService(transferRepo, tokenRepo, nil, logger)
	handler := NewHoldersHandler(service, logger)

	return handler, transferRepo, tokenRepo
}

func TestNewHoldersHandler(t *testing.T) {
	handler, _, _ := setupHoldersHandlerTest()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHoldersHandler_GetTopHolders_Success(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

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

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TopHoldersResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Data) != 3 {
		t.Errorf("expected 3 holders, got %d", len(response.Data))
	}

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
	// With total=100, limit=100, offset=0: 0+100 >= 100, so hasMore should be false
	if response.Pagination.HasMore {
		t.Error("expected has_more to be false (0 + 100 >= 100)")
	}
}

func TestHoldersHandler_GetTopHolders_WithLimit(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 100, nil
	}

	var capturedLimit int
	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		capturedLimit = limit
		return []repositories.HolderBalance{
			{Address: "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503", Balance: "1000", Rank: 1},
		}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders?limit=50", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if capturedLimit != 50 {
		t.Errorf("expected limit 50, got %d", capturedLimit)
	}
}

func TestHoldersHandler_GetTopHolders_MaxLimit(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 100, nil
	}

	var capturedLimit int
	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		capturedLimit = limit
		return []repositories.HolderBalance{}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders?limit=5000", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if capturedLimit != 1000 {
		t.Errorf("expected max limit 1000, got %d", capturedLimit)
	}
}

func TestHoldersHandler_GetTopHolders_NotFound(t *testing.T) {
	handler, _, _ := setupHoldersHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "token not found" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestHoldersHandler_GetTopHolders_InvalidAddress(t *testing.T) {
	handler, _, _ := setupHoldersHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	tests := []struct {
		name    string
		address string
	}{
		{"too short", "0x1234"},
		{"no prefix", "1111111111111111111111111111111111111111"},
		{"wrong prefix", "1x1111111111111111111111111111111111111111"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tokens/"+tt.address+"/holders", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", rec.Code)
			}

			var response map[string]string
			json.NewDecoder(rec.Body).Decode(&response)
			if response["error"] != "Invalid address format" {
				t.Errorf("unexpected error: %s", response["error"])
			}
		})
	}
}

func TestHoldersHandler_GetTopHolders_UppercaseAddress(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 100, nil
	}

	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		return []repositories.HolderBalance{
			{Address: "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503", Balance: "1000", Rank: 1},
		}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	// Use uppercase address
	upperAddr := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	req := httptest.NewRequest(http.MethodGet, "/tokens/"+upperAddr+"/holders", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHoldersHandler_GetTopHolders_ServiceError(t *testing.T) {
	handler, _, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database error")
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Failed to get top holders" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestHoldersHandler_GetTopHolders_EmptyResult(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 0, nil
	}

	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		return []repositories.HolderBalance{}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TopHoldersResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 holders, got %d", len(response.Data))
	}
}

func TestHoldersHandler_GetTopHolders_WithOffset(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 500, nil
	}

	var capturedOffset int
	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		capturedOffset = offset
		return []repositories.HolderBalance{
			{Address: "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503", Balance: "1000", Rank: offset + 1},
		}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders?offset=200", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if capturedOffset != 200 {
		t.Errorf("expected offset 200, got %d", capturedOffset)
	}

	var response services.TopHoldersResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Pagination.Offset != 200 {
		t.Errorf("expected pagination offset 200, got %d", response.Pagination.Offset)
	}
	if response.Pagination.Total != 500 {
		t.Errorf("expected pagination total 500, got %d", response.Pagination.Total)
	}
	if !response.Pagination.HasMore {
		t.Error("expected has_more to be true (200 + 100 < 500)")
	}
}

func TestHoldersHandler_GetHolderBalance_Success(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithSymbol("USDT"),
	))

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	transferRepo.GetHolderBalanceFunc = func(ctx context.Context, tokenAddr, holderAddr string) (*repositories.HolderBalance, error) {
		return &repositories.HolderBalance{
			Address: holderAddr,
			Balance: "999999999999999999999",
			Rank:    1,
		}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders/{holder_address}", handler.GetHolderBalance)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders/"+holderAddress, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.HolderBalanceResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
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

func TestHoldersHandler_GetHolderBalance_TokenNotFound(t *testing.T) {
	handler, _, _ := setupHoldersHandlerTest()

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders/{holder_address}", handler.GetHolderBalance)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders/"+holderAddress, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "token not found" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestHoldersHandler_GetHolderBalance_InvalidTokenAddress(t *testing.T) {
	handler, _, _ := setupHoldersHandlerTest()

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders/{holder_address}", handler.GetHolderBalance)

	req := httptest.NewRequest(http.MethodGet, "/tokens/0x1234/holders/"+holderAddress, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Invalid token address format" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestHoldersHandler_GetHolderBalance_InvalidHolderAddress(t *testing.T) {
	handler, _, _ := setupHoldersHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders/{holder_address}", handler.GetHolderBalance)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders/0x1234", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Invalid holder address format" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestHoldersHandler_GetHolderBalance_UppercaseAddresses(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	var capturedTokenAddr, capturedHolderAddr string
	transferRepo.GetHolderBalanceFunc = func(ctx context.Context, tokenAddr, holderAddr string) (*repositories.HolderBalance, error) {
		capturedTokenAddr = tokenAddr
		capturedHolderAddr = holderAddr
		return &repositories.HolderBalance{
			Address: holderAddr,
			Balance: "1000",
			Rank:    1,
		}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders/{holder_address}", handler.GetHolderBalance)

	// Use uppercase addresses
	upperToken := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	upperHolder := "0x47AC0FB4F2D84898E4D9E7B4DAB3C24507A6D503"
	req := httptest.NewRequest(http.MethodGet, "/tokens/"+upperToken+"/holders/"+upperHolder, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if capturedTokenAddr != testutil.USDTAddress {
		t.Errorf("expected lowercase token address %s, got %s", testutil.USDTAddress, capturedTokenAddr)
	}
	if capturedHolderAddr != "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503" {
		t.Errorf("expected lowercase holder address, got %s", capturedHolderAddr)
	}
}

func TestHoldersHandler_GetHolderBalance_ServiceError(t *testing.T) {
	handler, _, tokenRepo := setupHoldersHandlerTest()

	holderAddress := "0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503"

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database error")
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders/{holder_address}", handler.GetHolderBalance)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders/"+holderAddress, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Failed to get holder balance" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestHoldersHandler_ResponseContentType(t *testing.T) {
	handler, transferRepo, tokenRepo := setupHoldersHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	// Setup mock holder count
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 0, nil
	}

	transferRepo.GetTopHoldersWithOffsetFunc = func(ctx context.Context, tokenAddress string, limit, offset int) ([]repositories.HolderBalance, error) {
		return []repositories.HolderBalance{}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holders", handler.GetTopHolders)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holders", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}
