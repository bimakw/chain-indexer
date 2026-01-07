package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/application/services"
	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func setupStatsHandlerTest() (*StatsHandler, *testutil.MockTransferRepository, *testutil.MockTokenRepository) {
	transferRepo := testutil.NewMockTransferRepository()
	tokenRepo := testutil.NewMockTokenRepository()
	logger := zap.NewNop()

	service := services.NewStatsService(transferRepo, tokenRepo, nil, logger)
	handler := NewStatsHandler(service, logger)

	return handler, transferRepo, tokenRepo
}

func TestNewStatsHandler(t *testing.T) {
	handler, _, _ := setupStatsHandlerTest()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestStatsHandler_GetTokenStats_Success(t *testing.T) {
	handler, transferRepo, tokenRepo := setupStatsHandlerTest()

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

	r := chi.NewRouter()
	r.Get("/tokens/{address}/stats", handler.GetTokenStats)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/stats", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TokenStatsResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
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
	if stats.FirstTransferAt != "2024-01-15T10:30:00Z" {
		t.Errorf("expected first transfer at '2024-01-15T10:30:00Z', got %s", stats.FirstTransferAt)
	}
	if stats.LastTransferAt != "2024-06-20T15:45:00Z" {
		t.Errorf("expected last transfer at '2024-06-20T15:45:00Z', got %s", stats.LastTransferAt)
	}
}

func TestStatsHandler_GetTokenStats_NotFound(t *testing.T) {
	handler, _, _ := setupStatsHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}/stats", handler.GetTokenStats)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/stats", nil)
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

func TestStatsHandler_GetTokenStats_InvalidAddress(t *testing.T) {
	handler, _, _ := setupStatsHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}/stats", handler.GetTokenStats)

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
			req := httptest.NewRequest(http.MethodGet, "/tokens/"+tt.address+"/stats", nil)
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

func TestStatsHandler_GetTokenStats_UppercaseAddress(t *testing.T) {
	handler, transferRepo, tokenRepo := setupStatsHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	transferRepo.GetTokenStatsFunc = func(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error) {
		return &repositories.TokenStatsResult{
			TotalTransfers: 100,
			TotalVolume:    "1000",
			Volume24h:      "0",
			Volume7d:       "0",
		}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/stats", handler.GetTokenStats)

	// Use uppercase address
	upperAddr := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	req := httptest.NewRequest(http.MethodGet, "/tokens/"+upperAddr+"/stats", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TokenStatsResponse
	json.NewDecoder(rec.Body).Decode(&response)

	// Should return lowercase address
	if response.Data.TokenAddress != testutil.USDTAddress {
		t.Errorf("expected lowercase address %s, got %s", testutil.USDTAddress, response.Data.TokenAddress)
	}
}

func TestStatsHandler_GetTokenStats_ServiceError(t *testing.T) {
	handler, _, tokenRepo := setupStatsHandlerTest()

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database error")
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/stats", handler.GetTokenStats)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/stats", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Failed to get token stats" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestStatsHandler_GetTokenStats_NoTransfers(t *testing.T) {
	handler, transferRepo, tokenRepo := setupStatsHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

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

	r := chi.NewRouter()
	r.Get("/tokens/{address}/stats", handler.GetTokenStats)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/stats", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TokenStatsResponse
	json.NewDecoder(rec.Body).Decode(&response)

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

func TestStatsHandler_ResponseContentType(t *testing.T) {
	handler, transferRepo, tokenRepo := setupStatsHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	transferRepo.GetTokenStatsFunc = func(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error) {
		return &repositories.TokenStatsResult{
			TotalTransfers: 100,
			TotalVolume:    "1000",
			Volume24h:      "0",
			Volume7d:       "0",
		}, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/stats", handler.GetTokenStats)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/stats", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestStatsHandler_GetHolderCount_Success(t *testing.T) {
	handler, transferRepo, tokenRepo := setupStatsHandlerTest()

	// Add token
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithSymbol("USDT"),
	))

	// Setup mock holder count response
	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 4523891, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holder-count", handler.GetHolderCount)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holder-count", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.HolderCountResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data := response.Data
	if data.TokenAddress != testutil.USDTAddress {
		t.Errorf("expected token address %s, got %s", testutil.USDTAddress, data.TokenAddress)
	}
	if data.HolderCount != 4523891 {
		t.Errorf("expected holder count 4523891, got %d", data.HolderCount)
	}
}

func TestStatsHandler_GetHolderCount_NotFound(t *testing.T) {
	handler, _, _ := setupStatsHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holder-count", handler.GetHolderCount)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holder-count", nil)
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

func TestStatsHandler_GetHolderCount_InvalidAddress(t *testing.T) {
	handler, _, _ := setupStatsHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holder-count", handler.GetHolderCount)

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
			req := httptest.NewRequest(http.MethodGet, "/tokens/"+tt.address+"/holder-count", nil)
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

func TestStatsHandler_GetHolderCount_UppercaseAddress(t *testing.T) {
	handler, transferRepo, tokenRepo := setupStatsHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	transferRepo.GetHolderCountFunc = func(ctx context.Context, tokenAddress string) (int64, error) {
		return 1000, nil
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holder-count", handler.GetHolderCount)

	// Use uppercase address
	upperAddr := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	req := httptest.NewRequest(http.MethodGet, "/tokens/"+upperAddr+"/holder-count", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.HolderCountResponse
	json.NewDecoder(rec.Body).Decode(&response)

	// Should return lowercase address
	if response.Data.TokenAddress != testutil.USDTAddress {
		t.Errorf("expected lowercase address %s, got %s", testutil.USDTAddress, response.Data.TokenAddress)
	}
}

func TestStatsHandler_GetHolderCount_ServiceError(t *testing.T) {
	handler, _, tokenRepo := setupStatsHandlerTest()

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database error")
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}/holder-count", handler.GetHolderCount)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/holder-count", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Failed to get holder count" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}
